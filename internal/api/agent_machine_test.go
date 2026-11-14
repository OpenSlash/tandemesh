package api

import (
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestAgentMachineBinding(t *testing.T) {
	ts := newTestServer(t)
	machine := postJSON[domain.Machine](t, ts.URL+"/api/v1/machines/register", map[string]any{
		"token": "", "name": "builder", "agents": []string{"claude"}, "concurrency": 1,
	})
	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "bound", "backend": "claude", "machineId": machine.ID,
	})
	if agent.MachineID != machine.ID {
		t.Fatalf("machineId = %q", agent.MachineID)
	}
	if status := postStatus(t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "bad", "backend": "claude", "machineId": "mch_missing",
	}); status != 400 {
		t.Fatalf("invalid machine status = %d", status)
	}
}
