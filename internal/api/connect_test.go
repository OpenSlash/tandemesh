package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/integration/github"
	"github.com/OpenSlash/tandemesh/internal/integration/jira"
	"github.com/OpenSlash/tandemesh/internal/integration/linear"
	"github.com/OpenSlash/tandemesh/internal/integration/notion"
	"github.com/OpenSlash/tandemesh/internal/integration/slack"
)

func TestConnectFieldsManifestPriority(t *testing.T) {
	manifest := integration.Manifest{
		ConfigFields: []integration.ConfigField{
			{Key: "token", Required: true},
			{Key: "instance", Required: true},
			{Key: "email"},
		},
	}
	req := &connectRequest{
		Token: "legacy-token",
		Fields: map[string]string{
			"instance": "ENG",
			"email":    "dev@example.com",
		},
	}
	fields := connectFields(manifest, req)
	if fields["token"] != "legacy-token" {
		t.Fatalf("token = %q", fields["token"])
	}
	if fields["instance"] != "ENG" {
		t.Fatalf("instance = %q", fields["instance"])
	}
	if fields["email"] != "dev@example.com" {
		t.Fatalf("email = %q", fields["email"])
	}
}

func TestConfigFieldsExposeSetupInputs(t *testing.T) {
	checks := map[string][]string{
		"github": {"token", "instance", "baseUrl", "webhookSecret"},
		"linear": {"token", "instance"},
		"jira":   {"email", "token", "instance"},
		"slack":  {"token"},
		"notion": {"token"},
	}
	for id, want := range checks {
		adapter, ok := providerTestRegistry().Get(id)
		if !ok {
			t.Fatalf("provider %s missing", id)
		}
		have := map[string]bool{}
		for _, field := range adapter.Manifest.ConfigFields {
			have[field.Key] = true
		}
		for _, key := range want {
			if !have[key] {
				t.Fatalf("provider %s missing field %s: %+v", id, key, adapter.Manifest.ConfigFields)
			}
		}
	}
}

func providerTestRegistry() *integration.Registry {
	registry := integration.NewRegistry()
	for _, item := range []*integration.Integration{
		github.New(""), linear.New(""), jira.New(""), slack.New(), notion.New(""),
	} {
		_ = registry.Register(item)
	}
	return registry
}

func TestConnectChatProviderWithoutToken(t *testing.T) {
	ts := newTestServer(t)
	body := `{"provider":"feishu","workspaceId":"ws_default","name":"Feishu test","fields":{"webhookUrl":"https://example.com/hook"}}`
	resp, err := http.Post(ts.URL+"/api/v1/integrations/connect", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, data)
	}
	var out struct {
		Connection domain.Connection `json:"connection"`
		Automation any               `json:"automation"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Connection.Provider != "feishu" || out.Connection.Status != domain.ConnectionConnected {
		t.Fatalf("connection=%+v", out.Connection)
	}
	if out.Automation != nil {
		t.Fatalf("chat connection should not create an automation: %+v", out.Automation)
	}
}
