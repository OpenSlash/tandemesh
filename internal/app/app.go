// Package app assembles the control plane from its parts.
//
// It exists so the HTTP binary, tests and a future Wails shell can start the
// same object graph in-process instead of duplicating wiring.
package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/OpenSlash/tandemesh/internal/api"
	"github.com/OpenSlash/tandemesh/internal/automation"
	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/connections"
	"github.com/OpenSlash/tandemesh/internal/credentials"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/githubapp"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/integration/feishu"
	"github.com/OpenSlash/tandemesh/internal/integration/github"
	"github.com/OpenSlash/tandemesh/internal/integration/gitlab"
	"github.com/OpenSlash/tandemesh/internal/integration/jira"
	"github.com/OpenSlash/tandemesh/internal/integration/linear"
	"github.com/OpenSlash/tandemesh/internal/integration/notion"
	"github.com/OpenSlash/tandemesh/internal/integration/outline"
	"github.com/OpenSlash/tandemesh/internal/integration/outlook"
	"github.com/OpenSlash/tandemesh/internal/integration/plane"
	"github.com/OpenSlash/tandemesh/internal/integration/slack"
	"github.com/OpenSlash/tandemesh/internal/scheduler"
	"github.com/OpenSlash/tandemesh/internal/store"
	"github.com/OpenSlash/tandemesh/internal/writeback"
)

// App owns the running control plane.
type App struct {
	Store    *store.Store
	Vault    *credentials.Vault
	Registry *integration.Registry
	Hub      *hub.Hub
	Handler  http.Handler

	cancel context.CancelFunc
}

// New builds and starts the control plane.
func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return nil, err
	}
	if err := st.Migrate(ctx); err != nil {
		_ = st.Close()
		return nil, err
	}
	if os.Getenv("TANDEMESH_LOCAL_REBUILD") == "1" {
		if err := st.RebuildLocalOnly(ctx); err != nil {
			return nil, err
		}
	}
	vault, err := credentials.New(st, cfg.MasterKey, log)
	if err != nil {
		_ = st.Close()
		return nil, err
	}
	clients := map[string]credentials.OAuthClient{}
	if cfg.LinearClientID != "" {
		clients["linear"] = credentials.OAuthClient{ClientID: cfg.LinearClientID, ClientSecret: cfg.LinearClientSecret}
	}
	if cfg.JiraClientID != "" {
		clients["jira"] = credentials.OAuthClient{ClientID: cfg.JiraClientID, ClientSecret: cfg.JiraClientSecret}
	}
	if cfg.GraphClientID != "" {
		clients["outlook"] = credentials.OAuthClient{ClientID: cfg.GraphClientID, ClientSecret: cfg.GraphClientSecret}
	}
	if len(clients) > 0 {
		vault.OAuth = credentials.OAuthConfig{Clients: clients}
	}
	h := hub.New()
	registry := integration.NewRegistry()
	for _, item := range []*integration.Integration{
		github.New(cfg.GitHubSecret),
		plane.New(),
		linear.New(cfg.LinearSecret),
		gitlab.New(os.Getenv("TANDEMESH_GITLAB_WEBHOOK_SECRET")),
		jira.New(cfg.JiraSecret),
		feishu.New(cfg.FeishuWebhook),
		notion.New(cfg.NotionToken),
		outline.New(cfg.OutlineURL, cfg.OutlineToken),
		outlook.New(cfg.GraphToken),
		slack.New(),
	} {
		if err := registry.Register(item); err != nil {
			_ = st.Close()
			return nil, err
		}
	}

	runCtx, cancel := context.WithCancel(ctx)
	ghapp := githubapp.New(cfg.GitHubAppID, cfg.GitHubAppSlug, cfg.GitHubAppPrivateKey, os.Getenv("TANDEMESH_GITHUB_API_URL"))
	creds := func(provider, connectionID, accountID string) integration.Credential {
		if provider == "github" && connectionID != "" && ghapp.Enabled() {
			if conn, err := st.GetConnection(context.Background(), connectionID); err == nil && conn != nil && conn.AuthMethod == "github_app" {
				if accounts, err := st.ListConnectionAccounts(context.Background(), connectionID); err == nil {
					for _, account := range accounts {
						if account.IsDefault && account.ExternalID != "" {
							if id, err := strconv.ParseInt(account.ExternalID, 10, 64); err == nil {
								if token, _, err := ghapp.InstallationToken(context.Background(), id); err == nil {
									return integration.Credential{Provider: provider, AccessToken: token}
								}
							}
						}
					}
				}
			}
		}
		subject := connectionID
		if accountID != "" {
			if account, err := st.GetConnectionAccount(context.Background(), accountID); err == nil && account != nil && account.CredentialSubject != "" {
				subject = account.CredentialSubject
			}
		}
		if subject == "" {
			subject = "default"
		}
		if stored, ok, err := vault.Ensure(context.Background(), provider, subject); err == nil && ok && stored.AccessToken != "" {
			return stored
		}
		switch provider {
		case "github":
			return integration.Credential{Provider: provider, AccessToken: cfg.GitHubToken}
		case "linear":
			return integration.Credential{Provider: provider, AccessToken: cfg.LinearToken}
		case "jira":
			return integration.Credential{Provider: provider, Subject: cfg.JiraEmail, AccessToken: cfg.JiraToken}
		case "notion":
			return integration.Credential{Provider: provider, AccessToken: cfg.NotionToken}
		case "outlook":
			return integration.Credential{Provider: provider, AccessToken: cfg.GraphToken}
		case "outline":
			return integration.Credential{Provider: provider, AccessToken: cfg.OutlineToken}
		}
		return integration.Credential{Provider: provider}
	}
	wb := writeback.New(st, registry, creds, log)
	wb.Policy = func(ctx context.Context, w *domain.Writeback) bool {
		return st.WritebackAllowed(ctx, w.Provider, w.ConnectionID, w.Work.Instance)
	}
	go wb.Run(runCtx)
	go connections.New(st, registry, vault, log).Run(runCtx)
	go automation.New(st, log).Run(runCtx)

	sch := scheduler.New(st, h, cfg.LeaseTTL, cfg.MaxAttempts, log)
	sch.SetLimits(cfg.MaxActivePerWorkspace, cfg.MaxActivePerAgent)
	go sch.Run(runCtx)

	server := api.New(cfg, st, sch, h, registry, vault, log)
	server.ConfigureGitHubApp(ghapp)
	go server.SyncLoop(runCtx)
	go server.PullRequestSyncLoop(runCtx)
	server.ResumeImports(runCtx)
	go server.RetryDeliveriesLoop(runCtx)
	go server.StartReminders(runCtx)
	handler := server.Router()
	return &App{Store: st, Vault: vault, Registry: registry, Hub: h, Handler: handler, cancel: cancel}, nil
}

// Close stops background workers and closes the database.
func (a *App) Close() error {
	if a.cancel != nil {
		a.cancel()
	}
	if a.Store != nil {
		return a.Store.Close()
	}
	return nil
}
