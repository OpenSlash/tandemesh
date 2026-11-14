package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestTaskRelations(t *testing.T) {
	ts := newTestServer(t)
	parent := postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "501", "title": "parent",
	})
	child := postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "502", "title": "child",
	})

	// Parent link and children list.
	if status := patchStatus(t, ts.URL+"/api/v1/tasks/"+child.ID+"/parent", map[string]string{"parentId": parent.ID}); status != http.StatusOK {
		t.Fatalf("parent status = %d", status)
	}
	children := getJSON[struct {
		Items []domain.Task `json:"items"`
	}](t, ts.URL+"/api/v1/tasks/"+parent.ID+"/children")
	if len(children.Items) != 1 || children.Items[0].ID != child.ID {
		t.Fatalf("children = %+v", children.Items)
	}
	// Reverse link would cycle.
	if status := patchStatus(t, ts.URL+"/api/v1/tasks/"+parent.ID+"/parent", map[string]string{"parentId": child.ID}); status != http.StatusBadRequest {
		t.Fatalf("cycle status = %d", status)
	}

	// Dependencies and cycle rejection.
	if status := postStatus(t, ts.URL+"/api/v1/tasks/"+child.ID+"/dependencies", map[string]string{"dependsOn": parent.ID}); status != http.StatusCreated {
		t.Fatalf("dependency status = %d", status)
	}
	deps := getJSON[struct {
		Items []domain.Task `json:"items"`
	}](t, ts.URL+"/api/v1/tasks/"+child.ID+"/dependencies")
	if len(deps.Items) != 1 || deps.Items[0].ID != parent.ID {
		t.Fatalf("deps = %+v", deps.Items)
	}
	if status := postStatus(t, ts.URL+"/api/v1/tasks/"+parent.ID+"/dependencies", map[string]string{"dependsOn": child.ID}); status != http.StatusBadRequest {
		t.Fatalf("dependency cycle status = %d", status)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/tasks/"+child.ID+"/dependencies/"+parent.ID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}
}

func patchStatus(t *testing.T, url string, body any) int {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPatch, url, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func postStatus(t *testing.T, url string, body any) int {
	t.Helper()
	raw, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}
