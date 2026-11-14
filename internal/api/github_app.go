package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/githubapp"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func (s *Server) githubAppCredential(ctx context.Context, connectionID string) (integration.Credential, bool) {
	if s.githubApp == nil || !s.githubApp.Enabled() {
		return integration.Credential{}, false
	}
	conn, err := s.store.GetConnection(ctx, connectionID)
	if err != nil || conn == nil || conn.AuthMethod != "github_app" {
		return integration.Credential{}, false
	}
	accounts, err := s.store.ListConnectionAccounts(ctx, connectionID)
	if err != nil {
		return integration.Credential{}, false
	}
	for _, account := range accounts {
		if !account.IsDefault || account.ExternalID == "" {
			continue
		}
		installationID, err := strconv.ParseInt(account.ExternalID, 10, 64)
		if err != nil {
			continue
		}
		token, _, err := s.githubApp.InstallationToken(ctx, installationID)
		if err != nil {
			s.log.Warn("github app installation token", "connection", connectionID, "err", err)
			return integration.Credential{}, false
		}
		return integration.Credential{Provider: "github", AccessToken: token, BaseURL: conn.BaseURL}, true
	}
	return integration.Credential{}, false
}

func (s *Server) githubAppStatus(w http.ResponseWriter, r *http.Request) {
	enabled := s.githubApp != nil && s.githubApp.Enabled()
	writeJSON(w, http.StatusOK, map[string]any{"enabled": enabled})
}

func (s *Server) githubAppStart(w http.ResponseWriter, r *http.Request) {
	if s.githubApp == nil || !s.githubApp.Enabled() {
		writeErr(w, http.StatusBadRequest, "E_APP", "GitHub App is not configured")
		return
	}
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	state := githubOAuthState{
		Provider: "github_app", Workspace: workspaceID,
		Name: r.URL.Query().Get("name"), Nonce: ulid.Make().String(),
		Expires: time.Now().Add(10 * time.Minute).Unix(),
	}
	signed, err := s.signOAuthState(state)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_OAUTH", err.Error())
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "orca_oauth_state", Value: signed, Path: "/api/v1/integrations",
		HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 600,
	})
	http.Redirect(w, r, s.githubApp.InstallURL(signed), http.StatusFound)
}

func (s *Server) githubAppCallback(w http.ResponseWriter, r *http.Request) {
	if s.githubApp == nil || !s.githubApp.Enabled() {
		writeErr(w, http.StatusBadRequest, "E_APP", "GitHub App is not configured")
		return
	}
	cookie, err := r.Cookie("orca_oauth_state")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "E_OAUTH", "missing state cookie")
		return
	}
	state, ok := s.verifyOAuthState(cookie.Value)
	if !ok || state.Provider != "github_app" || r.URL.Query().Get("state") != cookie.Value {
		writeErr(w, http.StatusBadRequest, "E_OAUTH", "state mismatch")
		return
	}
	installationID, err := strconv.ParseInt(r.URL.Query().Get("installation_id"), 10, 64)
	if err != nil || installationID == 0 {
		writeErr(w, http.StatusBadRequest, "E_INSTALLATION", "installation_id is required")
		return
	}
	installations, err := s.githubApp.Installations(r.Context())
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_INSTALLATION", err.Error())
		return
	}
	var installation *githubapp.Installation
	for _, item := range installations {
		if item.ID == installationID {
			copy := item
			installation = &copy
			break
		}
	}
	if installation == nil {
		writeErr(w, http.StatusBadRequest, "E_INSTALLATION", "installation not found")
		return
	}
	token, _, err := s.githubApp.InstallationToken(r.Context(), installationID)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_INSTALLATION", err.Error())
		return
	}
	repos, err := s.githubApp.Repositories(r.Context(), token)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_REPOS", err.Error())
		return
	}
	name := state.Name
	if name == "" {
		name = "GitHub App · " + installation.AccountLogin
	}
	conn := &domain.Connection{
		ID: "conn_" + ulid.Make().String(), WorkspaceID: state.Workspace,
		Provider: "github", Name: name, Status: domain.ConnectionConnected,
		AuthMethod: "github_app", SyncDirection: "read_write",
		Instance: installation.AccountLogin, AccountLogin: installation.AccountLogin,
		BaseURL: s.githubApp.BaseURL, ExecutionMode: "cloud", Enabled: true,
	}
	if err := s.store.UpsertConnection(r.Context(), conn); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	account := &domain.ConnectionAccount{
		ID: "acct_" + ulid.Make().String(), WorkspaceID: state.Workspace,
		ConnectionID: conn.ID, Provider: "github", Kind: "org",
		Login: installation.AccountLogin, ExternalID: strconv.FormatInt(installationID, 10),
		CredentialSubject: conn.ID, IsDefault: true,
	}
	if err := s.store.UpsertConnectionAccount(r.Context(), account); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	for _, repo := range repos {
		binding := &domain.ResourceBinding{
			ID: "bind_" + ulid.Make().String(), WorkspaceID: state.Workspace,
			ConnectionID: conn.ID, Provider: "github", ExternalID: repo.FullName,
			Name: repo.FullName, Kind: "repo", Direction: "read_write",
			AccountID: account.ID, Enabled: true,
		}
		if err := s.store.UpsertBinding(r.Context(), binding); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
	}
	automation := &domain.Automation{
		ID: "auto_" + ulid.Make().String(), WorkspaceID: state.Workspace,
		Name: name, Provider: "github", TriggerKind: domain.TriggerCreated,
		ConnectionID: conn.ID, Writeback: map[string]any{"comment": true}, Enabled: true, Base: "main",
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
	http.SetCookie(w, &http.Cookie{Name: "orca_oauth_state", Value: "", Path: "/api/v1/integrations", MaxAge: -1})
	http.Redirect(w, r, strings.TrimRight(s.cfg.AppURL, "/")+"/providers?connected=github", http.StatusFound)
}
