package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestACPHelperProcess is not a real test: it acts as a minimal ACP server when
// re-executed by TestACPRunner with GO_WANT_ACP_HELPER=1.
func TestACPHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_ACP_HELPER") != "1" {
		return
	}
	enc := json.NewEncoder(os.Stdout)
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	notify := func(update map[string]any) {
		_ = enc.Encode(map[string]any{
			"jsonrpc": "2.0",
			"method":  "session/update",
			"params":  map[string]any{"sessionId": "sess_test", "update": update},
		})
	}

	for sc.Scan() {
		var msg map[string]any
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
			continue
		}
		method, _ := msg["method"].(string)
		id := msg["id"]

		switch method {
		case "initialize":
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{
				"protocolVersion": 1, "agentCapabilities": map[string]any{},
			}})
		case "session/new":
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{
				"sessionId": "sess_test",
			}})
		case "session/load":
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{}})
		case "session/prompt":
			notify(map[string]any{
				"sessionUpdate": "agent_message_chunk",
				"content":       map[string]any{"type": "text", "text": "hello from helper"},
			})
			notify(map[string]any{
				"sessionUpdate": "tool_call",
				"toolCallId":    "tc_1",
				"toolName":      "bash",
			})
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{
				"stopReason": "end_turn",
			}})
		}
	}
	os.Exit(0)
}

func TestACPRunner(t *testing.T) {
	var events []Event
	runner := NewACP(ACPConfig{
		Command: os.Args[0],
		Args:    []string{"-test.run=TestACPHelperProcess"},
		Env:     []string{"GO_WANT_ACP_HELPER=1"},
		Timeout: 10 * time.Second,
	})

	res, err := runner.Run(context.Background(), Spec{
		RunID:   "run_test",
		Prompt:  "say hello",
		Workdir: t.TempDir(),
	}, func(ev Event) { events = append(events, ev) })
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Status != "succeeded" {
		t.Fatalf("status = %q, message = %q", res.Status, res.Message)
	}

	var sawMessage, sawTool bool
	for _, ev := range events {
		if ev.Type == "agent.message" && ev.Payload["text"] == "hello from helper" {
			sawMessage = true
		}
		if ev.Type == "tool.call" && ev.Payload["name"] == "bash" {
			sawTool = true
		}
	}
	if !sawMessage {
		t.Fatalf("missing agent message event: %+v", events)
	}
	if !sawTool {
		t.Fatalf("missing tool call event: %+v", events)
	}
}

func TestACPMissingCommand(t *testing.T) {
	runner := NewACP(ACPConfig{Command: "orca-definitely-not-a-real-binary"})
	res, err := runner.Run(context.Background(), Spec{Prompt: "x"}, func(Event) {})
	if err != nil {
		t.Fatalf("run should return a result, got error: %v", err)
	}
	if res.Status != "failed" || res.Code != "E_AGENT_START" {
		t.Fatalf("result = %+v", res)
	}
}

func TestACPResolveConfinesPaths(t *testing.T) {
	a := NewACP(ACPConfig{})
	root := t.TempDir()
	if _, err := a.resolve(Spec{Workdir: root}, root+"/inside.txt"); err != nil {
		t.Fatalf("inside path rejected: %v", err)
	}
	if _, err := a.resolve(Spec{Workdir: root}, root+"/../escape.txt"); err == nil {
		t.Fatal("path traversal accepted")
	}
}

func TestACPResumesSession(t *testing.T) {
	var events []Event
	runner := NewACP(ACPConfig{
		Command: os.Args[0],
		Args:    []string{"-test.run=TestACPHelperProcess"},
		Env:     []string{"GO_WANT_ACP_HELPER=1"},
		Timeout: 10 * time.Second,
	})
	res, err := runner.Run(context.Background(), Spec{
		RunID:     "run_resume",
		SessionID: "sess_existing",
		Prompt:    "continue",
		Workdir:   t.TempDir(),
	}, func(ev Event) { events = append(events, ev) })
	if err != nil || res.Status != "succeeded" {
		t.Fatalf("run: %v status=%q", err, res.Status)
	}
	resumed := false
	for _, ev := range events {
		if ev.Type == "agent.session.started" && ev.Payload["resumed"] == true && ev.Payload["sessionId"] == "sess_existing" {
			resumed = true
		}
	}
	if !resumed {
		t.Fatalf("session was not resumed: %+v", events)
	}
}
func TestToolArgsAndOutput(t *testing.T) {
	if got := toolArgs(json.RawMessage(`{"command":"go test ./...","cwd":"/tmp"}`)); got != "go test ./..." {
		t.Fatalf("args = %q", got)
	}
	if got := toolOutput(json.RawMessage(`[{"type":"content","content":{"type":"text","text":"ok"}}]`)); got != "ok" {
		t.Fatalf("output = %q", got)
	}
}

func TestPromptContentImages(t *testing.T) {
	prompt := "look\n\n![shot](data:image/png;base64,aGVsbG8=)"
	blocks := promptContent(prompt)
	if len(blocks) != 2 {
		t.Fatalf("blocks = %+v", blocks)
	}
	text := blocks[0].(map[string]any)["text"].(string)
	if text != "look\n\n[image: shot]" {
		t.Fatalf("text = %q", text)
	}
	image := blocks[1].(map[string]any)
	if image["type"] != "image" || image["mimeType"] != "image/png" || image["data"] != "aGVsbG8=" {
		t.Fatalf("image = %+v", image)
	}
}

func TestReasoningEnv(t *testing.T) {
	env := reasoningEnv("claude", "high")
	if !contains(env, "MAX_THINKING_TOKENS=8192") || !contains(env, "TANDEMESH_REASONING=high") {
		t.Fatalf("claude env = %v", env)
	}
	env = reasoningEnv("codex", "low")
	if !contains(env, "CODEX_REASONING_EFFORT=low") {
		t.Fatalf("codex env = %v", env)
	}
	if env := reasoningEnv("claude", ""); env != nil {
		t.Fatalf("empty env = %v", env)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestPermissionQuestion(t *testing.T) {
	params := json.RawMessage(`{"toolCall":{"title":"AskUserQuestion","rawInput":{"questions":[{"question":"Which branch?","options":[{"label":"main"},{"label":"release"}]}]},"options":[{"optionId":"allow_once","kind":"allow_once"}]}}`)
	question, options, ok := permissionQuestion(params)
	if !ok || question != "Which branch?" || len(options) != 2 || options[1] != "release" {
		t.Fatalf("question=%q options=%v ok=%v", question, options, ok)
	}
	outcome := permissionQuestionOutcome(params, "release")
	selected, _ := outcome["outcome"].(map[string]any)
	if selected["optionId"] != "allow_once" {
		t.Fatalf("outcome=%v", outcome)
	}
}
