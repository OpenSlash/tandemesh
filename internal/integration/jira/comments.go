package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func (a *Adapter) ListComments(ctx context.Context, cred integration.Credential, ref domain.WorkRef) ([]integration.Comment, error) {
	instance := strings.TrimRight(ref.Instance, "/")
	if instance == "" {
		return nil, fmt.Errorf("jira: instance is required")
	}
	endpoint := fmt.Sprintf("%s/rest/api/2/issue/%s/comment?maxResults=100", instance, url.PathEscape(ref.ID))
	resp, err := a.do(ctx, http.MethodGet, endpoint, cred, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, nil
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jira comments: status %d", resp.StatusCode)
	}
	var out struct {
		Comments []struct {
			ID      string          `json:"id"`
			Body    json.RawMessage `json:"body"`
			Created string          `json:"created"`
			Author  *struct {
				DisplayName string `json:"displayName"`
			} `json:"author"`
		} `json:"comments"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	result := make([]integration.Comment, 0, len(out.Comments))
	for _, row := range out.Comments {
		body := string(row.Body)
		var text string
		if err := json.Unmarshal(row.Body, &text); err == nil {
			body = text
		} else if markdown := adfToMarkdown(row.Body); markdown != "" {
			body = markdown
		}
		author := ""
		if row.Author != nil {
			author = row.Author.DisplayName
		}
		created, _ := time.Parse("2006-01-02T15:04:05.000-0700", row.Created)
		result = append(result, integration.Comment{ID: row.ID, Author: author, Body: body, CreatedAt: created})
	}
	return result, nil
}
