package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
)

// listApprovals returns approvals, pending by default.
func (s *Server) listApprovals(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = domain.ApprovalPending
	}
	workspaceID := r.URL.Query().Get("workspace")
	all := workspaceID == "all"
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !all && !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	lookup := workspaceID
	if all {
		lookup = ""
	}
	items, err := s.store.ListApprovals(r.Context(), status, lookup)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if all {
		filtered := make([]*domain.Approval, 0, len(items))
		for _, item := range items {
			if s.allowWorkspace(r.Context(), item.WorkspaceID) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if items == nil {
		items = []*domain.Approval{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type createApprovalRequest struct {
	ID      string `json:"id"`
	RunID   string `json:"runId"`
	Tool    string `json:"tool"`
	Command string `json:"command"`
	Risk    string `json:"risk"`
	TTL     string `json:"ttl"`
}

// createApproval records a pending approval. The daemon normally creates these
// over the WebSocket, but the endpoint is useful for local testing and tooling.
func (s *Server) createApproval(w http.ResponseWriter, r *http.Request) {
	var req createApprovalRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.RunID == "" || req.Tool == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "runId and tool are required")
		return
	}
	run, err := s.store.GetRun(r.Context(), req.RunID)
	if err != nil || run == nil {
		writeErr(w, http.StatusNotFound, "E_RUN", "run not found")
		return
	}
	ttl := 15 * time.Minute
	if req.TTL != "" {
		if d, err := time.ParseDuration(req.TTL); err == nil && d > 0 {
			ttl = d
		}
	}
	if req.Risk == "" {
		req.Risk = "medium"
	}
	approvalID := req.ID
	if approvalID == "" {
		approvalID = "apr_" + ulid.Make().String()
	} else if !strings.HasPrefix(approvalID, "apr_") || len(approvalID) > 64 {
		writeErr(w, http.StatusBadRequest, "E_ID", "invalid approval id")
		return
	}
	approval := &domain.Approval{
		ID:          approvalID,
		RunID:       req.RunID,
		Tool:        req.Tool,
		Command:     req.Command,
		Risk:        req.Risk,
		Status:      domain.ApprovalPending,
		RequestedAt: time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(ttl),
	}
	if err := s.store.CreateApproval(r.Context(), approval); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	_, _ = s.store.AppendEvent(r.Context(), req.RunID, "approval.requested", map[string]any{
		"approvalId": approval.ID,
		"tool":       approval.Tool,
		"risk":       approval.Risk,
	})
	s.notifyApproval(r.Context(), approval)
	writeJSON(w, http.StatusCreated, approval)
}

// getApproval returns one approval so a daemon can poll for the decision.
func (s *Server) getApproval(w http.ResponseWriter, r *http.Request) {
	approval, err := s.store.GetApproval(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if approval == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "approval not found")
		return
	}
	writeJSON(w, http.StatusOK, approval)
}

type decisionRequest struct {
	Decision string `json:"decision"`
	By       string `json:"by"`
}

// recordApprovalTranscript appends the decision to the session conversation.
func (s *Server) recordApprovalTranscript(ctx context.Context, approval *domain.Approval, decision, by string) {
	run, err := s.store.GetRun(ctx, approval.RunID)
	if err != nil || run == nil || run.SessionID == "" {
		return
	}
	label := "Approved"
	if decision == domain.ApprovalDenied {
		label = "Denied"
	}
	text := fmt.Sprintf("%s %s by %s", label, approval.Tool, by)
	if approval.Command != "" {
		text += ": " + approval.Command
	}
	if _, err := s.store.AppendSessionMessage(ctx, &domain.SessionMessage{
		SessionID: run.SessionID, RunID: run.ID, Role: "system", Kind: "approval", Text: text,
	}); err != nil {
		s.log.Warn("record approval transcript", "approval", approval.ID, "err", err)
	}
}

// decideApproval records a decision and forwards it to the daemon.
func (s *Server) decideApproval(w http.ResponseWriter, r *http.Request) {
	var req decisionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	decision := strings.ToLower(req.Decision)
	if decision != domain.ApprovalApproved && decision != domain.ApprovalDenied {
		writeErr(w, http.StatusBadRequest, "E_DECISION", "decision must be approved or denied")
		return
	}
	id := chi.URLParam(r, "id")
	approval, err := s.store.GetApproval(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if approval == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "approval not found")
		return
	}
	by := req.By
	if by == "" {
		by = "api"
	}
	ok, err := s.store.DecideApproval(r.Context(), id, decision, by)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusConflict, "E_STATE", "approval is not pending")
		return
	}
	eventType := "approval.denied"
	if decision == domain.ApprovalApproved {
		eventType = "approval.granted"
	}
	_, _ = s.store.AppendEvent(r.Context(), approval.RunID, eventType, map[string]any{
		"approvalId": id,
		"by":         by,
	})
	s.recordApprovalTranscript(r.Context(), approval, decision, by)

	if run, err := s.store.GetRun(r.Context(), approval.RunID); err == nil && run != nil && run.MachineID != "" {
		payload, _ := json.Marshal(map[string]any{"approvalId": id, "decision": decision})
		s.hub.Send(run.MachineID, hub.Message{Type: "approval.decision", RunID: run.ID, Payload: payload})
	}
	s.updateApprovalCard(r.Context(), approval, by, decision)
	writeJSON(w, http.StatusOK, map[string]string{"status": decision})
}
