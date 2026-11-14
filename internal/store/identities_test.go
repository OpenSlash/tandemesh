package store

import (
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestExternalIdentityResolve(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "identities.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	identity := &domain.ExternalIdentity{
		ID: "ext_1", WorkspaceID: "ws_default", Provider: "github",
		ExternalID: "12345", Login: "alice", Email: "alice@example.com", MemberID: "alice",
	}
	if err := st.UpsertExternalIdentity(ctx, identity); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"12345", "alice"} {
		got, err := st.ResolveExternalIdentity(ctx, "ws_default", "github", key)
		if err != nil || got == nil || got.MemberID != "alice" {
			t.Fatalf("resolve %s = %+v, %v", key, got, err)
		}
	}
	items, err := st.ListExternalIdentities(ctx, "ws_default", "github")
	if err != nil || len(items) != 1 {
		t.Fatalf("items = %+v, %v", items, err)
	}
}

func TestMemberIdentityRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "member-identities.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	identity := &domain.MemberIdentity{
		ID: "mid_1", WorkspaceID: "ws_default", MemberID: "alice", Provider: "github",
		ExternalID: "123", Login: "alice", CredentialSubject: "mid:mid_1",
	}
	if err := st.UpsertMemberIdentity(ctx, identity); err != nil {
		t.Fatal(err)
	}
	items, err := st.ListMemberIdentities(ctx, "ws_default", "alice")
	if err != nil || len(items) != 1 || items[0].Provider != "github" {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
}
