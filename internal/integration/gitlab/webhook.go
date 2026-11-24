package gitlab

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Verify implements integration.Webhook.
func (a *Adapter) Verify(r *http.Request, body []byte) error {
	if a.Secret == "" {
		return nil
	}
	return a.VerifyWithSecret(r, body, a.Secret)
}

// VerifyWithSecret checks the X-Gitlab-Token header.
func (a *Adapter) VerifyWithSecret(r *http.Request, body []byte, secret string) error {
	token := r.Header.Get("X-Gitlab-Token")
	if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
		return errors.New("gitlab: invalid webhook token")
	}
	return nil
}

type gitlabEvent struct {
	ObjectKind string `json:"object_kind"`
	User       struct {
		Name string `json:"name"`
	} `json:"user"`
	Project struct {
		PathWithNamespace string `json:"path_with_namespace"`
		WebURL            string `json:"web_url"`
	} `json:"project"`
	ObjectAttributes struct {
		IID          int    `json:"iid"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		State        string `json:"state"`
		URL          string `json:"url"`
		UpdatedAt    string `json:"updated_at"`
		Action       string `json:"action"`
		Note         string `json:"note"`
		NoteableType string `json:"noteable_type"`
		NoteableIID  int    `json:"noteable_iid"`
		SourceBranch string `json:"source_branch"`
		Ref          string `json:"ref"`
		Status       string `json:"status"`
		SHA          string `json:"sha"`
	} `json:"object_attributes"`
	Issue *struct {
		IID         int    `json:"iid"`
		Title       string `json:"title"`
		Description string `json:"description"`
		State       string `json:"state"`
		URL         string `json:"url"`
		UpdatedAt   string `json:"updated_at"`
		Author      struct {
			Name string `json:"name"`
		} `json:"author"`
	} `json:"issue"`
	Commit struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Title   string `json:"title"`
	} `json:"commit"`
	MergeRequest *struct {
		IID          int    `json:"iid"`
		SourceBranch string `json:"source_branch"`
	} `json:"merge_request"`
	Labels    []string `json:"labels"`
	Assignees []struct {
		Name string `json:"name"`
	} `json:"assignees"`
}

// Parse implements integration.Webhook.
func (a *Adapter) Parse(body []byte) (integration.Event, error) {
	var p gitlabEvent
	if err := json.Unmarshal(body, &p); err != nil {
		return integration.Event{}, fmt.Errorf("gitlab: decode payload: %w", err)
	}
	if p.Project.PathWithNamespace == "" {
		return integration.Event{}, errors.New("gitlab: missing project")
	}
	kind := p.ObjectKind
	text := ""
	if kind == "note" {
		if p.ObjectAttributes.NoteableType != "Issue" {
			return integration.Event{Kind: "skip"}, nil
		}
		kind = "comment"
		text = p.ObjectAttributes.Note
	}
	if kind == "pipeline" {
		return integration.Event{Delivery: p.ObjectAttributes.URL + ":" + p.ObjectAttributes.SHA, Kind: "pipeline", Actor: p.User.Name, Raw: body}, nil
	}
	if kind == "merge_request" {
		return integration.Event{Delivery: p.ObjectAttributes.URL + ":" + p.ObjectAttributes.UpdatedAt, Kind: "merge_request", Actor: p.User.Name, Raw: body}, nil
	}
	if kind != "issue" && kind != "comment" {
		return integration.Event{Kind: "skip"}, nil
	}
	if kind == "issue" && p.ObjectAttributes.Action != "" {
		kind = p.ObjectAttributes.Action
	}
	labels := p.Labels
	if labels == nil {
		labels = []string{}
	}
	assignee := ""
	if len(p.Assignees) > 0 {
		assignee = p.Assignees[0].Name
	}
	description := p.ObjectAttributes.Description
	if p.Issue != nil {
		description = p.Issue.Description
	}
	return integration.Event{
		Delivery:    p.ObjectAttributes.URL + ":" + p.ObjectAttributes.UpdatedAt,
		Kind:        kind,
		Actor:       p.User.Name,
		Labels:      labels,
		Assignee:    assignee,
		Attachments: attachmentsFromDescription(description, p.Project.WebURL),
		Text:        text,
		Raw:         body,
	}, nil
}

// Normalize implements integration.Webhook.
func (a *Adapter) Normalize(ev integration.Event) (domain.WorkRef, domain.Snapshot, string, string, error) {
	var p gitlabEvent
	if err := json.Unmarshal(ev.Raw, &p); err != nil {
		return domain.WorkRef{}, domain.Snapshot{}, "", "", err
	}
	issue := p.Issue
	iid := p.ObjectAttributes.IID
	title, body, state, url, updated := p.ObjectAttributes.Title, p.ObjectAttributes.Description,
		p.ObjectAttributes.State, p.ObjectAttributes.URL, p.ObjectAttributes.UpdatedAt
	if issue != nil {
		iid = issue.IID
		title, body, state, url, updated = issue.Title, issue.Description, issue.State, issue.URL, issue.UpdatedAt
	}
	if iid == 0 {
		return domain.WorkRef{}, domain.Snapshot{}, "", "SKIP_EVENT_ACTION", nil
	}
	work := domain.WorkRef{
		Provider: "gitlab", Instance: p.Project.PathWithNamespace, Type: "issue",
		ID: strconv.Itoa(iid), URL: url, Revision: updated,
	}
	snap := domain.Snapshot{
		Title: title, Body: body, Labels: ev.Labels,
		Author: ev.Actor, Assignee: ev.Assignee, Status: state,
		StatusGroup: issueGroup(state),
	}
	if ev.Kind == "comment" {
		snap.BodyHash = hash(ev.Text)
		return work, snap, "评论：" + ev.Text, "", nil
	}
	if ev.Kind == "close" || ev.Kind == "closed" {
		snap.Status = "closed"
		snap.StatusGroup = "completed"
	}
	if ev.Kind == "reopen" || ev.Kind == "reopened" {
		snap.Status = "opened"
		snap.StatusGroup = "started"
	}
	prompt := "Fix the following GitLab issue.\n\nIID: " + work.ID + "\nTitle: " + title + "\nURL: " + url
	return work, snap, prompt, "", nil
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return fmt.Sprintf("%x", sum[:12])
}
