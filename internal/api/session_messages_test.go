package api

import (
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestSessionMessagesArePersisted(t *testing.T) {
	ts := newTestServer(t)
	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "chat agent", "backend": "claude",
	})
	sess := postJSON[domain.AgentSession](t, ts.URL+"/api/v1/sessions", map[string]any{
		"workspaceId": "ws_default", "title": "chat", "agentId": agent.ID, "mode": "chat",
	})
	run := postJSON[domain.Run](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/runs", map[string]any{
		"prompt": "hello",
	})
	postJSON[domain.Event](t, ts.URL+"/api/v1/agentd/runs/"+run.ID+"/events", map[string]any{
		"type": "agent.message", "payload": map[string]any{"text": "hi there"},
	})
	got := getJSON[struct {
		Messages []sessionMessage `json:"messages"`
	}](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/messages")
	if len(got.Messages) != 2 {
		t.Fatalf("messages = %+v", got.Messages)
	}
	if got.Messages[0].Role != "user" || got.Messages[0].Text != "hello" {
		t.Fatalf("user message = %+v", got.Messages[0])
	}
	if got.Messages[1].Role != "assistant" || got.Messages[1].Text != "hi there" {
		t.Fatalf("assistant message = %+v", got.Messages[1])
	}
}

func TestSessionWorktreeBinding(t *testing.T) {
	ts := newTestServer(t)
	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "worktree agent", "backend": "claude",
	})
	sess := postJSON[domain.AgentSession](t, ts.URL+"/api/v1/sessions", map[string]any{
		"workspaceId": "ws_default", "title": "worktree chat", "agentId": agent.ID,
		"repo": "file:///tmp/repo", "base": "main",
	})
	if sess.Repo == "" {
		t.Fatalf("session repo was not stored: %+v", sess)
	}
	run := postJSON[domain.Run](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/runs", map[string]any{"prompt": "hello"})
	got := getJSON[struct {
		Session domain.AgentSession `json:"session"`
	}](t, ts.URL+"/api/v1/sessions/"+sess.ID)
	if got.Session.WorktreeID == "" {
		t.Fatal("session is not bound to a worktree")
	}
	wt := getJSON[domain.Worktree](t, ts.URL+"/api/v1/worktrees/"+got.Session.WorktreeID)
	if wt.RunID != run.ID || wt.Repo != "file:///tmp/repo" {
		t.Fatalf("worktree = %+v", wt)
	}
}

func TestSessionModelOverride(t *testing.T) {
	ts := newTestServer(t)
	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "model agent", "backend": "claude", "model": "agent-default",
	})
	sess := postJSON[domain.AgentSession](t, ts.URL+"/api/v1/sessions", map[string]any{
		"workspaceId": "ws_default", "title": "model chat", "agentId": agent.ID, "model": "session-model",
	})
	if sess.Model != "session-model" {
		t.Fatalf("session model = %q", sess.Model)
	}
	run := postJSON[domain.Run](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/runs", map[string]any{"prompt": "hello"})
	if run.Model != "session-model" {
		t.Fatalf("run model = %q", run.Model)
	}
}
