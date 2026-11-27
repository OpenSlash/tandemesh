package store

import (
	"context"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestPruneUnboundTask(t *testing.T) {
	ctx := context.Background()
	st, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	ensure := func(id string) *domain.Task {
		task, err := st.EnsureTaskIn(ctx, "ws", domain.WorkRef{Provider: "plane", Instance: "acme/p1", Type: "issue", ID: id}, domain.Snapshot{Title: id})
		if err != nil {
			t.Fatal(err)
		}
		return task
	}
	idle, worked := ensure("idle"), ensure("worked")
	if _, err := st.DB().ExecContext(ctx, `INSERT INTO agent_sessions (id, task_id, created_at_ms, updated_at_ms) VALUES ('ses_1', ?, 0, 0)`, worked.ID); err != nil {
		t.Fatal(err)
	}

	if deleted, err := st.PruneUnboundTask(ctx, idle.ID); err != nil || !deleted {
		t.Fatalf("idle task: deleted=%v err=%v", deleted, err)
	}
	if got, _ := st.GetTask(ctx, idle.ID); got != nil {
		t.Fatalf("idle task still present: %+v", got)
	}
	if deleted, err := st.PruneUnboundTask(ctx, worked.ID); err != nil || deleted {
		t.Fatalf("task with a session: deleted=%v err=%v", deleted, err)
	}
	if got, _ := st.GetTask(ctx, worked.ID); got == nil || got.PlatformState != domain.PlatformStateUnbound {
		t.Fatalf("task with history should be kept as unbound: %+v", got)
	}
	if listed, _ := st.ListTasks(ctx, TaskFilter{Workspace: "ws"}); len(listed) != 0 {
		t.Fatalf("unbound task listed: %+v", listed)
	}
}
