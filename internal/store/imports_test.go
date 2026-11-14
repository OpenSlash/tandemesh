package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestImportBatchRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "imports.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	batch := &domain.ImportBatch{
		ID: "imp_1", WorkspaceID: "ws_default", Provider: "jira", ConnectionID: "conn_1",
		Query: "project = ENG", Status: "running", CreatedAt: time.Now().UTC(),
	}
	if err := st.CreateImportBatch(ctx, batch); err != nil {
		t.Fatal(err)
	}
	if err := st.AddImportItem(ctx, batch.ID, "task_1", "ENG-1"); err != nil {
		t.Fatal(err)
	}
	items, err := st.ListImportItems(ctx, batch.ID)
	if err != nil || len(items) != 1 || items[0] != "task_1" {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
	batch.Status, batch.Imported = "completed", 1
	if err := st.FinishImportBatch(ctx, batch); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetImportBatch(ctx, batch.ID)
	if err != nil || got == nil || got.Status != "completed" || got.Imported != 1 {
		t.Fatalf("batch = %+v, err = %v", got, err)
	}
}
