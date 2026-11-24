package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func newProfileServer(t *testing.T, cfg config.Config) *httptest.Server {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "profile.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	cfg.User = "me"
	srv := New(cfg, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)
	return ts
}

func TestCapabilitiesLocal(t *testing.T) {
	ts := newProfileServer(t, config.Config{Profile: "local", OAuthRedirect: "auto", Listen: ":8080"})
	resp, err := http.Get(ts.URL + "/api/v1/capabilities")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["profile"] != "local" || body["oauthRedirect"] != "http://127.0.0.1:8080" {
		t.Fatalf("body = %+v", body)
	}
}

func TestPublicProfileRequiresBasicAuth(t *testing.T) {
	ts := newProfileServer(t, config.Config{
		Profile: "public", AuthMode: "basic", AdminPassword: "secret",
		PublicURL: "https://example.test", OAuthRedirect: "public", Listen: ":8080",
	})
	resp, err := http.Get(ts.URL + "/api/v1/workspaces")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/workspaces", nil)
	req.SetBasicAuth("admin", "secret")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authorized status = %d", resp.StatusCode)
	}
}
