package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

type pullRow struct {
	Number            int    `json:"number"`
	HTMLURL           string `json:"html_url"`
	Title             string `json:"title"`
	State             string `json:"state"`
	Body              string `json:"body"`
	AuthorAssociation string `json:"author_association"`
	Head              struct {
		Ref string `json:"ref"`
	} `json:"head"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

// ListPullRequests implements integration.PullRequestReader.
func (a *Adapter) ListPullRequests(ctx context.Context, cred integration.Credential, repo string) ([]integration.PullRequest, error) {
	if cred.AccessToken == "" {
		return nil, fmt.Errorf("github: access token is required")
	}
	endpoint := fmt.Sprintf("%s/repos/%s/pulls?state=all&per_page=100", a.baseURLFor(cred), strings.TrimSpace(repo))
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
		return nil, fmt.Errorf("github pulls: status %d", resp.StatusCode)
	}
	var rows []pullRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]integration.PullRequest, 0, len(rows))
	for _, row := range rows {
		out = append(out, integration.PullRequest{
			Number: row.Number, URL: row.HTMLURL, Title: row.Title, Body: row.Body,
			State: row.State, Branch: row.Head.Ref, Author: row.User.Login, AuthorAssociation: row.AuthorAssociation,
		})
	}
	return out, nil
}
