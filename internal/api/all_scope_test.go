package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestAllWorkspacesScope(t *testing.T) {
	ts := newTestServer(t)
	second := postJSON[domain.Workspace](t, ts.URL+"/api/v1/workspaces", map[string]string{
		"name": "Second", "slug": "second",
	})
	postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "701", "title": "default task",
	})
	postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "702",
		"title": "second task", "workspaceId": second.ID,
	})
	postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "default agent", "backend": "claude",
	})
	postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": second.ID, "name": "second agent", "backend": "codex",
	})

	postJSON[domain.Run](t, ts.URL+"/api/v1/runs", map[string]any{
		"workspaceId": "ws_default", "repo": "acme/api", "prompt": "default run",
		"work": map[string]any{"provider": "manual", "instance": "web", "type": "task", "id": "run-default"},
	})
	postJSON[domain.Run](t, ts.URL+"/api/v1/runs", map[string]any{
		"workspaceId": second.ID, "repo": "acme/api", "prompt": "second run",
		"work": map[string]any{"provider": "manual", "instance": "web", "type": "task", "id": "run-second"},
	})

	runs := getJSON[struct {
		Items []domain.Run `json:"items"`
	}](t, ts.URL+"/api/v1/runs?workspace=all")
	if len(runs.Items) != 2 {
		t.Fatalf("all runs = %+v", runs.Items)
	}

	tasks := getJSON[struct {
		Items []domain.Task `json:"items"`
	}](t, ts.URL+"/api/v1/tasks?workspace=all")
	if len(tasks.Items) != 4 {
		t.Fatalf("all tasks = %+v", tasks.Items)
	}
	agents := getJSON[struct {
		Items []domain.Agent `json:"items"`
	}](t, ts.URL+"/api/v1/agents?workspace=all")
	if len(agents.Items) != 2 {
		t.Fatalf("all agents = %+v", agents.Items)
	}

	// The scope still respects local tenancy: a different identity sees nothing.
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/agents?workspace=all", nil)
	req.Header.Set("X-Orca-User", "charlie")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("charlie all agents status = %d", resp.StatusCode)
	}
	var visible struct {
		Items []domain.Agent `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&visible); err != nil {
		t.Fatal(err)
	}
	if len(visible.Items) != 0 {
		t.Fatalf("charlie sees %+v", visible.Items)
	}
}
