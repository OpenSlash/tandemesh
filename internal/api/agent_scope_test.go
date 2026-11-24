package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestAgentWorkspaceAssignments(t *testing.T) {
	ts := newTestServer(t)
	second := postJSON[domain.Workspace](t, ts.URL+"/api/v1/workspaces", map[string]string{
		"name": "Second", "slug": "second-assign",
	})
	third := postJSON[domain.Workspace](t, ts.URL+"/api/v1/workspaces", map[string]string{
		"name": "Third", "slug": "third-assign",
	})

	shared := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"name": "shared", "backend": "claude",
		"workspaceIds": []string{"ws_default", second.ID},
	})
	if len(shared.WorkspaceIDs) != 2 {
		t.Fatalf("workspaceIds = %+v", shared.WorkspaceIDs)
	}
	for _, workspaceID := range []string{"ws_default", second.ID} {
		list := getJSON[struct {
			Items []domain.Agent `json:"items"`
		}](t, ts.URL+"/api/v1/agents?workspace="+workspaceID)
		found := false
		for _, agent := range list.Items {
			if agent.ID == shared.ID {
				found = true
			}
		}
		if !found {
			t.Fatalf("agent missing from %s: %+v", workspaceID, list.Items)
		}
	}
	thirdList := getJSON[struct {
		Items []domain.Agent `json:"items"`
	}](t, ts.URL+"/api/v1/agents?workspace="+third.ID)
	if len(thirdList.Items) != 0 {
		t.Fatalf("third workspace should be empty: %+v", thirdList.Items)
	}

	postJSON[unifiedAutomation](t, ts.URL+"/api/v1/automations", map[string]any{
		"kind": "event", "workspaceId": second.ID, "provider": "github", "agentId": shared.ID,
		"triggerKind": "label", "triggerValue": "agent:shared",
	})

	local := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": second.ID, "name": "local", "backend": "codex",
	})
	if len(local.WorkspaceIDs) != 1 || local.WorkspaceIDs[0] != second.ID {
		t.Fatalf("local agent = %+v", local.WorkspaceIDs)
	}
}

func TestAgentsAndBindingsAreWorkspaceScoped(t *testing.T) {
	ts := newTestServer(t)
	second := postJSON[domain.Workspace](t, ts.URL+"/api/v1/workspaces", map[string]string{
		"name": "Second", "slug": "second",
	})

	defaultAgent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "default agent", "backend": "claude",
	})
	secondAgent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": second.ID, "name": "second agent", "backend": "codex",
	})

	first := getJSON[struct {
		Items []domain.Agent `json:"items"`
	}](t, ts.URL+"/api/v1/agents?workspace=ws_default")
	if len(first.Items) != 1 || first.Items[0].ID != defaultAgent.ID {
		t.Fatalf("default agents = %+v", first.Items)
	}
	other := getJSON[struct {
		Items []domain.Agent `json:"items"`
	}](t, ts.URL+"/api/v1/agents?workspace="+second.ID)
	if len(other.Items) != 1 || other.Items[0].ID != secondAgent.ID {
		t.Fatalf("second agents = %+v", other.Items)
	}

	// A binding cannot reference an agent from another workspace.
	body, _ := json.Marshal(map[string]any{
		"kind": "event", "workspaceId": "ws_default", "provider": "github", "agentId": secondAgent.ID,
		"triggerKind": "label", "triggerValue": "agent:fix",
	})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/automations", bytesReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("cross-workspace binding status = %d", resp.StatusCode)
	}

	valid := postJSON[unifiedAutomation](t, ts.URL+"/api/v1/automations", map[string]any{
		"kind": "event", "workspaceId": second.ID, "provider": "github", "agentId": secondAgent.ID,
		"triggerKind": "label", "triggerValue": "agent:fix",
	})
	if valid.AgentID != secondAgent.ID {
		t.Fatalf("binding = %+v", valid)
	}
}
