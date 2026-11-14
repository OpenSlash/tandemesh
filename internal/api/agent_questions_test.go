package api

import (
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestAgentQuestionLifecycle(t *testing.T) {
	ts := newTestServer(t)
	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "question agent", "backend": "claude",
	})
	sess := postJSON[domain.AgentSession](t, ts.URL+"/api/v1/sessions", map[string]any{
		"workspaceId": "ws_default", "title": "question chat", "agentId": agent.ID,
	})
	run := postJSON[domain.Run](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/runs", map[string]any{"prompt": "ask"})
	postJSON[domain.Event](t, ts.URL+"/api/v1/agentd/runs/"+run.ID+"/events", map[string]any{
		"type": "agent.question",
		"payload": map[string]any{
			"question": "Which branch?", "options": []string{"main", "release"},
		},
	})
	questions := getJSON[struct {
		Items []domain.AgentQuestion `json:"items"`
	}](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/questions")
	if len(questions.Items) != 1 || questions.Items[0].Question != "Which branch?" {
		t.Fatalf("questions = %+v", questions.Items)
	}
	postJSON[domain.AgentQuestion](t, ts.URL+"/api/v1/questions/"+questions.Items[0].ID+"/answer", map[string]any{
		"answer": "release",
	})
	messages := getJSON[struct {
		Messages []sessionMessage `json:"messages"`
	}](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/messages")
	found := false
	for _, message := range messages.Messages {
		if message.Kind == "question" && strings.Contains(message.Text, "A: release") {
			found = true
		}
	}
	if !found {
		t.Fatalf("question answer missing from transcript: %+v", messages.Messages)
	}
}
