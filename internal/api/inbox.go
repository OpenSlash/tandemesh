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

type inboxBatchRequest struct {
	IDs []string `json:"ids"`
	// Closed picks every done or cancelled task in Workspace ("all" or an
	// id) that is in From (default inbox), instead of IDs. It reaches tasks
	// the page has not loaded.
	Closed    bool   `json:"closed"`
	Workspace string `json:"workspace"`
	From      string `json:"from"`
	// State moves the tasks (inbox, later, cleared); Read marks them read.
	State string `json:"state"`
	Read  bool   `json:"read"`
}

// batchTaskInbox triages many tasks at once for the acting user.
func (s *Server) batchTaskInbox(w http.ResponseWriter, r *http.Request) {
	var req inboxBatchRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	switch req.State {
	case "", "inbox", "later", "cleared":
	default:
		writeErr(w, http.StatusBadRequest, "E_STATE", "state must be inbox, later or cleared")
		return
	}
	if req.State == "" && !req.Read {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "state or read is required")
		return
	}
	user := s.actingUser(r)
	ctx := r.Context()
	ids := req.IDs
	if req.Closed {
		workspace := req.Workspace
		if workspace == "all" {
			workspace = ""
		}
		closed, err := s.store.ListClosedTasks(ctx, workspace)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		from := req.From
		if from == "" {
			from = "inbox"
		}
		all := make([]string, 0, len(closed))
		for id := range closed {
			all = append(all, id)
		}
		states, err := s.store.TaskInboxStates(ctx, user, all)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		ids = ids[:0:0]
		for id := range closed {
			current := "inbox"
			if st, ok := states[id]; ok && st.State != "" {
				current = st.State
			}
			if current == from {
				ids = append(ids, id)
			}
		}
	}
	updated := 0
	for _, id := range ids {
		task, err := s.store.GetTask(ctx, id)
		if err != nil || task == nil || !s.allowWorkspace(ctx, task.WorkspaceID) {
			continue
		}
		if req.State != "" {
			if err := s.store.SetTaskInboxState(ctx, id, user, req.State); err != nil {
				writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
				return
			}
		}
		if req.Read {
			if err := s.store.MarkTaskRead(ctx, id, user); err != nil {
				writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
				return
			}
		}
		updated++
	}
	writeJSON(w, http.StatusOK, map[string]int{"updated": updated})
}
