package jira

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

// SearchIssues implements integration.ImportReader.
func (a *Adapter) SearchIssues(ctx context.Context, cred integration.Credential, instance, jql string, limit int) ([]integration.WorkItem, error) {
	return a.SearchIssuesPage(ctx, cred, instance, jql, 0, limit)
}

// SearchIssuesPage implements integration.ImportPager.
func (a *Adapter) SearchIssuesPage(ctx context.Context, cred integration.Credential, instance, jql string, startAt, limit int) ([]integration.WorkItem, error) {
	instance = strings.TrimRight(instance, "/")
	if instance == "" {
		return nil, fmt.Errorf("jira: instance is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if strings.TrimSpace(jql) == "" {
		return nil, fmt.Errorf("jira: jql is required")
	}
	endpoint := fmt.Sprintf("%s/rest/api/2/search?jql=%s&startAt=%d&maxResults=%d&fields=summary,labels,assignee,updated,status,priority,components,fixVersions,attachment",
		instance, url.QueryEscape(jql), startAt, limit)
	var resp *http.Response
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err = a.do(ctx, http.MethodGet, endpoint, cred, nil)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			break
		}
		wait := 5 * time.Second
		if raw := resp.Header.Get("Retry-After"); raw != "" {
			if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 && seconds <= 60 {
				wait = time.Duration(seconds) * time.Second
			}
		}
		resp.Body.Close()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jira import: status %d", resp.StatusCode)
	}
	var out struct {
		Issues []struct {
			Key    string `json:"key"`
			Fields struct {
				Summary  string   `json:"summary"`
				Labels   []string `json:"labels"`
				Updated  string   `json:"updated"`
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
				Status *struct {
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
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	items := make([]integration.WorkItem, 0, len(out.Issues))
	for _, issue := range out.Issues {
		assignee := ""
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
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
		updated, _ := time.Parse("2006-01-02T15:04:05.000-0700", issue.Fields.Updated)
		attachments := make([]integration.Attachment, 0, len(issue.Fields.Attachment))
		for _, attachment := range issue.Fields.Attachment {
			attachments = append(attachments, integration.Attachment{
				ID: attachment.ID, Name: attachment.Filename, URL: attachment.Content,
				Mime: attachment.MimeType, Size: attachment.Size, Author: attachment.Author.DisplayName,
			})
		}
		items = append(items, integration.WorkItem{
			Ref: domain.WorkRef{
				Provider: "jira", Instance: instance, Type: "issue",
				ID: issue.Key, URL: instance + "/browse/" + issue.Key, Revision: issue.Fields.Updated,
			},
			Title: issue.Fields.Summary, Labels: projectFields(issue.Fields.Labels, priority, components, versions),
			Fields:   jiraFields(priority, components, versions),
			Assignee: assignee, Attachments: attachments, Updated: updated, Status: status, StatusGroup: group,
		})
	}
	return items, nil
}
