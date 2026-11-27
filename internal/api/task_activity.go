package api

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// taskActivityItem is one normalized entry in a task timeline.
type taskActivityItem struct {
	Source  string         `json:"source"`
	Type    string         `json:"type"`
	TS      time.Time      `json:"ts"`
	Actor   string         `json:"actor,omitempty"`
	RunID   string         `json:"runId,omitempty"`
	AgentID string         `json:"agentId,omitempty"`
	Seq     int            `json:"seq,omitempty"`
	Payload map[string]any `json:"payload,omitempty"`
}

// taskActivity merges task-level actions with per-run events.
func (s *Server) taskActivity(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}

	items := make([]taskActivityItem, 0, 16)
	events, err := s.store.ListTaskEvents(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	for _, ev := range events {
		items = append(items, taskActivityItem{
			Source: "task", Type: ev.Type, TS: ev.CreatedAt, Actor: ev.Actor, Payload: ev.Payload,
		})
	}

	runs, err := s.store.ListRunsByTask(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	for _, run := range runs {
		runEvents, err := s.store.ListEvents(r.Context(), run.ID, 0)
		if err != nil {
			continue
		}
		if len(runEvents) > 200 {
			runEvents = runEvents[len(runEvents)-200:]
		}
		for _, ev := range runEvents {
			items = append(items, taskActivityItem{
				Source: "run", Type: ev.Type, TS: ev.TS, RunID: run.ID,
				AgentID: run.AgentID, Seq: ev.Seq, Payload: ev.Payload,
			})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].TS.Before(items[j].TS) })

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset := -1
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, _ = strconv.Atoi(raw)
	}
	total := len(items)
	if offset < 0 {
		offset = total - limit
	}
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items[offset:end], "total": total, "offset": offset, "hasMore": offset > 0, "task": task,
	})
}

type taskPatchRequest struct {
	Status string `json:"status"`
}

// patchTask updates the local projection status.
func (s *Server) patchTask(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	var req taskPatchRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	switch req.Status {
	case domain.TaskOpen, domain.TaskInProgress, domain.TaskDone, domain.TaskCancelled:
	default:
		writeErr(w, http.StatusBadRequest, "E_STATUS", "status must be open, in_progress or done")
		return
	}
	if err := s.store.SetTaskStatus(r.Context(), task.ID, req.Status); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if domain.TaskClosed(req.Status) {
		now := time.Now().UTC()
		_ = s.store.SetTaskDoneAt(r.Context(), task.ID, &now)
		_ = s.store.SetTaskExecutionState(r.Context(), task.ID, domain.TaskExecutionCompleted)
		_, _ = s.store.PromoteDependents(r.Context(), task.ID)
	} else {
		_ = s.store.SetTaskDoneAt(r.Context(), task.ID, nil)
		_ = s.store.EvaluateTaskReadiness(r.Context(), task.ID)
	}
	s.recordTaskEvent(r.Context(), task.ID, "task.status", map[string]any{
		"from": task.Status, "to": req.Status,
	})
	task.Status = req.Status
	s.enqueueStatusWriteback(r.Context(), task, req.Status)
	writeJSON(w, http.StatusOK, task)
}

// recordTaskEvent appends an activity entry authored by the current user.
func (s *Server) recordTaskEvent(ctx context.Context, taskID, typ string, payload map[string]any) {
	actor := s.cfg.User
	if actor == "" {
		actor = "human"
	}
	_ = s.store.AddTaskEvent(ctx, &domain.TaskEvent{
		ID:      "te_" + ulid.Make().String(),
		TaskID:  taskID,
		Actor:   actor,
		Type:    typ,
		Payload: payload,
	})
}
