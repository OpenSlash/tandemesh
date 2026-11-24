package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Adapter is the GitLab implementation of task-source slots.
type Adapter struct {
	Secret     string
	HTTPClient *http.Client
	BaseURL    string
}

// New builds the GitLab integration.
func New(secret string) *integration.Integration {
	a := &Adapter{Secret: secret, HTTPClient: &http.Client{Timeout: 20 * time.Second}, BaseURL: "https://gitlab.com"}
	return &integration.Integration{
		Manifest: integration.Manifest{
			ID:   "gitlab",
			Name: "GitLab",
			Kind: integration.KindTask,
			Auth: integration.AuthToken,
			AuthOptions: []integration.AuthOption{
				{Kind: integration.AuthOAuth2, LabelKey: "providers.authOAuth", Default: true, Fields: []integration.ConfigField{
					{Key: "instance", LabelKey: "providers.project", Type: "text", Required: true, Placeholder: "group/project"},
					{Key: "baseUrl", LabelKey: "providers.baseUrl", Type: "url", Placeholder: "https://gitlab.com"},
				}},
				{Kind: integration.AuthToken, LabelKey: "providers.authToken", Fields: []integration.ConfigField{
					{Key: "token", LabelKey: "providers.token", Type: "password", Required: true, Secret: true},
					{Key: "instance", LabelKey: "providers.project", Type: "text", Required: true, Placeholder: "group/project"},
					{Key: "baseUrl", LabelKey: "providers.baseUrl", Type: "url", Placeholder: "https://gitlab.com"},
					{Key: "webhookSecret", LabelKey: "providers.webhookSecret", Type: "password", Secret: true},
				}},
			},
			Capabilities: []integration.Capability{integration.CapWebhook, integration.CapComment, integration.CapTransition},
			ConfigFields: []integration.ConfigField{
				{Key: "token", LabelKey: "providers.token", Type: "password", Required: true, Secret: true},
				{Key: "instance", LabelKey: "providers.project", Type: "text", Required: true, Placeholder: "group/project"},
				{Key: "baseUrl", LabelKey: "providers.baseUrl", Type: "url", Placeholder: "https://gitlab.com"},
				{Key: "webhookSecret", LabelKey: "providers.webhookSecret", Type: "password", Secret: true},
			},
		},
		Webhook:   a,
		Registrar: a,
		Probe:     a,
		Reader:    a,
		Work:      a,
	}
}

func (a *Adapter) base(cred integration.Credential) string {
	base := cred.BaseURL
	if base == "" {
		base = a.BaseURL
	}
	if base == "" {
		base = "https://gitlab.com"
	}
	return strings.TrimRight(base, "/")
}

func (a *Adapter) client() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return http.DefaultClient
}

func (a *Adapter) do(ctx context.Context, method, endpoint string, cred integration.Credential, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, err
	}
	if cred.AccessToken != "" {
		req.Header.Set("PRIVATE-TOKEN", cred.AccessToken)
	}
	req.Header.Set("Content-Type", "application/json")
	return a.client().Do(req)
}

func projectEndpoint(base, project string) string {
	return base + "/api/v4/projects/" + url.PathEscape(project)
}

// Probe implements integration.Probe.
func (a *Adapter) Probe(ctx context.Context, cred integration.Credential) error {
	resp, err := a.do(ctx, http.MethodGet, a.base(cred)+"/api/v4/user", cred, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("gitlab probe: status %d", resp.StatusCode)
	}
	return nil
}

type issueRow struct {
	IID         int      `json:"iid"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	State       string   `json:"state"`
	Labels      []string `json:"labels"`
	WebURL      string   `json:"web_url"`
	UpdatedAt   string   `json:"updated_at"`
	Author      struct {
		Name string `json:"name"`
	} `json:"author"`
	Assignees []struct {
		Name string `json:"name"`
	} `json:"assignees"`
}

// ListAssigned implements integration.Reader.
func (a *Adapter) ListAssigned(ctx context.Context, cred integration.Credential, opts integration.ListOptions) ([]integration.WorkItem, error) {
	base := a.base(cred)
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	endpoint := base + "/api/v4/issues?scope=assigned_to_me&state=opened&per_page=" + strconv.Itoa(limit)
	resp, err := a.do(ctx, http.MethodGet, endpoint, cred, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gitlab inbox: status %d", resp.StatusCode)
	}
	var rows []issueRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]integration.WorkItem, 0, len(rows))
	for _, row := range rows {
		project := projectFromURL(row.WebURL, base)
		updated, _ := time.Parse(time.RFC3339, row.UpdatedAt)
		assignee := ""
		if len(row.Assignees) > 0 {
			assignee = row.Assignees[0].Name
		}
		out = append(out, integration.WorkItem{
			Ref: domain.WorkRef{
				Provider: "gitlab", Instance: project, Type: "issue",
				ID: strconv.Itoa(row.IID), URL: row.WebURL, Revision: row.UpdatedAt,
			},
			Title: row.Title, Body: row.Description, Labels: row.Labels, Author: row.Author.Name,
			Assignee: assignee, Status: row.State, StatusGroup: issueGroup(row.State),
			Attachments: attachmentsFromDescription(row.Description, a.base(cred)+"/"+project),
			Updated:     updated,
		})
	}
	return out, nil
}

func projectFromURL(webURL, base string) string {
	parsed, err := url.Parse(webURL)
	if err != nil {
		return ""
	}
	path := strings.TrimPrefix(parsed.Path, "/")
	if idx := strings.Index(path, "/-/issues/"); idx >= 0 {
		return path[:idx]
	}
	return ""
}

func issueGroup(state string) string {
	switch strings.ToLower(state) {
	case "closed":
		return "completed"
	case "opened":
		return "started"
	default:
		return ""
	}
}

// Comment implements integration.WorkWriter.
func (a *Adapter) Comment(ctx context.Context, work domain.WorkRef, cred integration.Credential, markdown string) (string, error) {
	endpoint := projectEndpoint(a.base(cred), work.Instance) + "/issues/" + url.PathEscape(work.ID) + "/notes"
	resp, err := a.do(ctx, http.MethodPost, endpoint, cred, map[string]string{"body": markdown})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("gitlab comment: status %d", resp.StatusCode)
	}
	var out struct {
		ID int `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return fmt.Sprintf("%s#note_%d", work.URL, out.ID), nil
}

// Attach implements integration.WorkWriter.
func (a *Adapter) Attach(ctx context.Context, work domain.WorkRef, cred integration.Credential, kind, link string) error {
	_, err := a.Comment(ctx, work, cred, fmt.Sprintf("%s: %s", kind, link))
	return err
}

// Transition implements integration.WorkWriter.
func (a *Adapter) Transition(ctx context.Context, work domain.WorkRef, cred integration.Credential, to string) error {
	event := ""
	switch strings.ToLower(to) {
	case "closed", "close", "done", "completed":
		event = "close"
	case "open", "reopen", "todo", "in_progress":
		event = "reopen"
	default:
		return integration.ErrUnsupported
	}
	endpoint := projectEndpoint(a.base(cred), work.Instance) + "/issues/" + url.PathEscape(work.ID)
	resp, err := a.do(ctx, http.MethodPut, endpoint, cred, map[string]string{"state_event": event})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("gitlab transition: status %d", resp.StatusCode)
	}
	return nil
}

// Assign implements integration.AssigneeWriter.
func (a *Adapter) Assign(ctx context.Context, work domain.WorkRef, cred integration.Credential, assignee string) error {
	id, err := strconv.Atoi(assignee)
	if err != nil {
		return fmt.Errorf("gitlab: assignee must be a numeric user id")
	}
	endpoint := projectEndpoint(a.base(cred), work.Instance) + "/issues/" + url.PathEscape(work.ID)
	resp, err := a.do(ctx, http.MethodPut, endpoint, cred, map[string]any{"assignee_ids": []int{id}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("gitlab assign: status %d", resp.StatusCode)
	}
	return nil
}

// SearchUsers implements integration.UserSearcher.
func (a *Adapter) SearchUsers(ctx context.Context, cred integration.Credential, query string, limit int) ([]integration.User, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	endpoint := a.base(cred) + "/api/v4/users?search=" + url.QueryEscape(query) + "&per_page=" + strconv.Itoa(limit)
	resp, err := a.do(ctx, http.MethodGet, endpoint, cred, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gitlab users: status %d", resp.StatusCode)
	}
	var rows []struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Name     string `json:"name"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]integration.User, 0, len(rows))
	for _, row := range rows {
		out = append(out, integration.User{ID: strconv.Itoa(row.ID), Login: row.Username, Name: row.Name, Email: row.Email})
	}
	return out, nil
}

// ListPullRequests implements integration.PullRequestReader.
func (a *Adapter) ListPullRequests(ctx context.Context, cred integration.Credential, repo string) ([]integration.PullRequest, error) {
	endpoint := projectEndpoint(a.base(cred), repo) + "/merge_requests?state=all&per_page=100"
	resp, err := a.do(ctx, http.MethodGet, endpoint, cred, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gitlab merge requests: status %d", resp.StatusCode)
	}
	var rows []struct {
		IID          int    `json:"iid"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		State        string `json:"state"`
		WebURL       string `json:"web_url"`
		SourceBranch string `json:"source_branch"`
		Author       struct {
			Name string `json:"name"`
		} `json:"author"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]integration.PullRequest, 0, len(rows))
	for _, row := range rows {
		out = append(out, integration.PullRequest{
			Number: row.IID, URL: row.WebURL, Title: row.Title, Body: row.Description,
			State: row.State, Branch: row.SourceBranch, Author: row.Author.Name,
		})
	}
	return out, nil
}

// RegisterWebhook implements integration.WebhookRegistrar.
func (a *Adapter) RegisterWebhook(ctx context.Context, cred integration.Credential, instance, hookURL, secret string) (string, error) {
	endpoint := projectEndpoint(a.base(cred), instance) + "/hooks"
	body := map[string]any{
		"url":                   hookURL,
		"token":                 secret,
		"issues_events":         true,
		"note_events":           true,
		"merge_requests_events": true,
		"pipeline_events":       true,
		"push_events":           false,
	}
	resp, err := a.do(ctx, http.MethodPost, endpoint, cred, body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("gitlab register webhook: status %d", resp.StatusCode)
	}
	var out struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return strconv.Itoa(out.ID), nil
}

// UnregisterWebhook implements integration.WebhookUnregistrar.
func (a *Adapter) UnregisterWebhook(ctx context.Context, cred integration.Credential, instance, hookURL string) error {
	base := projectEndpoint(a.base(cred), instance) + "/hooks"
	resp, err := a.do(ctx, http.MethodGet, base, cred, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("gitlab list webhooks: status %d", resp.StatusCode)
	}
	var hooks []struct {
		ID  int    `json:"id"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hooks); err != nil {
		return err
	}
	for _, hook := range hooks {
		if hook.URL != hookURL {
			continue
		}
		del, err := a.do(ctx, http.MethodDelete, base+"/"+strconv.Itoa(hook.ID), cred, nil)
		if err != nil {
			return err
		}
		del.Body.Close()
		if del.StatusCode >= 300 {
			return fmt.Errorf("gitlab delete webhook: status %d", del.StatusCode)
		}
		return nil
	}
	return nil
}

// ListCommits implements integration.CommitReader.
func (a *Adapter) ListCommits(ctx context.Context, cred integration.Credential, repo string, limit int) ([]integration.Commit, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	endpoint := projectEndpoint(a.base(cred), repo) + "/repository/commits?per_page=" + strconv.Itoa(limit)
	resp, err := a.do(ctx, http.MethodGet, endpoint, cred, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gitlab commits: status %d", resp.StatusCode)
	}
	var rows []struct {
		ID         string `json:"id"`
		ShortID    string `json:"short_id"`
		Title      string `json:"title"`
		Message    string `json:"message"`
		WebURL     string `json:"web_url"`
		AuthorName string `json:"author_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]integration.Commit, 0, len(rows))
	for _, row := range rows {
		message := row.Message
		if message == "" {
			message = row.Title
		}
		out = append(out, integration.Commit{
			SHA: row.ID, URL: row.WebURL, Message: message, Author: row.AuthorName,
		})
	}
	return out, nil
}

// SearchIssues implements integration.ImportReader. The query is a GitLab
// issue search string, not JQL.
func (a *Adapter) SearchIssues(ctx context.Context, cred integration.Credential, instance, query string, limit int) ([]integration.WorkItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	endpoint := projectEndpoint(a.base(cred), instance) + "/issues?scope=all&state=opened&per_page=" + strconv.Itoa(limit) +
		"&search=" + url.QueryEscape(query)
	resp, err := a.do(ctx, http.MethodGet, endpoint, cred, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gitlab import: status %d", resp.StatusCode)
	}
	var rows []issueRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]integration.WorkItem, 0, len(rows))
	for _, row := range rows {
		updated, _ := time.Parse(time.RFC3339, row.UpdatedAt)
		assignee := ""
		if len(row.Assignees) > 0 {
			assignee = row.Assignees[0].Name
		}
		out = append(out, integration.WorkItem{
			Ref: domain.WorkRef{
				Provider: "gitlab", Instance: instance, Type: "issue",
				ID: strconv.Itoa(row.IID), URL: row.WebURL, Revision: row.UpdatedAt,
			},
			Title: row.Title, Body: row.Description, Labels: row.Labels, Author: row.Author.Name,
			Assignee: assignee, Status: row.State, StatusGroup: issueGroup(row.State),
			Attachments: attachmentsFromDescription(row.Description, a.base(cred)+"/"+instance), Updated: updated,
		})
	}
	return out, nil
}

// ListComments implements integration.CommentReader.
func (a *Adapter) ListComments(ctx context.Context, cred integration.Credential, work domain.WorkRef) ([]integration.Comment, error) {
	endpoint := projectEndpoint(a.base(cred), work.Instance) + "/issues/" + url.PathEscape(work.ID) + "/notes?per_page=100&sort=asc&order_by=created_at"
	resp, err := a.do(ctx, http.MethodGet, endpoint, cred, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gitlab comments: status %d", resp.StatusCode)
	}
	var rows []struct {
		ID     int    `json:"id"`
		Body   string `json:"body"`
		System bool   `json:"system"`
		Author struct {
			Name string `json:"name"`
		} `json:"author"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]integration.Comment, 0, len(rows))
	for _, row := range rows {
		if row.System {
			continue
		}
		created, _ := time.Parse(time.RFC3339, row.CreatedAt)
		out = append(out, integration.Comment{
			ID: strconv.Itoa(row.ID), Author: row.Author.Name, Body: row.Body,
			Attachments: attachmentsFromDescription(row.Body, a.base(cred)+"/"+work.Instance), CreatedAt: created,
		})
	}
	return out, nil
}

var uploadPattern = regexp.MustCompile(`(?:!\[[^\]]*\]\()?(/uploads/[^\s\)]+)`)

// attachmentsFromDescription extracts GitLab upload links from issue markdown.
func attachmentsFromDescription(description, projectWebURL string) []integration.Attachment {
	matches := uploadPattern.FindAllStringSubmatch(description, -1)
	out := make([]integration.Attachment, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		path := match[1]
		if seen[path] {
			continue
		}
		seen[path] = true
		parts := strings.Split(strings.Trim(path, "/"), "/")
		name := parts[len(parts)-1]
		out = append(out, integration.Attachment{
			ID: path, Name: name, URL: strings.TrimRight(projectWebURL, "/") + path,
		})
	}
	return out
}

// OpenAttachment implements integration.AttachmentReader.
func (a *Adapter) OpenAttachment(ctx context.Context, cred integration.Credential, attachment integration.Attachment) (io.ReadCloser, string, error) {
	resp, err := a.do(ctx, http.MethodGet, attachment.URL, cred, nil)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, "", fmt.Errorf("gitlab attachment: status %d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = attachment.Mime
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return resp.Body, contentType, nil
}
