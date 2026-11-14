package api

import (
	"context"
	"os"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

// PullRequestSyncLoop periodically links PRs and commits to task projections.
func (s *Server) PullRequestSyncLoop(ctx context.Context) {
	interval := 10 * time.Minute
	if raw := os.Getenv("ORCA_PR_SYNC_INTERVAL"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed >= time.Minute {
			interval = parsed
		}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SyncPullRequests(ctx)
		}
	}
}

// SyncPullRequests scans enabled bindings for task-keyed PRs and commits.
func (s *Server) SyncPullRequests(ctx context.Context) int {
	locked, err := s.store.AcquireSyncLock(ctx, "sync:pull-requests", syncOwner(), 30*time.Minute)
	if err != nil || !locked {
		return 0
	}
	defer s.store.ReleaseSyncLock(ctx, "sync:pull-requests", syncOwner())
	bindings, err := s.store.ListAllBindings(ctx)
	if err != nil {
		return 0
	}
	linked := 0
	for _, binding := range bindings {
		if binding.Provider != "github" && binding.Provider != "gitlab" {
			continue
		}
		ad, ok := s.integrations.Get(binding.Provider)
		if !ok {
			continue
		}
		cred := s.providerCredential(ctx, binding.Provider, binding.ConnectionID)
		cred = s.accountCredential(ctx, binding.Provider, binding.AccountID, cred)
		tasks, err := s.store.ListTasks(ctx, store.TaskFilter{
			Workspace: binding.WorkspaceID, Provider: binding.Provider,
			Instance: binding.ExternalID, Limit: 200,
		})
		if err != nil || len(tasks) == 0 {
			continue
		}
		if reader, ok := ad.Reader.(integration.PullRequestReader); ok {
			pulls, err := reader.ListPullRequests(ctx, cred, binding.ExternalID)
			if err == nil {
				for _, pr := range pulls {
					if !trustedPullRequest(binding, pr) {
						continue
					}
					for _, task := range tasks {
						if !taskKeyMatches(task, pr) {
							continue
						}
						link := &domain.TaskPullRequest{
							ID: "pr_" + ulid.Make().String(), TaskID: task.ID, WorkspaceID: task.WorkspaceID,
							Provider: binding.Provider, Repo: binding.ExternalID, Number: pr.Number,
							URL: pr.URL, Title: pr.Title, State: pr.State, Branch: pr.Branch, Author: pr.Author,
						}
						if err := s.store.UpsertTaskPullRequest(ctx, link); err == nil {
							s.backlinkTaskPullRequest(ctx, task, link)
							linked++
						}
						if pr.Author != "" {
							_, _ = s.store.EnsureExternalIdentity(ctx, task.WorkspaceID, binding.Provider, pr.Author, pr.Author, "")
						}
					}
				}
			}
		}
		if reader, ok := ad.Reader.(integration.CommitReader); ok {
			commits, err := reader.ListCommits(ctx, cred, binding.ExternalID, 100)
			if err == nil {
				for _, commit := range commits {
					for _, task := range tasks {
						if !taskKeyInText(task, commit.Message) {
							continue
						}
						link := &domain.TaskCommitLink{
							ID: "commit_" + ulid.Make().String(), TaskID: task.ID, WorkspaceID: task.WorkspaceID,
							Provider: binding.Provider, Repo: binding.ExternalID, SHA: commit.SHA,
							URL: commit.URL, Message: commit.Message, Author: commit.Author,
						}
						if err := s.store.UpsertTaskCommit(ctx, link); err == nil {
							linked++
						}
					}
				}
			}
		}
	}
	return linked
}
