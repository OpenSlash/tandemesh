package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// updateController lets the web UI check for and install updates through
// the desktop server, alongside the tray menu and the background check.
type updateController struct {
	mu      sync.Mutex
	app     *application.App
	pending *updateManifest
}

type updateStatus struct {
	Version   string `json:"version"`
	Supported bool   `json:"supported"`
	Available string `json:"available,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

func (c *updateController) setApp(app *application.App) {
	c.mu.Lock()
	c.app = app
	c.mu.Unlock()
}

func (c *updateController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// A custom header forces a CORS preflight that this server never
	// answers, so other sites in a browser cannot trigger an install.
	if r.Header.Get("X-Tandemesh-Desktop") != "1" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	switch {
	case r.URL.Path == "/desktop/update" && r.Method == http.MethodGet:
		c.check(w, r)
	case r.URL.Path == "/desktop/update/install" && r.Method == http.MethodPost:
		c.install(w)
	default:
		http.NotFound(w, r)
	}
}

func (c *updateController) check(w http.ResponseWriter, r *http.Request) {
	status := updateStatus{Version: version}
	url := updateManifestURL()
	if url == "" {
		writeJSON(w, http.StatusOK, status)
		return
	}
	status.Supported = true
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	manifest, err := fetchUpdate(ctx, url)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	c.mu.Lock()
	c.pending = manifest
	c.mu.Unlock()
	if manifest != nil {
		status.Available = manifest.Version
		status.Notes = manifest.Notes
	}
	writeJSON(w, http.StatusOK, status)
}

// install downloads and hands over the update found by the last check, then
// quits so the installer can replace the app and relaunch it.
func (c *updateController) install(w http.ResponseWriter) {
	c.mu.Lock()
	manifest, app := c.pending, c.app
	c.mu.Unlock()
	if manifest == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "no update available; check first"})
		return
	}
	pkg, err := downloadUpdate(context.Background(), manifest)
	if err == nil {
		err = installUpdate(pkg)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"version": manifest.Version})
	if app != nil {
		// Let the response reach the page before the app goes away.
		time.AfterFunc(500*time.Millisecond, app.Quit)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
