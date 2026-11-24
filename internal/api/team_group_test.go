package api

import (
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestStartTeamFromGroup(t *testing.T) {
	ts := newTestServer(t)
	task := postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "401",
		"title": "group team task",
	})
	group := postJSON[domain.Group](t, ts.URL+"/api/v1/groups", map[string]string{
		"workspaceId": "ws_default", "name": "Launch crew",
	})
	for _, member := range []map[string]string{
		{"agentId": "claude", "role": "lead"},
		{"agentId": "codex", "role": "reviewer"},
	} {
		postJSON[domain.GroupMember](t, ts.URL+"/api/v1/groups/"+group.ID+"/members", member)
	}

	team := postJSON[struct {
		Items []domain.Run `json:"items"`
	}](t, ts.URL+"/api/v1/tasks/"+task.ID+"/team", map[string]any{
		"repo": "acme/api", "groupId": group.ID,
	})
	if len(team.Items) != 2 {
		t.Fatalf("team = %+v", team.Items)
	}
	if team.Items[0].Role != "lead" || team.Items[0].AgentID != "claude" {
		t.Fatalf("first = %+v", team.Items[0])
	}
	if team.Items[1].Role != "reviewer" || team.Items[1].AgentID != "codex" {
		t.Fatalf("second = %+v", team.Items[1])
	}
}
