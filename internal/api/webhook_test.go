package api

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/integration/feishu"
	"github.com/OpenSlash/tandemesh/internal/integration/github"
	"github.com/OpenSlash/tandemesh/internal/integration/jira"
	"github.com/OpenSlash/tandemesh/internal/integration/linear"
	"github.com/OpenSlash/tandemesh/internal/integration/notion"
	"github.com/OpenSlash/tandemesh/internal/integration/outline"
	"github.com/OpenSlash/tandemesh/internal/integration/outlook"
	"github.com/OpenSlash/tandemesh/internal/integration/slack"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func testRegistry() *integration.Registry {
	reg := integration.NewRegistry()
	_ = reg.Register(feishu.New(""))
	_ = reg.Register(github.New(""))
	_ = reg.Register(jira.New(""))
	_ = reg.Register(notion.New(""))
	_ = reg.Register(outline.New("", ""))
	_ = reg.Register(outlook.New(""))
	_ = reg.Register(linear.New(""))
	_ = reg.Register(slack.New())
	return reg
}

const issuePayload = `{
  "action": "labeled",
  "issue": {
    "number": 7,
    "title": "flaky test",
    "body": "sometimes fails",
    "html_url": "https://github.com/acme/api/issues/7",
    "updated_at": "2026-10-30T00:00:00Z",
    "labels": [{"name": "agent:fix"}],
    "user": {"login": "alice"}
  },
  "repository": {"full_name": "acme/api", "html_url": "https://github.com/acme/api"},
  "sender": {"login": "alice"}
}`

func TestGitHubWebhookUsesAutomation(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "wh.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	binding := &domain.Automation{
		ID:           "auto_github",
		Provider:     "github",
		Instance:     "acme/api",
		AgentID:      "agt_fix",
		TriggerKind:  domain.TriggerLabel,
		TriggerValue: "agent:fix",
		Enabled:      true,
	}
	if err := st.UpsertAutomation(ctx, binding); err != nil {
		t.Fatal(err)
	}

	srv := New(config.Config{GitHubSecret: ""}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	resp, err := http.Post(ts.URL+"/webhooks/github/"+binding.ID, "application/json", bytes.NewBufferString(issuePayload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d: %s", resp.StatusCode, body)
	}

	runs, err := st.ListRuns(ctx, "", "", 10)
	if err != nil || len(runs) != 1 {
		t.Fatalf("runs = %d, err = %v", len(runs), err)
	}
	if runs[0].AgentID != "agt_fix" || runs[0].AutomationID != binding.ID {
		t.Fatalf("run not bound: %+v", runs[0])
	}
}

func TestGitHubWebhookSkipsDisabledAutomation(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "wh2.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertAutomation(ctx, &domain.Automation{
		ID: "auto_off", Provider: "github", Instance: "acme/api", AgentID: "agt_fix",
		TriggerKind: domain.TriggerAny, Enabled: false,
	}); err != nil {
		t.Fatal(err)
	}

	srv := New(config.Config{GitHubSecret: ""}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	resp, err := http.Post(ts.URL+"/webhooks/github/auto_off", "application/json", bytes.NewBufferString(issuePayload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	runs, _ := st.ListRuns(ctx, "", "", 10)
	if len(runs) != 0 {
		t.Fatalf("disabled binding created %d runs", len(runs))
	}
}
