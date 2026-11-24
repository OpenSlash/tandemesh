package api

import (
	"context"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

type githubAppPRPayload struct {
	Action     string `json:"action"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	PullRequest struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
		Title   string `json:"title"`
		Body    string `json:"body"`
		State   string `json:"state"`
		Merged  bool   `json:"merged"`
		Head    struct {
			Ref string `json:"ref"`
		} `json:"head"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		AuthorAssociation string `json:"author_association"`
	} `json:"pull_request"`
}

type githubAppPushPayload struct {
	Ref        string `json:"ref"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Commits []struct {
		ID      string `json:"id"`
		URL     string `json:"url"`
		Message string `json:"message"`
		Author  struct {
			Name     string `json:"name"`
			Username string `json:"username"`
		} `json:"author"`
	} `json:"commits"`
	HeadCommit *struct {
		ID      string `json:"id"`
		URL     string `json:"url"`
		Message string `json:"message"`
		Author  struct {
			Name     string `json:"name"`
			Username string `json:"username"`
		} `json:"author"`
	} `json:"head_commit"`
}

func (s *Server) linkGitHubPullRequestEvent(ctx context.Context, payload githubAppPRPayload) int {
	if payload.Repository.FullName == "" || payload.PullRequest.Number == 0 {
		return 0
	}
	binding, _ := s.store.FindBinding(ctx, "github", payload.Repository.FullName)
	if binding == nil {
		return 0
	}
	tasks, err := s.store.ListTasks(ctx, store.TaskFilter{
		Workspace: binding.WorkspaceID, Provider: "github", Instance: payload.Repository.FullName, Limit: 200,
	})
	if err != nil {
		return 0
	}
	state := payload.PullRequest.State
	if payload.PullRequest.Merged {
		state = "merged"
	}
	pr := integration.PullRequest{
		Number: payload.PullRequest.Number, URL: payload.PullRequest.HTMLURL,
		Title: payload.PullRequest.Title, Body: payload.PullRequest.Body,
		State: state, Branch: payload.PullRequest.Head.Ref,
		Author: payload.PullRequest.User.Login, AuthorAssociation: payload.PullRequest.AuthorAssociation,
	}
	linked := 0
	for _, task := range tasks {
		if !trustedPullRequest(binding, pr) || !taskKeyMatches(task, pr) {
			continue
		}
		item := &domain.TaskPullRequest{
			ID: "pr_" + ulid.Make().String(), TaskID: task.ID, WorkspaceID: task.WorkspaceID,
			Provider: "github", Repo: payload.Repository.FullName, Number: pr.Number,
			URL: pr.URL, Title: pr.Title, State: pr.State, Branch: pr.Branch, Author: pr.Author,
		}
		if err := s.store.UpsertTaskPullRequest(ctx, item); err == nil {
			s.backlinkTaskPullRequest(ctx, task, item)
			linked++
		}
		if pr.Author != "" {
			_, _ = s.store.EnsureExternalIdentity(ctx, task.WorkspaceID, "github", pr.Author, pr.Author, "")
		}
	}
	return linked
}

func (s *Server) linkGitHubPushEvent(ctx context.Context, payload githubAppPushPayload) int {
	if payload.Repository.FullName == "" {
		return 0
	}
	binding, _ := s.store.FindBinding(ctx, "github", payload.Repository.FullName)
	if binding == nil {
		return 0
	}
	tasks, err := s.store.ListTasks(ctx, store.TaskFilter{
		Workspace: binding.WorkspaceID, Provider: "github", Instance: payload.Repository.FullName, Limit: 200,
	})
	if err != nil {
		return 0
	}
	commits := payload.Commits
	linked := 0
	for _, commit := range commits {
		for _, task := range tasks {
			if !taskKeyInText(task, commit.Message) {
				continue
			}
			author := commit.Author.Username
			if author == "" {
				author = commit.Author.Name
			}
			item := &domain.TaskCommitLink{
				ID: "commit_" + ulid.Make().String(), TaskID: task.ID, WorkspaceID: task.WorkspaceID,
				Provider: "github", Repo: payload.Repository.FullName, SHA: commit.ID,
				URL: commit.URL, Message: commit.Message, Author: author,
			}
			if err := s.store.UpsertTaskCommit(ctx, item); err == nil {
				linked++
			}
		}
	}
	return linked
}
