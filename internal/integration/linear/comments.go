package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func (a *Adapter) ListComments(ctx context.Context, cred integration.Credential, ref domain.WorkRef) ([]integration.Comment, error) {
	if cred.AccessToken == "" {
		return nil, fmt.Errorf("linear: missing token")
	}
	query := `query($id: String!, $first: Int!) {
	  issue(id: $id) {
	    comments(first: $first) {
	      nodes { id body createdAt user { name } }
	    }
	  }
	}`
	body, _ := json.Marshal(map[string]any{
		"query":     query,
		"variables": map[string]any{"id": ref.ID, "first": 100},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", cred.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("linear comments: status %d", resp.StatusCode)
	}
	var out struct {
		Data struct {
			Issue struct {
				Comments struct {
					Nodes []struct {
						ID        string    `json:"id"`
						Body      string    `json:"body"`
						CreatedAt time.Time `json:"createdAt"`
						User      *struct {
							Name string `json:"name"`
						} `json:"user"`
					} `json:"nodes"`
				} `json:"comments"`
			} `json:"issue"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("linear comments: %s", out.Errors[0].Message)
	}
	nodes := out.Data.Issue.Comments.Nodes
	result := make([]integration.Comment, 0, len(nodes))
	for _, node := range nodes {
		author := ""
		if node.User != nil {
			author = node.User.Name
		}
		result = append(result, integration.Comment{ID: node.ID, Author: author, Body: node.Body, CreatedAt: node.CreatedAt})
	}
	return result, nil
}
