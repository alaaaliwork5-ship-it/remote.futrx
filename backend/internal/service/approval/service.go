package approval

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	servicechat "github.com/futrx-com/remote.futrx.com/internal/service/chat"
	serviceproject "github.com/futrx-com/remote.futrx.com/internal/service/project"
)

var (
	ErrNotFound      = errors.New("approval not found")
	ErrGrantInvalid  = errors.New("invalid or expired approval grant")
	ErrExpired       = errors.New("approval expired")
	ErrNotActionable = errors.New("approval already decided")
)

type Decision string

const (
	DecisionAllow Decision = "allow"
	DecisionDeny  Decision = "deny"
)

const (
	decisionTimeout = 8 * time.Minute
	grantTTL        = 4 * time.Hour
)

// Request describes one pending human approval for a shell command.
type Request struct {
	ID          string
	ChatID      servicechat.ID
	ProjectID   serviceproject.ID
	Tool        string
	Command     string
	Reason      string
	RequestedAt time.Time
}

// Grant is the per-run capability carried by the container-side gate script.
type Grant struct {
	Token      string
	OwnerEmail string
	IsAdmin    bool
	ChatID     servicechat.ID
	ProjectID  serviceproject.ID
	ExpiresAt  time.Time
}

// GrantAccess is what a run issues into its runtime environment.
type GrantAccess struct {
	Token  string
	APIURL string
}

type pending struct {
	request Request
	decided chan Decision
}

// Emitter pushes transient chat events (approval requests and decisions) to
// connected chat clients. It is wired to the run hub's BroadcastTransient.
type Emitter func(chatID servicechat.ID, ev servicechat.Event)

// Service owns the pending-approval registry and the per-run grants that let
// the container gate script reach it.
type Service struct {
	mu      sync.Mutex
	apiURL  string
	now     func() time.Time
	grants  map[string]Grant
	pending map[string]*pending
	emit    Emitter
}

func New(baseURL string, emit Emitter) *Service {
	return &Service{
		apiURL:  strings.TrimRight(baseURL, "/") + "/agent-api/approvals",
		now:     time.Now,
		grants:  map[string]Grant{},
		pending: map[string]*pending{},
		emit:    emit,
	}
}

// APIURL is the endpoint the container gate script calls to request approval.
func (s *Service) APIURL() string {
	return s.apiURL
}

// IssueGrant creates a short-lived capability for one project chat run.
func (s *Service) IssueGrant(
	_ context.Context,
	email string,
	isAdmin bool,
	chatID servicechat.ID,
	projectID serviceproject.ID,
) (GrantAccess, error) {
	if chatID == "" || projectID == "" {
		return GrantAccess{}, errors.New("approval grant requires a project chat")
	}
	token, err := newToken()
	if err != nil {
		return GrantAccess{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteExpiredLocked()
	s.grants[token] = Grant{
		Token:      token,
		OwnerEmail: strings.ToLower(strings.TrimSpace(email)),
		IsAdmin:    isAdmin,
		ChatID:     chatID,
		ProjectID:  projectID,
		ExpiresAt:  s.now().Add(grantTTL),
	}
	return GrantAccess{Token: token, APIURL: s.apiURL}, nil
}

// ResolveGrant validates a gate token and returns its scope.
func (s *Service) ResolveGrant(token string) (Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	grant, ok := s.grants[token]
	if !ok || s.now().After(grant.ExpiresAt) {
		if ok {
			delete(s.grants, token)
		}
		return Grant{}, ErrGrantInvalid
	}
	return grant, nil
}

// RequestApproval registers a pending approval, notifies the chat, and blocks
// until a human decides. It is fail-closed: timeout and run cancellation both
// resolve to deny so a dangerous command never runs by default.
func (s *Service) RequestApproval(ctx context.Context, req Request) (Decision, error) {
	req.ID = newID()
	req.RequestedAt = s.now()
	p := &pending{request: req, decided: make(chan Decision, 1)}
	s.mu.Lock()
	s.deleteExpiredLocked()
	s.pending[req.ID] = p
	s.mu.Unlock()

	s.emitRequested(req)

	timer := time.NewTimer(decisionTimeout)
	defer timer.Stop()
	select {
	case decision := <-p.decided:
		s.forget(req.ID)
		return decision, nil
	case <-timer.C:
		s.forget(req.ID)
		s.emitDecided(req, DecisionDeny, "timed out waiting for approval")
		return DecisionDeny, ErrExpired
	case <-ctx.Done():
		s.forget(req.ID)
		s.emitDecided(req, DecisionDeny, "run canceled")
		return DecisionDeny, ErrExpired
	}
}

// Decide resolves a pending approval from the web UI.
func (s *Service) Decide(ctx context.Context, id string, decision Decision, decidedBy string) (Request, error) {
	s.mu.Lock()
	p, ok := s.pending[id]
	if !ok {
		s.mu.Unlock()
		return Request{}, ErrNotFound
	}
	delete(s.pending, id)
	s.mu.Unlock()

	select {
	case p.decided <- decision:
	default:
		return p.request, ErrNotActionable
	}
	s.emitDecided(p.request, decision, decidedBy)
	return p.request, nil
}

// Pending returns a pending approval for access checks in the transport layer.
func (s *Service) Pending(id string) (Request, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pending[id]
	if !ok {
		return Request{}, false
	}
	return p.request, true
}

func (s *Service) forget(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pending, id)
}

func (s *Service) deleteExpiredLocked() {
	now := s.now()
	for token, grant := range s.grants {
		if now.After(grant.ExpiresAt) {
			delete(s.grants, token)
		}
	}
	for id, p := range s.pending {
		if now.Sub(p.request.RequestedAt) > decisionTimeout+time.Minute {
			delete(s.pending, id)
		}
	}
}

func (s *Service) emitRequested(req Request) {
	if s.emit == nil {
		return
	}
	s.emit(req.ChatID, servicechat.Event{
		T:        req.RequestedAt.UnixMilli(),
		Type:     "permission_request",
		ID:       req.ID,
		ToolName: req.Tool,
		Name:     req.Tool,
		Input:    mustJSON(map[string]string{"command": req.Command}),
		Data:     mustJSON(map[string]string{"reason": req.Reason}),
	})
}

func (s *Service) emitDecided(req Request, decision Decision, decidedBy string) {
	if s.emit == nil {
		return
	}
	s.emit(req.ChatID, servicechat.Event{
		T:    s.now().UnixMilli(),
		Type: "permission_request",
		ID:   req.ID,
		Data: mustJSON(map[string]string{
			"decision":  string(decision),
			"decidedBy": decidedBy,
		}),
	})
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("{}")
	}
	return json.RawMessage(b)
}

func newToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "approval-" + time.Now().Format("150405.000")
	}
	return hex.EncodeToString(b)
}
