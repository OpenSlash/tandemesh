package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type agentMessageRequest struct {
	FromRunID  string `json:"fromRunId"`
	To         string `json:"to"`
	Type       string `json:"type"`
	Subject    string `json:"subject"`
	Body       string `json:"body"`
	DispatchID string `json:"dispatchId"`
}

func (s *Server) listTaskMessages(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	items, err := s.store.ListAgentMessages(r.Context(), task.ID, r.URL.Query().Get("runId"), r.URL.Query().Get("status"), 100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": items})
}

func (s *Server) sendTaskMessage(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	var req agentMessageRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if strings.TrimSpace(req.To) == "" {
		writeErr(w, http.StatusBadRequest, "E_TO", "to is required")
		return
	}
	if req.Type == "" {
		req.Type = "status"
	}
	runs, err := s.store.ListRunsByTask(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	targets := make([]*domain.Run, 0)
	switch {
	case req.To == "@all" || req.To == "@idle":
		for _, run := range runs {
			if run.ID == req.FromRunID {
				continue
			}
			if req.To == "@idle" && (run.Status == domain.RunRunning || run.Status == domain.RunPreparing) {
				continue
			}
			targets = append(targets, run)
		}
	case strings.HasPrefix(req.To, "@agent:"):
		agentID := strings.TrimPrefix(req.To, "@agent:")
		for _, run := range runs {
			if run.ID != req.FromRunID && run.AgentID == agentID {
				targets = append(targets, run)
			}
		}
	default:
		for _, run := range runs {
			if run.ID == req.To {
				targets = append(targets, run)
			}
		}
	}
	if len(targets) == 0 {
		writeErr(w, http.StatusNotFound, "E_RECIPIENT", "no matching run for recipient")
		return
	}
	sender := req.FromRunID
	if sender == "" {
		sender = userOf(r)
		if sender == "" {
			sender = "human"
		}
	}
	sent := make([]*domain.AgentMessage, 0, len(targets))
	for _, target := range targets {
		m := &domain.AgentMessage{
			WorkspaceID: task.WorkspaceID,
			TaskID:      task.ID,
			RunID:       target.ID,
			DispatchID:  req.DispatchID,
			Sender:      sender,
			Recipient:   target.ID,
			Type:        req.Type,
			Subject:     req.Subject,
			Body:        req.Body,
			Status:      "unread",
		}
		if err := s.store.SendAgentMessage(r.Context(), m); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		sent = append(sent, m)
	}
	s.recordTaskEvent(r.Context(), task.ID, "task.message.sent", map[string]any{
		"from": sender, "to": req.To, "type": req.Type, "count": len(sent),
	})
	writeJSON(w, http.StatusCreated, map[string]any{"messages": sent})
}

func (s *Server) checkTaskMessage(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	runID := r.URL.Query().Get("runId")
	if runID == "" {
		writeErr(w, http.StatusBadRequest, "E_RUN", "runId is required")
		return
	}
	m, err := s.store.CheckAgentMessage(r.Context(), runID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if m == nil {
		writeJSON(w, http.StatusOK, map[string]any{"messages": []*domain.AgentMessage{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": []*domain.AgentMessage{m}})
}

func (s *Server) ackTaskMessage(w http.ResponseWriter, r *http.Request) {
	if err := s.store.AckAgentMessage(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
