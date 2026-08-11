package minimax

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
	serviceproject "github.com/futrx-com/remote.futrx.com/internal/service/project"
)

const (
	// apiKeyEnvVar is the indirection Codex uses to read the bearer token. The
	// provider block references it via `env_key`, so the secret reaches the CLI
	// through the process environment instead of being written to config.toml
	// (which MiniMax's own docs suggest via `experimental_bearer_token`).
	apiKeyEnvVar = "MINIMAX_API_KEY"

	baseURL = "https://api.minimax.io/v1"

	// DefaultModel is used when the conversation has no explicit model. Codex
	// would otherwise fall back to its own OpenAI default, which the MiniMax
	// endpoint does not serve.
	DefaultModel = "MiniMax-M3"

	// contextWindow matches the window MiniMax documents for M3. Codex needs it
	// declared because it cannot infer the window for a non-OpenAI model.
	contextWindow = "1000000"
)

func (p *Provider) args(req agent.RunRequest) []string {
	common := []string{
		"--json",
		"--skip-git-repo-check",
		"--dangerously-bypass-approvals-and-sandbox",
	}
	common = append(common, providerConfigArgs()...)

	model := sanitizeModel(req.Model)
	if model == "" {
		model = DefaultModel
	}
	common = append(common, "--model", model)

	if effort := reasoningEffortArg(req.Preferences.ReasoningEffort); effort != "" {
		common = append(common, "-c", "model_reasoning_effort="+effort)
	}
	if req.EnableBrowser {
		common = append(common, browserMCPConfigArgs()...)
	}
	if req.ResumeID != "" {
		args := append([]string{"exec", "resume"}, common...)
		args = append(args, req.ResumeID, "-")
		return args
	}
	args := append([]string{"exec"}, common...)
	args = append(args, "-")
	return args
}

// providerConfigArgs declares the MiniMax model provider inline for this one
// invocation. Passing it as `-c` overrides rather than writing ~/.codex/
// config.toml keeps the host's Codex configuration untouched and means the two
// agents cannot drift out of sync with a stale file.
func providerConfigArgs() []string {
	return []string{
		"-c", `model_provider="minimax"`,
		"-c", `model_providers.minimax.name="MiniMax"`,
		"-c", `model_providers.minimax.base_url="` + baseURL + `"`,
		"-c", `model_providers.minimax.wire_api="responses"`,
		"-c", `model_providers.minimax.env_key="` + apiKeyEnvVar + `"`,
		"-c", "model_context_window=" + contextWindow,
	}
}

func browserMCPConfigArgs() []string {
	return []string{
		"-c", `mcp_servers.browser.command="npx"`,
		"-c", `mcp_servers.browser.args=["@playwright/mcp","--cdp-endpoint","http://127.0.0.1:9222","--caps=vision"]`,
	}
}

func sanitizeModel(model string) string {
	model = strings.TrimSpace(model)
	if idx := strings.Index(model, "["); idx > 0 {
		model = strings.TrimSpace(model[:idx])
	}
	return model
}

// reasoningEffortArg maps our ladder onto what MiniMax actually supports.
// MiniMax M3 exposes only two states — Adaptive Thinking off or on — so the
// richer Codex ladder is collapsed rather than forwarded verbatim.
func reasoningEffortArg(effort agent.ReasoningEffort) string {
	switch strings.ToLower(strings.TrimSpace(string(effort))) {
	case "none", "minimal":
		return "none"
	case "low", "medium", "high", "xhigh", "max", "ultra":
		return "high"
	default:
		return ""
	}
}

func (p *Provider) buildCmd(
	ctx context.Context,
	req agent.RunRequest,
	args []string,
	emit func(agent.Event),
) (*exec.Cmd, string, error) {
	apiKey, err := p.auth.Key()
	if err != nil {
		return nil, "", fmt.Errorf("read MiniMax API key: %w", err)
	}
	if apiKey == "" {
		return nil, "", ErrNoAPIKey
	}

	cwd := req.Cwd
	if cwd == "" {
		cwd = os.Getenv("HOME")
		if cwd == "" {
			cwd = "/root"
		}
	}

	if req.ProjectID == "" || p.projects == nil {
		cmd := exec.CommandContext(ctx, "codex", args...)
		cmd.Dir = cwd
		cmd.Env = agent.WithRuntimeEnvironment(hostEnv(os.Environ(), apiKey), req.RuntimeEnv)
		cmd.Stdin = strings.NewReader(req.Prompt)
		return cmd, "", nil
	}

	project, err := p.projects.Get(ctx, serviceproject.ID(req.ProjectID))
	if err != nil {
		return nil, "", fmt.Errorf("project not found (%s): %w", req.ProjectID, err)
	}
	if project.ContainerName == "" {
		return nil, "", fmt.Errorf("project %s has no container - recreate the project", project.ID)
	}

	// The container can be deleted out-of-band (e.g. a workspace recycle onto a
	// new base image), leaving the cached Status stale. Always reconcile via
	// Start — it relaunches a missing instance from the base image and is a
	// no-op when already running; the cached Status only gates the indicator.
	if project.Status != serviceproject.StatusRunning {
		emitSystem(req, emit, "container_starting")
	}
	if _, err := p.projects.Start(ctx, project.ID); err != nil {
		return nil, "", fmt.Errorf("start container: %w", err)
	}

	if err := p.containerDeps.Validate(); err != nil {
		return nil, "", err
	}
	if !p.containerDeps.IsZero() {
		emitSystem(req, emit, "container_preparing")
		// MiniMax's own profile declares no CLI, so ensure the codex binary
		// through the codex spec. This is a no-op once installed and keeps
		// MiniMax runnable even in a container built before codex was pinned.
		if err := p.containerDeps.CLI.Ensure(ctx, project.ContainerName, codexCLI()); err != nil {
			return nil, "", fmt.Errorf("codex CLI unavailable in container: %w", err)
		}
		// No credential seeding: the API key travels as an environment
		// variable below and is never persisted inside the container.
		if err := p.containerDeps.Workspace.EnsureAgentInstructions(ctx, project.ContainerName); err != nil {
			return nil, "", fmt.Errorf("push agent instructions to container: %w", err)
		}
		if err := p.containerDeps.Workspace.EnsureSkillLinks(ctx, project.ContainerName); err != nil {
			return nil, "", fmt.Errorf("prepare workspace skill links: %w", err)
		}
		if err := p.containerDeps.Browser.EnsureSkill(ctx, project.ContainerName); err != nil {
			// Best-effort migration for containers created before the skill.
			_ = err
		}
		if err := p.containerDeps.Browser.EnsureScript(ctx, project.ContainerName); err != nil {
			// Browser script provisioning is best-effort: its absence only
			// matters when the agent tries to run scripts/browser.mjs.
			_ = err
		}
		if req.EnableBrowser {
			if err := p.containerDeps.Browser.EnsureMCP(ctx, project.ContainerName); err != nil {
				return nil, "", fmt.Errorf("provision browser MCP: %w", err)
			}
			if err := p.containerDeps.Browser.EnsureCore(ctx, project.ContainerName); err != nil {
				return nil, "", fmt.Errorf("start browser core: %w", err)
			}
		}
		if req.EnableScheduleTools {
			if err := p.containerDeps.ScheduleTools.Ensure(ctx, project.ContainerName); err != nil {
				return nil, "", fmt.Errorf("provision scheduled-task tools: %w", err)
			}
		}
		if err := p.containerDeps.Lifecycle.EnsureBootAutostart(ctx, project.ContainerName); err != nil {
			return nil, "", fmt.Errorf("set container boot.autostart: %w", err)
		}
	}

	lxcArgs := []string{
		"exec",
		"--cwd", "/workspace",
		"--env", "HOME=/root",
		"--env", "CODEX_HOME=" + containerMiniMaxHome,
	}
	if p.projects != nil {
		if secrets, err := p.projects.ListSecrets(ctx, project.ID); err == nil {
			for _, sec := range secrets {
				// The stored MiniMax credential and Codex's ChatGPT auth both
				// win over a project secret of the same name.
				if sec.Key == apiKeyEnvVar || sec.Key == "OPENAI_API_KEY" {
					continue
				}
				if _, backendIssued := req.RuntimeEnv[sec.Key]; backendIssued {
					continue
				}
				lxcArgs = append(lxcArgs, "--env", sec.Key+"="+sec.Value)
			}
		}
	}
	// Blank OPENAI_API_KEY so a stray host or project value cannot be picked up
	// for the MiniMax provider block, and pass the MiniMax key env_key reads.
	lxcArgs = append(lxcArgs, "--env", "OPENAI_API_KEY=")
	lxcArgs = append(lxcArgs, "--env", apiKeyEnvVar+"="+apiKey)
	for _, entry := range agent.RuntimeEnvironment(req.RuntimeEnv) {
		lxcArgs = append(lxcArgs, "--env", entry)
	}
	lxcArgs = append(lxcArgs, project.ContainerName, "--", "codex")
	lxcArgs = append(lxcArgs, args...)
	cmd := exec.CommandContext(ctx, "lxc", lxcArgs...)
	cmd.Stdin = strings.NewReader(req.Prompt)
	return cmd, project.ContainerName, nil
}

// hostEnv builds the environment for a host-side run: MiniMax's own CODEX_HOME,
// its API key, and no inherited OPENAI_API_KEY.
func hostEnv(base []string, apiKey string) []string {
	out := make([]string, 0, len(base)+3)
	for _, env := range base {
		if strings.HasPrefix(env, "OPENAI_API_KEY=") ||
			strings.HasPrefix(env, "CODEX_HOME=") ||
			strings.HasPrefix(env, apiKeyEnvVar+"=") {
			continue
		}
		out = append(out, env)
	}
	out = append(out, "OPENAI_API_KEY=")
	out = append(out, "CODEX_HOME="+hostMiniMaxHome())
	out = append(out, apiKeyEnvVar+"="+apiKey)
	return out
}

// hostSessionsDir is where a host-side run keeps its rollouts.
func hostSessionsDir() string {
	return filepath.Join(hostMiniMaxHome(), "sessions")
}

func emitSystem(req agent.RunRequest, emit func(agent.Event), subtype string) {
	emit(agent.Event{
		T:              time.Now().UnixMilli(),
		Type:           agent.EventSystem,
		Provider:       agent.ProviderMiniMax,
		ConversationID: req.ConversationID,
		Subtype:        subtype,
	})
}
