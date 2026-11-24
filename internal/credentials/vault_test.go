package credentials

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "cred.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestVaultRoundTrip(t *testing.T) {
	st := testStore(t)
	v, err := New(st, "secret", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	cred := integration.Credential{
		Provider:     "github",
		Subject:      "default",
		AccessToken:  "ghp_secret",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(time.Hour).UTC().Truncate(time.Second),
		Scopes:       []string{"repo"},
	}
	if err := v.Put(ctx, cred); err != nil {
		t.Fatalf("put: %v", err)
	}

	got, ok, err := v.Get(ctx, "github", "")
	if err != nil || !ok {
		t.Fatalf("get: %v %v", ok, err)
	}
	if got.AccessToken != cred.AccessToken || got.RefreshToken != cred.RefreshToken {
		t.Fatalf("credential mismatch: %+v", got)
	}

	meta, err := v.List(ctx)
	if err != nil || len(meta) != 1 || meta[0].Kind != "oauth2" {
		t.Fatalf("meta = %+v err=%v", meta, err)
	}

	// A different key cannot decrypt the stored value.
	other, _ := New(st, "other-secret", nil)
	if _, _, err := other.Get(ctx, "github", ""); err == nil {
		t.Fatal("wrong key decrypted the credential")
	}

	if ok, err := v.Delete(ctx, "github", ""); err != nil || !ok {
		t.Fatalf("delete: %v %v", ok, err)
	}
}

func TestVaultRequiresProvider(t *testing.T) {
	v, _ := New(testStore(t), "s", nil)
	if err := v.Put(context.Background(), integration.Credential{}); err == nil {
		t.Fatal("empty provider accepted")
	}
}
