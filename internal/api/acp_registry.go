package api

import (
	"net/http"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/agent"
)

func (s *Server) acpRegistry(w http.ResponseWriter, r *http.Request) {
	snapshot, err := agent.FetchACPRegistry(r.Context())
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_REGISTRY", err.Error())
		return
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	items := make([]agent.ACPRegistryEntry, 0, len(snapshot.Items))
	for _, item := range snapshot.Items {
		if query == "" || strings.Contains(strings.ToLower(item.ID+" "+item.Name+" "+item.Description), query) {
			items = append(items, item)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"source": snapshot.Source, "fetchedAt": snapshot.FetchedAt, "items": items,
	})
}
