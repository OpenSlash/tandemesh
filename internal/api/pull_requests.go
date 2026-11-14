package api

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func (s *Server) listTaskPullRequests(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	items, err := s.store.ListTaskPullRequests(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.TaskPullRequest{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// syncTaskPullRequests reads pull requests for the task repository and links
// the ones that reference the task key in title, body or branch.
func (s *Server) syncTaskPullRequests(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	ad, ok := s.integrations.Get(task.Provider)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider not registered")
		return
	}
	reader, ok := ad.Reader.(integration.PullRequestReader)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_UNSUPPORTED", "provider does not expose pull requests")
		return
	}
	connectionID := ""
	if conns, err := s.store.ListConnections(r.Context(), task.WorkspaceID, task.Provider); err == nil {
		for _, conn := range conns {
			if conn.Instance == task.Instance {
				connectionID = conn.ID
				break
			}
		}
	}
	cred := s.providerCredential(r.Context(), task.Provider, connectionID)
	pulls, err := reader.ListPullRequests(r.Context(), cred, task.Instance)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_PULLS", err.Error())
		return
	}
	linked := 0
	binding, _ := s.store.FindBinding(r.Context(), task.Provider, task.Instance)
	for _, pr := range pulls {
		if !trustedPullRequest(binding, pr) {
			continue
		}
		if !taskKeyMatches(task, pr) {
			continue
		}
		if pr.Author != "" {
			_, _ = s.store.EnsureExternalIdentity(r.Context(), task.WorkspaceID, task.Provider, pr.Author, pr.Author, "")
		}
		item := &domain.TaskPullRequest{
			ID: "pr_" + ulid.Make().String(), TaskID: task.ID, WorkspaceID: task.WorkspaceID,
			Provider: task.Provider, Repo: task.Instance, Number: pr.Number, URL: pr.URL,
			Title: pr.Title, State: pr.State, Branch: pr.Branch, Author: pr.Author,
		}
		if err := s.store.UpsertTaskPullRequest(r.Context(), item); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		s.backlinkTaskPullRequest(r.Context(), task, item)
		linked++
	}
	items, _ := s.store.ListTaskPullRequests(r.Context(), task.ID)
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "linked": linked})
}

func taskKeyMatches(task *domain.Task, pr integration.PullRequest) bool {
	return taskKeyInText(task, pr.Title+" "+pr.Body+" "+pr.Branch)
}

func taskKeyInText(task *domain.Task, text string) bool {
	key := strings.TrimSpace(task.WorkID)
	if key == "" {
		return false
	}
	haystack := strings.ToLower(text)
	if strings.Contains(key, "-") {
		pattern := `(?i)\b` + regexp.QuoteMeta(key) + `\b`
		return regexp.MustCompile(pattern).MatchString(haystack)
	}
	if _, err := strconv.Atoi(key); err == nil {
		pattern := `(?:#|/)` + regexp.QuoteMeta(key) + `\b`
		return regexp.MustCompile(pattern).MatchString(haystack)
	}
	return strings.Contains(haystack, strings.ToLower(key))
}

func (s *Server) deleteTaskPullRequest(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetTaskPullRequest(r.Context(), chi.URLParam(r, "id"))
	if err != nil || item == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "pull request link not found")
		return
	}
	if !s.requireWorkspace(w, r, item.WorkspaceID) {
		return
	}
	if _, err := s.store.DeleteTaskPullRequest(r.Context(), item.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// backlinkTaskPullRequest posts the task link to the PR when the binding opts in.
func (s *Server) backlinkTaskPullRequest(ctx context.Context, task *domain.Task, pr *domain.TaskPullRequest) {
	if task == nil || pr == nil || task.URL == "" || pr.URL == "" {
		return
	}
	binding, err := s.store.FindBinding(ctx, task.Provider, task.Instance)
	if err != nil || binding == nil || binding.Settings["backlink"] != "true" {
		return
	}
	items, _ := s.store.ListTaskPullRequests(ctx, task.ID)
	for _, existing := range items {
		if existing.Repo == pr.Repo && existing.Number == pr.Number && existing.BacklinkedAt != nil {
			return
		}
	}
	ad, ok := s.integrations.Get(task.Provider)
	if !ok || ad.Work == nil {
		return
	}
	cred := s.providerCredential(ctx, task.Provider, binding.ConnectionID)
	cred = s.accountCredential(ctx, task.Provider, binding.AccountID, cred)
	ref := domain.WorkRef{Provider: task.Provider, Instance: pr.Repo, Type: "pr", ID: strconv.Itoa(pr.Number), URL: pr.URL}
	body := "Orca task: [" + task.Title + "](" + task.URL + ")"
	if _, err := ad.Work.Comment(ctx, ref, cred, body); err != nil {
		s.log.Warn("pr backlink", "task", task.ID, "pr", pr.URL, "err", err)
		return
	}
	now := time.Now().UTC()
	pr.BacklinkedAt = &now
	_ = s.store.UpsertTaskPullRequest(ctx, pr)
}

func trustedPullRequest(binding *domain.ResourceBinding, pr integration.PullRequest) bool {
	if binding == nil || binding.Settings["trustedAuthorsOnly"] != "true" {
		return true
	}
	switch strings.ToLower(pr.AuthorAssociation) {
	case "owner", "member", "collaborator":
		return true
	default:
		return false
	}
}
