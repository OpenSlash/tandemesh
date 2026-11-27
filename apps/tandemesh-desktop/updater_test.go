package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"changeme/updatesig"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"0.1.0", "0.2.0", -1},
		{"1.0.0", "1.0.0", 0},
		{"2.0.0", "1.9.9", 1},
		{"v0.2.0", "0.1.0-8-g02029c9", 1},
		{"0.1.0", "0.1.0-8-g02029c9", 0},
	}
	for _, tt := range tests {
		if got := compareVersions(tt.a, tt.b); got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func withVersion(t *testing.T, v string) {
	old := version
	version = v
	t.Cleanup(func() { version = old })
}

func TestUpdateManifestURL(t *testing.T) {
	withVersion(t, "dev")
	t.Setenv("TANDEMESH_UPDATE_URL", "")
	if got := updateManifestURL(); got != "" {
		t.Fatalf("dev build checks %q", got)
	}
	t.Setenv("TANDEMESH_UPDATE_URL", "http://mirror/update.json")
	if got := updateManifestURL(); got != "http://mirror/update.json" {
		t.Fatalf("override ignored: %q", got)
	}
}

func TestFetchUpdate(t *testing.T) {
	withVersion(t, "0.2.0")
	pub, priv, err := updatesig.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	oldKey := updatePublicKey
	updatePublicKey = pub
	t.Cleanup(func() { updatePublicKey = oldKey })

	signed := func(v string) []byte {
		m := updatesig.Manifest{Version: v, URL: "http://x/a.zip", SHA256: "ab"}
		if err := updatesig.Sign(&m, priv); err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(m)
		return b
	}
	tampered := updatesig.Manifest{}
	_ = json.Unmarshal(signed("0.3.0"), &tampered)
	tampered.URL = "http://evil/a.zip"
	tamperedJSON, _ := json.Marshal(tampered)

	feeds := map[string][]byte{
		"/newer":    signed("0.3.0"),
		"/same":     signed("0.2.0"),
		"/unsigned": []byte(`{"version":"0.3.0","url":"http://x/a.zip","sha256":"ab"}`),
		"/tampered": tamperedJSON,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := feeds[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	ctx := context.Background()

	if m, err := fetchUpdate(ctx, srv.URL+"/newer"); err != nil || m == nil || m.Version != "0.3.0" {
		t.Fatalf("newer: %+v %v", m, err)
	}
	if m, err := fetchUpdate(ctx, srv.URL+"/same"); err != nil || m != nil {
		t.Fatalf("same: %+v %v", m, err)
	}
	if m, err := fetchUpdate(ctx, srv.URL+"/missing"); err != nil || m != nil {
		t.Fatalf("no release yet should be quiet: %+v %v", m, err)
	}
	for _, path := range []string{"/unsigned", "/tampered"} {
		if _, err := fetchUpdate(ctx, srv.URL+path); err == nil {
			t.Errorf("%s manifest accepted", path)
		}
	}
}

func TestDownloadUpdateChecksum(t *testing.T) {
	body := []byte("package")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }))
	defer srv.Close()
	sum := sha256.Sum256(body)
	ctx := context.Background()

	path, err := downloadUpdate(ctx, &updateManifest{URL: srv.URL + "/Tandemesh.zip", SHA256: hex.EncodeToString(sum[:])})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "package" {
		t.Fatalf("downloaded %q", got)
	}
	if _, err := downloadUpdate(ctx, &updateManifest{URL: srv.URL + "/Tandemesh.zip", SHA256: "00"}); err == nil {
		t.Fatal("bad checksum accepted")
	}
}
