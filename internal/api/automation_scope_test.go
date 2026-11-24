package api

import (
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestAutomationsAreWorkspaceScoped(t *testing.T) {
	ts := newTestServer(t)
	second := postJSON[domain.Workspace](t, ts.URL+"/api/v1/workspaces", map[string]string{
		"name": "Second", "slug": "second",
	})

	first := postJSON[unifiedAutomation](t, ts.URL+"/api/v1/automations", map[string]any{
		"workspaceId": "ws_default", "name": "nightly", "schedule": "0 9 * * *", "agentId": "claude",
	})
	postJSON[unifiedAutomation](t, ts.URL+"/api/v1/automations", map[string]any{
		"workspaceId": second.ID, "name": "second nightly", "schedule": "0 10 * * *", "agentId": "codex",
	})

	list := getJSON[struct {
		Items []unifiedAutomation `json:"items"`
	}](t, ts.URL+"/api/v1/automations?workspace=ws_default")
	if len(list.Items) != 1 || list.Items[0].ID != first.ID {
		t.Fatalf("default automations = %+v", list.Items)
	}
	other := getJSON[struct {
		Items []unifiedAutomation `json:"items"`
	}](t, ts.URL+"/api/v1/automations?workspace="+second.ID)
	if len(other.Items) != 1 || other.Items[0].Name != "second nightly" {
		t.Fatalf("second automations = %+v", other.Items)
	}

	// Agents from another workspace are rejected.
	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": second.ID, "name": "second agent", "backend": "codex",
	})
	if status := postStatus(t, ts.URL+"/api/v1/automations", map[string]any{
		"workspaceId": "ws_default", "name": "bad", "schedule": "0 9 * * *", "agentId": agent.ID,
	}); status != http.StatusBadRequest {
		t.Fatalf("cross-workspace automation status = %d", status)
	}
}
