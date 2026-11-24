package api

import (
	"context"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) pushNotification(ctx context.Context, workspaceID, typ, title, body, link string) {
	if workspaceID == "" {
		return
	}
	notification := &domain.Notification{
		ID:          "ntf_" + ulid.Make().String(),
		WorkspaceID: workspaceID,
		Type:        typ,
		Title:       title,
		Body:        body,
		Link:        link,
	}
	if err := s.store.CreateNotification(ctx, notification); err != nil {
		s.log.Warn("create notification", "err", err)
	}
}

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")
	if workspaceID == "" {
		writeErr(w, http.StatusBadRequest, "E_WORKSPACE", "workspaceId is required")
		return
	}
	unread := r.URL.Query().Get("unread") == "1"
	if workspaceID == "all" {
		workspaces, err := s.store.ListWorkspaces(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		items := []*domain.Notification{}
		for _, workspace := range workspaces {
			if !s.allowWorkspace(r.Context(), workspace.ID) {
				continue
			}
			batch, err := s.store.ListNotifications(r.Context(), workspace.ID, unread, 200)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
				return
			}
			items = append(items, batch...)
		}
		sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
		if len(items) > 50 {
			items = items[:50]
		}
		writeJSON(w, http.StatusOK, map[string]any{"notifications": items})
		return
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListNotifications(r.Context(), workspaceID, unread, 50)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notifications": items})
}

func (s *Server) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	if err := s.store.MarkNotificationRead(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "read"})
}

func (s *Server) markAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")
	if workspaceID == "" {
		writeErr(w, http.StatusBadRequest, "E_WORKSPACE", "workspaceId is required")
		return
	}
	if workspaceID == "all" {
		workspaces, err := s.store.ListWorkspaces(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		for _, workspace := range workspaces {
			if !s.allowWorkspace(r.Context(), workspace.ID) {
				continue
			}
			_ = s.store.MarkAllNotificationsRead(r.Context(), workspace.ID)
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	if err := s.store.MarkAllNotificationsRead(r.Context(), workspaceID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
