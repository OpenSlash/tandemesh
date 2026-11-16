package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// registerBindingWebhook registers one webhook for a project binding.
func (s *Server) registerBindingWebhook(w http.ResponseWriter, r *http.Request) {
	conn, err := s.store.GetConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil || conn == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "connection not found")
		return
	}
	if !s.requireWorkspace(w, r, conn.WorkspaceID) {
		return
	}
	binding, err := s.store.GetBinding(r.Context(), chi.URLParam(r, "bindingId"))
	if err != nil || binding == nil || binding.ConnectionID != conn.ID {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "binding not found")
		return
	}
	ad, ok := s.integrations.Get(conn.Provider)
	if !ok || ad.Registrar == nil {
		writeErr(w, http.StatusBadRequest, "E_UNSUPPORTED", "provider does not support webhook registration")
		return
	}
	automationID := ""
	if binding.Settings != nil {
		automationID = binding.Settings["automationId"]
	}
	if automationID == "" {
		if automations, err := s.store.ListAutomations(r.Context(), conn.WorkspaceID); err == nil {
			for _, automation := range automations {
				if automation.ConnectionID == conn.ID && automation.Instance == binding.ExternalID {
					automationID = automation.ID
					break
				}
			}
		}
	}
	if automationID == "" {
		writeErr(w, http.StatusBadRequest, "E_AUTOMATION", "binding has no automation")
		return
	}
	secret := ""
	if s.vault != nil {
		if stored, ok, err := s.vault.Ensure(r.Context(), conn.Provider, "webhook:"+conn.ID); err == nil && ok {
			secret = stored.AccessToken
		}
	}
	url := strings.TrimRight(s.publicBaseURL(), "/") + "/webhooks/" + conn.Provider + "/" + automationID
	webhookID, err := ad.Registrar.RegisterWebhook(r.Context(), s.providerCredential(r.Context(), conn.Provider, conn.ID), binding.ExternalID, url, secret)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_WEBHOOK", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"webhookId": webhookID, "url": url, "bindingId": binding.ID})
}
