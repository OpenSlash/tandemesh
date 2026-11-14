package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/OpenSlash/tandemesh/internal/agent"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/workspace"
)

// wsWriter serialises writes on a WebSocket connection. coder/websocket allows
// one writer at a time; heartbeats, run events and completion share it.
type wsWriter struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func (w *wsWriter) write(ctx context.Context, msg hub.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	wctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return w.conn.Write(wctx, websocket.MessageText, data)
}

// wsSink publishes run facts over the WebSocket connection.
type wsSink struct {
	w      *wsWriter
	broker *approvalBroker
}

func (s *wsSink) Started(ctx context.Context, runID string, fence int64) {
	_ = s.w.write(ctx, hub.Message{Type: "run.started", RunID: runID, Fence: fence})
}

func (s *wsSink) Event(ctx context.Context, runID string, fence int64, typ string, payload map[string]any) {
	raw, _ := json.Marshal(map[string]any{"type": typ, "payload": payload})
	_ = s.w.write(ctx, hub.Message{Type: "run.event", RunID: runID, Fence: fence, Payload: raw})
}

func (s *wsSink) Finish(ctx context.Context, runID string, fence int64, status, code, message string) {
	raw, _ := json.Marshal(map[string]any{"status": status, "code": code, "message": message})
	_ = s.w.write(ctx, hub.Message{Type: "run.finished", RunID: runID, Fence: fence, Payload: raw})
}

// RequestApproval asks the control plane for a human decision and waits for it.
func (s *wsSink) RequestApproval(ctx context.Context, runID string, fence int64, tool, command, risk string) (string, error) {
	id := newApprovalID()
	raw, _ := json.Marshal(map[string]any{
		"approvalId": id,
		"tool":       tool,
		"command":    command,
		"risk":       risk,
	})
	if err := s.w.write(ctx, hub.Message{Type: "approval.request", RunID: runID, Fence: fence, Payload: raw}); err != nil {
		return "", err
	}
	return s.broker.wait(ctx, id, 15*time.Minute)
}

// wsLoop keeps an outbound WebSocket connection to the control plane. Leases
// arrive as offers; run events and completion travel back on the same socket.
func wsLoop(ctx context.Context, plane, token, machineID string, runner agent.Runner, workdir string, mgr *workspace.Manager, log *slog.Logger) error {
	wsURL := strings.Replace(plane, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	wsURL += "/ws?machineId=" + url.QueryEscape(machineID) + "&token=" + url.QueryEscape(token)

	backoff := time.Second
	broker := newApprovalBroker()
	for {
		if ctx.Err() != nil {
			return nil
		}
		conn, _, err := websocket.Dial(ctx, wsURL, nil)
		if err != nil {
			log.Warn("websocket dial", "err", err, "retryIn", backoff)
			sleepCtx(ctx, backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		log.Info("websocket connected", "machine", machineID)

		w := &wsWriter{conn: conn}
		if err := w.write(ctx, hub.Message{Type: "hello"}); err != nil {
			_ = conn.Close(websocket.StatusAbnormalClosure, "hello failed")
			sleepCtx(ctx, time.Second)
			continue
		}

		hbCtx, cancelHB := context.WithCancel(ctx)
		go heartbeatWS(hbCtx, w, log)

		err = readWS(ctx, w, broker, runner, workdir, mgr, plane, machineID, log)
		cancelHB()
		_ = conn.Close(websocket.StatusNormalClosure, "")
		broker.denyAll()
		if ctx.Err() != nil {
			return nil
		}
		log.Warn("websocket disconnected", "err", err)
		sleepCtx(ctx, time.Second)
	}
}

func readWS(ctx context.Context, w *wsWriter, broker *approvalBroker, runner agent.Runner, workdir string, mgr *workspace.Manager, plane, machineID string, log *slog.Logger) error {
	var (
		mu      sync.Mutex
		cancels = map[string]context.CancelFunc{}
	)
	events := &wsSink{w: w, broker: broker}

	for {
		typ, data, err := w.conn.Read(ctx)
		if err != nil {
			return err
		}
		if typ != websocket.MessageText {
			continue
		}
		var msg hub.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "lease.offer":
			var spec struct {
				Repo            string            `json:"repo"`
				Base            string            `json:"base"`
				Branch          string            `json:"branch"`
				Prompt          string            `json:"prompt"`
				SessionID       string            `json:"sessionId"`
				SessionRecordID string            `json:"sessionRecordId"`
				WorktreePath    string            `json:"worktreePath"`
				AgentID         string            `json:"agentId"`
				AskTools        []string          `json:"askTools"`
				Skills          []string          `json:"skills"`
				Model           string            `json:"model"`
				Reasoning       string            `json:"reasoning"`
				MCPServers      []agent.MCPServer `json:"mcpServers"`
			}
			if err := json.Unmarshal(msg.Payload, &spec); err != nil {
				log.Warn("decode lease offer", "err", err)
				continue
			}
			r := run{
				ID:              msg.RunID,
				Repo:            spec.Repo,
				Base:            spec.Base,
				Branch:          spec.Branch,
				Prompt:          spec.Prompt,
				SessionID:       spec.SessionID,
				SessionRecordID: spec.SessionRecordID,
				WorktreePath:    spec.WorktreePath,
				AgentID:         spec.AgentID,
				AskTools:        spec.AskTools,
				Skills:          spec.Skills,
				Model:           spec.Model,
				Reasoning:       spec.Reasoning,
				MCPServers:      spec.MCPServers,
				Fence:           msg.Fence,
			}
			runCtx, cancel := context.WithCancel(ctx)
			mu.Lock()
			cancels[r.ID] = cancel
			mu.Unlock()
			go func(r run, cancel context.CancelFunc) {
				defer func() {
					mu.Lock()
					delete(cancels, r.ID)
					mu.Unlock()
					cancel()
				}()
				execute(runCtx, events, &r, runner, workdir, mgr, &http.Client{Timeout: 10 * time.Second}, plane, machineID, log)
			}(r, cancel)
		case "run.cancel":
			mu.Lock()
			if cancel, ok := cancels[msg.RunID]; ok {
				cancel()
			}
			mu.Unlock()
			log.Info("run cancelled by control plane", "run", msg.RunID)
		case "hello.ok":
			log.Info("websocket ready")
		case "approval.decision":
			var payload struct {
				ApprovalID string `json:"approvalId"`
				Decision   string `json:"decision"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err == nil && payload.ApprovalID != "" {
				broker.resolve(payload.ApprovalID, payload.Decision)
			}
			log.Info("approval decision received", "run", msg.RunID)
		case "question.answer":
			var payload struct {
				QuestionID string `json:"questionId"`
				Answer     string `json:"answer"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err == nil && payload.QuestionID != "" {
				broker.resolve(payload.QuestionID, payload.Answer)
			}
			log.Info("question answer received", "run", msg.RunID)
		}
	}
}

func heartbeatWS(ctx context.Context, w *wsWriter, log *slog.Logger) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.write(ctx, hub.Message{Type: "heartbeat"}); err != nil {
				log.Warn("websocket heartbeat", "err", err)
				return
			}
		}
	}
}

func sleepCtx(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

// RequestQuestion sends a question over the WebSocket and waits for the answer.
func (s *wsSink) RequestQuestion(ctx context.Context, runID string, fence int64, question string, options []string) (string, error) {
	id := "q_" + newApprovalID()[4:]
	raw, _ := json.Marshal(map[string]any{
		"questionId": id, "question": question, "options": options,
	})
	if err := s.w.write(ctx, hub.Message{Type: "question.request", RunID: runID, Fence: fence, Payload: raw}); err != nil {
		return "", err
	}
	return s.broker.wait(ctx, id, 15*time.Minute)
}
