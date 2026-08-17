package opencode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
	serviceproject "github.com/futrx-com/remote.futrx.com/internal/service/project"
)

func (p *Provider) args(req agent.RunRequest) []string {
	// opencode run: non-interactive. --format json streams NDJSON events;
	// --auto auto-approves tool permissions (the container is single-user and
	// isolated, and without it opencode auto-rejects edits). The prompt is a
	// positional argument.
	args := []string{"run", "--format", "json", "--auto"}
	if model := sanitizeModel(req.Model); model != "" {
		args = append(args, "--model", model)
	}
	// opencode maps the reasoning-effort selection onto the model's variant
	// (--variant). An effort the model doesn't advertise is silently ignored
	// by opencode, so unsupported pairs fall back to the model default rather
	// than erroring. --thinking is display-only for the default (non-JSON)
	// format; with --format json the reasoning parts stream through anyway.
	if variant := variantArg(req.Preferences.ReasoningEffort); variant != "" {
		args = append(args, "--variant", variant)
	}
	if req.ResumeID != "" {
		args = append(args, "--session", req.ResumeID)
		// opencode has a real fork primitive; a forked chat resumes a copy.
		if req.Fork {
			args = append(args, "--fork")
		}
	}
	// opencode requires --continue/--session when --fork is passed; a fork of
	// a fresh chat is just a fresh chat.
	args = append(args, req.Prompt)
	return args
}

func sanitizeModel(model string) string {
	model = strings.TrimSpace(model)
	if idx := strings.Index(model, "["); idx > 0 {
		model = strings.TrimSpace(model[:idx])
	}
	return model
}

// variantArg maps the conversation's reasoning effort onto opencode's
// provider-specific --variant values. opencode passes the value through to
// the model's variants; unknown/unsupported efforts are ignored by opencode,
// so omitting here (Auto, ultra, and anything unrecognized) just lets the
// model pick its default.
func variantArg(effort agent.ReasoningEffort) string {
	switch strings.ToLower(strings.TrimSpace(string(effort))) {
	case "none":
		return "none"
	case "minimal":
		return "minimal"
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high":
		return "high"
	case "xhigh":
		return "xhigh"
	case "max":
		return "max"
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
	cwd := req.Cwd
	if cwd == "" {
		cwd = os.Getenv("HOME")
		if cwd == "" {
			cwd = "/root"
		}
	}

	if req.ProjectID == "" || p.projects == nil {
		cmd := exec.CommandContext(ctx, "opencode", args...)
		cmd.Dir = cwd
		cmd.Env = agent.WithRuntimeEnvironment(cmd.Env, req.RuntimeEnv)
		return cmd, "", nil
	}

	project, err := p.projects.Get(ctx, serviceproject.ID(req.ProjectID))
	if err != nil {
		return nil, "", fmt.Errorf("project not found (%s): %w", req.ProjectID, err)
	}
	if project.ContainerName == "" {
		return nil, "", fmt.Errorf("project %s has no container - recreate the project", project.ID)
	}

	// The container can be deleted out-of-band (e.g. a workspace recycle onto
	// a new base image), leaving the cached Status stale. Always reconcile via
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
		if err := p.containerDeps.CLI.Ensure(ctx, project.ContainerName, p.profile.CLI); err != nil {
			return nil, "", fmt.Errorf("install opencode in container: %w", err)
		}
		if err := p.containerDeps.Credentials.Ensure(ctx, project.ContainerName, p.profile.Credentials); err != nil {
			return nil, "", fmt.Errorf("seed opencode auth in container: %w", err)
		}
		if err := p.containerDeps.Workspace.EnsureAgentInstructions(ctx, project.ContainerName); err != nil {
			return nil, "", fmt.Errorf("push agent instructions to container: %w", err)
		}
		if err := p.containerDeps.Workspace.EnsureSkillLinks(ctx, project.ContainerName); err != nil {
			// Best-effort: a stale skill shim shouldn't block an opencode run.
			_ = err
		}
		if err := p.containerDeps.Browser.EnsureSkill(ctx, project.ContainerName); err != nil {
			// Best-effort migration for containers created before the skill.
			_ = err
		}
		if err := p.containerDeps.Browser.EnsureScript(ctx, project.ContainerName); err != nil {
			// Best-effort: only matters if the agent runs scripts/browser.mjs.
			_ = err
		}
		if req.EnableBrowser {
			// Pushes the browser MCP entry into opencode's global config and
			// starts the shared Chrome core, mirroring claude/codex. opencode
			// has no per-run MCP flag, so the template lands in the global
			// config file; it stays inert without the browser core running.
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
	}
	if p.projects != nil {
		if secrets, err := p.projects.ListSecrets(ctx, project.ID); err == nil {
			for _, sec := range secrets {
				if _, backendIssued := req.RuntimeEnv[sec.Key]; backendIssued {
					continue
				}
				lxcArgs = append(lxcArgs, "--env", sec.Key+"="+sec.Value)
			}
		}
	}
	for _, entry := range agent.RuntimeEnvironment(req.RuntimeEnv) {
		lxcArgs = append(lxcArgs, "--env", entry)
	}
	lxcArgs = append(lxcArgs, project.ContainerName, "--", "opencode")
	lxcArgs = append(lxcArgs, args...)
	cmd := exec.CommandContext(ctx, "lxc", lxcArgs...)
	return cmd, project.ContainerName, nil
}

func emitSystem(req agent.RunRequest, emit func(agent.Event), subtype string) {
	emit(agent.Event{
		T:              time.Now().UnixMilli(),
		Type:           agent.EventSystem,
		Provider:       agent.ProviderOpenCode,
		ConversationID: req.ConversationID,
		Subtype:        subtype,
	})
}
