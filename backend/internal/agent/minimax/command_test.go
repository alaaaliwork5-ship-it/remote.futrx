package minimax

import (
	"slices"
	"strings"
	"testing"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
)

func argValue(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// configValues collects every `-c key=value` override.
func configValues(args []string) map[string]string {
	out := map[string]string{}
	for i, arg := range args {
		if arg != "-c" || i+1 >= len(args) {
			continue
		}
		key, value, found := strings.Cut(args[i+1], "=")
		if found {
			out[key] = value
		}
	}
	return out
}

func TestArgsDeclareMiniMaxProvider(t *testing.T) {
	provider := &Provider{}
	args := provider.args(agent.RunRequest{})

	config := configValues(args)
	want := map[string]string{
		"model_provider":                   `"minimax"`,
		"model_providers.minimax.base_url": `"https://api.minimax.io/v1"`,
		"model_providers.minimax.wire_api": `"responses"`,
		"model_providers.minimax.env_key":  `"MINIMAX_API_KEY"`,
		"model_context_window":             contextWindow,
	}
	for key, expected := range want {
		if config[key] != expected {
			t.Errorf("config %s = %q, want %q", key, config[key], expected)
		}
	}

	// The key itself must never appear in an argument: it reaches the CLI only
	// through the environment that env_key names.
	for _, arg := range args {
		if strings.Contains(arg, "experimental_bearer_token") {
			t.Fatalf("args must not carry an inline bearer token: %q", arg)
		}
	}
}

func TestArgsDefaultModelWhenUnset(t *testing.T) {
	provider := &Provider{}

	// Codex would otherwise fall back to an OpenAI model the MiniMax endpoint
	// does not serve, so an empty request model must resolve to MiniMax's.
	if got := argValue(provider.args(agent.RunRequest{}), "--model"); got != DefaultModel {
		t.Fatalf("--model = %q, want %q", got, DefaultModel)
	}

	args := provider.args(agent.RunRequest{Model: "MiniMax-M3 [preview]"})
	if got := argValue(args, "--model"); got != "MiniMax-M3" {
		t.Fatalf("--model = %q, want %q", got, "MiniMax-M3")
	}
}

func TestArgsResumeUsesSessionID(t *testing.T) {
	provider := &Provider{}
	args := provider.args(agent.RunRequest{ResumeID: "session-1"})

	if !slices.Equal(args[:2], []string{"exec", "resume"}) {
		t.Fatalf("args = %v, want exec resume prefix", args[:2])
	}
	if args[len(args)-2] != "session-1" || args[len(args)-1] != "-" {
		t.Fatalf("args tail = %v, want [session-1 -]", args[len(args)-2:])
	}
}

func TestReasoningEffortCollapsesToMiniMaxLevels(t *testing.T) {
	// MiniMax M3 exposes only Adaptive Thinking off/on, so the wider Codex
	// ladder collapses rather than being forwarded verbatim.
	cases := map[agent.ReasoningEffort]string{
		"":        "",
		"none":    "none",
		"minimal": "none",
		"low":     "high",
		"medium":  "high",
		"high":    "high",
		"xhigh":   "high",
		"max":     "high",
		"ultra":   "high",
		"bogus":   "",
	}
	for effort, want := range cases {
		if got := reasoningEffortArg(effort); got != want {
			t.Errorf("reasoningEffortArg(%q) = %q, want %q", effort, got, want)
		}
	}
}

func TestHostEnvIsolatesFromCodex(t *testing.T) {
	env := hostEnv([]string{
		"PATH=/usr/bin",
		"OPENAI_API_KEY=leaked-chatgpt-key",
		"CODEX_HOME=/root/.codex",
		"MINIMAX_API_KEY=stale",
	}, "fresh-key")

	var codexHome, openAIKey, minimaxKey string
	for _, entry := range env {
		switch {
		case strings.HasPrefix(entry, "CODEX_HOME="):
			codexHome = strings.TrimPrefix(entry, "CODEX_HOME=")
		case strings.HasPrefix(entry, "OPENAI_API_KEY="):
			openAIKey = strings.TrimPrefix(entry, "OPENAI_API_KEY=")
		case strings.HasPrefix(entry, apiKeyEnvVar+"="):
			minimaxKey = strings.TrimPrefix(entry, apiKeyEnvVar+"=")
		}
	}

	// A MiniMax run must never inherit the Codex agent's home; that is what
	// keeps sessions, config, and auth.json separate between the two.
	if codexHome == "/root/.codex" {
		t.Fatalf("CODEX_HOME = %q, want MiniMax's own home", codexHome)
	}
	if openAIKey != "" {
		t.Fatalf("OPENAI_API_KEY = %q, want blank", openAIKey)
	}
	if minimaxKey != "fresh-key" {
		t.Fatalf("%s = %q, want the freshly read key", apiKeyEnvVar, minimaxKey)
	}
	if count := strings.Count(strings.Join(env, "\n"), "CODEX_HOME="); count != 1 {
		t.Fatalf("CODEX_HOME appears %d times, want 1", count)
	}
}

func TestForkScriptTargetsGivenSessionsDir(t *testing.T) {
	script := forkScript(containerMiniMaxHome+"/sessions", "parent-id", "new-id")

	// A fork must search MiniMax's own rollouts only — reaching into
	// /root/.codex/sessions would cross into the Codex agent's history.
	if !strings.Contains(script, containerMiniMaxHome+"/sessions") {
		t.Fatalf("script does not target MiniMax sessions: %s", script)
	}
	if strings.Contains(script, "/root/.codex/sessions") {
		t.Fatalf("script reaches into codex sessions: %s", script)
	}
	if !strings.Contains(script, "parent-id") || !strings.Contains(script, "new-id") {
		t.Fatalf("script missing session ids: %s", script)
	}
}
