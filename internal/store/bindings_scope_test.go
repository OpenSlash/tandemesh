package store

import (
	"context"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestFindBindingFallsBackToParentScope(t *testing.T) {
	ctx := context.Background()
	st, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, b := range []*domain.ResourceBinding{
		{ID: "bind_ws", WorkspaceID: "ws", ConnectionID: "conn", Provider: "plane", ExternalID: "acme", Enabled: true},
		{ID: "bind_p2", WorkspaceID: "ws", ConnectionID: "conn", Provider: "plane", ExternalID: "acme/p2", Enabled: true},
	} {
		if err := st.UpsertBinding(ctx, b); err != nil {
			t.Fatal(err)
		}
	}
	tests := map[string]string{"acme/p1": "bind_ws", "acme/p2": "bind_p2", "acme": "bind_ws", "other/p1": ""}
	for instance, want := range tests {
		b, err := st.FindBinding(ctx, "plane", instance)
		if err != nil {
			t.Fatal(err)
		}
		got := ""
		if b != nil {
			got = b.ID
		}
		if got != want {
			t.Errorf("FindBinding(%q) = %q, want %q", instance, got, want)
		}
	}
}
