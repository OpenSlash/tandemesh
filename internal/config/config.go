// Package config loads runtime configuration from environment variables.
// Every value has a working default except the master key, which is only
// required when credentials are stored.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config is the control-plane configuration.
type Config struct {
	Listen         string
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
	return Config{
		Listen:                 env("ORCA_LISTEN", ":8080"),
		BaseURL:                env("ORCA_BASE_URL", "http://localhost:8080"),
		WritebackState:         env("ORCA_WRITEBACK_STATE", "") == "1",
		DBPath:                 env("ORCA_DB_PATH", "orca.db"),
		MasterKey:              env("ORCA_MASTER_KEY", ""),
		RunTimeout:             duration("ORCA_RUN_TIMEOUT", 30*time.Minute),
		LeaseTTL:               duration("ORCA_LEASE_TTL", 30*time.Second),
		MaxAttempts:            integer("ORCA_MAX_ATTEMPTS", 3),
		MaxActivePerWorkspace:  integer("ORCA_MAX_ACTIVE_PER_WORKSPACE", 0),
		MaxActivePerAgent:      integer("ORCA_MAX_ACTIVE_PER_AGENT", 0),
		JoinToken:              env("ORCA_JOIN_TOKEN", "dev-join-token"),
		AgentdDir:              env("ORCA_AGENTD_DIR", "bin/agents"),
		SkillMarketFile:        env("ORCA_SKILL_MARKET_FILE", ""),
		SkillMarketURL:         env("ORCA_SKILL_MARKET_URL", ""),
		SkillMarketRepos:       env("ORCA_SKILL_MARKET_REPOS", ""),
		GitHubSecret:           env("ORCA_GITHUB_SECRET", ""),
		LinearSecret:           env("ORCA_LINEAR_SECRET", ""),
		JiraSecret:             env("ORCA_JIRA_SECRET", ""),
		GitHubToken:            env("ORCA_GITHUB_TOKEN", ""),
		GitHubClientID:         env("ORCA_GITHUB_CLIENT_ID", ""),
		GitHubClientSecret:     env("ORCA_GITHUB_CLIENT_SECRET", ""),
		GitHubAuthorizeURL:     env("ORCA_GITHUB_AUTHORIZE_URL", "https://github.com/login/oauth/authorize"),
		GitHubTokenURL:         env("ORCA_GITHUB_TOKEN_URL", "https://github.com/login/oauth/access_token"),
		GitHubAppID:            env("ORCA_GITHUB_APP_ID", ""),
		GitHubAppSlug:          env("ORCA_GITHUB_APP_SLUG", ""),
		GitHubAppPrivateKey:    env("ORCA_GITHUB_APP_PRIVATE_KEY", ""),
		GitHubAppWebhookSecret: env("ORCA_GITHUB_APP_WEBHOOK_SECRET", ""),
		AppURL:                 env("ORCA_APP_URL", env("ORCA_BASE_URL", "http://localhost:8080")),
		LinearToken:            env("ORCA_LINEAR_TOKEN", ""),
		JiraToken:              env("ORCA_JIRA_TOKEN", ""),
		JiraEmail:              env("ORCA_JIRA_EMAIL", ""),
		NotionToken:            env("ORCA_NOTION_TOKEN", ""),
		NotionClientID:         env("ORCA_NOTION_CLIENT_ID", ""),
		NotionClientSecret:     env("ORCA_NOTION_CLIENT_SECRET", ""),
		NotionAuthorizeURL:     env("ORCA_NOTION_AUTHORIZE_URL", "https://api.notion.com/v1/oauth/authorize"),
		NotionTokenURL:         env("ORCA_NOTION_TOKEN_URL", "https://api.notion.com/v1/oauth/token"),
		GraphToken:             env("ORCA_GRAPH_TOKEN", ""),
		OutlineURL:             env("ORCA_OUTLINE_URL", "https://app.getoutline.com"),
		OutlineToken:           env("ORCA_OUTLINE_TOKEN", ""),
		LinearClientID:         env("ORCA_LINEAR_CLIENT_ID", ""),
		LinearClientSecret:     env("ORCA_LINEAR_CLIENT_SECRET", ""),
		LinearAuthorizeURL:     env("ORCA_LINEAR_AUTHORIZE_URL", "https://linear.app/oauth/authorize"),
		LinearTokenURL:         env("ORCA_LINEAR_TOKEN_URL", "https://api.linear.app/oauth/token"),
		LinearScopes:           env("ORCA_LINEAR_SCOPES", "read write issues:create comments:create"),
		JiraClientID:           env("ORCA_JIRA_CLIENT_ID", ""),
		JiraClientSecret:       env("ORCA_JIRA_CLIENT_SECRET", ""),
		JiraAuthorizeURL:       env("ORCA_JIRA_AUTHORIZE_URL", "https://auth.atlassian.com/authorize"),
		JiraTokenURL:           env("ORCA_JIRA_TOKEN_URL", "https://auth.atlassian.com/oauth/token"),
		JiraScopes:             env("ORCA_JIRA_SCOPES", "read:jira-work write:jira-work offline_access"),
		GitLabClientID:         env("ORCA_GITLAB_CLIENT_ID", ""),
		GitLabClientSecret:     env("ORCA_GITLAB_CLIENT_SECRET", ""),
		GitLabAuthorizeURL:     env("ORCA_GITLAB_AUTHORIZE_URL", "https://gitlab.com/oauth/authorize"),
		GitLabTokenURL:         env("ORCA_GITLAB_TOKEN_URL", "https://gitlab.com/oauth/token"),
		GitLabScopes:           env("ORCA_GITLAB_SCOPES", "api read_user"),
		GraphClientID:          env("ORCA_GRAPH_CLIENT_ID", ""),
		GraphClientSecret:      env("ORCA_GRAPH_CLIENT_SECRET", ""),
		GraphAuthorizeURL:      env("ORCA_GRAPH_AUTHORIZE_URL", "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"),
		GraphTokenURL:          env("ORCA_GRAPH_TOKEN_URL", "https://login.microsoftonline.com/common/oauth2/v2.0/token"),
		GraphScopes:            env("ORCA_GRAPH_SCOPES", "offline_access Mail.Read"),
		SlackToken:             env("ORCA_SLACK_TOKEN", ""),
		SlackChannel:           env("ORCA_SLACK_CHANNEL", ""),
		SlackClientID:          env("ORCA_SLACK_CLIENT_ID", ""),
		SlackClientSecret:      env("ORCA_SLACK_CLIENT_SECRET", ""),
		SlackAuthorizeURL:      env("ORCA_SLACK_AUTHORIZE_URL", "https://slack.com/oauth/v2/authorize"),
		SlackTokenURL:          env("ORCA_SLACK_TOKEN_URL", "https://slack.com/api/oauth.v2.access"),
		SlackScopes:            env("ORCA_SLACK_SCOPES", "chat:write,channels:read,groups:read,im:write"),
		SlackSigningSecret:     env("ORCA_SLACK_SIGNING_SECRET", ""),
		FeishuWebhook:          env("ORCA_FEISHU_WEBHOOK", ""),
		ApprovalKey:            env("ORCA_APPROVAL_KEY", env("ORCA_JOIN_TOKEN", "dev-join-token")),
		User:                   env("ORCA_USER", "me"),
		SessionKey:             env("ORCA_SESSION_KEY", env("ORCA_APPROVAL_KEY", env("ORCA_JOIN_TOKEN", "dev-join-token"))),
		AllowHeaderAuth:        env("ORCA_ALLOW_HEADER_AUTH", "on") == "on",
		Telemetry:              env("ORCA_TELEMETRY", "off") == "on",
	}
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
