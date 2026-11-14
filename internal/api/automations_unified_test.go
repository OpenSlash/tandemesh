package api

import (
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestUnifiedAutomations(t *testing.T) {
	ts := newTestServer(t)
	postJSON[domain.Automation](t, ts.URL+"/api/v1/automations", map[string]any{
		"workspaceId": "ws_default", "name": "Daily", "schedule": "0 9 * * *",
		"agentId": "claude", "repo": "acme/api", "prompt": "triage", "enabled": true,
	})
	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "event agent", "backend": "claude",
	})
	binding := postJSON[unifiedAutomation](t, ts.URL+"/api/v1/automations", map[string]any{
		"kind": "event", "workspaceId": "ws_default", "name": "On label", "provider": "github",
		"instance": "acme/api", "agentId": agent.ID, "triggerKind": "label",
		"triggerValue": "agent:fix",
	})
	items := getJSON[struct {
		Items []unifiedAutomation `json:"items"`
	}](t, ts.URL+"/api/v1/automations?workspace=ws_default")
	if len(items.Items) != 2 {
		t.Fatalf("items = %+v", items.Items)
	}
	var event unifiedAutomation
	for _, item := range items.Items {
		if item.Kind == "event" {
			event = item
		}
	}
	if event.ID != binding.ID || event.TriggerValue != "agent:fix" {
		t.Fatalf("event = %+v", event)
	}
	status, updated := doAs[unifiedAutomation](t, http.MethodPut,
		ts.URL+"/api/v1/automations/"+binding.ID, "alice",
		map[string]any{"kind": "event", "enabled": false})
	if status != http.StatusOK || updated.Enabled {
		t.Fatalf("expected disabled event automation: %d %+v", status, updated)
	}
}
