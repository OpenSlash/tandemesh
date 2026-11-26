package api

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

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

// acpIcon serves a registry agent's icon from the plane's origin. The
// registry CDN sends no CORS headers, so the UI could not tint its
// currentColor SVGs, and fetching through the plane lets icons work offline.
func (s *Server) acpIcon(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !registryIDPattern.MatchString(id) {
		writeErr(w, http.StatusBadRequest, "E_REGISTRY_ID", "invalid registry id")
		return
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	raw, err := agent.ACPIcon(r.Context(), id, filepath.Join(cacheDir, "tandemesh", "acp-icons"))
	if errors.Is(err, agent.ErrIconNotFound) {
		writeErr(w, http.StatusNotFound, "E_ICON", err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_ICON", err.Error())
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Opened directly, the SVG must not run anything in the plane's origin.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	_, _ = w.Write(raw)
}
