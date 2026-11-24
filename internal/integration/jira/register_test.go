package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestRegisterWebhook(t *testing.T) {
	var payload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/webhook" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("authorization = %s", r.Header.Get("Authorization"))
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		_, _ = w.Write([]byte(`{"webhookRegistrationResult":[{"createdWebhookId":42}]}`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client()}
	id, err := a.RegisterWebhook(t.Context(), integration.Credential{AccessToken: "tok"}, srv.URL, "https://orca.test/webhooks/jira/auto_1", "")
	if err != nil {
		t.Fatal(err)
	}
	if id != "42" {
		t.Fatalf("id = %s", id)
	}
	if payload["url"] != "https://orca.test/webhooks/jira/auto_1" {
		t.Fatalf("payload = %+v", payload)
	}
}
