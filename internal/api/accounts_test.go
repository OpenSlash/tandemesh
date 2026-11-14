package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestConnectionAccountsAPI(t *testing.T) {
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
	accountsURL := ts.URL + "/api/v1/connections/" + connected.Connection.ID + "/accounts"
	resp, err = http.Get(accountsURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var initial struct {
		Items []domain.ConnectionAccount `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&initial); err != nil {
		t.Fatal(err)
	}
	if len(initial.Items) != 1 || !initial.Items[0].IsDefault {
		t.Fatalf("initial accounts = %+v", initial.Items)
	}

	body := `{"token":"second-token","login":"bob","email":"bob@example.com"}`
	resp, err = http.Post(accountsURL, "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create account status = %d", resp.StatusCode)
	}
	var added struct {
		Account domain.ConnectionAccount `json:"account"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&added); err != nil {
		t.Fatal(err)
	}
	if added.Account.IsDefault {
		t.Fatalf("second account should not be default: %+v", added.Account)
	}

	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/accounts/"+added.Account.ID, bytes.NewReader([]byte(`{"isDefault":true}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("default status = %d", resp.StatusCode)
	}
}
