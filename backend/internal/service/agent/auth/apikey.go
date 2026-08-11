package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ErrInvalidAPIKey marks a caller-supplied key that failed the provider's shape
// check. Transports map it to a 4xx rather than a 5xx.
var ErrInvalidAPIKey = errors.New("invalid API key")

// APIKeyState is the provider-neutral state of a stored static API key. The key
// itself is never included — only enough to render "configured" in a UI.
type APIKeyState struct {
	Configured bool   `json:"configured"`
	Hint       string `json:"hint,omitempty"`
	UpdatedAt  int64  `json:"updatedAt,omitempty"`
	Error      string `json:"error,omitempty"`
}

// APIKeyStatusBuilder lets a provider wrap the neutral key state in its own
// status shape, matching the code/device flows.
type APIKeyStatusBuilder[S any] func(APIKeyState) S

// APIKeyConfig supplies the provider-specific policy around a stored key.
type APIKeyConfig[S any] struct {
	// Path is the host file that stores the key. It is written 0600 and its
	// parent directory is created 0700 on first write.
	Path string
	// Validate rejects malformed keys before anything is persisted. Returning a
	// non-nil error surfaces as ErrInvalidAPIKey to callers.
	Validate    func(string) error
	BuildStatus func() APIKeyStatusBuilder[S]
}

// storedKey is the on-disk document. It mirrors codex's auth.json shape closely
// enough to be recognizable, while staying provider-neutral.
type storedKey struct {
	APIKey    string `json:"api_key"`
	UpdatedAt int64  `json:"updated_at,omitempty"`
}

// APIKeyService owns one provider's static API key: the host file is the single
// source of truth, and subscribers receive a snapshot on every mutation.
type APIKeyService[S any] struct {
	config APIKeyConfig[S]

	mu   sync.Mutex
	subs map[chan S]struct{}
}

func NewAPIKeyService[S any](config APIKeyConfig[S]) *APIKeyService[S] {
	return &APIKeyService[S]{config: config, subs: map[chan S]struct{}{}}
}

// Authenticated reports whether a usable key is on disk.
func (s *APIKeyService[S]) Authenticated() bool {
	key, err := s.Key()
	return err == nil && key != ""
}

// Key returns the stored key for run-time injection. Providers call this to
// build the child process environment; it is never sent to a transport.
func (s *APIKeyService[S]) Key() (string, error) {
	data, err := os.ReadFile(s.config.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	var stored storedKey
	if err := json.Unmarshal(data, &stored); err != nil {
		return "", fmt.Errorf("parse %s: %w", s.config.Path, err)
	}
	return strings.TrimSpace(stored.APIKey), nil
}

func (s *APIKeyService[S]) Status() S {
	return s.config.BuildStatus()(s.state())
}

func (s *APIKeyService[S]) Subscribe() (<-chan S, func()) {
	ch := make(chan S, subscriptionBuffer)
	s.mu.Lock()
	if s.subs == nil {
		s.subs = map[chan S]struct{}{}
	}
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	ch <- s.Status()

	var closeOnce sync.Once
	cancel := func() {
		closeOnce.Do(func() {
			s.mu.Lock()
			if _, ok := s.subs[ch]; ok {
				delete(s.subs, ch)
				close(ch)
			}
			s.mu.Unlock()
		})
	}
	return ch, cancel
}

// SubmitKey validates and persists a key, replacing any existing one.
func (s *APIKeyService[S]) SubmitKey(_ context.Context, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("%w: key is empty", ErrInvalidAPIKey)
	}
	if s.config.Validate != nil {
		if err := s.config.Validate(key); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidAPIKey, err.Error())
		}
	}

	document, err := json.Marshal(storedKey{APIKey: key, UpdatedAt: time.Now().Unix()})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.config.Path), 0o700); err != nil {
		return fmt.Errorf("create credential dir: %w", err)
	}
	// Write-then-rename so a crash mid-write cannot leave a truncated key that
	// would fail every subsequent run with a confusing auth error.
	temp := s.config.Path + ".tmp"
	if err := os.WriteFile(temp, document, 0o600); err != nil {
		return fmt.Errorf("write key: %w", err)
	}
	if err := os.Rename(temp, s.config.Path); err != nil {
		_ = os.Remove(temp)
		return fmt.Errorf("write key: %w", err)
	}
	s.Broadcast()
	return nil
}

// ClearKey removes the stored key. Clearing an absent key is a no-op so the
// caller's "sign out" is idempotent.
func (s *APIKeyService[S]) ClearKey(_ context.Context) error {
	if err := os.Remove(s.config.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove key: %w", err)
	}
	s.Broadcast()
	return nil
}

func (s *APIKeyService[S]) IsInputError(err error) bool {
	return errors.Is(err, ErrInvalidAPIKey)
}

func (s *APIKeyService[S]) state() APIKeyState {
	data, err := os.ReadFile(s.config.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return APIKeyState{}
		}
		return APIKeyState{Error: err.Error()}
	}
	var stored storedKey
	if err := json.Unmarshal(data, &stored); err != nil {
		return APIKeyState{Error: fmt.Sprintf("parse %s: %s", s.config.Path, err.Error())}
	}
	key := strings.TrimSpace(stored.APIKey)
	if key == "" {
		return APIKeyState{}
	}
	return APIKeyState{Configured: true, Hint: keyHint(key), UpdatedAt: stored.UpdatedAt}
}

func (s *APIKeyService[S]) Broadcast() {
	status := s.Status()
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- status:
		default:
			delete(s.subs, ch)
			close(ch)
		}
	}
}

// keyHint renders the last four characters so a user can tell which key is
// stored without the value being recoverable from the status endpoint.
func keyHint(key string) string {
	runes := []rune(key)
	if len(runes) <= 4 {
		return strings.Repeat("•", len(runes))
	}
	return "••••" + string(runes[len(runes)-4:])
}
