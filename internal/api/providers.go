package api

import (
	"net/http"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// listProviders exposes the integration manifests so the console and the
// policy engine can render capabilities without provider-specific code.
func (s *Server) listProviders(w http.ResponseWriter, r *http.Request) {
	kind := integration.Kind(r.URL.Query().Get("kind"))
	items := s.integrations.List()
	if kind != "" {
		filtered := items[:0]
		for _, m := range items {
			if m.Kind == kind {
				filtered = append(filtered, m)
			}
		}
		items = filtered
	}
	if items == nil {
		items = []integration.Manifest{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
