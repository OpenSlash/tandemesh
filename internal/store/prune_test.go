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

func TestListTasksForMember(t *testing.T) {
	ctx := context.Background()
	st, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	add := func(id, assignee string) {
		if _, err := st.EnsureTaskIn(ctx, "ws", domain.WorkRef{Provider: "plane", Instance: "acme/p1", Type: "issue", ID: id}, domain.Snapshot{Title: id, Assignee: assignee}); err != nil {
			t.Fatal(err)
		}
	}
	add("mine", "Alice Zhang")
	add("theirs", "Bob")
	add("nobody", "")
	if err := st.UpsertMemberIdentity(ctx, &domain.MemberIdentity{ID: "mid_1", WorkspaceID: "ws", MemberID: "me", Provider: "plane", ExternalID: "u-1", Login: "Alice Zhang"}); err != nil {
		t.Fatal(err)
	}
	tasks, err := st.ListTasks(ctx, TaskFilter{Workspace: "ws", Member: "me"})
	if err != nil || len(tasks) != 1 || tasks[0].WorkID != "mine" {
		t.Fatalf("tasks = %+v err=%v", tasks, err)
	}
}

func TestCancelledDependencyIsSatisfied(t *testing.T) {
	ctx := context.Background()
	st, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	dep, _ := st.EnsureTaskIn(ctx, "ws", domain.WorkRef{Provider: "plane", Instance: "acme/p1", Type: "issue", ID: "dep"}, domain.Snapshot{Title: "dep"})
	task, _ := st.EnsureTaskIn(ctx, "ws", domain.WorkRef{Provider: "plane", Instance: "acme/p1", Type: "issue", ID: "task"}, domain.Snapshot{Title: "task"})
	if _, err := st.DB().ExecContext(ctx, `INSERT INTO task_dependencies (task_id, depends_on) VALUES (?, ?)`, task.ID, dep.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.SetTaskStatus(ctx, dep.ID, domain.TaskCancelled); err != nil {
		t.Fatal(err)
	}
	if err := st.EvaluateTaskReadiness(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := st.GetTask(ctx, task.ID)
	if got.ExecutionState != domain.TaskExecutionReady {
		t.Fatalf("a cancelled dependency should not block: %q", got.ExecutionState)
	}
}
