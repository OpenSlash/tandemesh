package api

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestSkillsAndPromptInjection(t *testing.T) {
	ts := newTestServer(t)
	skill := postJSON[domain.Skill](t, ts.URL+"/api/v1/skills", map[string]any{
		"workspaceId": "ws_default", "name": "review-style",
		"description": "House review rules", "body": "Always check error handling first.",
	})
	if skill.ID == "" {
		t.Fatal("skill not created")
	}
	list := getJSON[struct {
		Items []domain.Skill `json:"items"`
	}](t, ts.URL+"/api/v1/skills?workspace=ws_default")
	if len(list.Items) != 1 || list.Items[0].Name != "review-style" {
		t.Fatalf("skills = %+v", list.Items)
	}

	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "style agent", "backend": "claude",
		"instructions": "Only report after tests pass.",
		"skills":       []string{"review-style"},
	})
	task := postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "601", "title": "skill task",
	})
	team := postJSON[struct {
		Items []domain.Run `json:"items"`
	}](t, ts.URL+"/api/v1/tasks/"+task.ID+"/team", map[string]any{
		"repo": "acme/api",
		"roles": []map[string]string{
			{"role": "lead", "agentId": agent.ID},
		},
	})
	if len(team.Items) != 1 {
		t.Fatalf("team = %+v", team.Items)
	}
	if !strings.Contains(team.Items[0].Prompt, "Always check error handling first.") {
		t.Fatalf("prompt missing skill body: %s", team.Items[0].Prompt)
	}
	if !strings.Contains(team.Items[0].Prompt, "Only report after tests pass.") {
		t.Fatalf("prompt missing agent instructions: %s", team.Items[0].Prompt)
	}
}

func TestSkillMarketInstall(t *testing.T) {
	ts := newTestServer(t)
	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "market agent", "backend": "claude",
	})
	market := getJSON[struct {
		Items []struct {
			ID        string `json:"id"`
			Installed bool   `json:"installed"`
		} `json:"items"`
	}](t, ts.URL+"/api/v1/skills/market?workspace=ws_default")
	if len(market.Items) < 3 || market.Items[0].Installed {
		t.Fatalf("market = %+v", market.Items)
	}
	result := postJSON[struct {
		Skill  domain.Skill `json:"skill"`
		Agents int          `json:"agents"`
	}](t, ts.URL+"/api/v1/skills/market/code-review/install", map[string]any{
		"workspaceId": "ws_default", "agentIds": []string{agent.ID},
	})
	if result.Skill.Name != "code-review" || result.Skill.Source != "market" || result.Agents != 1 {
		t.Fatalf("install = %+v", result)
	}
	updated := getJSON[domain.Agent](t, ts.URL+"/api/v1/agents/"+agent.ID)
	found := false
	for _, name := range updated.Skills {
		if name == "code-review" {
			found = true
		}
	}
	if !found {
		t.Fatalf("agent skills = %+v", updated.Skills)
	}
}

func TestSkillRegistryInstallAndToggle(t *testing.T) {
	ts := newTestServer(t)
	raw := "---\nname: review-style\ndescription: House review rules\nversion: 1.2.0\ntags: review, style\n---\n\nAlways check error handling first."
	installed := postJSON[domain.Skill](t, ts.URL+"/api/v1/skills/install", map[string]any{
		"workspaceId": "ws_default", "body": raw,
	})
	if installed.Name != "review-style" || installed.Version != "1.2.0" || len(installed.Tags) != 2 || installed.Checksum == "" {
		t.Fatalf("installed = %+v", installed)
	}
	if !installed.Enabled || installed.Source != "manual" {
		t.Fatalf("enabled/source = %+v", installed)
	}
	resp, err := http.Get(ts.URL + "/api/v1/skills/" + installed.ID + "/raw")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "name: review-style") || !strings.Contains(string(body), "Always check error handling first.") {
		t.Fatalf("raw = %s", body)
	}
	exported := getJSON[struct {
		Items []struct {
			Name     string `json:"name"`
			Markdown string `json:"markdown"`
		} `json:"items"`
	}](t, ts.URL+"/api/v1/skills/export?workspace=ws_default")
	if len(exported.Items) != 1 || !strings.Contains(exported.Items[0].Markdown, "version: 1.2.0") {
		t.Fatalf("export = %+v", exported)
	}
	disabled := postJSON[domain.Skill](t, ts.URL+"/api/v1/skills/"+installed.ID+"/enabled", map[string]any{"enabled": false})
	if disabled.Enabled {
		t.Fatalf("disabled = %+v", disabled)
	}

	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "disabled style agent", "backend": "claude",
		"skills": []string{"review-style"},
	})
	task := postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "602", "title": "disabled skill task",
	})
	team := postJSON[struct {
		Items []domain.Run `json:"items"`
	}](t, ts.URL+"/api/v1/tasks/"+task.ID+"/team", map[string]any{
		"repo": "acme/api", "roles": []map[string]string{{"role": "lead", "agentId": agent.ID}},
	})
	if strings.Contains(team.Items[0].Prompt, "Always check error handling first.") {
		t.Fatalf("disabled skill leaked into prompt: %s", team.Items[0].Prompt)
	}
}
