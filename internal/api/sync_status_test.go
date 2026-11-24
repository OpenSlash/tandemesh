package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestConnectionSyncStatusIncludesWebhook(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	body := []byte(`{"provider":"feishu","workspaceId":"ws_default","fields":{"webhookUrl":"https://example.com/hook"}}`)
	resp, err := http.Post(ts.URL+"/api/v1/integrations/connect", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("connect = %d %s", resp.StatusCode, data)
	}
	var connected struct {
		Connection domain.Connection `json:"connection"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&connected); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateWebhookDelivery(t.Context(), &domain.WebhookDelivery{
		ID: "whd_test", Provider: "feishu", Delivery: "d1", Kind: "message", Status: "accepted",
	}); err != nil {
		t.Fatal(err)
	}
	resp, err = http.Get(ts.URL + "/api/v1/connections/" + connected.Connection.ID + "/sync-status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(data), `"webhook"`) || !strings.Contains(string(data), "accepted") {
		t.Fatalf("body = %s", data)
	}
}
