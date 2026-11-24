package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// runDiff returns the latest unified diff reported by the agent daemon.
func (s *Server) runDiff(w http.ResponseWriter, r *http.Request) {
	run, err := s.store.GetRun(r.Context(), chi.URLParam(r, "id"))
	if err != nil || run == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "run not found")
		return
	}
	if !s.requireWorkspace(w, r, run.WorkspaceID) {
		return
	}
	events, err := s.store.ListEventsTail(r.Context(), run.ID, 200)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	diff := ""
	truncated := false
	for _, ev := range events {
		if ev.Type != "workspace.diff" {
			continue
		}
		diff, _ = ev.Payload["diff"].(string)
		truncated, _ = ev.Payload["truncated"].(bool)
		break
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"runId":     run.ID,
		"diff":      diff,
		"truncated": truncated,
	})
}
