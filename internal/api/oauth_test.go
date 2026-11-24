package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestExchangeOAuthCode(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		got = r.Form
		_, _ = w.Write([]byte(`{"access_token":"access-1","refresh_token":"refresh-1","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	token, err := exchangeOAuthCode(t.Context(), oauthClientConfig{
		ClientID: "client", ClientSecret: "secret", Scopes: "read write", TokenURL: srv.URL,
	}, "code-1", "https://orca.test/callback")
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "access-1" || token.RefreshToken != "refresh-1" {
		t.Fatalf("token = %+v", token)
	}
	if token.ExpiresAt.Before(time.Now().Add(50 * time.Minute)) {
		t.Fatalf("expiresAt = %v", token.ExpiresAt)
	}
	if got.Get("grant_type") != "authorization_code" || got.Get("code") != "code-1" {
		t.Fatalf("form = %v", got)
	}
	if got.Get("client_id") != "client" || got.Get("client_secret") != "secret" {
		t.Fatalf("client form = %v", got)
	}
	if got.Get("redirect_uri") != "https://orca.test/callback" || got.Get("scope") != "read write" {
		t.Fatalf("redirect/scope form = %v", got)
	}
}

func TestOAuthStateRoundTripCarriesProvider(t *testing.T) {
	s := &Server{cfg: config.Config{ApprovalKey: "state-key"}}
	state := githubOAuthState{
		Provider: "linear", Workspace: "ws_default", Instance: "ENG",
		Nonce: "n", Expires: time.Now().Add(time.Minute).Unix(),
	}
	signed, err := s.signOAuthState(state)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s.verifyOAuthState(signed)
	if !ok {
		t.Fatal("state did not verify")
	}
	if got.Provider != "linear" || got.Instance != "ENG" {
		t.Fatalf("state = %+v", got)
	}
}

type oauthProbe struct{}

func (oauthProbe) Probe(context.Context, integration.Credential) error { return nil }

func TestOAuthCallbackCreatesConnection(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		_, _ = w.Write([]byte(`{"access_token":"access-oauth","refresh_token":"refresh-oauth","expires_in":3600}`))
	}))
	defer tokenSrv.Close()
	st, err := store.Open(filepath.Join(t.TempDir(), "oauth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	reg := integration.NewRegistry()
	if err := reg.Register(&integration.Integration{
		Manifest: integration.Manifest{ID: "linear", Name: "Linear", Kind: integration.KindTask, Auth: integration.AuthOAuth2},
		Probe:    oauthProbe{},
	}); err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{
		LinearClientID: "client", LinearClientSecret: "secret", LinearTokenURL: tokenSrv.URL,
		BaseURL: "http://orca.test", AppURL: "http://orca.test", ApprovalKey: "state-key", User: "alice",
	}, st, nil, hub.New(), reg, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	state := githubOAuthState{Provider: "linear", Workspace: "ws_default", Instance: "ENG", SyncDirection: "read", Nonce: "n", Expires: time.Now().Add(time.Minute).Unix()}
	signed, err := srv.signOAuthState(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateOAuthNonce(t.Context(), "n", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/integrations/linear/oauth/callback?code=code&state="+url.QueryEscape(signed), nil)
	req.AddCookie(&http.Cookie{Name: "orca_oauth_state", Value: signed})
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, data)
	}
	replay, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/integrations/linear/oauth/callback?code=code&state="+url.QueryEscape(signed), nil)
	replay.AddCookie(&http.Cookie{Name: "orca_oauth_state", Value: signed})
	replayResp, err := client.Do(replay)
	if err != nil {
		t.Fatal(err)
	}
	defer replayResp.Body.Close()
	if replayResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("replay status = %d", replayResp.StatusCode)
	}
	conns, err := st.ListConnections(t.Context(), "ws_default", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(conns) != 1 || conns[0].Provider != "linear" || conns[0].Instance != "ENG" || conns[0].SyncDirection != "read" {
		t.Fatalf("connections = %+v", conns)
	}
}

func TestOAuthStatusListsConfiguredProviders(t *testing.T) {
	s := &Server{cfg: config.Config{LinearClientID: "id", LinearClientSecret: "secret"}}
	rec := httptest.NewRecorder()
	s.oauthStatus(rec, httptest.NewRequest(http.MethodGet, "/api/v1/integrations/oauth/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "linear") || strings.Contains(body, "jira") {
		t.Fatalf("body = %s", body)
	}
}

func TestExchangeOAuthCodeBasic(t *testing.T) {
	var got map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "client" || pass != "secret" {
			t.Errorf("basic auth = %q %q %v", user, pass, ok)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"access_token":"notion-access","workspace_name":"Acme"}`))
	}))
	defer srv.Close()
	token, err := exchangeOAuthCodeBasic(t.Context(), oauthClientConfig{
		ClientID: "client", ClientSecret: "secret", TokenURL: srv.URL,
	}, "code-notion", "https://orca.test/callback")
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "notion-access" {
		t.Fatalf("token = %+v", token)
	}
	if got["grant_type"] != "authorization_code" || got["code"] != "code-notion" {
		t.Fatalf("body = %v", got)
	}
}

func TestOAuthNonceCleanup(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "nonce.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateOAuthNonce(t.Context(), "expired", time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	n, err := st.CleanupOAuthNonces(t.Context())
	if err != nil || n != 1 {
		t.Fatalf("n = %d, err = %v", n, err)
	}
	used, err := st.ConsumeOAuthNonce(t.Context(), "expired")
	if err != nil || used {
		t.Fatalf("used = %v, err = %v", used, err)
	}
}

func TestExchangeSlackOAuthTeam(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"access_token":"xoxb","team":{"id":"T1","name":"Acme"},"bot_user_id":"B1"}`))
	}))
	defer srv.Close()
	token, err := exchangeOAuthCode(t.Context(), oauthClientConfig{ClientID: "id", ClientSecret: "secret", TokenURL: srv.URL}, "code", "https://orca.test/cb")
	if err != nil {
		t.Fatal(err)
	}
	if token.TeamID != "T1" || token.TeamName != "Acme" || token.BotUserID != "B1" {
		t.Fatalf("token = %+v", token)
	}
}

func TestOAuthResultPage(t *testing.T) {
	ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/oauth/result?provider=github&status=ok")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Authorization complete") || !strings.Contains(string(body), "Back to providers") {
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
}
