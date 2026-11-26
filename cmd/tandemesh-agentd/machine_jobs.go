package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/OpenSlash/tandemesh/internal/agent"
	"github.com/OpenSlash/tandemesh/internal/domain"
)

type machineJob struct {
	ID      string          `json:"id"`
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

type acpInstallPayload struct {
	RegistryID string `json:"registryId"`
	Version    string `json:"version,omitempty"`
}

// machineJobLoop runs plane-queued jobs one at a time; installs and checks
// both rewrite the host's runtime table, so serializing them keeps it simple.
func machineJobLoop(ctx context.Context, c *http.Client, plane, machineID string, host *runtimeHost, log *slog.Logger) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, plane+"/api/v1/agentd/machine-jobs/"+machineID, nil)
			resp, err := c.Do(req)
			if err != nil {
				continue
			}
			var body struct {
				Items []machineJob `json:"items"`
			}
			err = json.NewDecoder(resp.Body).Decode(&body)
			resp.Body.Close()
			if err != nil {
				continue
			}
			for _, job := range body.Items {
				result, err := runMachineJob(ctx, job, host)
				finishMachineJob(ctx, c, plane, machineID, job, result, err, host, log)
			}
		}
	}
}

func runMachineJob(ctx context.Context, job machineJob, host *runtimeHost) (any, error) {
	switch job.Kind {
	case domain.MachineJobRuntimeCheck:
		host.refresh(ctx)
		return nil, nil
	case domain.MachineJobACPInstall:
		var payload acpInstallPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return nil, fmt.Errorf("decode install payload: %w", err)
		}
		install, err := agent.InstallACP(ctx, payload.RegistryID, host.acpHome)
		host.refresh(ctx)
		return install, err
	case domain.MachineJobACPUninstall:
		var payload acpInstallPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return nil, fmt.Errorf("decode uninstall payload: %w", err)
		}
		err := uninstallACP(payload, host.acpHome)
		host.refresh(ctx)
		return nil, err
	}
	return nil, fmt.Errorf("unknown machine job kind %q", job.Kind)
}

// uninstallACP removes one version, or every installed version when the
// payload names none.
func uninstallACP(payload acpInstallPayload, home string) error {
	if payload.Version != "" {
		return agent.UninstallACP(payload.RegistryID, payload.Version, home)
	}
	installs, err := agent.ListACPInstalls(home)
	if err != nil {
		return err
	}
	removed := 0
	for _, install := range installs {
		if install.ID != payload.RegistryID {
			continue
		}
		if err := agent.UninstallACP(install.ID, install.Version, home); err != nil {
			return err
		}
		removed++
	}
	if removed == 0 {
		return fmt.Errorf("registry agent %q is not installed", payload.RegistryID)
	}
	return nil
}

func finishMachineJob(ctx context.Context, c *http.Client, plane, machineID string, job machineJob, result any, jobErr error, host *runtimeHost, log *slog.Logger) {
	body := host.report()
	body["status"] = "done"
	if jobErr != nil {
		body["status"] = "failed"
		body["error"] = jobErr.Error()
	}
	if result != nil {
		body["result"] = result
	}
	raw, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/api/v1/agentd/machine-jobs/%s/%s", plane, machineID, job.ID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		log.Warn("report machine job", "job", job.ID, "err", err)
		return
	}
	// The body only echoes the status; drain it so the connection is reused.
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	log.Info("machine job finished", "job", job.ID, "kind", job.Kind, "status", body["status"])
}
