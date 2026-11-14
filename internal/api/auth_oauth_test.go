package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestGitLabOAuthStart(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "gitlab-oauth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{User: "alice", GitLabClientID: "client", GitLabClientSecret: "secret", GitLabAuthorizeURL: "https://gitlab.test/oauth/authorize", BaseURL: "http://orca.test", AppURL: "http://app.test"}
	srv := New(cfg, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(ts.URL + "/api/v1/integrations/gitlab/oauth/start?workspace=ws_default&instance=acme/api")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); !strings.Contains(loc, "gitlab.test/oauth/authorize") {
		t.Fatalf("location = %s", loc)
	}
}

func TestCompleteMemberOAuth(t *testing.T) {
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":42,"login":"alice","email":"alice@example.com"}`))
	}))
	defer apiSrv.Close()
	t.Setenv("ORCA_GITHUB_API_URL", apiSrv.URL)
	st, err := store.Open(filepath.Join(t.TempDir(), "member-oauth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	s := &Server{store: st, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := s.completeMemberOAuth(t.Context(), "github", "ws_default", "alice", "", "token"); err != nil {
		t.Fatal(err)
	}
	items, err := st.ListMemberIdentities(t.Context(), "ws_default", "alice")
	if err != nil || len(items) != 1 || items[0].ExternalID != "42" {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
}

func TestCompleteGitLabMemberOAuth(t *testing.T) {
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":7,"username":"alice","email":"alice@example.com"}`))
	}))
	defer apiSrv.Close()
	st, err := store.Open(filepath.Join(t.TempDir(), "member-gitlab.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	s := &Server{store: st, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := s.completeMemberOAuth(t.Context(), "gitlab", "ws_default", "alice", apiSrv.URL, "token"); err != nil {
		t.Fatal(err)
	}
	items, err := st.ListMemberIdentities(t.Context(), "ws_default", "alice")
	if err != nil || len(items) != 1 || items[0].Provider != "gitlab" || items[0].ExternalID != "7" {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
}

func TestCompleteJiraMemberOAuth(t *testing.T) {
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"account_id":"acc-1","name":"alice","email":"alice@example.com"}`))
	}))
	defer apiSrv.Close()
	t.Setenv("ORCA_JIRA_API_URL", apiSrv.URL)
	st, err := store.Open(filepath.Join(t.TempDir(), "member-jira.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	s := &Server{store: st, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := s.completeMemberOAuth(t.Context(), "jira", "ws_default", "alice", "", "token"); err != nil {
		t.Fatal(err)
	}
	items, _ := st.ListMemberIdentities(t.Context(), "ws_default", "alice")
	if len(items) != 1 || items[0].Provider != "jira" || items[0].ExternalID != "acc-1" {
		t.Fatalf("items = %+v", items)
	}
}

func TestCompleteLinearMemberOAuth(t *testing.T) {
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"viewer":{"id":"usr-1","name":"alice","email":"alice@example.com"}}}`))
	}))
	defer apiSrv.Close()
	st, err := store.Open(filepath.Join(t.TempDir(), "member-linear.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	s := &Server{store: st, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := s.completeMemberOAuth(t.Context(), "linear", "ws_default", "alice", apiSrv.URL, "token"); err != nil {
		t.Fatal(err)
	}
	items, _ := st.ListMemberIdentities(t.Context(), "ws_default", "alice")
	if len(items) != 1 || items[0].Provider != "linear" || items[0].ExternalID != "usr-1" {
		t.Fatalf("items = %+v", items)
	}
}

func TestSlackOAuthStart(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "slack-oauth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{User: "alice", SlackClientID: "client", SlackClientSecret: "secret", SlackAuthorizeURL: "https://slack.test/oauth/v2/authorize", BaseURL: "http://orca.test", AppURL: "http://app.test"}
	srv := New(cfg, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(ts.URL + "/api/v1/integrations/slack/oauth/start?workspace=ws_default&channel=C1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); !strings.Contains(loc, "slack.test/oauth/v2/authorize") {
		t.Fatalf("location = %s", loc)
	}
}
