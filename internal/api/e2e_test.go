package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestE2EDocumentLifecycle(t *testing.T) {
	ts := newTestServer(t)

	created := postJSON[domain.Document](t, ts.URL+"/api/v1/documents", map[string]any{
		"workspaceId": "ws_default",
		"title":       "Runbook",
		"body":        "# Deploy\n\nSteps...",
		"tags":        []string{"ops"},
	})
	if created.ID == "" || created.Title != "Runbook" {
		t.Fatalf("created = %+v", created)
	}

	resp, err := http.Get(ts.URL + "/api/v1/documents?workspaceId=ws_default")
	if err != nil {
		t.Fatal(err)
	}
	var list struct {
		Documents []domain.Document `json:"documents"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list.Documents) != 1 {
		t.Fatalf("list = %+v", list)
	}

	payload, _ := json.Marshal(map[string]any{"title": "Runbook v2", "body": "updated"})
	updateReq, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/documents/"+created.ID, bytes.NewReader(payload))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, err := http.DefaultClient.Do(updateReq)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, updateResp.Body)
	updateResp.Body.Close()
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("update status = %d", updateResp.StatusCode)
	}

	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/documents/"+created.ID, nil)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatal(err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d", delResp.StatusCode)
	}
}
