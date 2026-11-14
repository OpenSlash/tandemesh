package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func (s *Server) listTaskCommits(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	items, err := s.store.ListTaskCommits(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.TaskCommitLink{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) syncTaskCommits(w http.ResponseWriter, r *http.Request) {
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
	reader, ok := ad.Reader.(integration.CommitReader)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_UNSUPPORTED", "provider does not expose commits")
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
	commits, err := reader.ListCommits(r.Context(), cred, task.Instance, 50)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_COMMITS", err.Error())
		return
	}
	linked := 0
	for _, commit := range commits {
		if !taskKeyInText(task, commit.Message) {
			continue
		}
		if commit.Author != "" {
			_, _ = s.store.EnsureExternalIdentity(r.Context(), task.WorkspaceID, task.Provider, commit.Author, commit.Author, "")
		}
		link := &domain.TaskCommitLink{
			ID: "commit_" + ulid.Make().String(), TaskID: task.ID, WorkspaceID: task.WorkspaceID,
			Provider: task.Provider, Repo: task.Instance, SHA: commit.SHA,
			URL: commit.URL, Message: commit.Message, Author: commit.Author,
		}
		if err := s.store.UpsertTaskCommit(r.Context(), link); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		linked++
	}
	items, _ := s.store.ListTaskCommits(r.Context(), task.ID)
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "linked": linked})
}
