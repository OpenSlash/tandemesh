package store

import (
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestTaskCommitRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "commits.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	link := &domain.TaskCommitLink{
		ID: "commit_1", TaskID: "task_1", WorkspaceID: "ws_default", Provider: "github",
		Repo: "acme/api", SHA: "abc123", URL: "https://github.test/c", Message: "fix ENG-42", Author: "alice",
	}
	if err := st.UpsertTaskCommit(ctx, link); err != nil {
		t.Fatal(err)
	}
	items, err := st.ListTaskCommits(ctx, "task_1")
	if err != nil || len(items) != 1 || items[0].SHA != "abc123" {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
}
