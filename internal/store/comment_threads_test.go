package store

import (
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestCommentThreadRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "threads.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	thread, err := st.EnsureCommentThread(ctx, &domain.CommentThread{
		ID: "ct_1", WorkspaceID: "ws_default", TaskID: "task_1",
		Provider: "github", ExternalID: "github:acme/api:issue:7", Source: "github",
	})
	if err != nil || thread == nil {
		t.Fatalf("thread = %+v, err = %v", thread, err)
	}
	if err := st.UpsertExternalComment(ctx, &domain.ExternalComment{
		ID: "pc_1", TaskID: "task_1", Provider: "github", ExternalID: "1",
		Author: "alice", Body: "hello", ThreadID: thread.ID, Direction: "inbound",
	}); err != nil {
		t.Fatal(err)
	}
	items, err := st.ListExternalComments(ctx, "task_1", 10)
	if err != nil || len(items) != 1 || items[0].ThreadID != thread.ID {
		t.Fatalf("comments = %+v, err = %v", items, err)
	}
	if err := st.SetCommentThreadResolved(ctx, thread.ID, true); err != nil {
		t.Fatal(err)
	}
	threads, _ := st.ListCommentThreads(ctx, "task_1")
	if len(threads) != 1 || !threads[0].Resolved {
		t.Fatalf("threads = %+v", threads)
	}
}
