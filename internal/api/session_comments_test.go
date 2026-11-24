package api

import (
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestSessionDiffComments(t *testing.T) {
	ts := newTestServer(t)
	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "review agent", "backend": "claude",
	})
	sess := postJSON[domain.AgentSession](t, ts.URL+"/api/v1/sessions", map[string]any{
		"workspaceId": "ws_default", "title": "review chat", "agentId": agent.ID,
	})
	comment := postJSON[domain.SessionComment](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/comments", map[string]any{
		"file": "internal/login.go", "line": 42, "side": "new", "body": "This lock is held twice.",
	})
	if comment.ID == "" || comment.Status != "open" {
		t.Fatalf("comment = %+v", comment)
	}
	run := postJSON[domain.Run](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/comments/send", map[string]any{})
	if !strings.Contains(run.Prompt, "internal/login.go:42") || !strings.Contains(run.Prompt, "held twice") {
		t.Fatalf("run prompt = %q", run.Prompt)
	}
	got := getJSON[struct {
		Items []domain.SessionComment `json:"items"`
	}](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/comments")
	if len(got.Items) != 1 || got.Items[0].Status != "sent" {
		t.Fatalf("comments = %+v", got.Items)
	}
}
