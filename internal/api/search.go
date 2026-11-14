package api

import (
	"net/http"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/store"
)

// search is a small global lookup for the command palette.
func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	if query == "" {
		writeJSON(w, http.StatusOK, map[string]any{"tasks": []any{}, "groups": []any{}})
		return
	}
	tasks, err := s.store.ListTasks(r.Context(), store.TaskFilter{
		Workspace: workspaceID,
		Query:     query,
		Limit:     20,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	groups, err := s.store.SearchGroups(r.Context(), workspaceID, query, 10)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks, "groups": groups})
}
