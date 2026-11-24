package api

import (
	"context"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestTerminalSessionLifecycle(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	if err := st.UpsertMachine(context.Background(), &domain.Machine{
		ID: "mch_term", Name: "terminal host", Status: "online", Concurrency: 1,
	}); err != nil {
		t.Fatal(err)
	}
	sess := postJSON[domain.AgentSession](t, ts.URL+"/api/v1/sessions", map[string]any{
		"workspaceId": "ws_default", "mode": "terminal", "agentId": "claude",
	})
	term := postJSON[domain.TerminalSession](t, ts.URL+"/api/v1/terminal-sessions", map[string]any{
		"workspaceId": "ws_default", "agentSessionId": sess.ID,
	})
	if term.ID == "" || term.Status != "pending" || term.MachineID != "mch_term" {
		t.Fatalf("terminal = %+v", term)
	}
	got := getAt[struct {
		Terminal domain.TerminalSession `json:"terminal"`
		Events   []domain.TerminalEvent `json:"events"`
	}](t, ts.URL+"/api/v1/terminal-sessions/"+term.ID+"/output?after=0")
	if got.Terminal.ID != term.ID || len(got.Events) != 0 {
		t.Fatalf("empty output = %+v", got)
	}
	postJSON[map[string]string](t, ts.URL+"/api/v1/agentd/terminals/"+term.ID+"/state", map[string]any{"status": "running"})
	postJSON[map[string]string](t, ts.URL+"/api/v1/terminal-sessions/"+term.ID+"/input", map[string]any{"data": "echo hi\r"})
	inputs := getAt[struct {
		Inputs []domain.TerminalInput `json:"inputs"`
	}](t, ts.URL+"/api/v1/agentd/terminals/"+term.ID+"/input")
	if len(inputs.Inputs) != 1 || inputs.Inputs[0].Data != "echo hi\r" {
		t.Fatalf("inputs = %+v", inputs.Inputs)
	}
	postJSON[map[string]string](t, ts.URL+"/api/v1/agentd/terminals/"+term.ID+"/output", map[string]any{"data": "hi\n"})
	got = getAt[struct {
		Terminal domain.TerminalSession `json:"terminal"`
		Events   []domain.TerminalEvent `json:"events"`
	}](t, ts.URL+"/api/v1/terminal-sessions/"+term.ID+"/output?after=0")
	if len(got.Events) != 1 || got.Events[0].Data != "hi\n" {
		t.Fatalf("events = %+v", got.Events)
	}
	postJSON[map[string]string](t, ts.URL+"/api/v1/agentd/terminals/"+term.ID+"/state", map[string]any{"status": "exited", "exitCode": 0})
	final := getAt[domain.TerminalSession](t, ts.URL+"/api/v1/terminal-sessions/"+term.ID)
	if final.Status != "exited" || final.ExitCode == nil || *final.ExitCode != 0 {
		t.Fatalf("final = %+v", final)
	}
}

func TestTerminalResumeCommand(t *testing.T) {
	cases := []struct {
		sess domain.AgentSession
		want string
	}{
		{domain.AgentSession{Backend: "claude", BackendSessionID: "abc"}, "claude --resume abc"},
		{domain.AgentSession{Backend: "codex", BackendSessionID: "xyz"}, "codex resume xyz"},
		{domain.AgentSession{Backend: "claude"}, "claude"},
		{domain.AgentSession{}, ""},
	}
	for _, tc := range cases {
		if got := terminalResumeCommand(&tc.sess); got != tc.want {
			t.Fatalf("terminalResumeCommand(%+v) = %q, want %q", tc.sess, got, tc.want)
		}
	}
}

func TestContinueTerminalInChat(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	if err := st.UpsertMachine(t.Context(), &domain.Machine{
		ID: "mch_continue", Name: "continue host", Status: "online", Concurrency: 1,
	}); err != nil {
		t.Fatal(err)
	}
	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "continue agent", "backend": "claude",
	})
	sess := postJSON[domain.AgentSession](t, ts.URL+"/api/v1/sessions", map[string]any{
		"workspaceId": "ws_default", "mode": "terminal", "agentId": agent.ID,
	})
	term := postJSON[domain.TerminalSession](t, ts.URL+"/api/v1/terminal-sessions", map[string]any{
		"workspaceId": "ws_default", "agentSessionId": sess.ID,
	})
	postJSON[map[string]string](t, ts.URL+"/api/v1/agentd/terminals/"+term.ID+"/output", map[string]any{
		"data": "claude: fixed the login race\\n",
	})
	chat := postJSON[domain.AgentSession](t, ts.URL+"/api/v1/terminal-sessions/"+term.ID+"/continue-chat", map[string]any{})
	if chat.Mode != "chat" || chat.AgentID != agent.ID {
		t.Fatalf("chat = %+v", chat)
	}
	stored, err := st.GetAgentSession(t.Context(), chat.ID)
	if err != nil || stored == nil {
		t.Fatalf("stored chat: %+v err=%v", stored, err)
	}
	if !strings.Contains(stored.Context, "fixed the login race") {
		t.Fatalf("stored context = %q", stored.Context)
	}
	run := postJSON[domain.Run](t, ts.URL+"/api/v1/sessions/"+chat.ID+"/runs", map[string]any{"prompt": "continue"})
	if !strings.Contains(run.Prompt, "fixed the login race") {
		t.Fatalf("run prompt lost terminal context: %q", run.Prompt)
	}
}
