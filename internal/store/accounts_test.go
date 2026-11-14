package store

import (
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestConnectionAccountDefaultSwitch(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "accounts.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	first := &domain.ConnectionAccount{
		ID: "acct_1", WorkspaceID: "ws_default", ConnectionID: "conn_1", Provider: "github",
		Kind: "user", Login: "alice", CredentialSubject: "conn_1", IsDefault: true,
	}
	second := &domain.ConnectionAccount{
		ID: "acct_2", WorkspaceID: "ws_default", ConnectionID: "conn_1", Provider: "github",
		Kind: "user", Login: "bob", CredentialSubject: "acct_2",
	}
	for _, account := range []*domain.ConnectionAccount{first, second} {
		if err := st.UpsertConnectionAccount(ctx, account); err != nil {
			t.Fatal(err)
		}
	}
	items, err := st.ListConnectionAccounts(ctx, "conn_1")
	if err != nil || len(items) != 2 {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
	if !items[0].IsDefault || items[0].ID != "acct_1" {
		t.Fatalf("default = %+v", items[0])
	}
	if _, err := st.SetDefaultConnectionAccount(ctx, "acct_2"); err != nil {
		t.Fatal(err)
	}
	items, _ = st.ListConnectionAccounts(ctx, "conn_1")
	if !items[0].IsDefault || items[0].ID != "acct_2" {
		t.Fatalf("default after switch = %+v", items[0])
	}
}
