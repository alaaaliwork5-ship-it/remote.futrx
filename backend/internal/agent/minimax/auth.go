package minimax

// Auth configures MiniMax's static-API-key credential. Unlike Codex or Kimi
// there is no interactive grant to drive: MiniMax issues a bearer token from
// its developer platform that the user pastes once. The key is stored on the
// host and handed to the Codex CLI through the `env_key` indirection, so it is
// never written into a container image or a config.toml on disk.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	agentauth "github.com/futrx-com/remote.futrx.com/internal/service/agent/auth"
)

// minKeyLength is a loose sanity floor. MiniMax does not publish a key format,
// so validation only rejects input that cannot be a bearer token at all — a
// stricter rule risks refusing keys after an upstream format change.
const minKeyLength = 16

var ErrNoAPIKey = errors.New("MiniMax API key is not configured - add it in Settings before running MiniMax")

type AuthStatus struct {
	Authenticated bool                  `json:"authenticated"`
	APIKey        agentauth.APIKeyState `json:"apiKey,omitempty"`
}

type Auth = agentauth.APIKeyService[AuthStatus]

func NewAuth() *Auth {
	return agentauth.NewAPIKeyService(agentauth.APIKeyConfig[AuthStatus]{
		Path:     APIKeyPath(),
		Validate: validateAPIKey,
		BuildStatus: func() agentauth.APIKeyStatusBuilder[AuthStatus] {
			return func(state agentauth.APIKeyState) AuthStatus {
				return AuthStatus{Authenticated: state.Configured, APIKey: state}
			}
		},
	})
}

// validateAPIKey rejects input that could not be a bearer token: MiniMax sends
// the value as an HTTP header, so embedded whitespace or control characters
// would produce a malformed request rather than an auth failure.
func validateAPIKey(key string) error {
	if len(key) < minKeyLength {
		return fmt.Errorf("key looks too short (%d characters)", len(key))
	}
	for _, r := range key {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return errors.New("key contains whitespace or control characters")
		}
		if r > unicode.MaxASCII {
			return errors.New("key contains non-ASCII characters")
		}
	}
	return nil
}

// hostMiniMaxHome is the host-side directory holding MiniMax's key. It mirrors
// the container's separate CODEX_HOME so the two never collide with ~/.codex.
func hostMiniMaxHome() string {
	if v := strings.TrimSpace(os.Getenv("MINIMAX_CODEX_HOME")); v != "" {
		return v
	}
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".minimax-codex")
	}
	return containerMiniMaxHome
}

// APIKeyPath is the host file backing the stored key.
func APIKeyPath() string {
	return filepath.Join(hostMiniMaxHome(), "auth.json")
}
