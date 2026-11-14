package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// retryRun clones a failed or cancelled run back into the queue.
func (s *Server) retryRun(w http.ResponseWriter, r *http.Request) {
	orig, err := s.store.GetRun(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if orig == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "run not found")
		return
	}
	if orig.Status != domain.RunFailed && orig.Status != domain.RunCancelled {
		writeErr(w, http.StatusConflict, "E_STATE", "only failed or cancelled runs can be retried")
		return
	}
	if !s.requireWorkspace(w, r, orig.WorkspaceID) {
		return
	}
	id := "run_" + ulid.Make().String()
	clone := *orig
	clone.ID = id
	clone.Status = domain.RunQueued
	clone.MachineID = ""
	clone.Fence = 0
	clone.Attempt = 0
	clone.ErrorCode = ""
	clone.ErrorMessage = ""
	clone.CreatedAt = time.Now().UTC()
	clone.StartedAt = nil
	clone.EndedAt = nil
	clone.LeaseExpiresAt = nil
	clone.Branch = "ap/" + id[len(id)-8:]
	clone.VariantOf = orig.ID
	clone.VariantLabel = "retry"
	if err := s.store.CreateRun(r.Context(), &clone); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	_, _ = s.store.AppendEvent(r.Context(), orig.ID, "run.retried", map[string]any{"newRunId": clone.ID})
	_, _ = s.store.AppendEvent(r.Context(), clone.ID, domain.EventRunCreated, map[string]any{
		"provider": clone.Work.Provider,
		"url":      clone.Work.URL,
		"retryOf":  orig.ID,
	})
	if orig.SessionID != "" {
		_, _ = s.store.AppendSessionMessage(r.Context(), &domain.SessionMessage{
			SessionID: orig.SessionID, RunID: clone.ID, Role: "system", Kind: "retry", Text: "Retried run",
		})
	}
	writeJSON(w, http.StatusCreated, &clone)
}
