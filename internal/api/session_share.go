package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type createSessionShareRequest struct {
	TTLHours int `json:"ttlHours"`
}

func (s *Server) createSessionShare(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	var req createSessionShareRequest
	if r.ContentLength > 0 {
		if err := decodeJSON(w, r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
			return
		}
	}
	ttl := 7 * 24 * time.Hour
	if req.TTLHours > 0 {
		ttl = time.Duration(req.TTLHours) * time.Hour
	}
	share, err := s.store.CreateSessionShare(r.Context(), sess.ID, userOf(r), ttl)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, share)
}

func (s *Server) revokeSessionShare(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	if err := s.store.RevokeSessionShare(r.Context(), sess.ID, chi.URLParam(r, "token")); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

type sharedMessage struct {
	Role string `json:"role"`
	Kind string `json:"kind"`
	Text string `json:"text"`
	TS   int64  `json:"ts"`
}

// getSharedSession is the unauthenticated read-only share endpoint.
func (s *Server) getSharedSession(w http.ResponseWriter, r *http.Request) {
	share, err := s.store.GetSessionShare(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if share == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "share link is invalid or expired")
		return
	}
	sess, err := s.store.GetAgentSession(r.Context(), share.SessionID)
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	messages, err := s.store.ListSessionMessages(r.Context(), sess.ID, 500)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	out := make([]sharedMessage, 0, len(messages))
	for _, message := range messages {
		text := message.Text
		if message.Kind == "tool" || message.Kind == "result" {
			text = strings.TrimSpace(text)
		}
		out = append(out, sharedMessage{
			Role: message.Role, Kind: message.Kind, Text: text, TS: message.CreatedAt.UnixMilli(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"session": map[string]any{
			"title": sess.Title, "repo": sess.Repo, "backend": sess.Backend,
			"model": sess.Model, "createdAt": sess.CreatedAt, "updatedAt": sess.UpdatedAt,
		},
		"messages":  out,
		"expiresAt": share.ExpiresAt,
	})
}
