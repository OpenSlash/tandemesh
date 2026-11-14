package api

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/store"
)

type gitlabMergePayload struct {
	User struct {
		Name string `json:"name"`
	} `json:"user"`
	Project struct {
		PathWithNamespace string `json:"path_with_namespace"`
	} `json:"project"`
	ObjectAttributes struct {
		IID          int    `json:"iid"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		State        string `json:"state"`
		URL          string `json:"url"`
		SourceBranch string `json:"source_branch"`
		UpdatedAt    string `json:"updated_at"`
	} `json:"object_attributes"`
}

// linkGitLabMergeRequest projects a merge request onto matching tasks without
// creating a task or run.
func (s *Server) linkGitLabMergeRequest(ctx context.Context, raw []byte, workspaceID string) int {
	var payload gitlabMergePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return 0
	}
	project := payload.Project.PathWithNamespace
	if project == "" || payload.ObjectAttributes.IID == 0 {
		return 0
	}
	tasks, err := s.store.ListTasks(ctx, store.TaskFilter{
		Workspace: workspaceID, Provider: "gitlab", Instance: project, Limit: 200,
	})
	if err != nil {
		return 0
	}
	linked := 0
	for _, task := range tasks {
		if !taskKeyInText(task, payload.ObjectAttributes.Title+" "+payload.ObjectAttributes.Description+" "+payload.ObjectAttributes.SourceBranch) {
			continue
		}
		item := &domain.TaskPullRequest{
			ID: "pr_" + ulid.Make().String(), TaskID: task.ID, WorkspaceID: task.WorkspaceID,
			Provider: "gitlab", Repo: project, Number: payload.ObjectAttributes.IID,
			URL: payload.ObjectAttributes.URL, Title: payload.ObjectAttributes.Title,
			State: payload.ObjectAttributes.State, Branch: payload.ObjectAttributes.SourceBranch,
			Author: payload.User.Name,
		}
		previousState := ""
		if existing, _ := s.store.ListTaskPullRequests(ctx, task.ID); existing != nil {
			for _, current := range existing {
				if current.Repo == item.Repo && current.Number == item.Number {
					previousState = current.State
				}
			}
		}
		if err := s.store.UpsertTaskPullRequest(ctx, item); err == nil {
			if previousState != item.State {
				s.recordTaskEvent(ctx, task.ID, "task.pull_request", map[string]any{
					"provider": "gitlab", "number": item.Number, "state": item.State, "url": item.URL,
				})
			}
			s.backlinkTaskPullRequest(ctx, task, item)
			linked++
		}
		if payload.User.Name != "" {
			_, _ = s.store.EnsureExternalIdentity(ctx, task.WorkspaceID, "gitlab", payload.User.Name, payload.User.Name, "")
		}
	}
	return linked
}

type gitlabPipelinePayload struct {
	User struct {
		Name string `json:"name"`
	} `json:"user"`
	Project struct {
		PathWithNamespace string `json:"path_with_namespace"`
	} `json:"project"`
	ObjectAttributes struct {
		ID     int    `json:"id"`
		Ref    string `json:"ref"`
		Status string `json:"status"`
		URL    string `json:"url"`
		SHA    string `json:"sha"`
	} `json:"object_attributes"`
	Commit struct {
		Message string `json:"message"`
		Title   string `json:"title"`
	} `json:"commit"`
}

// linkGitLabPipeline records pipeline status on tasks referenced by the ref or
// commit message.
func (s *Server) linkGitLabPipeline(ctx context.Context, raw []byte, workspaceID string) int {
	var payload gitlabPipelinePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return 0
	}
	project := payload.Project.PathWithNamespace
	if project == "" {
		return 0
	}
	tasks, err := s.store.ListTasks(ctx, store.TaskFilter{
		Workspace: workspaceID, Provider: "gitlab", Instance: project, Limit: 200,
	})
	if err != nil {
		return 0
	}
	text := payload.ObjectAttributes.Ref + " " + payload.Commit.Message + " " + payload.Commit.Title
	linked := 0
	for _, task := range tasks {
		if !taskKeyInText(task, text) {
			continue
		}
		s.recordTaskEvent(ctx, task.ID, "task.pipeline", map[string]any{
			"status": payload.ObjectAttributes.Status, "ref": payload.ObjectAttributes.Ref,
			"sha": payload.ObjectAttributes.SHA, "url": payload.ObjectAttributes.URL,
		})
		linked++
	}
	return linked
}

// enqueuePipelineFollowUp creates one repair run per matching task when a
// pipeline fails and the automation uses the pipeline trigger.
func (s *Server) enqueuePipelineFollowUp(ctx context.Context, raw []byte, automation *domain.Automation) int {
	if automation == nil || automation.TriggerKind != "pipeline" || !automation.Enabled {
		return 0
	}
	var payload gitlabPipelinePayload
	if err := json.Unmarshal(raw, &payload); err != nil || payload.ObjectAttributes.Status != "failed" {
		return 0
	}
	project := payload.Project.PathWithNamespace
	if project == "" {
		return 0
	}
	tasks, err := s.store.ListTasks(ctx, store.TaskFilter{
		Workspace: automation.WorkspaceID, Provider: "gitlab", Instance: project, Limit: 200,
	})
	if err != nil {
		return 0
	}
	text := payload.ObjectAttributes.Ref + " " + payload.Commit.Message
	queued := 0
	for _, task := range tasks {
		if !taskKeyInText(task, text) {
			continue
		}
		events, _ := s.store.ListTaskEvents(ctx, task.ID)
		if !pipelineFollowUpAllowed(events, automation, task, payload.ObjectAttributes.Ref) {
			continue
		}
		key := "gitlab:pipeline:" + project + ":" + strconv.Itoa(payload.ObjectAttributes.ID) + ":" + payload.ObjectAttributes.SHA
		claimed, err := s.store.ClaimIdempotency(ctx, key, task.ID)
		if err != nil || !claimed {
			continue
		}
		agentID := automation.AgentID
		if agentID == "" {
			agentID = "claude"
		}
		runID := "run_" + ulid.Make().String()
		run := &domain.Run{
			ID: runID, WorkspaceID: task.WorkspaceID, Work: domain.WorkRef{
				Provider: task.Provider, Instance: task.Instance, Type: task.WorkType,
				ID: task.WorkID, URL: task.URL, Revision: task.Revision,
			},
			Snapshot:     domain.Snapshot{Title: task.Title, Body: task.Body, Labels: task.Labels},
			AutomationID: automation.ID, AgentID: agentID, Status: domain.RunQueued,
			Priority: automation.Priority, Repo: task.Instance, Base: "main",
			Branch:    "orca/" + runID[len(runID)-8:],
			Prompt:    "GitLab pipeline failed. Investigate and fix it.\n\nPipeline: " + payload.ObjectAttributes.URL + "\nRef: " + payload.ObjectAttributes.Ref + "\nSHA: " + payload.ObjectAttributes.SHA,
			CreatedAt: time.Now().UTC(),
		}
		if task.ID != "" {
			run.TaskID = task.ID
		}
		if err := s.store.CreateRun(ctx, run); err != nil {
			continue
		}
		_ = s.store.SetTaskStatus(ctx, task.ID, domain.TaskInProgress)
		s.recordTaskEvent(ctx, task.ID, "task.pipeline_followup", map[string]any{
			"runId": run.ID, "pipeline": payload.ObjectAttributes.URL, "automationId": automation.ID,
		})
		queued++
	}
	return queued
}

func pipelineOptionInt(options map[string]any, key string, fallback int) int {
	if options == nil {
		return fallback
	}
	switch value := options[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	case string:
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func pipelineOptionString(options map[string]any, key string) string {
	if options == nil {
		return ""
	}
	value, _ := options[key].(string)
	return value
}

func pipelineFollowUpAllowed(events []*domain.TaskEvent, automation *domain.Automation, task *domain.Task, ref string) bool {
	maxRetries := pipelineOptionInt(automation.Writeback, "pipelineMaxRetries", 3)
	if maxRetries < 1 {
		maxRetries = 1
	}
	count := 0
	var last time.Time
	for _, event := range events {
		if event.Type != "task.pipeline_followup" {
			continue
		}
		if owner, _ := event.Payload["automationId"].(string); owner != "" && owner != automation.ID {
			continue
		}
		count++
		if event.CreatedAt.After(last) {
			last = event.CreatedAt
		}
	}
	if count >= maxRetries {
		return false
	}
	cooldown := pipelineOptionInt(automation.Writeback, "pipelineCooldownSeconds", 0)
	if cooldown > 0 && !last.IsZero() && time.Since(last) < time.Duration(cooldown)*time.Second {
		return false
	}
	branches := pipelineOptionString(automation.Writeback, "pipelineBranches")
	if branches != "" {
		matched := false
		for _, pattern := range strings.Split(branches, ",") {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				continue
			}
			if pattern == ref {
				matched = true
				break
			}
			if ok, _ := filepath.Match(pattern, ref); ok {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	labels := pipelineOptionString(automation.Writeback, "pipelineLabels")
	if labels != "" {
		matched := false
		for _, label := range strings.Split(labels, ",") {
			label = strings.TrimSpace(label)
			for _, have := range task.Labels {
				if label != "" && strings.EqualFold(label, have) {
					matched = true
				}
			}
		}
		if !matched {
			return false
		}
	}
	return true
}
