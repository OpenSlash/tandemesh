package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestCheckFence(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	run := &domain.Run{
		ID:        "run_fence",
		Work:      domain.WorkRef{Provider: "manual", Instance: "test", Type: "task", ID: "1"},
		Status:    domain.RunQueued,
		CreatedAt: time.Now().UTC(),
	}
	if err := st.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	ok, err := st.CheckFence(ctx, run.ID, 0)
	if err != nil || !ok {
		t.Fatalf("expected fence 0 to match: ok=%v err=%v", ok, err)
	}
	ok, err = st.CheckFence(ctx, run.ID, 1)
	if err != nil || ok {
		t.Fatalf("expected fence 1 to be stale: ok=%v err=%v", ok, err)
	}

	if _, err := st.DB().ExecContext(ctx, `UPDATE runs SET fence = 2 WHERE id = ?`, run.ID); err != nil {
		t.Fatalf("bump fence: %v", err)
	}
	ok, err = st.CheckFence(ctx, run.ID, 2)
	if err != nil || !ok {
		t.Fatalf("expected fence 2 to match: ok=%v err=%v", ok, err)
	}

	ok, err = st.CheckFence(ctx, "run_missing", 1)
	if err != nil || ok {
		t.Fatalf("missing run should not match: ok=%v err=%v", ok, err)
	}
}

func TestAgentAndBindingCRUD(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	agent := &domain.Agent{
		ID:          "agt_test",
		WorkspaceID: "ws_default",
		Name:        "fix-bug",
		Backend:     "claude",
		Skills:      []string{"go"},
		Repos:       []string{"acme/api"},
		Tools:       map[string]any{"ask": []any{"push"}},
	}
	if err := st.UpsertAgent(ctx, agent); err != nil {
		t.Fatalf("upsert agent: %v", err)
	}
	got, err := st.GetAgent(ctx, agent.ID)
	if err != nil || got == nil {
		t.Fatalf("get agent: %v %v", got, err)
	}
	if got.Name != "fix-bug" || len(got.Skills) != 1 {
		t.Fatalf("unexpected agent: %+v", got)
	}
	agents, err := st.ListAgents(ctx, "ws_default")
	if err != nil || len(agents) != 1 {
		t.Fatalf("list agents: %v %d", err, len(agents))
	}

	automation := &domain.Automation{
		ID:           "auto_test",
		WorkspaceID:  "ws_default",
		Name:         "label trigger",
		Provider:     "github",
		Instance:     "acme/api",
		AgentID:      agent.ID,
		TriggerKind:  domain.TriggerLabel,
		TriggerValue: "agent:fix",
		Enabled:      true,
	}
	if err := st.UpsertAutomation(ctx, automation); err != nil {
		t.Fatalf("upsert automation: %v", err)
	}
	a, err := st.GetAutomation(ctx, automation.ID)
	if err != nil || a == nil {
		t.Fatalf("get automation: %v %v", a, err)
	}
	if !a.Enabled || a.TriggerKind != domain.TriggerLabel {
		t.Fatalf("unexpected automation: %+v", a)
	}
	list, err := st.ListAutomations(ctx, "ws_default")
	if err != nil || len(list) != 1 {
		t.Fatalf("list automations: %v %d", err, len(list))
	}

	if ok, err := st.DeleteAutomation(ctx, automation.ID); err != nil || !ok {
		t.Fatalf("delete automation: %v %v", ok, err)
	}
	if ok, err := st.DeleteAgent(ctx, agent.ID); err != nil || !ok {
		t.Fatalf("delete agent: %v %v", ok, err)
	}
}

func TestApprovals(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	run := &domain.Run{
		ID:        "run_apr",
		Work:      domain.WorkRef{Provider: "manual", Type: "task", ID: "1"},
		Status:    domain.RunRunning,
		CreatedAt: time.Now().UTC(),
	}
	if err := st.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	approval := &domain.Approval{
		ID:          "apr_1",
		RunID:       run.ID,
		Tool:        "bash",
		Command:     "git push",
		Risk:        "high",
		RequestedAt: time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(15 * time.Minute),
	}
	if err := st.CreateApproval(ctx, approval); err != nil {
		t.Fatalf("create approval: %v", err)
	}
	pending, err := st.ListApprovals(ctx, domain.ApprovalPending, "")
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending = %d, err = %v", len(pending), err)
	}
	ok, err := st.DecideApproval(ctx, approval.ID, domain.ApprovalApproved, "alice")
	if err != nil || !ok {
		t.Fatalf("decide: %v %v", ok, err)
	}
	ok, err = st.DecideApproval(ctx, approval.ID, domain.ApprovalDenied, "bob")
	if err != nil || ok {
		t.Fatalf("second decision should fail: %v %v", ok, err)
	}
	got, err := st.GetApproval(ctx, approval.ID)
	if err != nil || got == nil || got.Status != domain.ApprovalApproved || got.DecidedBy != "alice" {
		t.Fatalf("unexpected approval: %+v err=%v", got, err)
	}
}
