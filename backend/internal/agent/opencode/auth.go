package opencode

// Auth configures OpenCode's shared API-key credential. OpenCode 1.18.x has
// no device-code URL flow: `opencode providers login` is an interactive TUI
// that prompts for a provider API key (or a self-hosted auth URL). Remote
// stores the key directly in OpenCode's auth.json the same way the CLI would,
// so the credential seeds into project containers and counts as
// authenticated. Users can also set provider API keys as project secrets
// (ANTHROPIC_API_KEY / OPENAI_API_KEY / GEMINI_API_KEY), which are injected
// as environment variables — those also count as authenticated.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	agentauth "github.com/futrx-com/remote.futrx.com/internal/service/agent/auth"
)

type AuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	AuthMode      string `json:"authMode,omitempty"`
	UsesAPIKey    bool   `json:"usesApiKey,omitempty"`
}

// Provider API key env vars OpenCode picks up at launch, in addition to
// auth.json. Any of these set on the host counts as authenticated.
var providerAPIKeyEnvs = []string{
	"ANTHROPIC_API_KEY",
	"OPENAI_API_KEY",
	"GEMINI_API_KEY",
	"OPENROUTER_API_KEY",
	"XAI_API_KEY",
	"DEEPSEEK_API_KEY",
}

type Auth = agentauth.APIKeyService[AuthStatus]

func NewAuth() *Auth {
	return agentauth.NewAPIKeyService(agentauth.APIKeyConfig[AuthStatus]{
		Save: saveAPIKey,
		Authenticated: func() bool {
			authenticated, _, _ := authenticated()
			return authenticated
		},
		BuildStatus: func() AuthStatus {
			authenticated, authMode, usesAPIKey := authenticated()
			return AuthStatus{
				Authenticated: authenticated,
				AuthMode:      authMode,
				UsesAPIKey:    usesAPIKey,
			}
		},
	})
}

// saveAPIKey writes {provider: {"type":"api","key":...}} into OpenCode's
// auth.json, merging with any existing credentials. Provider ids are
// normalized the way opencode normalizes them (lowercase, @ai-sdk/ prefix
// stripped).
func saveAPIKey(provider, key string) error {
	path := filepath.Join(hostOpenCodeHome(), "auth.json")
	entries := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &entries)
	}
	provider = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(provider)), "@ai-sdk/")
	if provider == "" {
		return errors.New("provider is required")
	}
	entries[provider] = map[string]any{"type": "api", "key": key}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create opencode credential dir: %w", err)
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encode opencode credentials: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write opencode credentials: %w", err)
	}
	return nil
}

// authenticated reports whether OpenCode has usable credentials: provider API
// keys in auth.json or the environment.
func authenticated() (bool, string, bool) {
	entries, err := readAuthEntries()
	if err == nil && len(entries) > 0 {
		for name := range entries {
			if isAPIKeyProvider(name) {
				return true, "apikey", true
			}
		}
		return true, "console", false
	}
	for _, env := range os.Environ() {
		key, _, ok := strings.Cut(env, "=")
		if !ok {
			continue
		}
		for _, candidate := range providerAPIKeyEnvs {
			if key == candidate {
				return true, "apikey", true
			}
		}
	}
	return false, "", false
}

// readAuthEntries returns the top-level provider keys in auth.json. OpenCode
// stores e.g. {"anthropic":{"type":"api","key":...}} plus console sessions.
func readAuthEntries() (map[string]struct{}, error) {
	data, err := os.ReadFile(filepath.Join(hostOpenCodeHome(), "auth.json"))
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	entries := make(map[string]struct{}, len(raw))
	for name := range raw {
		entries[name] = struct{}{}
	}
	return entries, nil
}

func isAPIKeyProvider(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "openai", "anthropic", "google", "gemini", "openrouter", "xai", "deepseek", "groq", "mistral", "together":
		return true
	default:
		return false
	}
}
