package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// connectionSyncStatus reports the last successful sync and error for a connection.
func (s *Server) connectionSyncStatus(w http.ResponseWriter, r *http.Request) {
	conn, err := s.store.GetConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil || conn == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "connection not found")
		return
	}
	if !s.requireWorkspace(w, r, conn.WorkspaceID) {
		return
	}
	cursor, err := s.store.GetSyncCursor(r.Context(), conn.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	var webhook any
	if delivery, err := s.store.LatestWebhookDelivery(r.Context(), conn.Provider, conn.AutomationID); err == nil {
		webhook = delivery
	}
	writeJSON(w, http.StatusOK, map[string]any{"cursor": cursor, "webhook": webhook})
}
