package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) listWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	automationID := chi.URLParam(r, "automation")
	if automationID != "" {
		automation, err := s.store.GetAutomation(r.Context(), automationID)
		if err != nil || automation == nil {
			writeErr(w, http.StatusNotFound, "E_AUTOMATION", "automation not found")
			return
		}
		if !s.requireWorkspace(w, r, automation.WorkspaceID) {
			return
		}
	}
	items, err := s.store.ListWebhookDeliveries(r.Context(), automationID, 50)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.WebhookDelivery{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
