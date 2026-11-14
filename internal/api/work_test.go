package api

import (
	"bytes"
	"encoding/json"
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

func TestImportWork(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "work.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "9",
		"url": "https://github.com/acme/api/issues/9", "revision": "r1",
		"title": "fix build", "body": "details", "labels": []string{"agent:fix"}, "assignee": "alice",
	})
	resp, err := http.Post(ts.URL+"/api/v1/work/import", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d body=%s", resp.StatusCode, raw)
	}
	var task domain.Task
	if err := json.Unmarshal(raw, &task); err != nil {
		t.Fatal(err)
	}
	if task.Title != "fix build" || task.Provider != "github" || task.Status != domain.TaskOpen {
		t.Fatalf("task = %+v", task)
	}
	listed, _ := st.ListTasks(t.Context(), store.TaskFilter{Limit: 10})
	if len(listed) != 1 || listed[0].ID != task.ID {
		t.Fatalf("listed = %+v", listed)
	}
}
