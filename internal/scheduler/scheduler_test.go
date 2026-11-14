package scheduler

import (
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestPickMachinePrefersBoundMachine(t *testing.T) {
	now := time.Now()
	machines := []*domain.Machine{
		{ID: "mch_a", Status: "online", Agents: []string{"claude"}, Concurrency: 2, LastSeen: now},
		{ID: "mch_b", Status: "online", Agents: []string{"claude"}, Concurrency: 2, LastSeen: now},
	}
	run := &domain.Run{ID: "run_1", AgentID: "claude"}

	if got := pickMachine(machines, map[string]int{}, run.AgentID, "", nil); got == nil || got.ID != "mch_a" {
		t.Fatalf("unbound pick = %+v", got)
	}
	if got := pickMachine(machines, map[string]int{}, run.AgentID, "mch_b", nil); got == nil || got.ID != "mch_b" {
		t.Fatalf("bound pick = %+v", got)
	}
	if got := pickMachine(machines, map[string]int{}, run.AgentID, "mch_missing", nil); got != nil {
		t.Fatalf("missing machine pick = %+v", got)
	}
}

func TestPickMachineUsesLowestLoad(t *testing.T) {
	now := time.Now()
	machines := []*domain.Machine{
		{ID: "mch_busy", Status: "online", Agents: []string{"claude"}, Concurrency: 4, LastSeen: now},
		{ID: "mch_idle", Status: "online", Agents: []string{"claude"}, Concurrency: 4, LastSeen: now},
	}
	busy := map[string]int{"mch_busy": 3, "mch_idle": 0}
	if got := pickMachine(machines, busy, "claude", "", nil); got == nil || got.ID != "mch_idle" {
		t.Fatalf("load-aware pick = %+v", got)
	}
}

func TestPickMachineRequiresNodeLabels(t *testing.T) {
	now := time.Now()
	machines := []*domain.Machine{
		{ID: "mch_gpu", Status: "online", Agents: []string{"claude"}, Labels: []string{"os=linux", "gpu"}, Concurrency: 1, LastSeen: now},
		{ID: "mch_cpu", Status: "online", Agents: []string{"claude"}, Labels: []string{"os=linux"}, Concurrency: 1, LastSeen: now},
	}
	if got := pickMachine(machines, map[string]int{}, "claude", "", []string{"gpu"}); got == nil || got.ID != "mch_gpu" {
		t.Fatalf("label pick = %+v", got)
	}
	if got := pickMachine(machines, map[string]int{}, "claude", "", []string{"arm64"}); got != nil {
		t.Fatalf("unsatisfied label pick = %+v", got)
	}
}

func TestHasAllLabels(t *testing.T) {
	if !hasAllLabels([]string{"gpu", "os=linux"}, []string{"gpu"}) {
		t.Fatal("expected gpu match")
	}
	if hasAllLabels([]string{"gpu"}, []string{"gpu", "arm64"}) {
		t.Fatal("unexpected partial match")
	}
}
func TestNodeSelectors(t *testing.T) {
	got := nodeSelectors([]string{"review", "node:gpu", "node:zone=cn"})
	if len(got) != 2 || got[0] != "gpu" || got[1] != "zone=cn" {
		t.Fatalf("selectors = %#v", got)
	}
	if len(nodeSelectors([]string{"review"})) != 0 {
		t.Fatal("review should not be a node selector")
	}
}
