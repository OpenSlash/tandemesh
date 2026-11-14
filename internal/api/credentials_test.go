package api

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/credentials"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestCredentialsAPI(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "credapi.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	vault, err := credentials.New(st, "test-secret", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	srv := New(config.Config{}, st, nil, hub.New(), testRegistry(), vault, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	body := []byte(`{"provider":"github","accessToken":"ghp_supersecret"}`)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/credentials", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("put status = %d", resp.StatusCode)
	}

	listResp, err := http.Get(ts.URL + "/api/v1/credentials")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(listResp.Body)
	listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d", listResp.StatusCode)
	}
	if bytes.Contains(raw, []byte("supersecret")) {
		t.Fatalf("token leaked: %s", raw)
	}
	if !bytes.Contains(raw, []byte(`"provider":"github"`)) {
		t.Fatalf("metadata missing: %s", raw)
	}

	del, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/credentials/github/default", nil)
	if err != nil {
		t.Fatal(err)
	}
	delResp, err := http.DefaultClient.Do(del)
	if err != nil {
		t.Fatal(err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", delResp.StatusCode)
	}
}
