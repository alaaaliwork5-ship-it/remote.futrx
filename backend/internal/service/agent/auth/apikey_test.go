package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type keyStatus struct {
	Authenticated bool
	State         APIKeyState
}

func newTestKeyService(t *testing.T) (*APIKeyService[keyStatus], string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nested", "auth.json")
	service := NewAPIKeyService(APIKeyConfig[keyStatus]{
		Path: path,
		Validate: func(key string) error {
			if strings.HasPrefix(key, "bad") {
				return errors.New("rejected by provider")
			}
			return nil
		},
		BuildStatus: func() APIKeyStatusBuilder[keyStatus] {
			return func(state APIKeyState) keyStatus {
				return keyStatus{Authenticated: state.Configured, State: state}
			}
		},
	})
	return service, path
}

func TestAPIKeyServiceStoresAndReportsKey(t *testing.T) {
	service, path := newTestKeyService(t)

	if service.Authenticated() {
		t.Fatal("expected no key before submit")
	}
	if err := service.SubmitKey(context.Background(), "  secret-value-1234  "); err != nil {
		t.Fatalf("SubmitKey: %v", err)
	}

	key, err := service.Key()
	if err != nil {
		t.Fatalf("Key: %v", err)
	}
	// Surrounding whitespace must be stripped: it would otherwise travel into
	// an HTTP header and produce a malformed request.
	if key != "secret-value-1234" {
		t.Fatalf("Key = %q, want %q", key, "secret-value-1234")
	}
	if !service.Authenticated() {
		t.Fatal("expected Authenticated after submit")
	}

	status := service.Status()
	if !status.State.Configured {
		t.Fatal("status should report configured")
	}
	if status.State.Hint != "••••1234" {
		t.Fatalf("hint = %q, want %q", status.State.Hint, "••••1234")
	}
	if strings.Contains(status.State.Hint, "secret") {
		t.Fatalf("hint leaks the key: %q", status.State.Hint)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat key file: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("key file mode = %o, want 600", mode)
	}
}

func TestAPIKeyServiceRejectsInvalidInput(t *testing.T) {
	service, path := newTestKeyService(t)

	for _, key := range []string{"", "   ", "bad-key-value"} {
		err := service.SubmitKey(context.Background(), key)
		if err == nil {
			t.Fatalf("SubmitKey(%q) succeeded, want rejection", key)
		}
		if !errors.Is(err, ErrInvalidAPIKey) {
			t.Fatalf("SubmitKey(%q) error = %v, want ErrInvalidAPIKey", key, err)
		}
		if !service.IsInputError(err) {
			t.Fatalf("IsInputError(%v) = false, want true", err)
		}
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("rejected key must not be persisted")
	}
}

func TestAPIKeyServiceClearIsIdempotent(t *testing.T) {
	service, _ := newTestKeyService(t)

	// Clearing an absent key must succeed so "sign out" is always safe.
	if err := service.ClearKey(context.Background()); err != nil {
		t.Fatalf("ClearKey on empty: %v", err)
	}
	if err := service.SubmitKey(context.Background(), "secret-value-1234"); err != nil {
		t.Fatalf("SubmitKey: %v", err)
	}
	if err := service.ClearKey(context.Background()); err != nil {
		t.Fatalf("ClearKey: %v", err)
	}
	if service.Authenticated() {
		t.Fatal("expected no key after clear")
	}
}

func TestAPIKeyServiceBroadcastsToSubscribers(t *testing.T) {
	service, _ := newTestKeyService(t)

	updates, unsubscribe := service.Subscribe()
	defer unsubscribe()

	if initial := <-updates; initial.Authenticated {
		t.Fatal("first snapshot should report no key")
	}
	if err := service.SubmitKey(context.Background(), "secret-value-1234"); err != nil {
		t.Fatalf("SubmitKey: %v", err)
	}
	if update := <-updates; !update.Authenticated {
		t.Fatal("subscriber should see the stored key")
	}
}

func TestAPIKeyBindingExposesKeyFlow(t *testing.T) {
	service, _ := newTestKeyService(t)
	binding := NewAPIKeyBinding("minimax", service)

	if binding.Flow() != FlowAPIKey {
		t.Fatalf("flow = %q, want %q", binding.Flow(), FlowAPIKey)
	}
	if !binding.Available() {
		t.Fatal("binding should be available")
	}
	if err := binding.SubmitKey(context.Background(), "secret-value-1234"); err != nil {
		t.Fatalf("SubmitKey: %v", err)
	}
	if !binding.Authenticated() {
		t.Fatal("binding should report authenticated")
	}

	// The code and device lifecycles must stay closed on this flow.
	if _, err := binding.StartDevice(context.Background()); !errors.Is(err, ErrUnsupportedFlow) {
		t.Fatalf("StartDevice error = %v, want ErrUnsupportedFlow", err)
	}
	if _, err := binding.StartCode(context.Background()); !errors.Is(err, ErrUnsupportedFlow) {
		t.Fatalf("StartCode error = %v, want ErrUnsupportedFlow", err)
	}

	if err := binding.ClearKey(context.Background()); err != nil {
		t.Fatalf("ClearKey: %v", err)
	}
	if binding.Authenticated() {
		t.Fatal("binding should report signed out after clear")
	}
}

func TestAPIKeyBindingWithoutServiceIsUnavailable(t *testing.T) {
	binding := NewAPIKeyBinding[keyStatus]("minimax", nil)

	if binding.Available() {
		t.Fatal("service-less binding should be unavailable")
	}
	if err := binding.SubmitKey(context.Background(), "x"); !errors.Is(err, ErrUnsupportedFlow) {
		t.Fatalf("SubmitKey error = %v, want ErrUnsupportedFlow", err)
	}
}
