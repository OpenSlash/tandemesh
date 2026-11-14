package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/OpenSlash/tandemesh/internal/gitops"
)

type gitJob struct {
	ID         string `json:"id"`
	SessionID  string `json:"sessionId"`
	WorktreeID string `json:"worktreeId"`
	Path       string `json:"path"`
	Action     string `json:"action"`
	Message    string `json:"message"`
}

func gitJobLoop(ctx context.Context, c *http.Client, plane, machineID, dataDir, gitPath string, log *slog.Logger) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	var active sync.Map
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, plane+"/api/v1/agentd/git-jobs?machineId="+machineID, nil)
			resp, err := c.Do(req)
			if err != nil {
				continue
			}
			var body struct {
				Items []gitJob `json:"items"`
			}
			err = json.NewDecoder(resp.Body).Decode(&body)
			resp.Body.Close()
			if err != nil {
				continue
			}
			for _, job := range body.Items {
				if _, loaded := active.LoadOrStore(job.ID, true); loaded {
					continue
				}
				go func(job gitJob) {
					defer active.Delete(job.ID)
					runGitJob(ctx, c, plane, dataDir, gitPath, job, log)
				}(job)
			}
		}
	}
}

func runGitJob(ctx context.Context, c *http.Client, plane, dataDir, gitPath string, job gitJob, log *slog.Logger) {
	path := strings.TrimSpace(job.Path)
	if path == "" && dataDir != "" && job.SessionID != "" {
		path = filepath.Join(dataDir, "sessions", job.SessionID, "workspace")
	}
	if path == "" {
		finishGitJob(ctx, c, plane, job.ID, "failed", nil, "worktree path is not configured")
		return
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		finishGitJob(ctx, c, plane, job.ID, "failed", nil, "worktree path is not available on this machine")
		return
	}
	svc := gitops.New()
	if gitPath != "" {
		svc.Git = gitPath
	}
	var result any
	var err error
	switch job.Action {
	case "pr":
		var url string
		url, err = svc.CreatePR(ctx, path)
		result = map[string]string{"url": url}
	case "pr_status":
		var state gitops.PRState
		state, err = svc.PRStatus(ctx, path)
		result = state
	default:
		var status gitops.Status
		status, err = svc.Ship(ctx, path, job.Message)
		result = status
	}
	if err != nil {
		log.Warn("git job failed", "job", job.ID, "action", job.Action, "err", err)
		finishGitJob(ctx, c, plane, job.ID, "failed", nil, err.Error())
		return
	}
	finishGitJob(ctx, c, plane, job.ID, "succeeded", result, "")
	log.Info("git job completed", "job", job.ID, "action", job.Action)
}

func finishGitJob(ctx context.Context, c *http.Client, plane, id, status string, result any, message string) {
	payload := map[string]any{"status": status, "error": message}
	if result != nil {
		payload["result"] = result
	}
	raw, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/v1/agentd/git-jobs/%s", plane, id), bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}
