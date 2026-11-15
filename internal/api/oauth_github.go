package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

type githubOAuthState struct {
	Provider      string `json:"provider,omitempty"`
	Workspace     string `json:"workspace"`
	Instance      string `json:"instance"`
	BaseURL       string `json:"baseUrl,omitempty"`
	Channel       string `json:"channel,omitempty"`
	MemberID      string `json:"memberId,omitempty"`
	Name          string `json:"name"`
	AgentID       string `json:"agentId"`
	Email         string `json:"email,omitempty"`
	SyncDirection string `json:"syncDirection,omitempty"`
	ConnectionID  string `json:"connectionId,omitempty"`
	Nonce         string `json:"nonce"`
	Expires       int64  `json:"expires"`
}

func (s *Server) oauthKey() []byte {
	if s.cfg.SessionKey != "" {
		return []byte(s.cfg.SessionKey)
	}
	return []byte(s.cfg.ApprovalKey)
}

func (s *Server) signOAuthState(state githubOAuthState) (string, error) {
	raw, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, s.oauthKey())
	mac.Write(raw)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (s *Server) verifyOAuthState(value string) (githubOAuthState, bool) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return githubOAuthState{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return githubOAuthState{}, false
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return githubOAuthState{}, false
	}
	mac := hmac.New(sha256.New, s.oauthKey())
	mac.Write(raw)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return githubOAuthState{}, false
	}
	var state githubOAuthState
	if err := json.Unmarshal(raw, &state); err != nil {
		return githubOAuthState{}, false
	}
	if time.Now().Unix() > state.Expires {
		return githubOAuthState{}, false
	}
	return state, true
}

// githubOAuthStatus reports whether OAuth is configured.
func (s *Server) githubOAuthStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{
		"enabled": s.cfg.GitHubClientID != "" && s.cfg.GitHubClientSecret != "",
	})
}

// githubOAuthStart redirects the browser to GitHub's consent screen.
func (s *Server) githubOAuthStart(w http.ResponseWriter, r *http.Request) {
	if s.cfg.GitHubClientID == "" || s.cfg.GitHubClientSecret == "" {
		writeErr(w, http.StatusBadRequest, "E_OAUTH", "GitHub OAuth is not configured")
		return
	}
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	instance := strings.TrimSpace(r.URL.Query().Get("instance"))
	memberID := strings.TrimSpace(r.URL.Query().Get("memberId"))
	if instance == "" && memberID == "" {
		writeErr(w, http.StatusBadRequest, "E_INSTANCE", "instance (owner/repo) or memberId is required")
		return
	}
	state := githubOAuthState{
		Workspace: workspaceID, Instance: instance, MemberID: memberID,
		Name: r.URL.Query().Get("name"), AgentID: r.URL.Query().Get("agentId"),
		ConnectionID: r.URL.Query().Get("connectionId"),
		Nonce:        ulid.Make().String(), Expires: time.Now().Add(10 * time.Minute).Unix(),
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
		Name: "orca_gh_state", Value: signed, Path: "/api/v1/integrations/github",
		HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 600,
	})
	scope := "repo read:user"
	if memberID != "" {
		scope = "read:user user:email"
	}
	redirect := s.cfg.GitHubAuthorizeURL + "?" + url.Values{
		"client_id":    {s.cfg.GitHubClientID},
		"redirect_uri": {s.redirectURI()},
		"scope":        {scope},
		"state":        {signed},
	}.Encode()
	http.Redirect(w, r, redirect, http.StatusFound)
}

func (s *Server) redirectURI() string {
	return strings.TrimRight(s.oauthBaseURL(), "/") + "/api/v1/integrations/github/oauth/callback"
}

// githubOAuthCallback exchanges the code, stores the token and creates a binding.
func (s *Server) githubOAuthCallback(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("orca_gh_state")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "E_OAUTH", "missing state cookie")
		return
	}
	state, ok := s.verifyOAuthState(cookie.Value)
	if !ok || r.URL.Query().Get("state") != cookie.Value {
		writeErr(w, http.StatusBadRequest, "E_OAUTH", "state mismatch")
		return
	}
	if used, err := s.store.ConsumeOAuthNonce(r.Context(), state.Nonce); err != nil || !used {
		writeErr(w, http.StatusBadRequest, "E_REPLAY", "OAuth state was already used")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		writeErr(w, http.StatusBadRequest, "E_OAUTH", "missing code")
		return
	}
	form := url.Values{
		"client_id":     {s.cfg.GitHubClientID},
		"client_secret": {s.cfg.GitHubClientSecret},
		"code":          {code},
		"redirect_uri":  {s.redirectURI()},
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.cfg.GitHubTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_OAUTH", err.Error())
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_OAUTH", err.Error())
		return
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var token struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(raw, &token); err != nil || token.AccessToken == "" {
		writeErr(w, http.StatusBadGateway, "E_OAUTH", fmt.Sprintf("token exchange failed: %s", strings.TrimSpace(string(raw))))
		return
	}
	if state.MemberID != "" {
		if err := s.completeMemberOAuth(r.Context(), "github", state.Workspace, state.MemberID, "", token.AccessToken); err != nil {
			writeErr(w, http.StatusBadGateway, "E_OAUTH", err.Error())
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "orca_gh_state", Value: "", Path: "/api/v1/integrations/github", MaxAge: -1})
		http.Redirect(w, r, strings.TrimRight(s.cfg.AppURL, "/")+"/oauth/result?provider=github&status=ok", http.StatusFound)
		return
	}
	if state.ConnectionID != "" {
		conn, err := s.store.GetConnection(r.Context(), state.ConnectionID)
		if err != nil || conn == nil {
			writeErr(w, http.StatusBadRequest, "E_OAUTH", "connection not found")
			return
		}
		cred := integration.Credential{Provider: conn.Provider, Subject: conn.ID, AccessToken: token.AccessToken}
		if ad, ok := s.integrations.Get(conn.Provider); ok && ad.Probe != nil {
			if err := ad.Probe.Probe(r.Context(), cred); err != nil {
				writeErr(w, http.StatusBadGateway, "E_PROBE", err.Error())
				return
			}
		}
		if s.vault != nil {
			if err := s.vault.Put(r.Context(), cred); err != nil {
				writeErr(w, http.StatusInternalServerError, "E_VAULT", err.Error())
				return
			}
		}
		conn.Status = domain.ConnectionConnected
		conn.AuthMethod = "oauth"
		if err := s.store.UpsertConnection(r.Context(), conn); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "orca_gh_state", Value: "", Path: "/api/v1/integrations/github", MaxAge: -1})
		http.Redirect(w, r, strings.TrimRight(s.cfg.AppURL, "/")+"/oauth/result?provider=github&status=ok", http.StatusFound)
		return
	}
	conn := &domain.Connection{
		ID: "conn_" + ulid.Make().String(), WorkspaceID: state.Workspace,
		Provider: "github", Name: state.Name, Status: domain.ConnectionConnected,
		AuthMethod: "oauth", Instance: state.Instance, ExecutionMode: "cloud", Enabled: true,
	}
	if conn.Name == "" {
		conn.Name = "GitHub · " + state.Instance
	}
	cred := integration.Credential{Provider: "github", Subject: conn.ID, AccessToken: token.AccessToken}
	if ad, ok := s.integrations.Get("github"); ok && ad.Probe != nil {
		if err := ad.Probe.Probe(r.Context(), cred); err != nil {
			writeErr(w, http.StatusBadGateway, "E_PROBE", err.Error())
			return
		}
	}
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
	automation := &domain.Automation{
		ID: "auto_" + ulid.Make().String(), WorkspaceID: state.Workspace,
		Name: conn.Name, Provider: "github", Instance: state.Instance,
		AgentID: state.AgentID, TriggerKind: domain.TriggerAny, ConnectionID: conn.ID,
		Writeback: map[string]any{"comment": true}, Enabled: true, Base: "main",
	}
	if err := s.store.UpsertAutomation(r.Context(), automation); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "orca_gh_state", Value: "", Path: "/api/v1/integrations/github", MaxAge: -1})
	target := strings.TrimRight(s.cfg.AppURL, "/") + "/providers?connected=github"
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Server) completeMemberOAuth(ctx context.Context, provider, workspaceID, memberID, baseURL, accessToken string) error {
	switch provider {
	case "github":
		return s.completeGitHubMember(ctx, workspaceID, memberID, accessToken)
	case "gitlab":
		return s.completeGitLabMember(ctx, workspaceID, memberID, baseURL, accessToken)
	case "jira":
		return s.completeJiraMember(ctx, workspaceID, memberID, accessToken)
	case "linear":
		return s.completeLinearMember(ctx, workspaceID, memberID, baseURL, accessToken)
	default:
		return fmt.Errorf("member OAuth is unsupported for %s", provider)
	}
}

func (s *Server) completeGitHubMember(ctx context.Context, workspaceID, memberID, accessToken string) error {
	base := os.Getenv("TANDEMESH_GITHUB_API_URL")
	if base == "" {
		base = "https://api.github.com"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/user", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("github user: status %d", resp.StatusCode)
	}
	var user struct {
		ID    int    `json:"id"`
		Login string `json:"login"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return err
	}
	return s.saveMemberIdentity(ctx, "github", workspaceID, memberID, strconv.Itoa(user.ID), user.Login, user.Email, accessToken)
}

func (s *Server) completeGitLabMember(ctx context.Context, workspaceID, memberID, baseURL, accessToken string) error {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/v4/user", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("gitlab user: status %d", resp.StatusCode)
	}
	var user struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return err
	}
	return s.saveMemberIdentity(ctx, "gitlab", workspaceID, memberID, strconv.Itoa(user.ID), user.Username, user.Email, accessToken)
}

func (s *Server) completeJiraMember(ctx context.Context, workspaceID, memberID, accessToken string) error {
	base := os.Getenv("TANDEMESH_JIRA_API_URL")
	if base == "" {
		base = "https://api.atlassian.com"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/me", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("jira user: status %d", resp.StatusCode)
	}
	var user struct {
		AccountID string `json:"account_id"`
		Name      string `json:"name"`
		Email     string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return err
	}
	return s.saveMemberIdentity(ctx, "jira", workspaceID, memberID, user.AccountID, user.Name, user.Email, accessToken)
}

func (s *Server) completeLinearMember(ctx context.Context, workspaceID, memberID, baseURL, accessToken string) error {
	if baseURL == "" {
		baseURL = "https://api.linear.app/graphql"
	}
	body := strings.NewReader(`{"query":"query { viewer { id name email } }"}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("linear user: status %d", resp.StatusCode)
	}
	var payload struct {
		Data struct {
			Viewer struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"viewer"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}
	return s.saveMemberIdentity(ctx, "linear", workspaceID, memberID, payload.Data.Viewer.ID, payload.Data.Viewer.Name, payload.Data.Viewer.Email, accessToken)
}
func (s *Server) saveMemberIdentity(ctx context.Context, provider, workspaceID, memberID, externalID, login, email, accessToken string) error {
	if login == "" {
		return fmt.Errorf("%s user: missing login", provider)
	}
	identity := &domain.MemberIdentity{
		ID: "mid_" + ulid.Make().String(), WorkspaceID: workspaceID, MemberID: memberID,
		Provider: provider, ExternalID: externalID, Login: login, Email: email,
	}
	identity.CredentialSubject = "mid:" + identity.ID
	if s.vault != nil {
		if err := s.vault.Put(ctx, integration.Credential{Provider: provider, Subject: identity.CredentialSubject, AccessToken: accessToken, Account: email}); err != nil {
			return err
		}
	}
	return s.store.UpsertMemberIdentity(ctx, identity)
}
