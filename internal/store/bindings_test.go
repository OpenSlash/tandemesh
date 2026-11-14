package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestResourceBindingRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "bindings.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	binding := &domain.ResourceBinding{
		ID: "bind_test", WorkspaceID: "ws_default", ConnectionID: "conn_test",
		Provider: "github", ExternalID: "acme/api", Name: "acme/api",
		Kind: "repo", Direction: "read", Settings: map[string]string{"label": "agent"}, Enabled: true,
	}
	if err := st.UpsertBinding(ctx, binding); err != nil {
		t.Fatal(err)
	}
	items, err := st.ListBindings(ctx, "ws_default", "conn_test")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ExternalID != "acme/api" || items[0].Direction != "read" {
		t.Fatalf("bindings = %+v", items)
	}
	if items[0].Settings["label"] != "agent" {
		t.Fatalf("settings = %+v", items[0].Settings)
	}
}

func TestBindingSyncCursorRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "binding-cursor.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	cursor, err := st.GetBindingSyncCursor(ctx, "conn_1", "bind_1")
	if err != nil || cursor.Cursor.IsZero() == false {
		t.Fatalf("empty cursor = %+v, %v", cursor, err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	cursor.Cursor = now
	cursor.LastSyncedAt = now
	if err := st.SaveBindingSyncCursor(ctx, cursor); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetBindingSyncCursor(ctx, "conn_1", "bind_1")
	if err != nil || !got.Cursor.Equal(now) {
		t.Fatalf("cursor = %+v, %v", got, err)
	}
}
