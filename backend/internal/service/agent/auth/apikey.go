package auth

// APIKeyService owns a provider's shared API-key credential. Unlike the code
// and device flows it does not drive a CLI process: agents like OpenCode
// 1.18.x authenticate with provider API keys stored in a credential file, so
// the backend writes the key straight to the provider's store and broadcasts
// the resulting status to subscribers.

import (
	"context"
	"errors"
	"strings"
	"sync"
)

const apiKeySubscriptionBuffer = 8

// APIKeyConfig supplies provider policy around a shared API-key credential.
type APIKeyConfig[T any] struct {
	// Save persists the key for the named provider. Providers normalize the
	// provider id and key before writing; the returned error is shown as-is.
	Save func(provider, key string) error
	// Authenticated reports whether a usable credential already exists.
	Authenticated func() bool
	// BuildStatus renders the current status snapshot.
	BuildStatus func() T
}

// APIKeyService owns one provider's API-key credential and streams status
// snapshots to subscribers.
type APIKeyService[T any] struct {
	config APIKeyConfig[T]

	mu   sync.Mutex
	subs map[chan T]struct{}
}

func NewAPIKeyService[T any](config APIKeyConfig[T]) *APIKeyService[T] {
	return &APIKeyService[T]{
		config: config,
		subs:   map[chan T]struct{}{},
	}
}

func (s *APIKeyService[T]) Authenticated() bool {
	if s.config.Authenticated == nil {
		return false
	}
	return s.config.Authenticated()
}

func (s *APIKeyService[T]) Status() T {
	var zero T
	if s.config.BuildStatus == nil {
		return zero
	}
	return s.config.BuildStatus()
}

// Subscribe registers a status channel and immediately delivers the current
// snapshot. The returned function unsubscribes and closes the channel.
func (s *APIKeyService[T]) Subscribe() (<-chan T, func()) {
	ch := make(chan T, apiKeySubscriptionBuffer)
	s.mu.Lock()
	if s.subs == nil {
		s.subs = map[chan T]struct{}{}
	}
	s.subs[ch] = struct{}{}
	status := s.Status()
	s.mu.Unlock()
	ch <- status

	cancel := func() {
		s.mu.Lock()
		if _, ok := s.subs[ch]; ok {
			delete(s.subs, ch)
			close(ch)
		}
		s.mu.Unlock()
	}
	return ch, cancel
}

// SaveKey validates and persists a provider API key, then broadcasts the new
// status. The provider's Save owns idempotency and error wording.
func (s *APIKeyService[T]) SaveKey(_ context.Context, provider, key string) error {
	if s.config.Save == nil {
		return ErrUnsupportedFlow
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return errors.New("provider is required")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("API key is required")
	}
	if err := s.config.Save(provider, key); err != nil {
		return err
	}
	s.broadcast()
	return nil
}

func (s *APIKeyService[T]) broadcast() {
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
