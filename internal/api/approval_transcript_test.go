package api

import (
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestApprovalDecisionAppearsInTranscript(t *testing.T) {
	ts := newTestServer(t)
	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "approval agent", "backend": "claude",
	})
	sess := postJSON[domain.AgentSession](t, ts.URL+"/api/v1/sessions", map[string]any{
		"workspaceId": "ws_default", "title": "approval chat", "agentId": agent.ID,
	})
	run := postJSON[domain.Run](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/runs", map[string]any{"prompt": "do it"})
	approval := postJSON[domain.Approval](t, ts.URL+"/api/v1/approvals", map[string]any{
		"runId": run.ID, "tool": "bash", "command": "echo ok", "risk": "low",
	})
	postJSON[map[string]string](t, ts.URL+"/api/v1/approvals/"+approval.ID+"/decision", map[string]any{
		"decision": "approved", "by": "tester",
	})
	got := getJSON[struct {
		Messages []sessionMessage `json:"messages"`
	}](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/messages")
	found := false
	for _, message := range got.Messages {
		if message.Kind == "approval" && message.Text == "Approved bash by tester: echo ok" {
			found = true
		}
	}
	if !found {
		t.Fatalf("approval transcript missing: %+v", got.Messages)
	}
}
