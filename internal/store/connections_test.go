package store

import (
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestConnectionSyncDirectionRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "connections.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	conn := &domain.Connection{
		ID: "conn_test", WorkspaceID: "ws_default", Provider: "linear",
		Name: "Linear", Status: domain.ConnectionConnected, SyncDirection: "read", AutomationID: "auto_test", Enabled: true,
	}
	if err := st.UpsertConnection(ctx, conn); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetConnection(ctx, "conn_test")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.SyncDirection != "read" || got.AutomationID != "auto_test" {
		t.Fatalf("connection = %+v", got)
	}
}
