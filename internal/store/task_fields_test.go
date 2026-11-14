package store

import (
	"path/filepath"
	"testing"
)

func TestTaskFieldsRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "task-fields.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.ReplaceTaskFields(ctx, "task_1", "jira", map[string][]string{
		"priority":   {"High"},
		"components": {"API", "Web"},
	}); err != nil {
		t.Fatal(err)
	}
	items, err := st.ListTaskFields(ctx, "task_1")
	if err != nil || len(items) != 3 {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
	if err := st.ReplaceTaskFields(ctx, "task_1", "jira", nil); err != nil {
		t.Fatal(err)
	}
	items, _ = st.ListTaskFields(ctx, "task_1")
	if len(items) != 0 {
		t.Fatalf("items = %+v", items)
	}
}
