package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ACPConfig configures the ACP runner.
type ACPConfig struct {
	Command string        // e.g. "npx" or "claude-code-acp"
	Args    []string      // e.g. []string{"-y", "@zed-industries/claude-code-acp"}
	Env     []string      // extra environment variables
	Timeout time.Duration // overall prompt timeout (0 = spec timeout or 30m)
	Logger  *slog.Logger
}

// ACP drives an Agent Client Protocol server over stdio.
//
// Transport: JSON-RPC 2.0, one JSON object per line on stdin/stdout. The client
// implements initialize, session/new and session/prompt, streams session/update
// notifications as run events, and answers the agent's fs and permission
// requests. Terminal requests are rejected in M1; they land with the workspace
// manager.
type ACP struct {
	cfg ACPConfig
}

// NewACP creates an ACP runner.
func NewACP(cfg ACPConfig) *ACP {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Minute
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &ACP{cfg: cfg}
}

// Name implements Runner.
func (a *ACP) Name() string { return "acp" }

// Run implements Runner.
func (a *ACP) Run(ctx context.Context, spec Spec, emit Emitter) (Result, error) {
	timeout := spec.Timeout
	if timeout <= 0 {
		timeout = a.cfg.Timeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, a.cfg.Command, a.cfg.Args...)
	cmd.WaitDelay = 3 * time.Second
	if spec.Workdir != "" {
		cmd.Dir = spec.Workdir
	}
	cmd.Env = append(os.Environ(), a.cfg.Env...)
	for key, value := range spec.Env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	cmd.Env = append(cmd.Env, reasoningEnv(spec.Backend, spec.Reasoning)...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return failed("E_AGENT_START", err), nil
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return failed("E_AGENT_START", err), nil
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return failed("E_AGENT_START", err), nil
	}

	a.cfg.Logger.Info("acp process started", "command", a.cfg.Command, "pid", cmd.Process.Pid)
	client := newRPCClient(stdin, stdout)
	textBytes := 0
	var assistantText, thoughtText strings.Builder
	trackedEmit := func(ev Event) {
		if ev.Type == "agent.message" {
			if text, ok := ev.Payload["text"].(string); ok {
				textBytes += len(text)
				if role, _ := ev.Payload["role"].(string); role == "thought" {
					thoughtText.WriteString(text)
				} else {
					assistantText.WriteString(text)
				}
			}
		}
		emit(ev)
	}
	client.onNotify = func(method string, params json.RawMessage) {
		a.handleNotification(method, params, trackedEmit)
	}
	client.onRequest = func(method string, params json.RawMessage) (any, *rpcError) {
		return a.handleRequest(runCtx, spec, method, params)
	}

	stop := func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
	defer stop()

	if _, err := client.call(runCtx, "initialize", map[string]any{
		"protocolVersion": 1,
		"clientCapabilities": map[string]any{
			"fs":       map[string]any{"readTextFile": true, "writeTextFile": true},
			"terminal": false,
		},
		"clientInfo": map[string]any{"name": "orca", "version": "0.1.0"},
	}, 30*time.Second); err != nil {
		return rpcFailure(err, stderr.String()), nil
	}

	a.cfg.Logger.Info("acp initialize ok")
	var session struct {
		SessionID string `json:"sessionId"`
	}
	resumed := false
	if spec.SessionID != "" {
		_, err := client.call(runCtx, "session/load", map[string]any{
			"sessionId":  spec.SessionID,
			"cwd":        cwdOrDot(spec.Workdir),
			"mcpServers": mcpServersPayload(spec.MCPServers),
		}, 30*time.Second)
		if err != nil {
			a.cfg.Logger.Warn("acp session load failed; starting new", "session", spec.SessionID, "err", err)
		} else {
			session.SessionID = spec.SessionID
			resumed = true
		}
	}
	if session.SessionID == "" {
		raw, err := client.call(runCtx, "session/new", map[string]any{
			"cwd":        cwdOrDot(spec.Workdir),
			"mcpServers": mcpServersPayload(spec.MCPServers),
		}, 30*time.Second)
		if err != nil {
			return rpcFailure(err, stderr.String()), nil
		}
		if err := json.Unmarshal(raw, &session); err != nil || session.SessionID == "" {
			return failed("E_AGENT_RPC", fmt.Errorf("session/new returned no sessionId: %s", string(raw))), nil
		}
	}

	a.cfg.Logger.Info("acp session ready", "session", session.SessionID, "resumed", resumed)
	emit(Event{Type: "agent.session.started", Payload: map[string]any{
		"sessionId": session.SessionID,
		"backend":   spec.Backend,
		"resumed":   resumed,
	}})

	raw, err := client.call(runCtx, "session/prompt", map[string]any{
		"sessionId": session.SessionID,
		"prompt":    promptContent(spec.Prompt),
	}, timeout)
	if err != nil {
		return rpcFailure(err, stderr.String()), nil
	}
	var promptResult struct {
		StopReason string `json:"stopReason"`
	}
	_ = json.Unmarshal(raw, &promptResult)
	if promptResult.StopReason == "cancelled" {
		return Result{Status: "failed", Code: "E_CANCELLED", Message: "prompt cancelled"}, nil
	}

	if assistantText.Len() > 0 {
		emit(Event{Type: "agent.message.complete", Payload: map[string]any{
			"text": assistantText.String(), "role": "assistant",
		}})
	}
	if thoughtText.Len() > 0 {
		emit(Event{Type: "agent.message.complete", Payload: map[string]any{
			"text": thoughtText.String(), "role": "thought",
		}})
	}
	emit(Event{Type: "agent.message", Payload: map[string]any{
		"text": "prompt finished: " + promptResult.StopReason,
	}})
	usage := readClaudeUsage(session.SessionID)
	if usage != nil && usage.OutputTokens == 0 && textBytes > 0 {
		usage.OutputTokens = textBytes / 4
		usage.Estimated = true
	}
	return Result{Status: "succeeded", Usage: usage}, nil
}

// mcpServersPayload converts MCP servers into the ACP session shape.
func mcpServersPayload(servers []MCPServer) []any {
	out := make([]any, 0, len(servers))
	for _, s := range servers {
		if s.Name == "" {
			continue
		}
		if s.URL != "" {
			out = append(out, map[string]any{"name": s.Name, "url": s.URL})
			continue
		}
		entry := map[string]any{"name": s.Name, "command": s.Command}
		if len(s.Args) > 0 {
			entry["args"] = s.Args
		}
		if len(s.Env) > 0 {
			entry["env"] = s.Env
		}
		out = append(out, entry)
	}
	return out
}

func (a *ACP) handleNotification(method string, params json.RawMessage, emit Emitter) {
	if method != "session/update" {
		return
	}
	var update struct {
		Update struct {
			SessionUpdate string          `json:"sessionUpdate"`
			Content       json.RawMessage `json:"content"`
			RawInput      json.RawMessage `json:"rawInput"`
			Title         string          `json:"title"`
			Kind          string          `json:"kind"`
			ToolName      string          `json:"toolName"`
			Status        string          `json:"status"`
			ToolCallID    string          `json:"toolCallId"`
		} `json:"update"`
	}
	if err := json.Unmarshal(params, &update); err != nil {
		return
	}
	switch update.Update.SessionUpdate {
	case "agent_message_chunk":
		emit(Event{Type: "agent.message", Payload: map[string]any{"text": contentText(update.Update.Content), "chunk": true}})
	case "agent_thought_chunk":
		emit(Event{Type: "agent.message", Payload: map[string]any{"text": contentText(update.Update.Content), "role": "thought", "chunk": true}})
	case "tool_call":
		name := update.Update.ToolName
		if name == "" {
			name = update.Update.Title
		}
		emit(Event{Type: "tool.call", Payload: map[string]any{
			"name":       name,
			"args":       toolArgs(update.Update.RawInput),
			"kind":       update.Update.Kind,
			"toolCallId": update.Update.ToolCallID,
		}})
	case "tool_call_update":
		emit(Event{Type: "tool.result", Payload: map[string]any{
			"status":     update.Update.Status,
			"output":     toolOutput(update.Update.Content),
			"toolCallId": update.Update.ToolCallID,
		}})
	case "plan":
		emit(Event{Type: "agent.plan", Payload: map[string]any{}})
	}
}

var inlineImage = regexp.MustCompile(`!\[([^\]]*)\]\(data:(image/[a-zA-Z0-9.+-]+);base64,([A-Za-z0-9+/=]+)\)`)

// reasoningEnv maps Orca's reasoning level to backend process settings.
func reasoningEnv(backend, level string) []string {
	level = strings.ToLower(strings.TrimSpace(level))
	if level == "" {
		return nil
	}
	env := []string{"ORCA_REASONING=" + level}
	switch strings.ToLower(backend) {
	case "claude":
		tokens := map[string]string{"low": "1024", "medium": "4096", "high": "8192"}[level]
		if tokens != "" {
			env = append(env, "MAX_THINKING_TOKENS="+tokens)
		}
	case "codex":
		env = append(env, "CODEX_REASONING_EFFORT="+level)
	}
	return env
}

// promptContent converts markdown data-URL images into ACP image blocks.
func promptContent(prompt string) []any {
	matches := inlineImage.FindAllStringSubmatch(prompt, 4)
	if len(matches) == 0 {
		return []any{map[string]any{"type": "text", "text": prompt}}
	}
	text := inlineImage.ReplaceAllString(prompt, "[image: $1]")
	blocks := []any{}
	if strings.TrimSpace(text) != "" {
		blocks = append(blocks, map[string]any{"type": "text", "text": text})
	}
	for _, match := range matches {
		blocks = append(blocks, map[string]any{
			"type":     "image",
			"mimeType": match[2],
			"data":     match[3],
		})
	}
	return blocks
}

func contentText(raw json.RawMessage) string {
	var content struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &content) == nil {
		return content.Text
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	return ""
}

func toolArgs(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var args map[string]any
	if json.Unmarshal(raw, &args) == nil {
		for _, key := range []string{"command", "cmd", "path", "query", "url"} {
			if value, ok := args[key].(string); ok && value != "" {
				return value
			}
		}
		compact, err := json.Marshal(args)
		if err == nil {
			return truncate(string(compact), 500)
		}
	}
	return truncate(string(raw), 500)
}

func toolOutput(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var blocks []struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Content struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if json.Unmarshal(raw, &blocks) == nil {
		parts := make([]string, 0, len(blocks))
		for _, block := range blocks {
			if block.Text != "" {
				parts = append(parts, block.Text)
			} else if block.Content.Text != "" {
				parts = append(parts, block.Content.Text)
			}
		}
		if len(parts) > 0 {
			return truncate(strings.Join(parts, "\n"), 1200)
		}
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return truncate(text, 1200)
	}
	return truncate(string(raw), 1200)
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}

func (a *ACP) handleRequest(ctx context.Context, spec Spec, method string, params json.RawMessage) (any, *rpcError) {
	switch method {
	case "session/request_permission":
		if question, options, ok := permissionQuestion(params); ok && spec.Ask != nil {
			answer, err := spec.Ask(ctx, question, options)
			if err != nil {
				return nil, &rpcError{Code: -32603, Message: err.Error()}
			}
			return permissionQuestionOutcome(params, answer), nil
		}
		tool, command := permissionRequest(params)
		allow := true
		if spec.Approve != nil && ShouldAsk(tool, spec.AskTools) {
			ok, err := spec.Approve(ctx, tool, command, RiskFor(tool))
			if err != nil || !ok {
				allow = false
			}
		}
		return permissionOutcome(params, allow), nil
	case "fs/read_text_file":
		var req struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, &rpcError{Code: -32602, Message: "invalid params"}
		}
		path, err := a.resolve(spec, req.Path)
		if err != nil {
			return nil, &rpcError{Code: -32001, Message: err.Error()}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, &rpcError{Code: -32002, Message: err.Error()}
		}
		return map[string]any{"content": string(data)}, nil
	case "fs/write_text_file":
		var req struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, &rpcError{Code: -32602, Message: "invalid params"}
		}
		path, err := a.resolve(spec, req.Path)
		if err != nil {
			return nil, &rpcError{Code: -32001, Message: err.Error()}
		}
		if err := os.WriteFile(path, []byte(req.Content), 0o644); err != nil {
			return nil, &rpcError{Code: -32003, Message: err.Error()}
		}
		return map[string]any{}, nil
	default:
		return nil, &rpcError{Code: -32601, Message: "method not supported by orca: " + method}
	}
}

// resolve confines fs access to the run workdir.
func (a *ACP) resolve(spec Spec, path string) (string, error) {
	if spec.Workdir == "" {
		return path, nil
	}
	root, err := filepath.Abs(spec.Workdir)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if abs != root && !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("path outside workdir: %s", path)
	}
	return abs, nil
}

// permissionRequest extracts a human-readable tool name and command summary.
func permissionRequest(params json.RawMessage) (tool, command string) {
	var req struct {
		ToolCall struct {
			Title    string         `json:"title"`
			Kind     string         `json:"kind"`
			RawInput map[string]any `json:"rawInput"`
		} `json:"toolCall"`
	}
	_ = json.Unmarshal(params, &req)
	tool = strings.TrimSpace(req.ToolCall.Title)
	if tool == "" {
		tool = strings.TrimSpace(req.ToolCall.Kind)
	}
	command = tool
	if cmd, ok := req.ToolCall.RawInput["command"].(string); ok && cmd != "" {
		command = cmd
	}
	return tool, command
}

// ShouldAsk reports whether a tool call matches the agent's ask policy.
// Matching is a case-insensitive substring test so policies like "push" cover
// titles such as "Run git push".
func ShouldAsk(tool string, ask []string) bool {
	lower := strings.ToLower(tool)
	for _, pattern := range ask {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if pattern != "" && strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// RiskFor classifies a tool call for the approval card.
func RiskFor(tool string) string {
	lower := strings.ToLower(tool)
	for _, word := range []string{"push", "delete", "drop", "merge", "deploy", "rm ", "sudo"} {
		if strings.Contains(lower, word) {
			return "high"
		}
	}
	return "medium"
}

func permissionOutcome(params json.RawMessage, allow bool) map[string]any {
	var req struct {
		Options []struct {
			OptionID string `json:"optionId"`
			Kind     string `json:"kind"`
		} `json:"options"`
	}
	if err := json.Unmarshal(params, &req); err != nil || len(req.Options) == 0 {
		if allow {
			return map[string]any{"outcome": map[string]any{"outcome": "selected", "optionId": "allow_once"}}
		}
		return map[string]any{"outcome": map[string]any{"outcome": "cancelled"}}
	}
	prefix := "allow"
	if !allow {
		prefix = "reject"
	}
	for _, o := range req.Options {
		if strings.HasPrefix(o.Kind, prefix) {
			return map[string]any{"outcome": map[string]any{"outcome": "selected", "optionId": o.OptionID}}
		}
	}
	if allow {
		return map[string]any{"outcome": map[string]any{"outcome": "selected", "optionId": req.Options[0].OptionID}}
	}
	return map[string]any{"outcome": map[string]any{"outcome": "cancelled"}}
}

func cwdOrDot(dir string) string {
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "."
		}
		return wd
	}
	return dir
}

func failed(code string, err error) Result {
	return Result{Status: "failed", Code: code, Message: err.Error()}
}

func rpcFailure(err error, stderr string) Result {
	code := "E_AGENT_RPC"
	if err == context.DeadlineExceeded || err == context.Canceled {
		code = "E_AGENT_TIMEOUT"
	}
	msg := err.Error()
	if stderr != "" {
		msg += ": " + strings.TrimSpace(stderr)
	}
	return Result{Status: "failed", Code: code, Message: msg}
}

// ---- JSON-RPC client ----

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string { return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message) }

type rpcMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *rpcError        `json:"error,omitempty"`
}

type rpcClient struct {
	mu        sync.Mutex
	enc       *json.Encoder
	nextID    atomic.Int64
	pending   sync.Map // int64 -> chan rpcMessage
	closed    chan struct{}
	closeOnce sync.Once
	readErr   error

	onNotify  func(method string, params json.RawMessage)
	onRequest func(method string, params json.RawMessage) (any, *rpcError)
}

func newRPCClient(w io.Writer, r io.Reader) *rpcClient {
	c := &rpcClient{
		enc:    json.NewEncoder(w),
		closed: make(chan struct{}),
	}
	go func() {
		dec := json.NewDecoder(bufio.NewReader(r))
		for {
			var msg rpcMessage
			if err := dec.Decode(&msg); err != nil {
				c.fail(err)
				return
			}
			c.dispatch(msg)
		}
	}()
	return c
}

func (c *rpcClient) dispatch(msg rpcMessage) {
	switch {
	case msg.Method == "" && msg.ID != nil:
		var id int64
		if err := json.Unmarshal(*msg.ID, &id); err != nil {
			return
		}
		slog.Default().Info("acp rpc response", "id", id)
		if ch, ok := c.pending.LoadAndDelete(id); ok {
			ch.(chan rpcMessage) <- msg
		}
	case msg.Method != "" && msg.ID != nil:
		slog.Default().Info("acp rpc request", "method", msg.Method)
		go func(m rpcMessage) {
			result, rerr := c.onRequest(m.Method, m.Params)
			c.respond(*m.ID, result, rerr)
		}(msg)
	case msg.Method != "":
		if c.onNotify != nil {
			c.onNotify(msg.Method, msg.Params)
		}
	}
}

func (c *rpcClient) respond(id json.RawMessage, result any, rerr *rpcError) {
	resp := map[string]any{"jsonrpc": "2.0", "id": id}
	if rerr != nil {
		resp["error"] = rerr
	} else {
		resp["result"] = result
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.enc.Encode(resp)
}

func (c *rpcClient) call(ctx context.Context, method string, params any, timeout time.Duration) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	ch := make(chan rpcMessage, 1)
	c.pending.Store(id, ch)
	defer c.pending.Delete(id)

	req := map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}
	c.mu.Lock()
	err := c.enc.Encode(req)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.closed:
		return nil, fmt.Errorf("agent connection closed: %v", c.readErr)
	case <-timer.C:
		return nil, fmt.Errorf("timeout waiting for %s", method)
	case msg := <-ch:
		if msg.Error != nil {
			return nil, msg.Error
		}
		return msg.Result, nil
	}
}

func (c *rpcClient) fail(err error) {
	c.closeOnce.Do(func() {
		c.readErr = err
		close(c.closed)
	})
}

// readClaudeUsage sums token usage from the Claude Code session transcript.
func readClaudeUsage(sessionID string) *Usage {
	for i := 0; i < 12; i++ {
		if usage := readClaudeUsageOnce(sessionID); usage != nil && usage.InputTokens+usage.OutputTokens > 0 {
			return usage
		}
		time.Sleep(250 * time.Millisecond)
	}
	return nil
}

func readClaudeUsageOnce(sessionID string) *Usage {
	if sessionID == "" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	files, _ := filepath.Glob(filepath.Join(home, ".claude", "projects", "*", sessionID+".jsonl"))
	if len(files) == 0 {
		return nil
	}
	type usageLine struct {
		Type    string `json:"type"`
		Message struct {
			ID    string `json:"id"`
			Model string `json:"model"`
			Usage struct {
				InputTokens      int `json:"input_tokens"`
				OutputTokens     int `json:"output_tokens"`
				CacheReadTokens  int `json:"cache_read_input_tokens"`
				CacheWriteTokens int `json:"cache_creation_input_tokens"`
			} `json:"usage"`
		} `json:"message"`
	}
	perMessage := map[string]usageLine{}
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 1024*1024), 8*1024*1024)
		for scanner.Scan() {
			var line usageLine
			if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
				continue
			}
			if line.Type == "assistant" && line.Message.ID != "" {
				perMessage[line.Message.ID] = line
			}
		}
		f.Close()
	}
	if len(perMessage) == 0 {
		return nil
	}
	var u Usage
	for _, line := range perMessage {
		u.InputTokens += line.Message.Usage.InputTokens
		u.OutputTokens += line.Message.Usage.OutputTokens
		u.CacheReadTokens += line.Message.Usage.CacheReadTokens
		u.CacheWriteTokens += line.Message.Usage.CacheWriteTokens
		if line.Message.Model != "" {
			u.Model = line.Message.Model
		}
	}
	return &u
}

func permissionQuestion(params json.RawMessage) (string, []string, bool) {
	var req struct {
		ToolCall struct {
			ToolCallID string         `json:"toolCallId"`
			Title      string         `json:"title"`
			RawInput   map[string]any `json:"rawInput"`
		} `json:"toolCall"`
	}
	if json.Unmarshal(params, &req) != nil {
		return "", nil, false
	}
	if question, ok := req.ToolCall.RawInput["question"].(string); ok && strings.TrimSpace(question) != "" {
		return strings.TrimSpace(question), nil, true
	}
	raw, ok := req.ToolCall.RawInput["questions"].([]any)
	if !ok || len(raw) == 0 {
		return "", nil, false
	}
	first, ok := raw[0].(map[string]any)
	if !ok {
		return "", nil, false
	}
	question, _ := first["question"].(string)
	if question == "" {
		question, _ = first["text"].(string)
	}
	options := []string{}
	if list, ok := first["options"].([]any); ok {
		for _, item := range list {
			switch value := item.(type) {
			case string:
				options = append(options, value)
			case map[string]any:
				if label, ok := value["label"].(string); ok && label != "" {
					options = append(options, label)
				} else if name, ok := value["name"].(string); ok && name != "" {
					options = append(options, name)
				}
			}
		}
	}
	question = strings.TrimSpace(question)
	return question, options, question != ""
}

func permissionQuestionOutcome(params json.RawMessage, answer string) map[string]any {
	var req struct {
		Options []struct {
			OptionID string `json:"optionId"`
			Name     string `json:"name"`
			Label    string `json:"label"`
			Kind     string `json:"kind"`
		} `json:"options"`
	}
	if json.Unmarshal(params, &req) == nil {
		for _, option := range req.Options {
			if option.OptionID == answer || option.Name == answer || option.Label == answer {
				return map[string]any{"outcome": map[string]any{"outcome": "selected", "optionId": option.OptionID}}
			}
		}
		for _, option := range req.Options {
			if strings.HasPrefix(option.Kind, "allow") {
				return map[string]any{"outcome": map[string]any{"outcome": "selected", "optionId": option.OptionID}}
			}
		}
	}
	return map[string]any{"outcome": map[string]any{"outcome": "selected", "optionId": "allow_once"}}
}
