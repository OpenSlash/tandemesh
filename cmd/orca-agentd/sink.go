package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
)

// sink publishes execution facts for a run. The HTTP transport and the
// WebSocket transport implement it, so the executor never cares which one is
// active.
type sink interface {
	Started(ctx context.Context, runID string, fence int64)
	Event(ctx context.Context, runID string, fence int64, typ string, payload map[string]any)
	Finish(ctx context.Context, runID string, fence int64, status, code, message string)
	RequestApproval(ctx context.Context, runID string, fence int64, tool, command, risk string) (string, error)
	RequestQuestion(ctx context.Context, runID string, fence int64, question string, options []string) (string, error)
}

func newApprovalID() string {
	return "apr_" + ulid.Make().String()
}

type httpSink struct {
	c     *http.Client
	plane string
}

func (h *httpSink) Started(ctx context.Context, runID string, fence int64) {
	post(ctx, h.c, h.plane+"/api/v1/agentd/runs/"+runID+"/start", map[string]any{"fence": fence})
}

func (h *httpSink) Event(ctx context.Context, runID string, fence int64, typ string, payload map[string]any) {
	event(ctx, h.c, h.plane, runID, fence, typ, payload)
}

func (h *httpSink) Finish(ctx context.Context, runID string, fence int64, status, code, message string) {
	post(ctx, h.c, h.plane+"/api/v1/agentd/runs/"+runID+"/finish", map[string]any{
		"status":  status,
		"code":    code,
		"message": message,
		"fence":   fence,
	})
}

// RequestApproval creates an approval and polls until it is decided.
func (h *httpSink) RequestApproval(ctx context.Context, runID string, fence int64, tool, command, risk string) (string, error) {
	id := newApprovalID()
	body, _ := json.Marshal(map[string]any{
		"id":      id,
		"runId":   runID,
		"tool":    tool,
		"command": command,
		"risk":    risk,
		"ttl":     "15m",
	})
	resp, err := h.c.Post(h.plane+"/api/v1/approvals", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("create approval: status %d", resp.StatusCode)
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	deadline := time.NewTimer(15 * time.Minute)
	defer deadline.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-deadline.C:
			return "denied", nil
		case <-ticker.C:
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, h.plane+"/api/v1/approvals/"+id, nil)
			resp, err := h.c.Do(req)
			if err != nil {
				continue
			}
			var approval struct {
				Status string `json:"status"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&approval)
			resp.Body.Close()
			if approval.Status == "approved" || approval.Status == "denied" {
				return approval.Status, nil
			}
		}
	}
}

// RequestQuestion asks the control plane for a human answer and polls for it.
func (h *httpSink) RequestQuestion(ctx context.Context, runID string, fence int64, question string, options []string) (string, error) {
	id := "q_" + ulid.Make().String()
	raw, _ := json.Marshal(map[string]any{
		"id": id, "question": question, "options": options,
	})
	resp, err := h.c.Post(h.plane+"/api/v1/agentd/runs/"+runID+"/questions", "application/json", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("create question: status %d", resp.StatusCode)
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, h.plane+"/api/v1/agentd/questions/"+id, nil)
			resp, err := h.c.Do(req)
			if err != nil {
				continue
			}
			var result struct {
				Status string `json:"status"`
				Answer string `json:"answer"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()
			if result.Status == "answered" {
				return result.Answer, nil
			}
			if result.Status == "expired" {
				return "", fmt.Errorf("question expired")
			}
		}
	}
}
