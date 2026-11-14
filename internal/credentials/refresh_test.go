package credentials

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestEnsureRefreshesExpiredToken(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "refresh.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "old-refresh" {
			t.Errorf("unexpected form: %v", r.Form)
		}
		_, _ = w.Write([]byte(`{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`))
	}))
	defer srv.Close()

	v, _ := New(st, "secret", slog.New(slog.NewTextHandler(io.Discard, nil)))
	v.HTTPClient = srv.Client()
	v.OAuth = OAuthConfig{Clients: map[string]OAuthClient{
		"linear": {ClientID: "id", ClientSecret: "sec", TokenURL: srv.URL},
	}}

	if err := v.Put(ctx, integration.Credential{
		Provider: "linear", AccessToken: "old-access", RefreshToken: "old-refresh",
		ExpiresAt: time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatal(err)
	}

	cred, ok, err := v.Ensure(ctx, "linear", "default")
	if err != nil || !ok {
		t.Fatalf("ensure: %v %v", ok, err)
	}
	if cred.AccessToken != "new-access" || cred.RefreshToken != "new-refresh" {
		t.Fatalf("credential = %+v", cred)
	}
	if calls != 1 {
		t.Fatalf("refresh calls = %d", calls)
	}

	// The refreshed token is valid, so a second call does not refresh.
	if _, _, err := v.Ensure(ctx, "linear", "default"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("unexpected refresh, calls = %d", calls)
	}
}

func TestEnsureWithoutRefreshConfig(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "norefresh.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	_ = st.Migrate(ctx)

	v, _ := New(st, "secret", nil)
	_ = v.Put(ctx, integration.Credential{Provider: "github", AccessToken: "tok"})
	cred, ok, err := v.Ensure(ctx, "github", "default")
	if err != nil || !ok || cred.AccessToken != "tok" {
		t.Fatalf("cred = %+v ok=%v err=%v", cred, ok, err)
	}
}
