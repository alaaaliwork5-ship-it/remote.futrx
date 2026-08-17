package httphandlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	serviceapproval "github.com/futrx-com/remote.futrx.com/internal/service/approval"
	servicechat "github.com/futrx-com/remote.futrx.com/internal/service/chat"
	serviceproject "github.com/futrx-com/remote.futrx.com/internal/service/project"
)

type fakeApprovalService struct {
	mu        sync.Mutex
	approvals *serviceapproval.Service
	events    []servicechat.Event
}

func newFakeApprovalService() *fakeApprovalService {
	f := &fakeApprovalService{}
	f.approvals = serviceapproval.New("https://remote.example.com", func(_ servicechat.ID, ev servicechat.Event) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.events = append(f.events, ev)
	})
	return f
}

func (f *fakeApprovalService) RequestApproval(ctx context.Context, req serviceapproval.Request) (serviceapproval.Decision, error) {
	return f.approvals.RequestApproval(ctx, req)
}

func (f *fakeApprovalService) Decide(ctx context.Context, id string, decision serviceapproval.Decision, decidedBy string) (serviceapproval.Request, error) {
	return f.approvals.Decide(ctx, id, decision, decidedBy)
}

func (f *fakeApprovalService) ResolveGrant(token string) (serviceapproval.Grant, error) {
	return f.approvals.ResolveGrant(token)
}

func (f *fakeApprovalService) Pending(id string) (serviceapproval.Request, bool) {
	return f.approvals.Pending(id)
}

// firstRequestedID waits for the first permission_request event and returns
// its approval id.
func (f *fakeApprovalService) firstRequestedID(t *testing.T) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		var id string
		for _, ev := range f.events {
			if ev.Type == "permission_request" && ev.ID != "" {
				id = ev.ID
				break
			}
		}
		f.mu.Unlock()
		if id != "" {
			return id
		}
		time.Sleep(5 * time.Millisecond)
	}
	return ""
}

type fakeApprovalAccess struct {
	email   string
	isAdmin bool
	access  bool
}

func (f *fakeApprovalAccess) CallerAndAdmin(ctx context.Context, r *http.Request) (string, bool, error) {
	return f.email, f.isAdmin, nil
}

func (f *fakeApprovalAccess) HasAccess(ctx context.Context, projectID serviceproject.ID, email string) (bool, error) {
	return f.access, nil
}

func TestAgentApprovalRequest(t *testing.T) {
	fake := newFakeApprovalService()
	access, err := fake.approvals.IssueGrant(context.Background(), "admin@example.com", true, "chat-1", "project-1")
	if err != nil {
		t.Fatalf("IssueGrant: %v", err)
	}
	handler := NewApprovalHandler(fake, &fakeApprovalAccess{})

	t.Run("safe command auto-allows", func(t *testing.T) {
		body := `{"tool":"Bash","command":"go test ./..."}`
		req := httptest.NewRequest(http.MethodPost, "/agent-api/approvals", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+access.Token)
		rec := httptest.NewRecorder()
		handler.HandleAgentRequest(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Decision string `json:"decision"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.Decision != "allow" {
			t.Fatalf("decision = %q, want allow", resp.Decision)
		}
	})

	t.Run("rejects missing grant", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/agent-api/approvals", strings.NewReader(`{"tool":"Bash","command":"rm -rf /"}`))
		rec := httptest.NewRecorder()
		handler.HandleAgentRequest(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("dangerous command waits for decision", func(t *testing.T) {
		body := `{"tool":"Bash","command":"rm -rf /"}`
		req := httptest.NewRequest(http.MethodPost, "/agent-api/approvals", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+access.Token)
		rec := httptest.NewRecorder()
		done := make(chan struct{})
		go func() {
			handler.HandleAgentRequest(rec, req)
			close(done)
		}()

		id := fake.firstRequestedID(t)
		if id == "" {
			t.Fatal("no permission_request event emitted")
		}
		if _, err := fake.approvals.Decide(context.Background(), id, serviceapproval.DecisionDeny, "admin@example.com"); err != nil {
			t.Fatalf("Decide: %v", err)
		}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("agent request did not return after decision")
		}
		var resp struct {
			Decision string `json:"decision"`
			Reason   string `json:"reason"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.Decision != "deny" || resp.Reason == "" {
			t.Fatalf("resp = %+v, want deny with reason", resp)
		}
	})
}

func TestUserDecisionEndpoint(t *testing.T) {
	fake := newFakeApprovalService()
	handler := NewApprovalHandler(fake, &fakeApprovalAccess{email: "member@example.com", access: true})

	reqCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		_, _ = fake.approvals.RequestApproval(reqCtx, serviceapproval.Request{
			ChatID:    "chat-1",
			ProjectID: "project-1",
			Tool:      "Bash",
			Command:   "mkfs /dev/sda",
			Reason:    "creating a filesystem",
		})
		close(done)
	}()
	id := fake.firstRequestedID(t)
	if id == "" {
		t.Fatal("no pending approval")
	}

	body := `{"decision":"allow"}`
	req := httptest.NewRequest(http.MethodPost, "/api/approvals/"+id+"/decision", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.HandleUserDecision(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RequestApproval did not resolve")
	}
}

func TestUserDecisionDeniedWithoutAccess(t *testing.T) {
	fake := newFakeApprovalService()
	handler := NewApprovalHandler(fake, &fakeApprovalAccess{email: "outsider@example.com", isAdmin: false, access: false})

	reqCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		_, _ = fake.approvals.RequestApproval(reqCtx, serviceapproval.Request{
			ChatID:    "chat-1",
			ProjectID: "project-1",
			Tool:      "Bash",
			Command:   "reboot",
			Reason:    "shutting down",
		})
		close(done)
	}()
	id := fake.firstRequestedID(t)
	if id == "" {
		t.Fatal("no pending approval")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/approvals/"+id+"/decision", strings.NewReader(`{"decision":"allow"}`))
	rec := httptest.NewRecorder()
	handler.HandleUserDecision(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	cancel()
	<-done
}
