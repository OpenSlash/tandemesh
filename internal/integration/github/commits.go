package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

type commitRow struct {
	SHA     string `json:"sha"`
	HTMLURL string `json:"html_url"`
	Commit  struct {
		Message string `json:"message"`
		Author  struct {
			Name string `json:"name"`
		} `json:"author"`
	} `json:"commit"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
}

// ListCommits implements integration.CommitReader.
func (a *Adapter) ListCommits(ctx context.Context, cred integration.Credential, repo string, limit int) ([]integration.Commit, error) {
	return a.ListCommitsForRef(ctx, cred, repo, "", limit)
}

// ListCommitsForRef lists commits for a branch or ref.
func (a *Adapter) ListCommitsForRef(ctx context.Context, cred integration.Credential, repo, ref string, limit int) ([]integration.Commit, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	endpoint := fmt.Sprintf("%s/repos/%s/commits?per_page=%d", a.baseURLFor(cred), strings.TrimSpace(repo), limit)
	if ref != "" {
		endpoint += "&sha=" + url.QueryEscape(ref)
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
		return nil, fmt.Errorf("github commits: status %d", resp.StatusCode)
	}
	var rows []commitRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]integration.Commit, 0, len(rows))
	for _, row := range rows {
		author := row.Author.Login
		if author == "" {
			author = row.Commit.Author.Name
		}
		out = append(out, integration.Commit{
			SHA: row.SHA, URL: row.HTMLURL, Message: row.Commit.Message, Author: author,
		})
	}
	return out, nil
}
