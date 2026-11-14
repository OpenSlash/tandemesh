package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestGitHubAppWebhookRoutesToAutomation(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "gh-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	conn := &domain.Connection{ID: "conn_gh", WorkspaceID: "ws_default", Provider: "github", Instance: "acme/api", Status: "connected", Enabled: true}
	if err := st.UpsertConnection(t.Context(), conn); err != nil {
		t.Fatal(err)
	}
	binding := &domain.ResourceBinding{ID: "bind_gh", WorkspaceID: "ws_default", ConnectionID: conn.ID, Provider: "github", ExternalID: "acme/api", Direction: "read_write", Enabled: true}
	if err := st.UpsertBinding(t.Context(), binding); err != nil {
		t.Fatal(err)
	}
	automation := &domain.Automation{ID: "auto_gh", WorkspaceID: "ws_default", Name: "gh", Provider: "github", Instance: "acme/api", ConnectionID: conn.ID, TriggerKind: "any", AgentID: "claude", Enabled: true}
	if err := st.UpsertAutomation(t.Context(), automation); err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{User: "alice", GitHubAppWebhookSecret: "secret", BaseURL: "http://orca.test"}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	body := []byte(`{"action":"opened","issue":{"number":7,"title":"Fix","body":"body","html_url":"https://github.test/acme/api/issues/7","updated_at":"2026-01-01T00:00:00Z"},"repository":{"full_name":"acme/api","html_url":"https://github.test/acme/api"},"sender":{"login":"alice"}}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(body)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/webhooks/github/app", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("X-GitHub-Delivery", "delivery-1")
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	task, err := st.GetTask(t.Context(), store.TaskID(domain.WorkRef{Provider: "github", Instance: "acme/api", Type: "issue", ID: "7"}))
	if err != nil || task == nil {
		t.Fatalf("task = %+v, err = %v", task, err)
	}
	prBody := []byte(`{"action":"opened","repository":{"full_name":"acme/api"},"pull_request":{"number":9,"html_url":"https://github.test/acme/api/pull/9","title":"#7 fix","state":"open","head":{"ref":"fix/7"},"user":{"login":"alice"},"author_association":"OWNER"}}`)
	mac = hmac.New(sha256.New, []byte("secret"))
	mac.Write(prBody)
	prReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/webhooks/github/app", bytes.NewReader(prBody))
	prReq.Header.Set("X-GitHub-Event", "pull_request")
	prReq.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	prResp, err := http.DefaultClient.Do(prReq)
	if err != nil {
		t.Fatal(err)
	}
	defer prResp.Body.Close()
	if prResp.StatusCode < 200 || prResp.StatusCode >= 300 {
		t.Fatalf("pr status = %d", prResp.StatusCode)
	}
	links, err := st.ListTaskPullRequests(t.Context(), task.ID)
	if err != nil || len(links) != 1 || links[0].Number != 9 {
		t.Fatalf("links = %+v, err = %v", links, err)
	}
}
