package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) listIterations(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListIterations(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.Iteration{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type iterationRequest struct {
	Name      string `json:"name"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Status    string `json:"status"`
}

func (s *Server) createIteration(w http.ResponseWriter, r *http.Request) {
	var req iterationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Name == "" {
		writeErr(w, http.StatusBadRequest, "E_NAME", "name is required")
		return
	}
	switch req.Status {
	case "", "planned", "active", "done":
	default:
		writeErr(w, http.StatusBadRequest, "E_STATUS", "status must be planned, active or done")
		return
	}
	if req.Status == "" {
		req.Status = "planned"
	}
	it := &domain.Iteration{
		ID: "it_" + ulid.Make().String(), WorkspaceID: chi.URLParam(r, "id"),
		Name: req.Name, StartDate: req.StartDate, EndDate: req.EndDate, Status: req.Status,
	}
	if err := s.store.CreateIteration(r.Context(), it); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, it)
}

type iterationPatch struct {
	Status string `json:"status"`
}

func (s *Server) patchIteration(w http.ResponseWriter, r *http.Request) {
	var req iterationPatch
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	switch req.Status {
	case "planned", "active", "done":
	default:
		writeErr(w, http.StatusBadRequest, "E_STATUS", "status must be planned, active or done")
		return
	}
	ok, err := s.store.SetIterationStatus(r.Context(), chi.URLParam(r, "id"), req.Status)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "iteration not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func (s *Server) iterationBurndown(w http.ResponseWriter, r *http.Request) {
	it, points, stats, err := s.store.IterationBurndown(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if it == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "iteration not found")
		return
	}
	if !s.requireWorkspace(w, r, it.WorkspaceID) {
		return
	}
	if points == nil {
		points = []domain.BurndownPoint{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"iteration": it, "points": points, "stats": stats})
}

func (s *Server) deleteIteration(w http.ResponseWriter, r *http.Request) {
	ok, err := s.store.DeleteIteration(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "iteration not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type taskIterationRequest struct {
	IterationID string `json:"iterationId"`
}

func (s *Server) setTaskIteration(w http.ResponseWriter, r *http.Request) {
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
	var req taskIterationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.IterationID != "" {
		it, err := s.store.GetIteration(r.Context(), req.IterationID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		if it == nil || it.WorkspaceID != task.WorkspaceID {
			writeErr(w, http.StatusBadRequest, "E_ITERATION", "iteration is not in this workspace")
			return
		}
	}
	if err := s.store.SetTaskIteration(r.Context(), task.ID, req.IterationID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.recordTaskEvent(r.Context(), task.ID, "task.iteration", map[string]any{
		"from": task.IterationID, "to": req.IterationID,
	})
	task.IterationID = req.IterationID
	writeJSON(w, http.StatusOK, task)
}
