package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestConnectionBindingsAPI(t *testing.T) {
	ts := newTestServer(t)
	connectBody := `{"provider":"feishu","workspaceId":"ws_default","name":"Feishu","fields":{"webhookUrl":"https://example.com/hook"}}`
	resp, err := http.Post(ts.URL+"/api/v1/integrations/connect", "application/json", bytes.NewReader([]byte(connectBody)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("connect status = %d", resp.StatusCode)
	}
	var connected struct {
		Connection domain.Connection `json:"connection"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&connected); err != nil {
		t.Fatal(err)
	}

	body := `{"externalId":"C123","name":"backend","kind":"channel","direction":"read"}`
	resp, err = http.Post(ts.URL+"/api/v1/connections/"+connected.Connection.ID+"/bindings", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create binding status = %d", resp.StatusCode)
	}
	var created struct {
		Binding domain.ResourceBinding `json:"binding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Binding.ExternalID != "C123" || created.Binding.Direction != "read" {
		t.Fatalf("binding = %+v", created.Binding)
	}

	resp, err = http.Get(ts.URL + "/api/v1/connections/" + connected.Connection.ID + "/bindings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list bindings status = %d", resp.StatusCode)
	}
	var listed struct {
		Items []domain.ResourceBinding `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Items) != 1 {
		t.Fatalf("items = %+v", listed.Items)
	}
}
