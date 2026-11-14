package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user := userOf(r)
	if user == "" {
		user = s.cfg.User
	}
	writeJSON(w, http.StatusOK, map[string]string{"name": user})
}

type inboxRequest struct {
	State string `json:"state"`
}

// setTaskInbox moves a task between inbox, later and cleared.
func (s *Server) setTaskInbox(w http.ResponseWriter, r *http.Request) {
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
	var req inboxRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	switch req.State {
	case "inbox", "later", "cleared":
	default:
		writeErr(w, http.StatusBadRequest, "E_STATE", "state must be inbox, later or cleared")
		return
	}
	user := s.actingUser(r)
	from := "inbox"
	if states, err := s.store.TaskInboxStates(r.Context(), user, []string{task.ID}); err == nil {
		if state, ok := states[task.ID]; ok {
			from = state.State
		}
	}
	if err := s.store.SetTaskInboxState(r.Context(), task.ID, user, req.State); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.recordTaskEvent(r.Context(), task.ID, "task.inbox", map[string]any{
		"from": from, "to": req.State, "user": user,
	})
	_ = s.applyTaskInbox(r.Context(), user, []*domain.Task{task})
	writeJSON(w, http.StatusOK, task)
}

// markTaskRead records that the user opened the task.
func (s *Server) markTaskRead(w http.ResponseWriter, r *http.Request) {
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
	if err := s.store.MarkTaskRead(r.Context(), task.ID, s.actingUser(r)); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
