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
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestProviders(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "providers.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := New(config.Config{}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/api/v1/providers")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Items []integration.Manifest `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Items) != 8 {
		t.Fatalf("providers = %+v", out.Items)
	}
	var github *integration.Manifest
	for i := range out.Items {
		if out.Items[i].ID == "github" {
			github = &out.Items[i]
		}
	}
	if github == nil || github.Kind != integration.KindTask {
		t.Fatalf("github provider = %+v", github)
	}

	resp2, err := http.Get(ts.URL + "/api/v1/providers?kind=chat")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var chats struct {
		Items []integration.Manifest `json:"items"`
	}
	_ = json.NewDecoder(resp2.Body).Decode(&chats)
	if len(chats.Items) != 2 {
		t.Fatalf("chat providers = %+v", chats.Items)
	}
}
