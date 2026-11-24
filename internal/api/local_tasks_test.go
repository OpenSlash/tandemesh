package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestLocalTaskCreateAndAssign(t *testing.T) {
	ts, _ := newTestServerWithStore(t)
	body := []byte(`{"workspaceId":"ws_default","title":"Local task","body":"do it","labels":["agent"]}`)
	resp, err := http.Post(ts.URL+"/api/v1/tasks", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d", resp.StatusCode)
	}
	var task domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	if task.Provider != "local" || task.Origin != "local" {
		t.Fatalf("task = %+v", task)
	}
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/tasks/"+task.ID+"/member-assignee", bytes.NewReader([]byte(`{"memberId":"alice","login":"alice"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("assign = %d", resp.StatusCode)
	}
	var updated domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.AssigneeMemberID != "alice" || updated.Assignee != "alice" {
		t.Fatalf("updated = %+v", updated)
	}
}
