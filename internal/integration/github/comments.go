package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

type commentRow struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

func (a *Adapter) ListComments(ctx context.Context, cred integration.Credential, ref domain.WorkRef) ([]integration.Comment, error) {
	if cred.AccessToken == "" {
		return nil, fmt.Errorf("github: missing token")
	}
	endpoint := fmt.Sprintf("%s/repos/%s/issues/%s/comments?per_page=100", a.baseURLFor(cred), ref.Instance, ref.ID)
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
	if resp.StatusCode == 404 {
		return nil, nil
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github comments: status %d", resp.StatusCode)
	}
	var rows []commentRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]integration.Comment, 0, len(rows))
	for _, row := range rows {
		out = append(out, integration.Comment{
			ID:          fmt.Sprintf("%d", row.ID),
			Author:      row.User.Login,
			Body:        row.Body,
			Attachments: attachmentsFromText(row.Body),
			CreatedAt:   row.CreatedAt,
		})
	}
	return out, nil
}
