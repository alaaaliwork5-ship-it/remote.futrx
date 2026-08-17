package httphandlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	serviceapproval "github.com/futrx-com/remote.futrx.com/internal/service/approval"
	serviceproject "github.com/futrx-com/remote.futrx.com/internal/service/project"
	httptransport "github.com/futrx-com/remote.futrx.com/internal/transport/http"
)

const approvalRequestLimit = 64 << 10

// ApprovalService is the HTTP layer's narrow view of the approval service.
type ApprovalService interface {
	RequestApproval(ctx context.Context, req serviceapproval.Request) (serviceapproval.Decision, error)
	Decide(ctx context.Context, id string, decision serviceapproval.Decision, decidedBy string) (serviceapproval.Request, error)
	ResolveGrant(token string) (serviceapproval.Grant, error)
	Pending(id string) (serviceapproval.Request, bool)
}

// ApprovalAccessChecker authorizes human decisions: the caller must be able to
// reach the approval's project (membership or admin).
type ApprovalAccessChecker interface {
	CallerAndAdmin(ctx context.Context, r *http.Request) (string, bool, error)
	HasAccess(ctx context.Context, projectID serviceproject.ID, email string) (bool, error)
}

type ApprovalHandler struct {
	approvals ApprovalService
	access    ApprovalAccessChecker
}

func NewApprovalHandler(approvals ApprovalService, access ApprovalAccessChecker) *ApprovalHandler {
	return &ApprovalHandler{
		approvals: approvals,
		access:    access,
	}
}

func (h *ApprovalHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/agent-api/approvals", h.HandleAgentRequest)
	mux.HandleFunc("/api/approvals/", h.HandleUserDecision)
}

// HandleAgentRequest is called by the container-side gate hook. It is
// authorized by the per-run grant token, never by a browser session.
func (h *ApprovalHandler) HandleAgentRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httptransport.SendErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if r.URL.Path != "/agent-api/approvals" {
		httptransport.SendErr(w, http.StatusNotFound, "not found")
		return
	}

	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		httptransport.SendErr(w, http.StatusUnauthorized, "invalid or expired approval grant")
		return
	}
	grant, err := h.approvals.ResolveGrant(token)
	if err != nil {
		httptransport.SendErr(w, http.StatusUnauthorized, "invalid or expired approval grant")
		return
	}

	var body struct {
		Tool    string `json:"tool"`
		Command string `json:"command"`
	}
	if err := decodeApprovalBody(r, &body); err != nil {
		httptransport.SendErr(w, http.StatusBadRequest, err.Error())
		return
	}
	command := strings.TrimSpace(body.Command)
	if command == "" {
		httptransport.SendErr(w, http.StatusBadRequest, "command is required")
		return
	}

	reason := serviceapproval.Reason(command)
	if reason == "" {
		httptransport.SendJSON(w, http.StatusOK, map[string]string{"decision": "allow"})
		return
	}

	decision, waitErr := h.approvals.RequestApproval(r.Context(), serviceapproval.Request{
		ChatID:    grant.ChatID,
		ProjectID: grant.ProjectID,
		Tool:      body.Tool,
		Command:   command,
		Reason:    reason,
	})
	response := map[string]string{"decision": string(decision), "reason": reason}
	if errors.Is(waitErr, serviceapproval.ErrExpired) {
		httptransport.SendJSON(w, http.StatusOK, response)
		return
	}
	if waitErr != nil {
		httptransport.SendErr(w, http.StatusInternalServerError, waitErr.Error())
		return
	}
	httptransport.SendJSON(w, http.StatusOK, response)
}

// HandleUserDecision resolves a pending approval from the web UI. The caller
// must be authenticated and a member of the approval's project (or an admin).
func (h *ApprovalHandler) HandleUserDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httptransport.SendErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/approvals/")
	rest = strings.TrimSuffix(rest, "/decision")
	if rest == "" || strings.Contains(rest, "/") {
		httptransport.SendErr(w, http.StatusNotFound, "not found")
		return
	}
	id := rest

	email, isAdmin, err := h.access.CallerAndAdmin(r.Context(), r)
	if err != nil || email == "" {
		httptransport.SendErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	pending, ok := h.approvals.Pending(id)
	if !ok {
		httptransport.SendErr(w, http.StatusNotFound, "approval not found")
		return
	}
	if !isAdmin {
		ok, err := h.access.HasAccess(r.Context(), pending.ProjectID, email)
		if err != nil {
			httptransport.SendErr(w, http.StatusInternalServerError, "access check failed")
			return
		}
		if !ok {
			httptransport.SendErr(w, http.StatusForbidden, "not a member of this project")
			return
		}
	}

	var body struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason,omitempty"`
	}
	if err := decodeApprovalBody(r, &body); err != nil {
		httptransport.SendErr(w, http.StatusBadRequest, err.Error())
		return
	}
	decision := serviceapproval.Decision(strings.ToLower(strings.TrimSpace(body.Decision)))
	if decision != serviceapproval.DecisionAllow && decision != serviceapproval.DecisionDeny {
		httptransport.SendErr(w, http.StatusBadRequest, "decision must be allow or deny")
		return
	}

	if _, err := h.approvals.Decide(r.Context(), id, decision, email); err != nil {
		httptransport.SendErr(w, http.StatusNotFound, err.Error())
		return
	}
	httptransport.SendJSON(w, http.StatusOK, map[string]string{"decision": string(decision)})
}

func decodeApprovalBody(r *http.Request, out any) error {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, approvalRequestLimit))
	if err != nil {
		return errors.New("read body: " + err.Error())
	}
	if len(raw) == 0 {
		return errors.New("request body is required")
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return errors.New("invalid JSON body")
	}
	return nil
}
