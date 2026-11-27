package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"changeme/updatesig"
)

func TestUpdateControllerRequiresHeader(t *testing.T) {
	c := &updateController{}
	rec := httptest.NewRecorder()
	c.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/desktop/update/install", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("install without header: %d", rec.Code)
	}
}

func TestUpdateControllerCheck(t *testing.T) {
	withVersion(t, "0.2.0")
	pub, priv, _ := updatesig.GenerateKey()
	oldKey := updatePublicKey
	updatePublicKey = pub
	t.Cleanup(func() { updatePublicKey = oldKey })
	m := updatesig.Manifest{Version: "0.3.0", URL: "http://x/a.zip", SHA256: "ab", Notes: "notes"}
	if err := updatesig.Sign(&m, priv); err != nil {
		t.Fatal(err)
	}
	feed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(m) }))
	defer feed.Close()
	t.Setenv("TANDEMESH_UPDATE_URL", feed.URL)

	c := &updateController{}
	req := httptest.NewRequest(http.MethodGet, "/desktop/update", nil)
	req.Header.Set("X-Tandemesh-Desktop", "1")
	rec := httptest.NewRecorder()
	c.ServeHTTP(rec, req)
	var got updateStatus
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || !got.Supported || got.Available != "0.3.0" || got.Version != "0.2.0" || c.pending == nil {
		t.Fatalf("status %d %+v pending=%v", rec.Code, got, c.pending)
	}
}
