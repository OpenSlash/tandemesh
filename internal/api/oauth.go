package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

type oauthClientConfig struct {
	ClientID     string
	ClientSecret string
	AuthorizeURL string
	TokenURL     string
	Scopes       string
	BasicToken   bool
}

func (s *Server) oauthProviderConfig(provider string) (oauthClientConfig, bool) {
	var cfg oauthClientConfig
	switch provider {
	case "linear":
		cfg = oauthClientConfig{
			ClientID: s.cfg.LinearClientID, ClientSecret: s.cfg.LinearClientSecret,
			AuthorizeURL: s.cfg.LinearAuthorizeURL, Scopes: s.cfg.LinearScopes,
			TokenURL: s.cfg.LinearTokenURL,
		}
	case "slack":
		cfg = oauthClientConfig{
			ClientID: s.cfg.SlackClientID, ClientSecret: s.cfg.SlackClientSecret,
			AuthorizeURL: s.cfg.SlackAuthorizeURL, Scopes: s.cfg.SlackScopes,
			TokenURL: s.cfg.SlackTokenURL,
		}
	case "gitlab":
		cfg = oauthClientConfig{
			ClientID: s.cfg.GitLabClientID, ClientSecret: s.cfg.GitLabClientSecret,
			AuthorizeURL: s.cfg.GitLabAuthorizeURL, Scopes: s.cfg.GitLabScopes,
			TokenURL: s.cfg.GitLabTokenURL,
		}
	case "jira":
		cfg = oauthClientConfig{
			ClientID: s.cfg.JiraClientID, ClientSecret: s.cfg.JiraClientSecret,
			AuthorizeURL: s.cfg.JiraAuthorizeURL, Scopes: s.cfg.JiraScopes,
			TokenURL: s.cfg.JiraTokenURL,
		}
	case "outlook":
		cfg = oauthClientConfig{
			ClientID: s.cfg.GraphClientID, ClientSecret: s.cfg.GraphClientSecret,
			AuthorizeURL: s.cfg.GraphAuthorizeURL, Scopes: s.cfg.GraphScopes,
			TokenURL: s.cfg.GraphTokenURL,
		}
	case "notion":
		cfg = oauthClientConfig{
			ClientID: s.cfg.NotionClientID, ClientSecret: s.cfg.NotionClientSecret,
			AuthorizeURL: s.cfg.NotionAuthorizeURL, TokenURL: s.cfg.NotionTokenURL,
			BasicToken: true,
		}
	default:
		return oauthClientConfig{}, false
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return oauthClientConfig{}, false
	}
	return cfg, true
}

// oauthStatus reports which providers have a complete OAuth app configuration.
func (s *Server) oauthStatus(w http.ResponseWriter, r *http.Request) {
	providers := []string{}
	for _, id := range []string{"linear", "jira", "outlook", "notion"} {
		if _, ok := s.oauthProviderConfig(id); ok {
			providers = append(providers, id)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": providers})
}

func (s *Server) oauthRedirectURI(provider string) string {
	return strings.TrimRight(s.oauthBaseURL(), "/") + "/api/v1/integrations/" + provider + "/oauth/callback"
}

func (s *Server) oauthStart(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	cfg, ok := s.oauthProviderConfig(provider)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_OAUTH", "OAuth is not configured for "+provider)
		return
	}
	baseURL := strings.TrimRight(strings.TrimSpace(r.URL.Query().Get("baseUrl")), "/")
	if provider == "gitlab" && baseURL != "" && baseURL != "https://gitlab.com" {
		cfg.AuthorizeURL = baseURL + "/oauth/authorize"
		cfg.TokenURL = baseURL + "/oauth/token"
	}
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	state := githubOAuthState{
		Provider: provider, Workspace: workspaceID,
		Instance: strings.TrimSpace(r.URL.Query().Get("instance")),
		BaseURL:  strings.TrimSpace(r.URL.Query().Get("baseUrl")),
		MemberID: strings.TrimSpace(r.URL.Query().Get("memberId")),
		Channel:  strings.TrimSpace(r.URL.Query().Get("channel")),
		Name:     r.URL.Query().Get("name"), AgentID: r.URL.Query().Get("agentId"),
		Email: r.URL.Query().Get("email"), SyncDirection: r.URL.Query().Get("syncDirection"), ConnectionID: r.URL.Query().Get("connectionId"),
		Nonce: ulid.Make().String(), Expires: time.Now().Add(10 * time.Minute).Unix(),
	}
	signed, err := s.signOAuthState(state)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_OAUTH", err.Error())
		return
	}
	if err := s.store.CreateOAuthNonce(r.Context(), state.Nonce, time.Unix(state.Expires, 0)); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_OAUTH", err.Error())
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "orca_oauth_state", Value: signed, Path: "/api/v1/integrations",
		HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 600,
	})
	params := url.Values{
		"client_id":     {cfg.ClientID},
		"redirect_uri":  {s.oauthRedirectURI(provider)},
		"response_type": {"code"},
		"state":         {signed},
	}
	if cfg.Scopes != "" {
		params.Set("scope", cfg.Scopes)
	}
	if provider == "jira" {
		params.Set("audience", "api.atlassian.com")
		params.Set("prompt", "consent")
	}
	http.Redirect(w, r, cfg.AuthorizeURL+"?"+params.Encode(), http.StatusFound)
}

func (s *Server) oauthCallback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	cfg, ok := s.oauthProviderConfig(provider)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_OAUTH", "OAuth is not configured for "+provider)
		return
	}
	cookie, err := r.Cookie("orca_oauth_state")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "E_OAUTH", "missing state cookie")
		return
	}
	state, ok := s.verifyOAuthState(cookie.Value)
	if !ok || state.Provider != provider || r.URL.Query().Get("state") != cookie.Value {
		writeErr(w, http.StatusBadRequest, "E_OAUTH", "state mismatch")
		return
	}
	if used, err := s.store.ConsumeOAuthNonce(r.Context(), state.Nonce); err != nil || !used {
		writeErr(w, http.StatusBadRequest, "E_REPLAY", "OAuth state was already used")
		return
	}
	if provider == "gitlab" && state.BaseURL != "" && state.BaseURL != "https://gitlab.com" {
		cfg.AuthorizeURL = strings.TrimRight(state.BaseURL, "/") + "/oauth/authorize"
		cfg.TokenURL = strings.TrimRight(state.BaseURL, "/") + "/oauth/token"
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		writeErr(w, http.StatusBadRequest, "E_OAUTH", "missing code")
		return
	}
	var token oauthToken
	if cfg.BasicToken {
		token, err = exchangeOAuthCodeBasic(r.Context(), cfg, code, s.oauthRedirectURI(provider))
	} else {
		token, err = exchangeOAuthCode(r.Context(), cfg, code, s.oauthRedirectURI(provider))
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_OAUTH", err.Error())
		return
	}
	if state.SyncDirection != "read" {
		state.SyncDirection = "read_write"
	}
	cred := integration.Credential{
		Provider: provider, AccessToken: token.AccessToken, RefreshToken: token.RefreshToken,
		ExpiresAt: token.ExpiresAt, Account: state.Email,
		Fields: map[string]string{"auth": "oauth"},
	}
	if state.Instance != "" {
		cred.Fields["instance"] = state.Instance
	}
	if state.Email != "" {
		cred.Fields["email"] = state.Email
	}
	if provider == "jira" {
		cred.BaseURL = state.Instance
	}
	if provider == "gitlab" {
		cred.BaseURL = state.BaseURL
		if cred.BaseURL == "" {
			cred.BaseURL = "https://gitlab.com"
		}
	}
	if ad, ok := s.integrations.Get(provider); ok && ad.Probe != nil {
		if err := ad.Probe.Probe(r.Context(), cred); err != nil {
			writeErr(w, http.StatusBadGateway, "E_PROBE", err.Error())
			return
		}
	}

	if state.MemberID != "" {
		if err := s.completeMemberOAuth(r.Context(), provider, state.Workspace, state.MemberID, cred.BaseURL, token.AccessToken); err != nil {
			writeErr(w, http.StatusBadGateway, "E_OAUTH", err.Error())
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "orca_oauth_state", Value: "", Path: "/api/v1/integrations", MaxAge: -1})
		http.Redirect(w, r, strings.TrimRight(s.cfg.AppURL, "/")+"/oauth/result?provider="+provider+"&status=ok"+"&member="+url.QueryEscape(state.MemberID), http.StatusFound)
		return
	}
	var conn *domain.Connection
	if state.ConnectionID != "" {
		conn, err = s.store.GetConnection(r.Context(), state.ConnectionID)
		if err != nil || conn == nil {
			writeErr(w, http.StatusBadRequest, "E_OAUTH", "connection not found")
			return
		}
		conn.Status = domain.ConnectionConnected
		conn.AuthMethod = "oauth2"
	} else {
		name := state.Name
		if name == "" {
			name = provider + " connection"
		}
		conn = &domain.Connection{
			ID: "conn_" + ulid.Make().String(), WorkspaceID: state.Workspace,
			Provider: provider, Name: name, Status: domain.ConnectionConnected,
			AuthMethod: "oauth2", SyncDirection: state.SyncDirection, Instance: connectionInstance(provider, state), BaseURL: baseURLForState(provider, state),
			AccountLogin: state.Email, ExecutionMode: "cloud", Enabled: true,
		}
	}
	cred.Subject = conn.ID
	if s.vault != nil {
		if err := s.vault.Put(r.Context(), cred); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_VAULT", err.Error())
			return
		}
	}
	if err := s.store.UpsertConnection(r.Context(), conn); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if ad, ok := s.integrations.Get(provider); ok && ad.Manifest.Kind == integration.KindTask {
		automation := &domain.Automation{
			ID: "auto_" + ulid.Make().String(), WorkspaceID: conn.WorkspaceID,
			Name: conn.Name, Provider: provider, Instance: state.Instance,
			AgentID: state.AgentID, TriggerKind: domain.TriggerCreated, ConnectionID: conn.ID,
			Writeback: map[string]any{"comment": state.SyncDirection != "read"}, Enabled: true, Base: "main",
		}
		if err := s.store.UpsertAutomation(r.Context(), automation); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		conn.AutomationID = automation.ID
		if err := s.store.UpsertConnection(r.Context(), conn); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		account := &domain.ConnectionAccount{
			ID: "acct_" + ulid.Make().String(), WorkspaceID: conn.WorkspaceID,
			ConnectionID: conn.ID, Provider: provider, Kind: accountKind(ad.Manifest.Auth),
			Login: state.Email, Email: state.Email, ExternalID: state.Instance,
			CredentialSubject: conn.ID, IsDefault: true,
		}
		if account.Login == "" {
			account.Login = state.Instance
		}
		if err := s.store.UpsertConnectionAccount(r.Context(), account); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		if state.Instance != "" {
			binding := &domain.ResourceBinding{
				ID: "bind_" + ulid.Make().String(), WorkspaceID: conn.WorkspaceID,
				ConnectionID: conn.ID, Provider: provider, ExternalID: state.Instance,
				Name: state.Instance, Kind: bindingKind(provider), Direction: state.SyncDirection,
				AccountID: account.ID, Enabled: true,
			}
			if err := s.store.UpsertBinding(r.Context(), binding); err != nil {
				writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
				return
			}
		}
		if ad.Registrar != nil {
			hookURL := strings.TrimRight(s.publicBaseURL(), "/") + "/webhooks/" + provider + "/" + automation.ID
			if _, err := ad.Registrar.RegisterWebhook(r.Context(), cred, state.Instance, hookURL, ""); err != nil {
				s.log.Warn("oauth auto-register webhook", "provider", provider, "err", err)
			}
		}
	}
	if ad, ok := s.integrations.Get(provider); ok && ad.Manifest.Kind != integration.KindTask {
		login, externalID := state.Email, state.Instance
		if provider == "slack" {
			if token.TeamName != "" {
				login = token.TeamName
			}
			if token.TeamID != "" {
				externalID = token.TeamID
			}
		}
		account := &domain.ConnectionAccount{
			ID: "acct_" + ulid.Make().String(), WorkspaceID: conn.WorkspaceID,
			ConnectionID: conn.ID, Provider: provider, Kind: accountKind(ad.Manifest.Auth),
			Login: login, Email: state.Email, ExternalID: externalID,
			CredentialSubject: conn.ID, IsDefault: true,
		}
		if account.Login == "" {
			account.Login = state.Instance
		}
		if err := s.store.UpsertConnectionAccount(r.Context(), account); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: "orca_oauth_state", Value: "", Path: "/api/v1/integrations", MaxAge: -1})
	http.Redirect(w, r, strings.TrimRight(s.cfg.AppURL, "/")+"/oauth/result?provider="+provider+"&status=ok", http.StatusFound)
}

type oauthToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	TeamID       string
	TeamName     string
	BotUserID    string
}

func exchangeOAuthCode(ctx context.Context, cfg oauthClientConfig, code, redirectURI string) (oauthToken, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
	}
	if cfg.Scopes != "" {
		form.Set("scope", cfg.Scopes)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return oauthToken{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return oauthToken{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Error        string `json:"error"`
		Description  string `json:"error_description"`
		Team         struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"team"`
		BotUserID string `json:"bot_user_id"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return oauthToken{}, fmt.Errorf("oauth token: invalid response: %s", strings.TrimSpace(string(raw)))
	}
	if resp.StatusCode >= 300 || body.AccessToken == "" {
		return oauthToken{}, fmt.Errorf("oauth token: status %d %s %s", resp.StatusCode, body.Error, body.Description)
	}
	out := oauthToken{AccessToken: body.AccessToken, RefreshToken: body.RefreshToken, TeamID: body.Team.ID, TeamName: body.Team.Name, BotUserID: body.BotUserID}
	if body.ExpiresIn > 0 {
		out.ExpiresAt = time.Now().UTC().Add(time.Duration(body.ExpiresIn) * time.Second)
	}
	return out, nil
}

func exchangeOAuthCodeBasic(ctx context.Context, cfg oauthClientConfig, code, redirectURI string) (oauthToken, error) {
	body, _ := json.Marshal(map[string]string{
		"grant_type": "authorization_code", "code": code, "redirect_uri": redirectURI,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL, bytes.NewReader(body))
	if err != nil {
		return oauthToken{}, err
	}
	req.SetBasicAuth(cfg.ClientID, cfg.ClientSecret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return oauthToken{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var decoded struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Error        string `json:"error"`
		Description  string `json:"error_description"`
		Team         struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"team"`
		BotUserID string `json:"bot_user_id"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return oauthToken{}, fmt.Errorf("oauth token: invalid response: %s", strings.TrimSpace(string(raw)))
	}
	if resp.StatusCode >= 300 || decoded.AccessToken == "" {
		return oauthToken{}, fmt.Errorf("oauth token: status %d %s %s", resp.StatusCode, decoded.Error, decoded.Description)
	}
	out := oauthToken{AccessToken: decoded.AccessToken, RefreshToken: decoded.RefreshToken}
	if decoded.ExpiresIn > 0 {
		out.ExpiresAt = time.Now().UTC().Add(time.Duration(decoded.ExpiresIn) * time.Second)
	}
	return out, nil
}

func connectionInstance(provider string, state githubOAuthState) string {
	if provider == "slack" && state.Channel != "" {
		return state.Channel
	}
	return state.Instance
}

func baseURLForState(provider string, state githubOAuthState) string {
	if provider == "gitlab" {
		if state.BaseURL != "" {
			return state.BaseURL
		}
		return "https://gitlab.com"
	}
	return state.Instance
}
