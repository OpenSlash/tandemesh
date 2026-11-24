package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// ListAssigned implements integration.Reader. Instance must be the Jira base
// URL, e.g. https://acme.atlassian.net.
func (a *Adapter) ListAssigned(ctx context.Context, cred integration.Credential, opts integration.ListOptions) ([]integration.WorkItem, error) {
	instance := strings.TrimRight(opts.Instance, "/")
	if instance == "" {
		return nil, fmt.Errorf("jira: instance is required")
	}
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	jql := url.QueryEscape("assignee = currentUser() AND resolution = Unresolved ORDER BY updated DESC")
	fieldList := "summary,labels,assignee,updated,status,priority,components,fixVersions,attachment"
	for id := range opts.Fields {
		fieldList += "," + id
	}
	endpoint := fmt.Sprintf("%s/rest/api/2/search?jql=%s&maxResults=%d&fields=%s", instance, jql, limit, fieldList)
	resp, err := a.do(ctx, http.MethodGet, endpoint, cred, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jira inbox: status %d", resp.StatusCode)
	}
	var out struct {
		Issues []struct {
			Key    string `json:"key"`
			Fields struct {
				Summary  string   `json:"summary"`
				Labels   []string `json:"labels"`
				Priority *struct {
					Name string `json:"name"`
				} `json:"priority"`
				Components []struct {
					Name string `json:"name"`
				} `json:"components"`
				FixVersions []struct {
					Name string `json:"name"`
				} `json:"fixVersions"`
				Attachment []struct {
					ID       string `json:"id"`
					Filename string `json:"filename"`
					Content  string `json:"content"`
					MimeType string `json:"mimeType"`
					Size     int64  `json:"size"`
					Author   struct {
						DisplayName string `json:"displayName"`
					} `json:"author"`
				} `json:"attachment"`
				Updated string `json:"updated"`
				Status  *struct {
					Name           string `json:"name"`
					StatusCategory struct {
						Key string `json:"key"`
					} `json:"statusCategory"`
				} `json:"status"`
				Assignee *struct {
					DisplayName string `json:"displayName"`
				} `json:"assignee"`
			} `json:"fields"`
		} `json:"issues"`
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	var raw struct {
		Issues []struct {
			Key    string                     `json:"key"`
			Fields map[string]json.RawMessage `json:"fields"`
		} `json:"issues"`
	}
	_ = json.Unmarshal(body, &raw)
	custom := map[string]map[string][]string{}
	for _, issue := range raw.Issues {
		values := map[string][]string{}
		for id, name := range opts.Fields {
			if rawValue, ok := issue.Fields[id]; ok {
				if parsed := jiraRawValues(rawValue); len(parsed) > 0 {
					values[name] = parsed
				}
			}
		}
		custom[issue.Key] = values
	}
	items := make([]integration.WorkItem, 0, len(out.Issues))
	for _, issue := range out.Issues {
		assignee := ""
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}
		updated, _ := time.Parse("2006-01-02T15:04:05.000-0700", issue.Fields.Updated)
		status, group := "", ""
		if issue.Fields.Status != nil {
			status = issue.Fields.Status.Name
			switch issue.Fields.Status.StatusCategory.Key {
			case "new":
				group = "unstarted"
			case "indeterminate":
				group = "started"
			case "done":
				group = "completed"
			}
		}
		priority := ""
		if issue.Fields.Priority != nil {
			priority = issue.Fields.Priority.Name
		}
		components := make([]string, 0, len(issue.Fields.Components))
		for _, component := range issue.Fields.Components {
			components = append(components, component.Name)
		}
		versions := make([]string, 0, len(issue.Fields.FixVersions))
		for _, version := range issue.Fields.FixVersions {
			versions = append(versions, version.Name)
		}
		attachments := make([]integration.Attachment, 0, len(issue.Fields.Attachment))
		for _, attachment := range issue.Fields.Attachment {
			attachments = append(attachments, integration.Attachment{
				ID: attachment.ID, Name: attachment.Filename, URL: attachment.Content,
				Mime: attachment.MimeType, Size: attachment.Size, Author: attachment.Author.DisplayName,
			})
		}
		fields := jiraFields(priority, components, versions)
		for key, values := range custom[issue.Key] {
			fields[key] = values
		}
		items = append(items, integration.WorkItem{
			Ref: domain.WorkRef{
				Provider: "jira", Instance: instance, Type: "issue",
				ID: issue.Key, URL: instance + "/browse/" + issue.Key, Revision: issue.Fields.Updated,
			},
			Title: issue.Fields.Summary, Labels: projectFields(issue.Fields.Labels, priority, components, versions),
			Fields:   fields,
			Assignee: assignee, Attachments: attachments, Updated: updated, Status: status, StatusGroup: group,
		})
	}
	return items, nil
}
