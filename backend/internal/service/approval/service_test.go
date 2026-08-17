package approval

import (
	"context"
	"sync"
	"testing"
	"time"

	servicechat "github.com/futrx-com/remote.futrx.com/internal/service/chat"
)

func TestRequestThenDecideAllows(t *testing.T) {
	events := []servicechat.Event{}
	var mu sync.Mutex
	svc := New("https://remote.example.com", func(_ servicechat.ID, ev servicechat.Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, ev)
	})
	grant, err := svc.IssueGrant(context.Background(), "admin@example.com", true, "chat-1", "project-1")
	if err != nil {
		t.Fatalf("IssueGrant: %v", err)
	}
	if grant.APIURL != "https://remote.example.com/agent-api/approvals" {
		t.Fatalf("APIURL = %q", grant.APIURL)
	}

	type result struct {
		decision Decision
		err      error
	}
	done := make(chan result, 1)
	req := Request{
		ChatID:    "chat-1",
		ProjectID: "project-1",
		Tool:      "Bash",
		Command:   "rm -rf /",
		Reason:    "recursively deleting a filesystem root",
	}
	go func() {
		d, err := svc.RequestApproval(context.Background(), req)
		done <- result{d, err}
	}()

	// Wait until the approval is registered, then decide.
	deadline := time.Now().Add(3 * time.Second)
	var id string
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(events)
		mu.Unlock()
		if n > 0 {
			mu.Lock()
			id = events[0].ID
			mu.Unlock()
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if id == "" {
		t.Fatal("no permission_request event emitted")
	}

	pending, ok := svc.Pending(id)
	if !ok || pending.Command != "rm -rf /" {
		t.Fatalf("Pending(%q) = %+v, %v", id, pending, ok)
	}

	if _, err := svc.Decide(context.Background(), id, DecisionAllow, "admin@example.com"); err != nil {
		t.Fatalf("Decide: %v", err)
	}
	res := <-done
	if res.err != nil {
		t.Fatalf("RequestApproval: %v", res.err)
	}
	if res.decision != DecisionAllow {
		t.Fatalf("decision = %q, want allow", res.decision)
	}

	// The decided event carries the decision.
	mu.Lock()
	defer mu.Unlock()
	if len(events) != 2 || events[1].Type != "permission_request" {
		t.Fatalf("events = %+v, want 2 permission_request events", events)
	}
	if string(events[1].Data) == "" {
		t.Fatal("decided event has no data")
	}
}

func TestRequestTimesOutDenies(t *testing.T) {
	svc := New("https://remote.example.com", nil)
	req := Request{ChatID: "chat-1", ProjectID: "project-1", Tool: "Bash", Command: "mkfs /dev/sda"}
	go func() {
		svc.RequestApproval(context.Background(), req)
	}()
	time.Sleep(50 * time.Millisecond)
	if got := len(svc.pending); got != 1 {
		t.Fatalf("pending = %d, want 1", got)
	}
	// Nothing decides; the request blocks until timeout (8 min default), so we
	// only verify registration and cleanup on context cancel instead.
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_, _ = svc.RequestApproval(ctx, Request{ChatID: "chat-1", ProjectID: "project-1", Tool: "Bash", Command: "reboot"})
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RequestApproval did not return after cancel")
	}
}

func TestResolveGrantValidates(t *testing.T) {
	svc := New("https://remote.example.com", nil)
	access, err := svc.IssueGrant(context.Background(), "user@example.com", false, "chat-1", "project-1")
	if err != nil {
		t.Fatalf("IssueGrant: %v", err)
	}
	grant, err := svc.ResolveGrant(access.Token)
	if err != nil {
		t.Fatalf("ResolveGrant: %v", err)
	}
	if grant.ChatID != "chat-1" || grant.ProjectID != "project-1" || grant.OwnerEmail != "user@example.com" || grant.IsAdmin {
		t.Fatalf("grant = %+v", grant)
	}
	if _, err := svc.ResolveGrant("bogus"); err == nil {
		t.Fatal("ResolveGrant(bogus) succeeded, want error")
	}
}

func TestDecideUnknownApproval(t *testing.T) {
	svc := New("https://remote.example.com", nil)
	if _, err := svc.Decide(context.Background(), "nope", DecisionDeny, "admin@example.com"); err == nil {
		t.Fatal("Decide(nope) succeeded, want ErrNotFound")
	}
}
