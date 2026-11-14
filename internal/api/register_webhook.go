package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) connectionWebhookURL(w http.ResponseWriter, r *http.Request) {
	conn, err := s.store.GetConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil || conn == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "connection not found")
		return
	}
	if !s.requireWorkspace(w, r, conn.WorkspaceID) {
		return
	}
	automationID := conn.AutomationID
	if automationID == "" {
		automations, err := s.store.ListAutomations(r.Context(), conn.WorkspaceID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		for _, item := range automations {
			if item.ConnectionID == conn.ID {
				automationID = item.ID
				break
			}
		}
	}
	if automationID == "" {
		writeErr(w, http.StatusBadRequest, "E_AUTOMATION", "connection has no task automation")
		return
	}
	url := strings.TrimRight(s.cfg.BaseURL, "/") + "/webhooks/" + conn.Provider + "/" + automationID
	writeJSON(w, http.StatusOK, map[string]any{"url": url, "automationId": automationID})
}

func (s *Server) registerConnectionWebhook(w http.ResponseWriter, r *http.Request) {
	conn, err := s.store.GetConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil || conn == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "connection not found")
		return
	}
	if !s.requireWorkspace(w, r, conn.WorkspaceID) {
		return
	}
	ad, ok := s.integrations.Get(conn.Provider)
	if !ok || ad.Registrar == nil {
		writeErr(w, http.StatusBadRequest, "E_UNSUPPORTED", "provider does not support automatic webhook registration")
		return
	}
	automations, err := s.store.ListAutomations(r.Context(), conn.WorkspaceID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	var automation *domain.Automation
	for _, item := range automations {
		if item.ConnectionID == conn.ID && item.Provider == conn.Provider {
			automation = item
			break
		}
	}
	if automation == nil {
		writeErr(w, http.StatusBadRequest, "E_AUTOMATION", "connection has no event automation")
		return
	}
	secret := ""
	if stored, ok, err := s.vault.Ensure(r.Context(), conn.Provider, "webhook:"+conn.ID); err == nil && ok {
		secret = stored.AccessToken
	}
	url := strings.TrimRight(s.cfg.BaseURL, "/") + "/webhooks/" + conn.Provider + "/" + automation.ID
	webhookID, err := ad.Registrar.RegisterWebhook(r.Context(), s.providerCredential(r.Context(), conn.Provider, conn.ID), conn.Instance, url, secret)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_WEBHOOK", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"webhookId": webhookID, "url": url})
}
