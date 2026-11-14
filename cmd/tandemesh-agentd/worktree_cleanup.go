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
	"sync"
	"time"

	"github.com/OpenSlash/tandemesh/internal/workspace"
)

type worktreeCleanup struct {
	ID         string `json:"id"`
	WorktreeID string `json:"worktreeId"`
	RunID      string `json:"runId"`
	Path       string `json:"path"`
	Repo       string `json:"repo"`
}

func worktreeCleanupLoop(ctx context.Context, c *http.Client, plane, machineID string, mgr *workspace.Manager, log *slog.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	var active sync.Map
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, plane+"/api/v1/agentd/worktree-cleanups?machineId="+machineID, nil)
			resp, err := c.Do(req)
			if err != nil {
				continue
			}
			var body struct {
				Items []worktreeCleanup `json:"items"`
			}
			err = json.NewDecoder(resp.Body).Decode(&body)
			resp.Body.Close()
			if err != nil {
				continue
			}
			for _, item := range body.Items {
				if _, loaded := active.LoadOrStore(item.ID, true); loaded {
					continue
				}
				go func(item worktreeCleanup) {
					defer active.Delete(item.ID)
					runWorktreeCleanup(ctx, c, plane, mgr, item, log)
				}(item)
			}
		}
	}
}

func runWorktreeCleanup(ctx context.Context, c *http.Client, plane string, mgr *workspace.Manager, item worktreeCleanup, log *slog.Logger) {
	status := "done"
	message := ""
	if mgr != nil && item.RunID != "" {
		if err := mgr.Cleanup(ctx, item.RunID); err != nil {
			status = "failed"
			message = err.Error()
		}
	} else if item.Path != "" {
		if err := os.RemoveAll(item.Path); err != nil {
			status = "failed"
			message = err.Error()
		}
	}
	if status == "failed" {
		log.Warn("worktree cleanup failed", "worktree", item.WorktreeID, "err", message)
	} else {
		log.Info("worktree cleaned", "worktree", item.WorktreeID)
	}
	raw, _ := json.Marshal(map[string]string{"status": status, "error": message})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/v1/agentd/worktree-cleanups/%s", plane, item.ID), bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func workspaceGCLoop(ctx context.Context, mgr *workspace.Manager, log *slog.Logger) {
	if mgr == nil {
		return
	}
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n, err := mgr.GC(ctx, time.Now()); err != nil {
				log.Warn("workspace gc", "err", err)
			} else if n > 0 {
				log.Info("workspace gc removed", "count", n)
			}
		}
	}
}
