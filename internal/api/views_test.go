package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{User: "alice"}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)
	return ts
}

func TestSavedViews(t *testing.T) {
	ts := newTestServer(t)

	view := postJSON[domain.SavedView](t, ts.URL+"/api/v1/workspaces/ws_default/views", map[string]any{
		"name":    "Agent fixes",
		"filters": map[string]string{"status": "open", "label": "agent:fix", "bogus": "x"},
	})
	if view.ID == "" || view.Name != "Agent fixes" || view.Filters["label"] != "agent:fix" {
		t.Fatalf("view = %+v", view)
	}
	if _, ok := view.Filters["bogus"]; ok {
		t.Fatalf("unknown filter accepted: %+v", view.Filters)
	}

	list := getJSON[struct {
		Items []domain.SavedView `json:"items"`
	}](t, ts.URL+"/api/v1/workspaces/ws_default/views")
	if len(list.Items) != 1 || list.Items[0].ID != view.ID {
		t.Fatalf("views = %+v", list.Items)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/workspaces/ws_default/views/"+view.ID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}
}
