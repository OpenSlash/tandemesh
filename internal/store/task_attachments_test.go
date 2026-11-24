package store

import (
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestTaskAttachmentRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "attachments.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	item := &domain.TaskAttachment{
		ID: "att_1", TaskID: "task_1", Provider: "jira", ExternalID: "10001",
		Name: "spec.pdf", URL: "https://example.test/spec.pdf", Mime: "application/pdf", Size: 42, Author: "Alice",
	}
	if err := st.UpsertTaskAttachment(ctx, item); err != nil {
		t.Fatal(err)
	}
	items, err := st.ListTaskAttachments(ctx, "task_1")
	if err != nil || len(items) != 1 || items[0].Name != "spec.pdf" {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
}

func TestReplaceTaskAttachmentsClears(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "attachments-replace.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.ReplaceTaskAttachments(ctx, "task_1", "jira", "issue", []*domain.TaskAttachment{{
		ID: "att_1", TaskID: "task_1", Provider: "jira", ExternalID: "1", Name: "a",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.ReplaceTaskAttachments(ctx, "task_1", "jira", "comment", []*domain.TaskAttachment{{
		ID: "att_2", TaskID: "task_1", Provider: "jira", ExternalID: "2", Name: "b", Source: "comment",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.ReplaceTaskAttachments(ctx, "task_1", "jira", "issue", nil); err != nil {
		t.Fatal(err)
	}
	items, _ := st.ListTaskAttachments(ctx, "task_1")
	if len(items) != 1 || items[0].Source != "comment" {
		t.Fatalf("items = %+v", items)
	}
}
