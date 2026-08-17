package httphandlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
	service "github.com/futrx-com/remote.futrx.com/internal/service"
	agentauth "github.com/futrx-com/remote.futrx.com/internal/service/agent/auth"
)

const missingAgentCLI = "futrx-test-agent-cli-that-does-not-exist"

var (
	testCodeRequired = errors.New("code is required")
	testNoSession    = errors.New("no login session in progress - call /api/claude/login/start first")
)

type agentAuthDeviceStatus struct {
	Authenticated bool                  `json:"authenticated"`
	DeviceLogin   agentauth.DeviceState `json:"deviceLogin,omitempty"`
}

type agentAuthAPIKeyStatus struct {
	Authenticated bool `json:"authenticated"`
}

func TestAgentAuthStatusRoutesPreserveProviderPayloads(t *testing.T) {
	handler := newTestAgentAuthHandler()
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	tests := []struct {
		path string
		want string
	}{
		{path: "/api/claude/auth-status", want: `{"authenticated":false,"login":{"active":false}}` + "\n"},
		{path: "/api/codex/auth-status", want: `{"authenticated":false,"deviceLogin":{"active":false}}` + "\n"},
		{path: "/api/kimi/auth-status", want: `{"authenticated":false,"deviceLogin":{"active":false}}` + "\n"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, test.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK || rec.Body.String() != test.want {
				t.Fatalf("response = %d %q, want %d %q", rec.Code, rec.Body.String(), http.StatusOK, test.want)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
		})
	}
}

func TestAgentAuthMutationRoutesRemainPostOnly(t *testing.T) {
	handler := newTestAgentAuthHandler()
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	paths := []string{
		"/api/claude/login/start",
		"/api/claude/login/code",
		"/api/claude/login/cancel",
		"/api/codex/login/device",
		"/api/kimi/login/device",
		"/api/opencode/login/key",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusMethodNotAllowed || rec.Body.String() != `{"error":"method not allowed"}`+"\n" {
				t.Fatalf("response = %d %q", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAgentAuthCodeErrorsKeepTheirHTTPMapping(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
		want int
		text string
	}{
		{
			name: "malformed json",
			path: "/api/claude/login/code",
			body: `{`,
			want: http.StatusBadRequest,
			text: `{"error":"invalid json: unexpected EOF"}` + "\n",
		},
		{
			name: "blank code",
			path: "/api/claude/login/code",
			body: `{"code":"  "}`,
			want: http.StatusBadRequest,
			text: `{"error":"code is required"}` + "\n",
		},
		{
			name: "missing session",
			path: "/api/claude/login/code",
			body: `{"code":"abc"}`,
			want: http.StatusBadRequest,
			text: `{"error":"no login session in progress - call /api/claude/login/start first"}` + "\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := newTestAgentAuthHandler()
			mux := http.NewServeMux()
			handler.RegisterRoutes(mux)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body)))
			if rec.Code != test.want || rec.Body.String() != test.text {
				t.Fatalf("response = %d %q, want %d %q", rec.Code, rec.Body.String(), test.want, test.text)
			}
		})
	}
}

func TestAgentAuthCatalogRouteListsBindingsWithAuthMethods(t *testing.T) {
	handler := newTestAgentAuthHandler()
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/agents", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/agents status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Agents []service.AgentInfo `json:"agents"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if len(body.Agents) != 4 {
		t.Fatalf("catalog has %d agents, want 4", len(body.Agents))
	}

	want := []struct {
		id         string
		authMethod string
	}{
		{id: "claude", authMethod: "code"},
		{id: "codex", authMethod: "device"},
		{id: "kimi", authMethod: "device"},
		{id: "opencode", authMethod: "apikey"},
	}
	for index, entry := range want {
		agent := body.Agents[index]
		if agent.ID != entry.id || agent.AuthMethod != entry.authMethod {
			t.Fatalf("catalog[%d] = %s/%s, want %s/%s",
				index, agent.ID, agent.AuthMethod, entry.id, entry.authMethod)
		}
		if agent.Name == "" || agent.Description == "" {
			t.Fatalf("catalog[%d] %q missing display metadata", index, agent.ID)
		}
	}
}

func TestAgentAuthCatalogRouteRejectsNonGet(t *testing.T) {
	handler := newTestAgentAuthHandler()
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/agents", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /api/agents status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestAgentAuthOperationalErrorsAndCancelShape(t *testing.T) {
	handler := newTestAgentAuthHandler()
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	tests := []struct {
		path string
		want int
		body string
	}{
		{
			path: "/api/claude/login/start",
			want: http.StatusInternalServerError,
			body: `{"error":"claude CLI not found on PATH - install it first"}` + "\n",
		},
		{
			path: "/api/codex/login/device",
			want: http.StatusInternalServerError,
			body: `{"error":"codex CLI not found on PATH - install it first"}` + "\n",
		},
		{
			path: "/api/kimi/login/device",
			want: http.StatusInternalServerError,
			body: `{"error":"kimi CLI not found on PATH - install it first"}` + "\n",
		},
		{
			path: "/api/claude/login/cancel",
			want: http.StatusOK,
			body: `{"ok":true}` + "\n",
		},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, test.path, nil))
			if rec.Code != test.want || rec.Body.String() != test.body {
				t.Fatalf("response = %d %q, want %d %q", rec.Code, rec.Body.String(), test.want, test.body)
			}
		})
	}
}

func TestAgentAuthAPIKeySaveRoute(t *testing.T) {
	handler := newTestAgentAuthHandler()
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Save captures into the shared APIKeyService so the status route reflects
	// the new credential.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(
		http.MethodPost,
		"/api/opencode/login/key",
		strings.NewReader(`{"provider":"anthropic","key":"sk-ant-test"}`),
	))
	if rec.Code != http.StatusOK || rec.Body.String() != `{"ok":true}`+"\n" {
		t.Fatalf("save response = %d %q, want 200 ok", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/opencode/auth-status", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != `{"authenticated":true}`+"\n" {
		t.Fatalf("status after save = %d %q, want authenticated", rec.Code, rec.Body.String())
	}

	tests := []struct {
		name string
		body string
		want int
		text string
	}{
		{
			name: "rejects blank key",
			body: `{"provider":"anthropic","key":"  "}`,
			want: http.StatusBadRequest,
			text: `{"error":"API key is required"}` + "\n",
		},
		{
			name: "rejects blank provider",
			body: `{"provider":" ","key":"sk-ant-test"}`,
			want: http.StatusBadRequest,
			text: `{"error":"provider is required"}` + "\n",
		},
		{
			name: "rejects malformed json",
			body: `{`,
			want: http.StatusBadRequest,
			text: `{"error":"invalid json: unexpected EOF"}` + "\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/opencode/login/key", strings.NewReader(test.body)))
			if rec.Code != test.want || rec.Body.String() != test.text {
				t.Fatalf("response = %d %q, want %d %q", rec.Code, rec.Body.String(), test.want, test.text)
			}
		})
	}
}

func newTestAgentAuthHandler() *AgentAuthHandler {
	code := agentauth.NewCodeService(agentauth.CodeConfig{
		Command:       missingAgentCLI,
		Authenticated: func() bool { return false },
		NotFound:      errors.New("claude CLI not found on PATH - install it first"),
		CodeRequired:  testCodeRequired,
		NoSession:     testNoSession,
	})
	device := func(notFound string) *agentauth.DeviceService[agentAuthDeviceStatus] {
		return agentauth.NewDeviceService(agentauth.DeviceConfig[agentAuthDeviceStatus]{
			Command:  missingAgentCLI,
			NotFound: errors.New(notFound),
			BuildStatus: func() agentauth.DeviceStatusBuilder[agentAuthDeviceStatus] {
				return func(state agentauth.DeviceState) agentAuthDeviceStatus {
					return agentAuthDeviceStatus{DeviceLogin: state}
				}
			},
		})
	}

	var savedProvider string
	apiKey := agentauth.NewAPIKeyService(agentauth.APIKeyConfig[agentAuthAPIKeyStatus]{
		Save: func(provider, _ string) error {
			savedProvider = provider
			return nil
		},
		Authenticated: func() bool { return savedProvider != "" },
		BuildStatus: func() agentAuthAPIKeyStatus {
			return agentAuthAPIKeyStatus{Authenticated: savedProvider != ""}
		},
	})

	bindings := []agentauth.Binding{
		agentauth.NewCodeBinding(agent.ProviderClaude, code),
		agentauth.NewDeviceBinding(agent.ProviderCodex, device("codex CLI not found on PATH - install it first")),
		agentauth.NewDeviceBinding(agent.ProviderKimi, device("kimi CLI not found on PATH - install it first")),
		agentauth.NewAPIKeyBinding(agent.ProviderOpenCode, apiKey),
	}
	return NewAgentAuthHandler(bindings, nil, []service.AgentInfo{
		{ID: "claude", Name: "Claude", Description: "Anthropic's Claude Code", AuthMethod: "code", AuthAvailable: true},
		{ID: "codex", Name: "Codex", Description: "OpenAI's Codex CLI", AuthMethod: "device", AuthAvailable: true},
		{ID: "kimi", Name: "Kimi", Description: "Moonshot's Kimi Code", AuthMethod: "device", AuthAvailable: true},
		{ID: "opencode", Name: "OpenCode", Description: "Open-source coding agent", AuthMethod: "apikey", AuthAvailable: true},
	})
}
