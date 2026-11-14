package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestWorkspaceGroups(t *testing.T) {
	ts := newTestServer(t)

	spaces := getJSON[struct {
		Items []domain.Workspace `json:"items"`
	}](t, ts.URL+"/api/v1/workspaces")
	if len(spaces.Items) != 1 || spaces.Items[0].ID != "ws_default" || spaces.Items[0].Slug != "default" {
		t.Fatalf("workspaces = %+v", spaces.Items)
	}

	ws := postJSON[domain.Workspace](t, ts.URL+"/api/v1/workspaces", map[string]string{
		"name": "Payments", "slug": "payments",
	})
	if ws.ID == "" || ws.Slug != "payments" {
		t.Fatalf("workspace = %+v", ws)
	}

	wsRenameReq, _ := http.NewRequest(http.MethodPatch,
		ts.URL+"/api/v1/workspaces/"+ws.ID,
		bytes.NewBufferString(`{"name":"Payments v2"}`))
	wsRenameReq.Header.Set("Content-Type", "application/json")
	wsRenameResp, err := http.DefaultClient.Do(wsRenameReq)
	if err != nil {
		t.Fatal(err)
	}
	wsRenamedRaw, _ := io.ReadAll(wsRenameResp.Body)
	wsRenameResp.Body.Close()
	if wsRenameResp.StatusCode != http.StatusOK {
		t.Fatalf("rename workspace = %d %s", wsRenameResp.StatusCode, wsRenamedRaw)
	}
	var wsRenamed domain.Workspace
	if err := json.Unmarshal(wsRenamedRaw, &wsRenamed); err != nil {
		t.Fatal(err)
	}
	if wsRenamed.Name != "Payments v2" || wsRenamed.Slug != "payments" {
		t.Fatalf("renamed workspace = %+v", wsRenamed)
	}

	group := postJSON[domain.Group](t, ts.URL+"/api/v1/groups", map[string]string{
		"workspaceId": ws.ID, "name": "Checkout crew", "description": "payments work",
	})
	if group.ID == "" || group.WorkspaceID != ws.ID {
		t.Fatalf("group = %+v", group)
	}
	member := postJSON[domain.GroupMember](t, ts.URL+"/api/v1/groups/"+group.ID+"/members", map[string]string{
		"agentId": "claude", "role": "lead",
	})
	if member.Role != "lead" {
		t.Fatalf("member = %+v", member)
	}
	detail := getJSON[struct {
		Group   domain.Group         `json:"group"`
		Members []domain.GroupMember `json:"members"`
	}](t, ts.URL+"/api/v1/groups/"+group.ID)
	if len(detail.Members) != 1 || detail.Members[0].AgentID != "claude" {
		t.Fatalf("members = %+v", detail.Members)
	}

	// Import into the new workspace and scope the task list.
	postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/pay", "type": "issue", "id": "5",
		"title": "charge twice", "workspaceId": ws.ID,
	})
	scoped := getJSON[struct {
		Items []domain.Task `json:"items"`
	}](t, ts.URL+"/api/v1/tasks?workspace="+ws.ID)
	if len(scoped.Items) != 1 || scoped.Items[0].WorkspaceID != ws.ID {
		t.Fatalf("scoped = %+v", scoped.Items)
	}
	def := getJSON[struct {
		Items []domain.Task `json:"items"`
	}](t, ts.URL+"/api/v1/tasks?workspace=ws_default")
	if len(def.Items) != 0 {
		t.Fatalf("default workspace should be empty: %+v", def.Items)
	}
}

// TestTenancyRoutesRemoved pins the open-source surface: organization and
// member routes no longer exist.
func TestTenancyRoutesRemoved(t *testing.T) {
	ts := newTestServer(t)
	for _, path := range []string{
		"/api/v1/organizations",
		"/api/v1/organizations/org_openslash/members",
		"/api/v1/workspaces/ws_default/members",
	} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("GET %s = %d, want 404", path, resp.StatusCode)
		}
	}
}

func getJSON[T any](t *testing.T, url string) T {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d %s", url, resp.StatusCode, raw)
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s: %v (%s)", url, err, raw)
	}
	return out
}

func doAs[T any](t *testing.T, method, url, user string, body any) (int, T) {
	t.Helper()
	var payload io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		payload = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if user != "" {
		req.Header.Set("X-Orca-User", user)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out T
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return resp.StatusCode, out
}

func postJSON[T any](t *testing.T, url string, body any) T {
	t.Helper()
	raw, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		t.Fatalf("POST %s = %d %s", url, resp.StatusCode, out)
	}
	var result T
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("decode %s: %v (%s)", url, err, out)
	}
	return result
}
