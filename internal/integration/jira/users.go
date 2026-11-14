package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// SearchUsers implements integration.UserSearcher.
func (a *Adapter) SearchUsers(ctx context.Context, cred integration.Credential, query string, limit int) ([]integration.User, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	base := strings.TrimRight(cred.BaseURL, "/")
	if base == "" {
		return nil, fmt.Errorf("jira: site URL is required")
	}
	endpoint := base + "/rest/api/2/user/search?query=" + url.QueryEscape(query) + "&maxResults=" + strconv.Itoa(limit)
	resp, err := a.do(ctx, http.MethodGet, endpoint, cred, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jira users: status %d", resp.StatusCode)
	}
	var rows []struct {
		AccountID   string `json:"accountId"`
		DisplayName string `json:"displayName"`
		Email       string `json:"emailAddress"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]integration.User, 0, len(rows))
	for _, row := range rows {
		out = append(out, integration.User{
			ID: row.AccountID, Login: row.DisplayName, Name: row.DisplayName, Email: row.Email,
		})
	}
	return out, nil
}
