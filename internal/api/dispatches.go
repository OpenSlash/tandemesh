package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/store"
)

// ---- dispatches ----

type dispatchRequest struct {
	RunID   string `json:"runId"`
	AgentID string `json:"agentId"`
	Attempt int    `json:"attempt"`
}

func (s *Server) listTaskDispatches(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	items, err := s.store.ListDispatchesByTask(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dispatches": items})
}

func (s *Server) createTaskDispatch(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	var req dispatchRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.RunID == "" {
		writeErr(w, http.StatusBadRequest, "E_RUN", "runId is required")
		return
	}
	if req.AgentID == "" {
		req.AgentID = task.AssigneeAgentID
	}
	if req.Attempt <= 0 {
		req.Attempt = 1
	}
	d, err := s.store.CreateDispatch(r.Context(), task, req.RunID, req.AgentID, req.Attempt)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (s *Server) getDispatch(w http.ResponseWriter, r *http.Request) {
	d, err := s.store.GetDispatch(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if d == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "dispatch not found")
		return
	}
	task, _ := s.store.GetTask(r.Context(), d.TaskID)
	if !s.requireTask(w, r, task) {
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) startDispatch(w http.ResponseWriter, r *http.Request) {
	d, err := s.store.GetDispatch(r.Context(), chi.URLParam(r, "id"))
	if err != nil || d == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "dispatch not found")
		return
	}
	task, _ := s.store.GetTask(r.Context(), d.TaskID)
	if !s.requireTask(w, r, task) {
		return
	}
	if err := s.store.StartDispatch(r.Context(), d.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	d.Status = domain.DispatchRunning
	writeJSON(w, http.StatusOK, d)
}

type completeDispatchRequest struct {
	TaskID        string   `json:"taskId"`
	Outcome       string   `json:"outcome"`
	Result        string   `json:"result"`
	FilesModified []string `json:"filesModified"`
}

func (s *Server) completeDispatch(w http.ResponseWriter, r *http.Request) {
	d, err := s.store.GetDispatch(r.Context(), chi.URLParam(r, "id"))
	if err != nil || d == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "dispatch not found")
		return
	}
	task, _ := s.store.GetTask(r.Context(), d.TaskID)
	if !s.requireTask(w, r, task) {
		return
	}
	var req completeDispatchRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.TaskID != "" && req.TaskID != d.TaskID {
		writeErr(w, http.StatusBadRequest, "E_TASK", "taskId does not match dispatch")
		return
	}
	if req.Outcome != "succeeded" && req.Outcome != "failed" {
		writeErr(w, http.StatusBadRequest, "E_OUTCOME", "outcome must be succeeded or failed")
		return
	}
	updated, promoted, err := s.store.CompleteDispatch(r.Context(), d.ID, d.TaskID, req.Outcome, req.Result, req.FilesModified)
	if errors.Is(err, store.ErrOpenGates) {
		writeErr(w, http.StatusConflict, "E_GATE", "decision gates are still open")
		return
	}
	if errors.Is(err, store.ErrDispatchDone) {
		writeJSON(w, http.StatusOK, map[string]any{"task": updated, "duplicate": true})
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if d.RunID != "" {
		_, _ = s.store.AppendEvent(r.Context(), d.RunID, "dispatch.completed", map[string]any{
			"dispatchId": d.ID, "taskId": d.TaskID, "outcome": req.Outcome,
		})
	}
	s.recordTaskEvent(r.Context(), d.TaskID, "task.dispatch.completed", map[string]any{
		"dispatchId": d.ID, "outcome": req.Outcome, "filesModified": req.FilesModified,
	})
	if req.Outcome == "failed" {
		s.pushNotification(r.Context(), d.WorkspaceID, "dispatch.failed", "任务执行失败："+task.Title, req.Result, "/tasks/"+d.TaskID)
	}
	for _, pid := range promoted {
		s.recordTaskEvent(r.Context(), pid, "task.promoted", map[string]any{"completed": d.TaskID})
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": updated, "promoted": promoted})
}

// ---- decision gates ----

type gateRequest struct {
	Question   string   `json:"question"`
	Options    []string `json:"options"`
	DispatchID string   `json:"dispatchId"`
}

func (s *Server) listGates(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	items, err := s.store.ListGates(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"gates": items})
}

func (s *Server) createGate(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	var req gateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if strings.TrimSpace(req.Question) == "" {
		writeErr(w, http.StatusBadRequest, "E_QUESTION", "question is required")
		return
	}
	g, err := s.store.CreateGate(r.Context(), task.ID, req.DispatchID, req.Question, req.Options)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	_ = s.store.SetTaskExecutionState(r.Context(), task.ID, domain.TaskExecutionBlocked)
	s.recordTaskEvent(r.Context(), task.ID, "task.gate.created", map[string]any{"gateId": g.ID, "question": req.Question})
	writeJSON(w, http.StatusCreated, g)
}

type resolveGateRequest struct {
	Resolution string `json:"resolution"`
}

func (s *Server) resolveGate(w http.ResponseWriter, r *http.Request) {
	g, err := s.store.GetGate(r.Context(), chi.URLParam(r, "id"))
	if err != nil || g == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "gate not found")
		return
	}
	task, _ := s.store.GetTask(r.Context(), g.TaskID)
	if !s.requireTask(w, r, task) {
		return
	}
	var req resolveGateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if strings.TrimSpace(req.Resolution) == "" {
		writeErr(w, http.StatusBadRequest, "E_RESOLUTION", "resolution is required")
		return
	}
	updated, err := s.store.ResolveGate(r.Context(), g.ID, req.Resolution)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if n, _ := s.store.OpenGateCount(r.Context(), g.TaskID); n == 0 {
		_ = s.store.SetTaskExecutionState(r.Context(), g.TaskID, domain.TaskExecutionDispatched)
	}
	s.recordTaskEvent(r.Context(), g.TaskID, "task.gate.resolved", map[string]any{"gateId": g.ID, "resolution": req.Resolution})
	writeJSON(w, http.StatusOK, updated)
}
