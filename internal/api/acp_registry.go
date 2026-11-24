package api

import (
	"net/http"
	"os"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/agent"
)

func (s *Server) acpInstall(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID string `json:"id"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	home := os.Getenv("TANDEMESH_ACP_HOME")
	result, err := agent.InstallACP(r.Context(), body.ID, home)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_INSTALL", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) acpUninstall(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	version := r.URL.Query().Get("version")
	if err := agent.UninstallACP(id, version, os.Getenv("TANDEMESH_ACP_HOME")); err != nil {
		writeErr(w, http.StatusBadRequest, "E_UNINSTALL", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
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
