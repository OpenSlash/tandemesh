package tenancy

import (
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestLocalTenancy(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "tenancy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateWorkspace(t.Context(), &domain.Workspace{ID: "ws_1", Name: "One", Slug: "one"}); err != nil {
		t.Fatal(err)
	}
	local := Local{Store: st}
	if !local.CanAccess(t.Context(), "me", "ws_1") || local.Role(t.Context(), "me", "ws_1") != "owner" {
		t.Fatal("local tenancy must grant owner access")
	}
	ids, err := local.Workspaces(t.Context(), "me")
	if err != nil || len(ids) < 1 {
		t.Fatalf("workspaces = %v, err = %v", ids, err)
	}
}
