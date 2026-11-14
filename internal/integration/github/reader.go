package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// ListAssigned implements integration.Reader.
func (a *Adapter) ListAssigned(ctx context.Context, cred integration.Credential, opts integration.ListOptions) ([]integration.WorkItem, error) {
	if cred.AccessToken == "" {
		return nil, fmt.Errorf("github: missing token")
	}
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	endpoint := fmt.Sprintf("%s/issues?filter=assigned&state=open&per_page=%d", a.baseURLFor(cred), limit)
	if !opts.Since.IsZero() {
		endpoint += "&since=" + url.QueryEscape(opts.Since.UTC().Format(time.RFC3339))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := a.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github inbox: status %d", resp.StatusCode)
	}
	var rows []struct {
		Number        int    `json:"number"`
		Title         string `json:"title"`
		Body          string `json:"body"`
		HTMLURL       string `json:"html_url"`
		RepositoryURL string `json:"repository_url"`
		UpdatedAt     string `json:"updated_at"`
		State         string `json:"state"`
		Labels        []struct {
			Name string `json:"name"`
		} `json:"labels"`
		Assignee *struct {
			Login string `json:"login"`
		} `json:"assignee"`
		User *struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]integration.WorkItem, 0, len(rows))
	for _, row := range rows {
		labels := make([]string, 0, len(row.Labels))
		for _, l := range row.Labels {
			labels = append(labels, l.Name)
		}
		updated, _ := time.Parse(time.RFC3339, row.UpdatedAt)
		assignee := ""
		if row.Assignee != nil {
			assignee = row.Assignee.Login
		}
		author := ""
		if row.User != nil {
			author = row.User.Login
		}
		out = append(out, integration.WorkItem{
			Ref: domain.WorkRef{
				Provider: "github", Instance: repoFromURL(row.RepositoryURL), Type: "issue",
				ID: strconv.Itoa(row.Number), URL: row.HTMLURL, Revision: row.UpdatedAt,
			},
			Title: row.Title, Body: row.Body, Labels: labels, Author: author, Assignee: assignee, Updated: updated,
			Status: row.State, StatusGroup: map[string]string{"closed": "completed", "open": "unstarted"}[row.State],
		})
	}
	return out, nil
}

func repoFromURL(raw string) string {
	if i := strings.Index(raw, "/repos/"); i >= 0 {
		return strings.Trim(raw[i+len("/repos/"):], "/")
	}
	return raw
}
