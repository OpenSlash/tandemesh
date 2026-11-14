package plane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// ListAssigned implements integration.Reader. Plane has no assignee filter on
// the legacy list endpoint, so Orca reads the project's work items.
func (a *Adapter) ListAssigned(ctx context.Context, cred integration.Credential, opts integration.ListOptions) ([]integration.WorkItem, error) {
	slug, project, err := splitInstance(opts.Instance)
	if err != nil {
		return nil, err
	}
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	paths := []string{
		fmt.Sprintf("/workspaces/%s/projects/%s/work-items/?per_page=%d", url.PathEscape(slug), url.PathEscape(project), limit),
		fmt.Sprintf("/workspaces/%s/projects/%s/issues/?per_page=%d", url.PathEscape(slug), url.PathEscape(project), limit),
	}
	var rows []issue
	for _, path := range paths {
		raw, status, err := a.do(ctx, "GET", a.apiAt(cred, path), cred, nil)
		if err != nil {
			if status == 404 {
				continue
			}
			return nil, err
		}
		rows, err = decodeIssues(raw)
		if err != nil {
			return nil, err
		}
		break
	}
	labels := a.labelIndex(ctx, cred, slug, project)
	members := a.memberIndex(ctx, cred, slug)
	states, _ := a.ListStates(ctx, cred, slug+"/"+project)
	stateByID := map[string]integration.State{}
	for _, state := range states {
		stateByID[state.ID] = state
	}
	out := make([]integration.WorkItem, 0, len(rows))
	for _, row := range rows {
		updated, _ := time.Parse(time.RFC3339, row.UpdatedAt)
		body := row.Description
		if body == "" {
			body = stripHTML(row.DescriptionHTML)
		}
		out = append(out, integration.WorkItem{
			Ref: domain.WorkRef{
				Provider: "plane", Instance: slug + "/" + project, Type: "issue",
				ID: row.ID, URL: a.issueURL(slug, project, row), Revision: row.UpdatedAt,
			},
			Title: row.Name, Body: body, Labels: mappedLabels(row, labels),
			Author: mappedAuthor(row, members), Assignee: mappedAssignee(row, members), Updated: updated,
			Status: stateByID[issueStateID(row.State)].Name, StatusGroup: stateByID[issueStateID(row.State)].Group,
		})
	}
	return out, nil
}

func issueStateID(raw json.RawMessage) string {
	var id string
	if err := json.Unmarshal(raw, &id); err == nil {
		return id
	}
	var obj struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(raw, &obj)
	return obj.ID
}

func decodeIssues(raw []byte) ([]issue, error) {
	var list []issue
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, nil
	}
	var page struct {
		Results []issue `json:"results"`
	}
	if err := json.Unmarshal(raw, &page); err != nil {
		return nil, fmt.Errorf("plane: decode issues: %w", err)
	}
	return page.Results, nil
}

// Probe implements integration.Prober.
func (a *Adapter) Probe(ctx context.Context, cred integration.Credential) error {
	_, _, err := a.do(ctx, "GET", a.apiAt(cred, "/users/me/"), cred, nil)
	return err
}

func (a *Adapter) labelIndex(ctx context.Context, cred integration.Credential, slug, project string) map[string]string {
	path := fmt.Sprintf("/workspaces/%s/projects/%s/labels/", url.PathEscape(slug), url.PathEscape(project))
	raw, _, err := a.do(ctx, "GET", a.apiAt(cred, path), cred, nil)
	if err != nil {
		return nil
	}
	var items []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		var page struct {
			Results []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"results"`
		}
		if json.Unmarshal(raw, &page) != nil {
			return nil
		}
		items = page.Results
	}
	out := map[string]string{}
	for _, item := range items {
		if item.ID != "" && item.Name != "" {
			out[item.ID] = item.Name
		}
	}
	return out
}

func (a *Adapter) memberIndex(ctx context.Context, cred integration.Credential, slug string) map[string]string {
	path := fmt.Sprintf("/workspaces/%s/members/", url.PathEscape(slug))
	raw, _, err := a.do(ctx, "GET", a.apiAt(cred, path), cred, nil)
	if err != nil {
		return nil
	}
	var items []struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		var page struct {
			Results []struct {
				ID          string `json:"id"`
				DisplayName string `json:"display_name"`
				Email       string `json:"email"`
			} `json:"results"`
		}
		if json.Unmarshal(raw, &page) != nil {
			return nil
		}
		items = page.Results
	}
	out := map[string]string{}
	for _, item := range items {
		name := item.DisplayName
		if name == "" {
			name = item.Email
		}
		if item.ID != "" && name != "" {
			out[item.ID] = name
		}
	}
	return out
}

func mappedLabels(row issue, index map[string]string) []string {
	if len(index) == 0 {
		return labelNames(row)
	}
	out := []string{}
	for _, raw := range row.Labels {
		if id, ok := raw.(string); ok {
			if name, ok := index[id]; ok {
				out = append(out, name)
			}
		}
	}
	if len(out) == 0 {
		return labelNames(row)
	}
	return out
}

func mappedAssignee(row issue, index map[string]string) string {
	if len(index) > 0 {
		for _, raw := range row.Assignees {
			if id, ok := raw.(string); ok {
				if name, ok := index[id]; ok {
					return name
				}
			}
		}
	}
	return assigneeName(row)
}

func mappedAuthor(row issue, index map[string]string) string {
	if len(index) == 0 || row.CreatedBy == "" {
		return ""
	}
	return index[row.CreatedBy]
}
