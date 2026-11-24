package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestExternalIdentitiesAPI(t *testing.T) {
	ts := newTestServer(t)
	body := `{"provider":"github","externalId":"12345","login":"alice","memberId":"alice"}`
	resp, err := http.Post(ts.URL+"/api/v1/workspaces/ws_default/identities", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", resp.StatusCode)
	}
	var created struct {
		Identity domain.ExternalIdentity `json:"identity"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Identity.MemberID != "alice" {
		t.Fatalf("identity = %+v", created.Identity)
	}

	resp, err = http.Get(ts.URL + "/api/v1/workspaces/ws_default/identities?provider=github")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var listed struct {
		Items []domain.ExternalIdentity `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Items) != 1 || listed.Items[0].Login != "alice" {
		t.Fatalf("items = %+v", listed.Items)
	}

	patch, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/identities/"+created.Identity.ID,
		bytes.NewReader([]byte(`{"memberId":"bob"}`)))
	patch.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(patch)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d", resp.StatusCode)
	}
	var patched struct {
		Identity domain.ExternalIdentity `json:"identity"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&patched); err != nil {
		t.Fatal(err)
	}
	if patched.Identity.MemberID != "bob" {
		t.Fatalf("patched = %+v", patched.Identity)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/identities/"+created.Identity.ID, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}
}
