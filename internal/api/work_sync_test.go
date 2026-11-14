package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

type stubReader struct{}

func (stubReader) ListAssigned(context.Context, integration.Credential, integration.ListOptions) ([]integration.WorkItem, error) {
	return []integration.WorkItem{{
		Ref: domain.WorkRef{
			Provider: "github", Instance: "acme/api", Type: "issue", ID: "11",
			URL: "https://github.com/acme/api/issues/11", Revision: "r1",
		},
		Title: "sync me", Labels: []string{"agent:fix"}, Assignee: "alice", Updated: time.Now().UTC(),
	}}, nil
}

func TestSyncWork(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "sync.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	reg := integration.NewRegistry()
	if err := reg.Register(&integration.Integration{
		Manifest: integration.Manifest{ID: "github", Kind: integration.KindTask},
		Reader:   stubReader{},
	}); err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{GitHubToken: "tok"}, st, nil, hub.New(), reg, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]string{"provider": "github"})
	resp, err := http.Post(ts.URL+"/api/v1/work/sync", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !bytes.Contains(raw, []byte(`"synced":1`)) {
		t.Fatalf("status = %d body=%s", resp.StatusCode, raw)
	}
	tasks, _ := st.ListTasks(t.Context(), store.TaskFilter{Limit: 10})
	if len(tasks) != 1 || tasks[0].Title != "sync me" {
		t.Fatalf("tasks = %+v", tasks)
	}
}
