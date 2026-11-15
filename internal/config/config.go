// Package config loads runtime configuration from environment variables.
// Every value has a working default except the master key, which is only
// required when credentials are stored.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the control-plane configuration.
type Config struct {
	Listen         string
	Profile        string
	PublicURL      string
	OAuthRedirect  string
	AuthMode       string
	AdminPassword  string
	BaseURL        string
	WritebackState bool
	DBPath         string
	MasterKey      string

	RunTimeout             time.Duration
	LeaseTTL               time.Duration
	MaxAttempts            int
	MaxActivePerWorkspace  int
	MaxActivePerAgent      int
	JoinToken              string
	AgentdDir              string
	SkillMarketFile        string
	SkillMarketURL         string
	SkillMarketRepos       string
	GitHubSecret           string
	GitHubClientID         string
	GitHubClientSecret     string
	GitHubAuthorizeURL     string
	GitHubTokenURL         string
	GitHubAppID            string
	GitHubAppSlug          string
	GitHubAppPrivateKey    string
	GitHubAppWebhookSecret string
	AppURL                 string
	LinearSecret           string
	JiraSecret             string
	GitHubToken            string
	LinearToken            string
	JiraToken              string
	JiraEmail              string
	NotionToken            string
	NotionClientID         string
	NotionClientSecret     string
	NotionAuthorizeURL     string
	NotionTokenURL         string
	GraphToken             string
	OutlineURL             string
	OutlineToken           string
	LinearClientID         string
	LinearClientSecret     string
	LinearAuthorizeURL     string
	LinearTokenURL         string
	LinearScopes           string
	JiraClientID           string
	JiraClientSecret       string
	JiraAuthorizeURL       string
	JiraTokenURL           string
	JiraScopes             string
	GitLabClientID         string
	GitLabClientSecret     string
	GitLabAuthorizeURL     string
	GitLabTokenURL         string
	GitLabScopes           string
	GraphClientID          string
	GraphClientSecret      string
	GraphAuthorizeURL      string
	GraphTokenURL          string
	GraphScopes            string
	SlackToken             string
	SlackChannel           string
	SlackClientID          string
	SlackClientSecret      string
	SlackAuthorizeURL      string
	SlackTokenURL          string
	SlackScopes            string
	SlackSigningSecret     string
	FeishuWebhook          string
	ApprovalKey            string
	User                   string
	SessionKey             string
	AllowHeaderAuth        bool

	Telemetry bool
}

// Load reads configuration from the environment.
func Load() Config {
	applyLegacyEnv()
	return Config{
		Listen:                 env("TANDEMESH_LISTEN", ":8080"),
		Profile:                env("TANDEMESH_PROFILE", "local"),
		PublicURL:              env("TANDEMESH_PUBLIC_URL", ""),
		OAuthRedirect:          env("TANDEMESH_OAUTH_REDIRECT", "auto"),
		AuthMode:               env("TANDEMESH_AUTH", "none"),
		AdminPassword:          env("TANDEMESH_ADMIN_PASSWORD", ""),
		BaseURL:                env("TANDEMESH_BASE_URL", "http://localhost:8080"),
		WritebackState:         env("TANDEMESH_WRITEBACK_STATE", "") == "1",
		DBPath:                 env("TANDEMESH_DB_PATH", defaultDBPath()),
		MasterKey:              env("TANDEMESH_MASTER_KEY", ""),
		RunTimeout:             duration("TANDEMESH_RUN_TIMEOUT", 30*time.Minute),
		LeaseTTL:               duration("TANDEMESH_LEASE_TTL", 30*time.Second),
		MaxAttempts:            integer("TANDEMESH_MAX_ATTEMPTS", 3),
		MaxActivePerWorkspace:  integer("TANDEMESH_MAX_ACTIVE_PER_WORKSPACE", 0),
		MaxActivePerAgent:      integer("TANDEMESH_MAX_ACTIVE_PER_AGENT", 0),
		JoinToken:              env("TANDEMESH_JOIN_TOKEN", "dev-join-token"),
		AgentdDir:              env("TANDEMESH_AGENTD_DIR", "bin/agents"),
		SkillMarketFile:        env("TANDEMESH_SKILL_MARKET_FILE", ""),
		SkillMarketURL:         env("TANDEMESH_SKILL_MARKET_URL", ""),
		SkillMarketRepos:       env("TANDEMESH_SKILL_MARKET_REPOS", ""),
		GitHubSecret:           env("TANDEMESH_GITHUB_SECRET", ""),
		LinearSecret:           env("TANDEMESH_LINEAR_SECRET", ""),
		JiraSecret:             env("TANDEMESH_JIRA_SECRET", ""),
		GitHubToken:            env("TANDEMESH_GITHUB_TOKEN", ""),
		GitHubClientID:         env("TANDEMESH_GITHUB_CLIENT_ID", ""),
		GitHubClientSecret:     env("TANDEMESH_GITHUB_CLIENT_SECRET", ""),
		GitHubAuthorizeURL:     env("TANDEMESH_GITHUB_AUTHORIZE_URL", "https://github.com/login/oauth/authorize"),
		GitHubTokenURL:         env("TANDEMESH_GITHUB_TOKEN_URL", "https://github.com/login/oauth/access_token"),
		GitHubAppID:            env("TANDEMESH_GITHUB_APP_ID", ""),
		GitHubAppSlug:          env("TANDEMESH_GITHUB_APP_SLUG", ""),
		GitHubAppPrivateKey:    env("TANDEMESH_GITHUB_APP_PRIVATE_KEY", ""),
		GitHubAppWebhookSecret: env("TANDEMESH_GITHUB_APP_WEBHOOK_SECRET", ""),
		AppURL:                 env("TANDEMESH_APP_URL", env("TANDEMESH_BASE_URL", "http://localhost:8080")),
		LinearToken:            env("TANDEMESH_LINEAR_TOKEN", ""),
		JiraToken:              env("TANDEMESH_JIRA_TOKEN", ""),
		JiraEmail:              env("TANDEMESH_JIRA_EMAIL", ""),
		NotionToken:            env("TANDEMESH_NOTION_TOKEN", ""),
		NotionClientID:         env("TANDEMESH_NOTION_CLIENT_ID", ""),
		NotionClientSecret:     env("TANDEMESH_NOTION_CLIENT_SECRET", ""),
		NotionAuthorizeURL:     env("TANDEMESH_NOTION_AUTHORIZE_URL", "https://api.notion.com/v1/oauth/authorize"),
		NotionTokenURL:         env("TANDEMESH_NOTION_TOKEN_URL", "https://api.notion.com/v1/oauth/token"),
		GraphToken:             env("TANDEMESH_GRAPH_TOKEN", ""),
		OutlineURL:             env("TANDEMESH_OUTLINE_URL", "https://app.getoutline.com"),
		OutlineToken:           env("TANDEMESH_OUTLINE_TOKEN", ""),
		LinearClientID:         env("TANDEMESH_LINEAR_CLIENT_ID", ""),
		LinearClientSecret:     env("TANDEMESH_LINEAR_CLIENT_SECRET", ""),
		LinearAuthorizeURL:     env("TANDEMESH_LINEAR_AUTHORIZE_URL", "https://linear.app/oauth/authorize"),
		LinearTokenURL:         env("TANDEMESH_LINEAR_TOKEN_URL", "https://api.linear.app/oauth/token"),
		LinearScopes:           env("TANDEMESH_LINEAR_SCOPES", "read write issues:create comments:create"),
		JiraClientID:           env("TANDEMESH_JIRA_CLIENT_ID", ""),
		JiraClientSecret:       env("TANDEMESH_JIRA_CLIENT_SECRET", ""),
		JiraAuthorizeURL:       env("TANDEMESH_JIRA_AUTHORIZE_URL", "https://auth.atlassian.com/authorize"),
		JiraTokenURL:           env("TANDEMESH_JIRA_TOKEN_URL", "https://auth.atlassian.com/oauth/token"),
		JiraScopes:             env("TANDEMESH_JIRA_SCOPES", "read:jira-work write:jira-work offline_access"),
		GitLabClientID:         env("TANDEMESH_GITLAB_CLIENT_ID", ""),
		GitLabClientSecret:     env("TANDEMESH_GITLAB_CLIENT_SECRET", ""),
		GitLabAuthorizeURL:     env("TANDEMESH_GITLAB_AUTHORIZE_URL", "https://gitlab.com/oauth/authorize"),
		GitLabTokenURL:         env("TANDEMESH_GITLAB_TOKEN_URL", "https://gitlab.com/oauth/token"),
		GitLabScopes:           env("TANDEMESH_GITLAB_SCOPES", "api read_user"),
		GraphClientID:          env("TANDEMESH_GRAPH_CLIENT_ID", ""),
		GraphClientSecret:      env("TANDEMESH_GRAPH_CLIENT_SECRET", ""),
		GraphAuthorizeURL:      env("TANDEMESH_GRAPH_AUTHORIZE_URL", "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"),
		GraphTokenURL:          env("TANDEMESH_GRAPH_TOKEN_URL", "https://login.microsoftonline.com/common/oauth2/v2.0/token"),
		GraphScopes:            env("TANDEMESH_GRAPH_SCOPES", "offline_access Mail.Read"),
		SlackToken:             env("TANDEMESH_SLACK_TOKEN", ""),
		SlackChannel:           env("TANDEMESH_SLACK_CHANNEL", ""),
		SlackClientID:          env("TANDEMESH_SLACK_CLIENT_ID", ""),
		SlackClientSecret:      env("TANDEMESH_SLACK_CLIENT_SECRET", ""),
		SlackAuthorizeURL:      env("TANDEMESH_SLACK_AUTHORIZE_URL", "https://slack.com/oauth/v2/authorize"),
		SlackTokenURL:          env("TANDEMESH_SLACK_TOKEN_URL", "https://slack.com/api/oauth.v2.access"),
		SlackScopes:            env("TANDEMESH_SLACK_SCOPES", "chat:write,channels:read,groups:read,im:write"),
		SlackSigningSecret:     env("TANDEMESH_SLACK_SIGNING_SECRET", ""),
		FeishuWebhook:          env("TANDEMESH_FEISHU_WEBHOOK", ""),
		ApprovalKey:            env("TANDEMESH_APPROVAL_KEY", env("TANDEMESH_JOIN_TOKEN", "dev-join-token")),
		User:                   env("TANDEMESH_USER", "me"),
		SessionKey:             env("TANDEMESH_SESSION_KEY", env("TANDEMESH_APPROVAL_KEY", env("TANDEMESH_JOIN_TOKEN", "dev-join-token"))),
		AllowHeaderAuth:        env("TANDEMESH_ALLOW_HEADER_AUTH", "on") == "on",
		Telemetry:              env("TANDEMESH_TELEMETRY", "off") == "on",
	}
}

func applyLegacyEnv() {
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || !strings.HasPrefix(key, "ORCA_") {
			continue
		}
		target := "TANDEMESH_" + strings.TrimPrefix(key, "ORCA_")
		if os.Getenv(target) == "" {
			_ = os.Setenv(target, value)
		}
	}
}

func defaultDBPath() string {
	if _, err := os.Stat("orca.db"); err == nil {
		return "orca.db"
	}
	return "tandemesh.db"
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func duration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func integer(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
