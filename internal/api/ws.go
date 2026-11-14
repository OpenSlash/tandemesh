package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
)

// websocket upgrades a daemon connection.
//
// Authentication uses the join token as a query parameter for M1; machine
// tokens replace it when credential storage lands. Origin checks are disabled
// because daemons are not browsers; the token is the boundary.
func (s *Server) websocket(w http.ResponseWriter, r *http.Request) {
	machineID := r.URL.Query().Get("machineId")
	token := r.URL.Query().Get("token")
	if machineID == "" || token != s.cfg.JoinToken {
		writeErr(w, http.StatusUnauthorized, "E_JOIN_TOKEN", "invalid machine credentials")
		return
	}

	conn, unregister := s.hub.Register(machineID)
	defer unregister()

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		s.log.Warn("websocket accept", "machine", machineID, "err", err)
		return
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-conn.Done():
				return
			case msg := <-conn.Messages():
				data, err := json.Marshal(msg)
				if err != nil {
					continue
				}
				wctx, wcancel := context.WithTimeout(ctx, 10*time.Second)
				werr := c.Write(wctx, websocket.MessageText, data)
				wcancel()
				if werr != nil {
					cancel()
					return
				}
			}
		}
	}()

	conn.Send(hub.Message{Type: "hello.ok", Payload: mustJSON(map[string]any{
		"serverTime": time.Now().UTC().Format(time.RFC3339),
		"heartbeat":  "10s",
	})})

	for {
		typ, data, err := c.Read(ctx)
		if err != nil {
			return
		}
		if typ != websocket.MessageText {
			continue
		}
		var msg hub.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		s.handleDaemonMessage(ctx, machineID, msg)
	}
}

func (s *Server) handleDaemonMessage(ctx context.Context, machineID string, msg hub.Message) {
	switch msg.Type {
	case "hello", "heartbeat":
		if err := s.store.Heartbeat(ctx, machineID); err != nil {
			s.log.Warn("daemon heartbeat", "machine", machineID, "err", err)
		}
	case "lease.accept":
		s.log.Info("lease accepted", "machine", machineID, "run", msg.RunID, "fence", msg.Fence)
	case "run.started":
		run, err := s.store.GetRun(ctx, msg.RunID)
		if err != nil || run == nil || run.Fence != msg.Fence {
			s.log.Warn("stale start rejected", "run", msg.RunID, "fence", msg.Fence)
			return
		}
		if _, err := s.store.UpdateStatus(ctx, msg.RunID, domain.RunLeased, domain.RunRunning, "", ""); err != nil {
			s.log.Warn("start run", "run", msg.RunID, "err", err)
			return
		}
		_, _ = s.store.AppendEvent(ctx, msg.RunID, domain.EventRunStarted, map[string]any{})
	case "approval.request":
		var payload struct {
			ApprovalID string `json:"approvalId"`
			Tool       string `json:"tool"`
			Command    string `json:"command"`
			Risk       string `json:"risk"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil || payload.Tool == "" {
			return
		}
		run, err := s.store.GetRun(ctx, msg.RunID)
		if err != nil || run == nil || run.Fence != msg.Fence {
			return
		}
		if payload.Risk == "" {
			payload.Risk = "medium"
		}
		approvalID := payload.ApprovalID
		if approvalID == "" {
			approvalID = "apr_" + ulid.Make().String()
		}
		approval := &domain.Approval{
			ID:          approvalID,
			RunID:       msg.RunID,
			Tool:        payload.Tool,
			Command:     payload.Command,
			Risk:        payload.Risk,
			Status:      domain.ApprovalPending,
			RequestedAt: time.Now().UTC(),
			ExpiresAt:   time.Now().UTC().Add(15 * time.Minute),
		}
		if err := s.store.CreateApproval(ctx, approval); err != nil {
			s.log.Warn("create approval", "run", msg.RunID, "err", err)
			return
		}
		_, _ = s.store.AppendEvent(ctx, msg.RunID, "approval.requested", map[string]any{
			"approvalId": approval.ID,
			"tool":       approval.Tool,
			"risk":       approval.Risk,
		})
		s.notifyApproval(ctx, approval)
	case "question.request":
		var payload struct {
			QuestionID string   `json:"questionId"`
			Question   string   `json:"question"`
			Options    []string `json:"options"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil || payload.Question == "" {
			return
		}
		run, err := s.store.GetRun(ctx, msg.RunID)
		if err != nil || run == nil || run.Fence != msg.Fence || run.SessionID == "" {
			return
		}
		if _, err := s.store.CreateAgentQuestion(ctx, &domain.AgentQuestion{
			ID: payload.QuestionID, SessionID: run.SessionID, RunID: run.ID,
			Question: payload.Question, Options: payload.Options,
		}); err != nil {
			s.log.Warn("create agent question", "run", msg.RunID, "err", err)
		}
	case "run.event":
		var payload struct {
			Type    string         `json:"type"`
			Payload map[string]any `json:"payload"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil || payload.Type == "" {
			return
		}
		ok, err := s.store.CheckFence(ctx, msg.RunID, msg.Fence)
		if err != nil || !ok {
			s.log.Warn("stale run event rejected", "run", msg.RunID, "fence", msg.Fence)
			return
		}
		if payload.Type == "agent.session.started" {
			backendID, _ := payload.Payload["sessionId"].(string)
			if backendID != "" {
				if run, _ := s.store.GetRun(ctx, msg.RunID); run != nil && run.SessionID != "" {
					if err := s.store.RecordSessionRun(ctx, run.SessionID, backendID, msg.RunID); err != nil {
						s.log.Warn("record session backend id", "session", run.SessionID, "err", err)
					}
				}
			}
		}
		if payload.Type == "workspace.ready" {
			path, _ := payload.Payload["path"].(string)
			branch, _ := payload.Payload["branch"].(string)
			if err := s.store.MarkWorktreeReady(ctx, msg.RunID, path, branch); err != nil {
				s.log.Warn("record worktree", "run", msg.RunID, "err", err)
			} else if run, _ := s.store.GetRun(ctx, msg.RunID); run != nil {
				s.recordTaskEvent(ctx, run.TaskID, "task.worktree.ready", map[string]any{
					"runId": msg.RunID, "path": path, "branch": branch,
				})
			}
		}
		if _, err := s.store.AppendEvent(ctx, msg.RunID, payload.Type, payload.Payload); err != nil {
			s.log.Warn("append run event", "run", msg.RunID, "err", err)
		}
	case "run.finished":
		var payload struct {
			Status  string `json:"status"`
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return
		}
		status := domain.RunStatus(payload.Status)
		if status != domain.RunSucceeded && status != domain.RunFailed {
			return
		}
		run, err := s.store.GetRun(ctx, msg.RunID)
		if err != nil || run == nil || run.Fence != msg.Fence {
			s.log.Warn("stale finish rejected", "run", msg.RunID, "fence", msg.Fence)
			return
		}
		if _, err := s.store.UpdateStatus(ctx, msg.RunID, run.Status, status, payload.Code, payload.Message); err != nil {
			s.log.Warn("finish run", "run", msg.RunID, "err", err)
			return
		}
		_, _ = s.store.AppendEvent(ctx, msg.RunID, domain.EventRunFinished, map[string]any{
			"status": string(status),
			"code":   payload.Code,
		})
		if status == domain.RunFailed {
			s.pushNotification(ctx, run.WorkspaceID, "run.failed", "运行失败："+run.Snapshot.Title, payload.Message, "/runs/"+run.ID)
		}
		if dispatch, err := s.store.ActiveDispatchByRun(ctx, run.ID); err == nil && dispatch != nil {
			outcome := "succeeded"
			if status == domain.RunFailed {
				outcome = "failed"
			}
			task, promoted, err := s.store.CompleteDispatch(ctx, dispatch.ID, dispatch.TaskID, outcome, payload.Message, nil)
			if err == nil && task != nil {
				s.recordTaskEvent(ctx, dispatch.TaskID, "task.dispatch.completed", map[string]any{
					"dispatchId": dispatch.ID, "outcome": outcome, "source": "run.finished",
				})
				for _, pid := range promoted {
					s.recordTaskEvent(ctx, pid, "task.promoted", map[string]any{"completed": dispatch.TaskID})
				}
			}
		}
		s.enqueueWritebacks(ctx, run)
	default:
		s.log.Debug("unhandled daemon message", "type", msg.Type, "machine", machineID)
	}
}

func mustJSON(v any) json.RawMessage {
	raw, _ := json.Marshal(v)
	return raw
}
