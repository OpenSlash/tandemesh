package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// mergeConnection moves a source connection's projects into a target connection.
func (s *Server) mergeConnection(w http.ResponseWriter, r *http.Request) {
	target, err := s.store.GetConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil || target == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "target connection not found")
		return
	}
	source, err := s.store.GetConnection(r.Context(), chi.URLParam(r, "source"))
	if err != nil || source == nil || source.Provider != target.Provider || source.WorkspaceID != target.WorkspaceID || source.ID == target.ID {
		writeErr(w, http.StatusBadRequest, "E_SOURCE", "source connection is not compatible")
		return
	}
	if !s.requireWorkspace(w, r, target.WorkspaceID) {
		return
	}
	existing := map[string]bool{}
	targetBindings, _ := s.store.ListBindings(r.Context(), target.WorkspaceID, target.ID)
	for _, binding := range targetBindings {
		existing[binding.ExternalID] = true
	}
	moved := 0
	if source.Instance != "" && !existing[source.Instance] {
		_ = s.store.UpsertBinding(r.Context(), &domain.ResourceBinding{
			ID: "bind_" + ulid.Make().String(), WorkspaceID: target.WorkspaceID,
			ConnectionID: target.ID, Provider: target.Provider, ExternalID: source.Instance,
			Name: source.Instance, Kind: "resource", Direction: source.SyncDirection, Enabled: true,
		})
		existing[source.Instance] = true
		moved++
	}
	sourceBindings, _ := s.store.ListBindings(r.Context(), source.WorkspaceID, source.ID)
	for _, binding := range sourceBindings {
		if existing[binding.ExternalID] {
			continue
		}
		binding.ConnectionID = target.ID
		binding.WorkspaceID = target.WorkspaceID
		if err := s.store.UpsertBinding(r.Context(), binding); err == nil {
			existing[binding.ExternalID] = true
			moved++
		}
	}
	if automations, err := s.store.ListAutomations(r.Context(), source.WorkspaceID); err == nil {
		for _, automation := range automations {
			if automation.ConnectionID == source.ID {
				_, _ = s.store.DeleteAutomation(r.Context(), automation.ID)
			}
		}
	}
	_, _ = s.store.DeleteConnection(r.Context(), source.ID)
	if s.vault != nil {
		_, _ = s.vault.Delete(r.Context(), source.Provider, source.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"moved": moved, "connection": target})
}
