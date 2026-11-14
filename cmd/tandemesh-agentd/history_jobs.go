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

	"github.com/OpenSlash/tandemesh/internal/history"
)

type historyScanJob struct {
	ID        string `json:"id"`
	MachineID string `json:"machineId"`
}

func historyScanLoop(ctx context.Context, c *http.Client, plane, machineID string, log *slog.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	var active sync.Map
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, plane+"/api/v1/agentd/history-scans/"+machineID, nil)
			resp, err := c.Do(req)
			if err != nil {
				continue
			}
			var body struct {
				Items []historyScanJob `json:"items"`
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
				go func(job historyScanJob) {
					defer active.Delete(job.ID)
					runHistoryScan(ctx, c, plane, machineID, job, log)
				}(job)
			}
		}
	}
}

func runHistoryScan(ctx context.Context, c *http.Client, plane, machineID string, job historyScanJob, log *slog.Logger) {
	payload := map[string]any{"items": []history.Entry{}}
	home, err := os.UserHomeDir()
	if err != nil {
		payload["error"] = err.Error()
	} else if entries, err := history.Scan(home, 200); err != nil {
		payload["error"] = err.Error()
	} else {
		payload["items"] = entries
	}
	raw, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/v1/agentd/history-scans/%s/%s", plane, machineID, job.ID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		log.Warn("history scan upload", "scan", job.ID, "err", err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	log.Info("history scan completed", "scan", job.ID, "entries", len(payload["items"].([]history.Entry)))
}
