package api

import (
	"context"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestWorktreeCleanupLifecycle(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	ctx := context.Background()
	if err := st.UpsertMachine(ctx, &domain.Machine{ID: "mch_wt", Name: "wt host", Status: "online", Concurrency: 1}); err != nil {
		t.Fatal(err)
	}
	task := mustTask(t, st, "wt", "Worktree task")
	run := postJSON[domain.Run](t, ts.URL+"/api/v1/tasks/"+task.ID+"/runs", map[string]any{
		"agentId": "claude", "repo": "acme/api",
	})
	if _, ok, err := st.AssignRun(ctx, run.ID, "mch_wt", time.Now().Add(time.Hour)); err != nil || !ok {
		t.Fatalf("assign run ok=%v err=%v", ok, err)
	}
	if err := st.MarkWorktreeReady(ctx, run.ID, "/tmp/orca/wt", "orca/test"); err != nil {
		t.Fatal(err)
	}
	worktrees := getAt[struct {
		Worktrees []domain.Worktree `json:"worktrees"`
	}](t, ts.URL+"/api/v1/worktrees?taskId="+task.ID)
	if len(worktrees.Worktrees) != 1 {
		t.Fatalf("worktrees = %+v", worktrees.Worktrees)
	}
	wt := worktrees.Worktrees[0]
	archived := postJSON[domain.Worktree](t, ts.URL+"/api/v1/worktrees/"+wt.ID+"/archive", map[string]any{})
	if archived.Status != "archived" {
		t.Fatalf("archived = %+v", archived)
	}
	cleanups := getAt[struct {
		Items []domain.WorktreeCleanup `json:"items"`
	}](t, ts.URL+"/api/v1/agentd/worktree-cleanups?machineId=mch_wt")
	if len(cleanups.Items) != 1 || cleanups.Items[0].WorktreeID != wt.ID {
		t.Fatalf("cleanups = %+v", cleanups.Items)
	}
	postJSON[map[string]string](t, ts.URL+"/api/v1/agentd/worktree-cleanups/"+cleanups.Items[0].ID, map[string]any{"status": "done"})
	final := getAt[domain.Worktree](t, ts.URL+"/api/v1/worktrees/"+wt.ID)
	if final.Status != "removed" {
		t.Fatalf("final = %+v", final)
	}
}

func TestWorktreeRetentionCandidates(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	task := mustTask(t, st, "ret", "Retention")
	_ = postJSON[domain.Run](t, ts.URL+"/api/v1/tasks/"+task.ID+"/runs", map[string]any{"agentId": "claude", "repo": "acme/api"})
	res := getAt[struct {
		Worktrees []domain.Worktree `json:"worktrees"`
	}](t, ts.URL+"/api/v1/worktrees/retention?workspaceId=ws_default&olderThanDays=7")
	if res.Worktrees == nil {
		t.Fatalf("retention response missing")
	}
}
