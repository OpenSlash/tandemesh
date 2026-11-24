package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestTaskPullRequestBacklinkPreserved(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "pr-backlink.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	pr := &domain.TaskPullRequest{
		ID: "pr_1", TaskID: "task_1", WorkspaceID: "ws_default", Provider: "github",
		Repo: "acme/api", Number: 7, URL: "https://github.test/pr/7", Title: "fix",
		BacklinkedAt: &now,
	}
	if err := st.UpsertTaskPullRequest(ctx, pr); err != nil {
		t.Fatal(err)
	}
	pr.BacklinkedAt = nil
	if err := st.UpsertTaskPullRequest(ctx, pr); err != nil {
		t.Fatal(err)
	}
	items, err := st.ListTaskPullRequests(ctx, "task_1")
	if err != nil || len(items) != 1 || items[0].BacklinkedAt == nil {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
}
