package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type pullRequestCommentRequest struct {
	Body string `json:"body"`
}

// addPullRequestComment writes a comment to a linked pull request.
func (s *Server) addPullRequestComment(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	pr, err := s.store.GetTaskPullRequest(r.Context(), chi.URLParam(r, "pr"))
	if err != nil || pr == nil || pr.TaskID != task.ID {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "pull request not found")
		return
	}
	var req pullRequestCommentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "body is required")
		return
	}
	ad, ok := s.integrations.Get(task.Provider)
	if !ok || ad.Work == nil {
		writeErr(w, http.StatusBadRequest, "E_UNSUPPORTED", "provider does not support comments")
		return
	}
	connectionID := ""
	if conns, err := s.store.ListConnections(r.Context(), task.WorkspaceID, task.Provider); err == nil {
		for _, conn := range conns {
			if domain.InstanceCovers(conn.Instance, task.Instance) {
				connectionID = conn.ID
				break
			}
		}
	}
	cred := s.providerCredential(r.Context(), task.Provider, connectionID)
	if personal, ok := s.memberCredential(r.Context(), task.WorkspaceID, task.Provider); ok {
		cred = personal
	}
	ref := domain.WorkRef{
		Provider: task.Provider, Instance: pr.Repo, Type: "pull_request",
		ID: strconv.Itoa(pr.Number), URL: pr.URL,
	}
	url, err := ad.Work.Comment(r.Context(), ref, cred, req.Body)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_COMMENT", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"url": url})
}
