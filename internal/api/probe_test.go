package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/integration/github"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestProbeProvider(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "probe.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}

	var auth string
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if r.URL.Path != "/user" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"login":"orca"}`))
	}))
	defer fake.Close()

	adapter := &github.Adapter{BaseURL: fake.URL, HTTPClient: fake.Client()}
	reg := integration.NewRegistry()
	if err := reg.Register(&integration.Integration{
		Manifest: integration.Manifest{ID: "github", Name: "GitHub", Kind: integration.KindTask},
		Probe:    adapter,
	}); err != nil {
		t.Fatal(err)
	}

	srv := New(config.Config{GitHubToken: "tok"}, st, nil, hub.New(), reg, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	resp, err := http.Post(ts.URL+"/api/v1/providers/github/probe", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.StatusCode, raw)
	}
	if auth != "Bearer tok" {
		t.Fatalf("auth = %q", auth)
	}
	if !strings.Contains(string(raw), `"ok":true`) {
		t.Fatalf("body = %s", raw)
	}
}
