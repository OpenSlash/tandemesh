package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) listPlatformComments(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	items, err := s.store.ListExternalComments(r.Context(), task.ID, 100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.ExternalComment{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type platformCommentRequest struct {
	Body     string `json:"body"`
	ThreadID string `json:"threadId"`
}

func (s *Server) addPlatformComment(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	var req platformCommentRequest
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
	if req.ThreadID != "" {
		threads, _ := s.store.ListCommentThreads(r.Context(), task.ID)
		found := false
		for _, thread := range threads {
			if thread.ID == req.ThreadID {
				found = true
				break
			}
		}
		if !found {
			writeErr(w, http.StatusNotFound, "E_THREAD", "comment thread not found")
			return
		}
		if !s.store.WritebackAllowed(r.Context(), task.Provider, connectionID, task.Instance) {
			writeErr(w, http.StatusConflict, "E_READ_ONLY", "binding is read-only")
			return
		}
	}
	cred := s.providerCredential(r.Context(), task.Provider, connectionID)
	if personal, ok := s.memberCredential(r.Context(), task.WorkspaceID, task.Provider); ok {
		cred = personal
	}
	ref := domain.WorkRef{
		Provider: task.Provider, Instance: task.Instance, Type: task.WorkType,
		ID: task.WorkID, URL: task.URL, Revision: task.Revision,
	}
	commentID := "pc_" + ulid.Make().String()
	delivery := &domain.CommentDelivery{
		ID: "cd_" + ulid.Make().String(), WorkspaceID: task.WorkspaceID, TaskID: task.ID,
		CommentID: commentID, Provider: task.Provider, Status: "pending", Attempts: 1,
	}
	_ = s.store.UpsertCommentDelivery(r.Context(), delivery)
	url, err := ad.Work.Comment(r.Context(), ref, cred, req.Body)
	if err != nil {
		delivery.Status = commentFailureStatus(err)
		delivery.Error = err.Error()
		_ = s.store.UpsertCommentDelivery(r.Context(), delivery)
		writeErr(w, http.StatusBadGateway, "E_COMMENT", err.Error())
		return
	}
	delivery.Status = "sent"
	delivery.ExternalID = url
	_ = s.store.UpsertCommentDelivery(r.Context(), delivery)
	comment := &domain.ExternalComment{
		ID: commentID, TaskID: task.ID, Provider: task.Provider,
		ExternalID: url, Author: "me", Body: req.Body, ThreadID: req.ThreadID,
		Direction: "outbound", AuthorMemberID: "me",
	}
	if err := s.store.UpsertExternalComment(r.Context(), comment); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"comment": comment, "url": url})
}

func (s *Server) listCommentThreads(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	threads, err := s.store.ListCommentThreads(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	comments, _ := s.store.ListExternalComments(r.Context(), task.ID, 200)
	grouped := map[string][]*domain.ExternalComment{}
	for _, comment := range comments {
		grouped[comment.ThreadID] = append(grouped[comment.ThreadID], comment)
	}
	if threads == nil {
		threads = []*domain.CommentThread{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": threads, "comments": grouped})
}

func (s *Server) resolveCommentThread(w http.ResponseWriter, r *http.Request) {
	threads, err := s.store.ListCommentThreads(r.Context(), chi.URLParam(r, "taskId"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	var target *domain.CommentThread
	for _, thread := range threads {
		if thread.ID == chi.URLParam(r, "threadId") {
			target = thread
			break
		}
	}
	if target == nil {
		writeErr(w, http.StatusNotFound, "E_THREAD", "comment thread not found")
		return
	}
	if !s.requireWorkspace(w, r, target.WorkspaceID) {
		return
	}
	var req struct {
		Resolved bool `json:"resolved"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if err := s.store.SetCommentThreadResolved(r.Context(), target.ID, req.Resolved); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"resolved": req.Resolved})
}

func commentFailureStatus(err error) string {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "uncertain"
	}
	text := strings.ToLower(err.Error())
	if strings.Contains(text, "timeout") || strings.Contains(text, "deadline") || strings.Contains(text, "connection reset") {
		return "uncertain"
	}
	return "failed"
}

func (s *Server) listCommentDeliveries(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	items, err := s.store.ListCommentDeliveries(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.CommentDelivery{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
