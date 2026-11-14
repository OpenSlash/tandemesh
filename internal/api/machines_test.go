package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/agent"
	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestMachineRuntimes(t *testing.T) {
	ts := newTestServer(t)
	machine := postJSON[domain.Machine](t, ts.URL+"/api/v1/machines/register", map[string]any{
		"token": "", "name": "build-01", "agents": []string{"claude"},
		"runtimes": map[string]string{"claude": "1.2.3"}, "concurrency": 2,
	})
	if machine.Runtimes["claude"] != "1.2.3" {
		t.Fatalf("machine = %+v", machine)
	}

	list := getJSON[struct {
		Items []domain.Machine `json:"items"`
	}](t, ts.URL+"/api/v1/machines")
	if len(list.Items) != 1 || list.Items[0].Runtimes["claude"] != "1.2.3" {
		t.Fatalf("machines = %+v", list.Items)
	}

	detail := getJSON[domain.Machine](t, ts.URL+"/api/v1/machines/"+machine.ID)
	if detail.ID != machine.ID || detail.Runtimes["claude"] != "1.2.3" {
		t.Fatalf("machine detail = %+v", detail)
	}

	raw, _ := json.Marshal(map[string]any{"runtimes": map[string]string{"claude": "1.3.0", "codex": "0.9"}})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/machines/"+machine.ID+"/heartbeat", bytesReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("heartbeat status = %d", resp.StatusCode)
	}

	updated := getJSON[struct {
		Items []domain.Machine `json:"items"`
	}](t, ts.URL+"/api/v1/machines")
	if updated.Items[0].Runtimes["codex"] != "0.9" || updated.Items[0].Runtimes["claude"] != "1.3.0" {
		t.Fatalf("updated = %+v", updated.Items[0].Runtimes)
	}
}

func TestMachineJoinToken(t *testing.T) {
	ts := newTestServer(t)
	created := postJSON[struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expiresAt"`
	}](t, ts.URL+"/api/v1/machines/join-token", map[string]any{})
	if created.Token == "" || created.ExpiresAt == "" {
		t.Fatalf("join token = %+v", created)
	}
	postJSON[domain.Machine](t, ts.URL+"/api/v1/machines/register", map[string]any{
		"token": created.Token, "name": "one-time-host", "concurrency": 1,
	})

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/machines/register",
		bytesReader([]byte(`{"token":"`+created.Token+`","name":"replay-host"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("replayed token = %d, want 401", resp.StatusCode)
	}

	info := getJSON[struct {
		BaseURL string `json:"baseUrl"`
	}](t, ts.URL+"/api/v1/system/info")
	if info.BaseURL == "" {
		t.Fatal("missing base url")
	}
}

func TestRuntimeCatalog(t *testing.T) {
	ts := newTestServer(t)
	catalog := getJSON[struct {
		Items []agent.Runtime `json:"items"`
	}](t, ts.URL+"/api/v1/runtimes")
	if len(catalog.Items) < 3 {
		t.Fatalf("catalog = %+v", catalog.Items)
	}
	for _, item := range catalog.Items {
		if item.Install == "" || item.Docs == "" {
			t.Fatalf("incomplete runtime = %+v", item)
		}
	}
}

func TestMachineIdentityIsStable(t *testing.T) {
	ts := newTestServer(t)
	body := map[string]any{"token": "", "name": "stable-host", "agents": []string{"claude"}, "concurrency": 1}
	first := postJSON[domain.Machine](t, ts.URL+"/api/v1/machines/register", body)
	second := postJSON[domain.Machine](t, ts.URL+"/api/v1/machines/register", body)
	if first.ID != second.ID {
		t.Fatalf("ids differ: %s vs %s", first.ID, second.ID)
	}
	list := getJSON[struct {
		Items []domain.Machine `json:"items"`
	}](t, ts.URL+"/api/v1/machines")
	if len(list.Items) != 1 {
		t.Fatalf("machines = %d", len(list.Items))
	}
}

func TestMachineTakeover(t *testing.T) {
	ts := newTestServer(t)
	a := postJSON[domain.Machine](t, ts.URL+"/api/v1/machines/register", map[string]any{"token": "", "name": "old-host", "agents": []string{"claude"}, "concurrency": 1})
	b := postJSON[domain.Machine](t, ts.URL+"/api/v1/machines/register", map[string]any{"token": "", "name": "new-host", "agents": []string{"claude"}, "concurrency": 1})
	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{"workspaceId": "ws_default", "name": "bound", "backend": "claude", "machineId": a.ID})
	res := postJSON[struct {
		Agents int `json:"agents"`
		Runs   int `json:"runs"`
	}](t, ts.URL+"/api/v1/machines/"+a.ID+"/takeover", map[string]string{"targetId": b.ID})
	if res.Agents != 1 {
		t.Fatalf("takeover agents = %d", res.Agents)
	}
	updated := getJSON[domain.Agent](t, ts.URL+"/api/v1/agents/"+agent.ID)
	if updated.MachineID != b.ID {
		t.Fatalf("machineId = %s", updated.MachineID)
	}

	postJSON[map[string]string](t, ts.URL+"/api/v1/machines/"+b.ID+"/status", map[string]string{"status": "offline"})
	req, _ := http.NewRequest(http.MethodPost,
		ts.URL+"/api/v1/machines/"+a.ID+"/takeover",
		bytesReader([]byte(`{"targetId":"`+b.ID+`"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("offline target takeover = %d, want 400", resp.StatusCode)
	}
}
