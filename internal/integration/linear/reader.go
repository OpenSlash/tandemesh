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

// ListAssigned implements integration.Reader.
func (a *Adapter) ListAssigned(ctx context.Context, cred integration.Credential, opts integration.ListOptions) ([]integration.WorkItem, error) {
	if cred.AccessToken == "" {
		return nil, fmt.Errorf("linear: missing token")
	}
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	query := `query($first: Int!) {
	  viewer {
	    assignedIssues(first: $first) {
	      nodes { id identifier title description url updatedAt state { name } assignee { name } labels { nodes { name } } }
	    }
	  }
	}`
	body, _ := json.Marshal(map[string]any{"query": query, "variables": map[string]any{"first": limit}})
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
		return nil, fmt.Errorf("linear inbox: status %d", resp.StatusCode)
	}
	var out struct {
		Data struct {
			Viewer struct {
				AssignedIssues struct {
					Nodes []struct {
						ID          string `json:"id"`
						Identifier  string `json:"identifier"`
						Title       string `json:"title"`
						Description string `json:"description"`
						URL         string `json:"url"`
						UpdatedAt   string `json:"updatedAt"`
						State       *struct {
							Name string `json:"name"`
						} `json:"state"`
						Assignee *struct {
							Name string `json:"name"`
						} `json:"assignee"`
						Labels struct {
							Nodes []struct {
								Name string `json:"name"`
							} `json:"nodes"`
						} `json:"labels"`
					} `json:"nodes"`
				} `json:"assignedIssues"`
			} `json:"viewer"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("linear inbox: %s", out.Errors[0].Message)
	}
	items := make([]integration.WorkItem, 0, len(out.Data.Viewer.AssignedIssues.Nodes))
	for _, node := range out.Data.Viewer.AssignedIssues.Nodes {
		labels := make([]string, 0, len(node.Labels.Nodes))
		for _, l := range node.Labels.Nodes {
			labels = append(labels, l.Name)
		}
		assignee := ""
		if node.Assignee != nil {
			assignee = node.Assignee.Name
		}
		updated, _ := time.Parse(time.RFC3339, node.UpdatedAt)
		status := ""
		if node.State != nil {
			status = node.State.Name
		}
		items = append(items, integration.WorkItem{
			Ref: domain.WorkRef{
				Provider: "linear", Instance: "linear", Type: "issue",
				ID: node.Identifier, URL: node.URL, Revision: node.UpdatedAt,
			},
			Title: node.Title, Body: node.Description, Labels: labels, Assignee: assignee, Updated: updated, Status: status,
		})
	}
	return items, nil
}
