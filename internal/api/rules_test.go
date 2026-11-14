package api

import (
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestAssignmentRulesAutoDispatch(t *testing.T) {
	ts := newTestServer(t)

	high := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "high priority", "backend": "claude",
	})
	low := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "low priority", "backend": "codex",
	})

	postJSON[domain.AssignmentRule](t, ts.URL+"/api/v1/assignment-rules", map[string]any{
		"workspaceId": "ws_default", "name": "low", "provider": "github",
		"label": "agent:fix", "agentId": low.ID, "autoDispatch": true, "priority": 1,
	})
	postJSON[domain.AssignmentRule](t, ts.URL+"/api/v1/assignment-rules", map[string]any{
		"workspaceId": "ws_default", "name": "high", "provider": "github",
		"label": "agent:fix", "agentId": high.ID, "repo": "acme/api",
		"autoDispatch": true, "priority": 10,
	})
	postJSON[domain.AssignmentRule](t, ts.URL+"/api/v1/assignment-rules", map[string]any{
		"workspaceId": "ws_default", "name": "disabled", "provider": "github",
		"label": "skip", "agentId": low.ID, "autoDispatch": true, "enabled": false,
	})

	rules := getJSON[struct {
		Items []domain.AssignmentRule `json:"items"`
	}](t, ts.URL+"/api/v1/assignment-rules?workspace=ws_default")
	if len(rules.Items) != 3 || rules.Items[0].Name != "high" {
		t.Fatalf("rules = %+v", rules.Items)
	}

	task := postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "501",
		"title": "auto dispatched task", "body": "fix the flaky test",
		"labels": []string{"agent:fix"}, "author": "alice", "assignee": "bob",
	})
	if task.AssigneeAgentID != high.ID {
		t.Fatalf("assignee = %q, want %q", task.AssigneeAgentID, high.ID)
	}
	if task.AutoDispatchedAt == nil {
		t.Fatalf("task was not marked auto dispatched: %+v", task)
	}

	detail := getJSON[struct {
		Task domain.Task  `json:"task"`
		Runs []domain.Run `json:"runs"`
	}](t, ts.URL+"/api/v1/tasks/"+task.ID)
	if len(detail.Runs) != 1 {
		t.Fatalf("runs = %d, want 1", len(detail.Runs))
	}
	run := detail.Runs[0]
	if run.AgentID != high.ID || run.TaskID != task.ID || run.Role != domain.RoleLead || run.Repo != "acme/api" {
		t.Fatalf("run = %+v", run)
	}
	if !strings.Contains(run.Prompt, "auto dispatched task") {
		t.Fatalf("prompt = %q", run.Prompt)
	}
	if detail.Task.Status != domain.TaskInProgress || detail.Task.ActiveRuns != 1 {
		t.Fatalf("dispatched task = %+v", detail.Task)
	}

	// Re-importing the same work item must not dispatch it twice.
	again := postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "501",
		"title": "auto dispatched task", "labels": []string{"agent:fix"},
	})
	after := getJSON[struct {
		Task domain.Task  `json:"task"`
		Runs []domain.Run `json:"runs"`
	}](t, ts.URL+"/api/v1/tasks/"+again.ID)
	if len(after.Runs) != 1 {
		t.Fatalf("re-import created %d runs, want 1", len(after.Runs))
	}

	// A task that matches no enabled rule stays unassigned and undispatched.
	plain := postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "502",
		"title": "plain task", "labels": []string{"bug"},
	})
	if plain.AssigneeAgentID != "" || plain.AutoDispatchedAt != nil {
		t.Fatalf("unmatched task = %+v", plain)
	}
	plainDetail := getJSON[struct {
		Task domain.Task  `json:"task"`
		Runs []domain.Run `json:"runs"`
	}](t, ts.URL+"/api/v1/tasks/"+plain.ID)
	if len(plainDetail.Runs) != 0 || plainDetail.Task.RunCount != 0 {
		t.Fatalf("unmatched task runs = %+v", plainDetail)
	}

	// A disabled rule never assigns or dispatches.
	skipped := postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "503",
		"title": "skipped task", "labels": []string{"skip"},
	})
	if skipped.AssigneeAgentID != "" || skipped.AutoDispatchedAt != nil {
		t.Fatalf("disabled rule applied: %+v", skipped)
	}
}
