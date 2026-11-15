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
	"time"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestIterations(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "iterations.db"))
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

	it := postJSON[domain.Iteration](t, ts.URL+"/api/v1/workspaces/ws_default/iterations", map[string]string{
		"name": "Sprint 1", "startDate": time.Now().AddDate(0, 0, -7).Format("2006-01-02"), "endDate": time.Now().AddDate(0, 0, 7).Format("2006-01-02"),
	})
	if it.ID == "" || it.Status != "planned" {
		t.Fatalf("iteration = %+v", it)
	}

	task := postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "61",
		"title": "iteration task",
	})
	body, _ := json.Marshal(map[string]string{"iterationId": it.ID})
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/tasks/"+task.ID+"/iteration", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("assign status = %d", resp.StatusCode)
	}

	scoped := getJSON[struct {
		Items []domain.Task `json:"items"`
	}](t, ts.URL+"/api/v1/tasks?iteration="+it.ID)
	if len(scoped.Items) != 1 || scoped.Items[0].ID != task.ID {
		t.Fatalf("scoped = %+v", scoped.Items)
	}

	doneBody, _ := json.Marshal(map[string]string{"status": "done"})
	req, _ = http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/tasks/"+task.ID, bytes.NewReader(doneBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("done status = %d", resp.StatusCode)
	}
	burndown := getJSON[struct {
		Points []domain.BurndownPoint `json:"points"`
		Stats  domain.IterationStats  `json:"stats"`
	}](t, ts.URL+"/api/v1/iterations/"+it.ID+"/burndown")
	if burndown.Stats.Total != 1 || burndown.Stats.Done != 1 || len(burndown.Points) == 0 {
		t.Fatalf("burndown = %+v", burndown)
	}
	if last := burndown.Points[len(burndown.Points)-1]; last.Remaining != 0 {
		t.Fatalf("remaining = %d", last.Remaining)
	}

	status, _ := json.Marshal(map[string]string{"status": "active"})
	req, _ = http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/iterations/"+it.ID, bytes.NewReader(status))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status patch = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/iterations/"+it.ID, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}
	after, _ := st.GetTask(t.Context(), task.ID)
	if after.IterationID != "" {
		t.Fatalf("iteration not cleared: %+v", after)
	}
}
