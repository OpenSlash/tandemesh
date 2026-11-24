package store

import (
	"context"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestGitJobLifecycle(t *testing.T) {
	st, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	job, err := st.CreateGitJob(context.Background(), &domain.GitJob{
		SessionID: "ses_test", WorktreeID: "wt_test", MachineID: "mch_test",
		Path: "/tmp/worktree", Action: "ship", Message: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	pending, err := st.PendingGitJobs(context.Background(), "mch_test", 10)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending = %+v err=%v", pending, err)
	}
	if err := st.FinishGitJob(context.Background(), job.ID, "succeeded", `{"branch":"main"}`, ""); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetGitJob(context.Background(), job.ID)
	if err != nil || got == nil || got.Status != "succeeded" {
		t.Fatalf("job = %+v err=%v", got, err)
	}
}
