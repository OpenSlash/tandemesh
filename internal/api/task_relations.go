package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/store"
)

type parentRequest struct {
	ParentID string `json:"parentId"`
}

func (s *Server) setTaskParent(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	var req parentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.ParentID != "" {
		parent, err := s.store.GetTask(r.Context(), req.ParentID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		if parent == nil || parent.WorkspaceID != task.WorkspaceID {
			writeErr(w, http.StatusBadRequest, "E_PARENT", "parent is not in this workspace")
			return
		}
		// Walk up to prevent a cycle.
		current := parent
		for current != nil {
			if current.ID == task.ID {
				writeErr(w, http.StatusBadRequest, "E_CYCLE", "parent relation would create a cycle")
				return
			}
			if current.ParentID == "" {
				break
			}
			current, _ = s.store.GetTask(r.Context(), current.ParentID)
		}
	}
	if err := s.store.SetTaskParent(r.Context(), task.ID, req.ParentID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.recordTaskEvent(r.Context(), task.ID, "task.parent", map[string]any{
		"from": task.ParentID, "to": req.ParentID,
	})
	task.ParentID = req.ParentID
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) listTaskChildren(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	items, err := s.store.ListTaskChildren(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) listTaskDependencies(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	ids, err := s.store.ListTaskDependencyIDs(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	items := make([]any, 0, len(ids))
	for _, id := range ids {
		dep, err := s.store.GetTask(r.Context(), id)
		if err != nil || dep == nil {
			continue
		}
		items = append(items, dep)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type dependencyRequest struct {
	DependsOn string `json:"dependsOn"`
}

func (s *Server) addTaskDependency(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	var req dependencyRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	dep, err := s.store.GetTask(r.Context(), req.DependsOn)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if dep == nil || dep.WorkspaceID != task.WorkspaceID {
		writeErr(w, http.StatusBadRequest, "E_DEPENDENCY", "dependency is not in this workspace")
		return
	}
	if err := s.store.AddTaskDependency(r.Context(), task.ID, dep.ID); err != nil {
		if errors.Is(err, store.ErrTaskCycle) {
			writeErr(w, http.StatusBadRequest, "E_CYCLE", "dependency would create a cycle")
			return
		}
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	_ = s.store.EvaluateTaskReadiness(r.Context(), task.ID)
	s.recordTaskEvent(r.Context(), task.ID, "task.dependency", map[string]any{"dependsOn": dep.ID})
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) removeTaskDependency(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	dependsOn := chi.URLParam(r, "dep")
	ok, err := s.store.RemoveTaskDependency(r.Context(), task.ID, dependsOn)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "dependency not found")
		return
	}
	_ = s.store.EvaluateTaskReadiness(r.Context(), task.ID)
	s.recordTaskEvent(r.Context(), task.ID, "task.dependency.removed", map[string]any{"dependsOn": dependsOn})
	w.WriteHeader(http.StatusNoContent)
}
