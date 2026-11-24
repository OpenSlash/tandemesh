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

func TestInboxStatesAndMyTasks(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "inbox.db"))
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

	task := postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "31",
		"title": "inbox state task", "assignee": "alice",
	})
	if task.InboxState != "inbox" || task.InboxReadAt != nil {
		t.Fatalf("new task = %+v", task)
	}

	body, _ := json.Marshal(map[string]string{"state": "later"})
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/tasks/"+task.ID+"/inbox", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d", resp.StatusCode)
	}
	later := getJSON[struct {
		Items []domain.Task `json:"items"`
	}](t, ts.URL+"/api/v1/tasks?inbox=later")
	if len(later.Items) != 1 || later.Items[0].InboxState != "later" {
		t.Fatalf("later = %+v", later.Items)
	}

	readResp, err := http.Post(ts.URL+"/api/v1/tasks/"+task.ID+"/read", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	readResp.Body.Close()
	if readResp.StatusCode != http.StatusNoContent {
		t.Fatalf("read status = %d", readResp.StatusCode)
	}
	states, err := st.TaskInboxStates(t.Context(), "alice", []string{task.ID})
	if err != nil {
		t.Fatal(err)
	}
	if states[task.ID] == nil || states[task.ID].ReadAt == nil {
		t.Fatalf("task not marked read: %+v", states[task.ID])
	}
	bobStates, err := st.TaskInboxStates(t.Context(), "bob", []string{task.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(bobStates) != 0 {
		t.Fatalf("bob should not inherit alice's inbox: %+v", bobStates)
	}
	status, _ := doAs[map[string]any](t, http.MethodGet, ts.URL+"/api/v1/tasks?inbox=later", "bob", nil)
	if status != http.StatusForbidden {
		t.Fatalf("stranger later = %d, want 403", status)
	}

	mine := getJSON[struct {
		Items []domain.Task `json:"items"`
	}](t, ts.URL+"/api/v1/tasks?assignee=alice")
	if len(mine.Items) != 1 || mine.Items[0].ID != task.ID {
		t.Fatalf("mine = %+v", mine.Items)
	}
	me := getJSON[struct {
		Name string `json:"name"`
	}](t, ts.URL+"/api/v1/me")
	if me.Name != "alice" {
		t.Fatalf("me = %+v", me)
	}
}
