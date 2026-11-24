package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestMobilePairingFlow(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "mobile.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{User: "me", LocalToken: "desk-token"}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	// Creating a pairing requires the local (desktop) credential.
	resp, err := http.Post(ts.URL+"/api/v1/mobile/pairings", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated pairing status = %d", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/mobile/pairings", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "Bearer desk-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create pairing status = %d body=%s", resp.StatusCode, raw)
	}
	var created struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(raw, &created); err != nil {
		t.Fatal(err)
	}
	if created.Code == "" {
		t.Fatal("empty pairing code")
	}

	// Exchanging the code does not require the desktop token.
	exchange, _ := json.Marshal(map[string]string{"code": created.Code, "deviceName": "Pixel", "platform": "android"})
	resp, err = http.Post(ts.URL+"/api/v1/mobile/pairings/exchange", "application/json", bytes.NewReader(exchange))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("exchange status = %d body=%s", resp.StatusCode, raw)
	}
	var token struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &token); err != nil || token.Token == "" {
		t.Fatalf("exchange token: %s %v", raw, err)
	}

	// The bearer token identifies the local user and can list devices.
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/v1/mobile/devices", nil)
	req.Header.Set("Authorization", "Bearer "+token.Token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !bytes.Contains(raw, []byte(`"name":"Pixel"`)) {
		t.Fatalf("devices status = %d body=%s", resp.StatusCode, raw)
	}

	// The same code cannot be exchanged twice.
	resp, err = http.Post(ts.URL+"/api/v1/mobile/pairings/exchange", "application/json", bytes.NewReader(exchange))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("reused code status = %d", resp.StatusCode)
	}

	// Revoking the device invalidates the token.
	var devices struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &devices); err != nil || len(devices.Items) != 1 {
		t.Fatalf("decode devices: %s %v", raw, err)
	}
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/mobile/devices/"+devices.Items[0].ID, nil)
	req.Header.Set("Authorization", "Bearer "+token.Token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token.Token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked token status = %d", resp.StatusCode)
	}
}
