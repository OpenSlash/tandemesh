package api

import (
	"context"
	"net/http"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type userContextKey struct{}

// identify resolves the acting user from the signed session or dev header.
func (s *Server) identify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := s.identityUser(r)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey{}, user)))
	})
}

func userOf(r *http.Request) string {
	if user, ok := r.Context().Value(userContextKey{}).(string); ok {
		return user
	}
	return ""
}

// allowWorkspace permits access when the local account owns the workspace.
// The open-source core has no organization or workspace membership model;
// tenancy.Local grants the single local account access to every workspace.
func (s *Server) allowWorkspace(ctx context.Context, workspaceID string) bool {
	if s.tenancy == nil {
		return false
	}
	return s.tenancy.CanAccess(ctx, userOfContext(ctx), workspaceID)
}

// requireWorkspace writes 403 when the user cannot access the workspace.
func (s *Server) requireWorkspace(w http.ResponseWriter, r *http.Request, workspaceID string) bool {
	if s.allowWorkspace(r.Context(), workspaceID) {
		return true
	}
	writeErr(w, http.StatusForbidden, "E_FORBIDDEN", "not a member of this workspace")
	return false
}

// requireTask checks access to the workspace that owns the task.
func (s *Server) requireTask(w http.ResponseWriter, r *http.Request, task *domain.Task) bool {
	if task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return false
	}
	return s.requireWorkspace(w, r, task.WorkspaceID)
}

// workspaceRole resolves the acting user's role. The local account is always
// the workspace owner; any other identity has no role.
func (s *Server) workspaceRole(ctx context.Context, workspaceID string) string {
	if s.tenancy == nil {
		return ""
	}
	return s.tenancy.Role(ctx, userOfContext(ctx), workspaceID)
}

// requireAdmin allows workspace owner or admin roles.
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request, workspaceID string) bool {
	role := s.workspaceRole(r.Context(), workspaceID)
	if role == "owner" || role == "admin" {
		return true
	}
	writeErr(w, http.StatusForbidden, "E_FORBIDDEN", "owner or admin role required")
	return false
}

// enforceViewer blocks writes for read-only workspace roles when the request
// identifies a workspace through the query string or X-Orca-Workspace header.
func (s *Server) enforceViewer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		workspaceID := r.URL.Query().Get("workspaceId")
		if workspaceID == "" {
			workspaceID = r.URL.Query().Get("workspace")
		}
		if workspaceID == "" {
			workspaceID = headerWithLegacy(r, "X-Tandemesh-Workspace", "X-Orca-Workspace")
		}
		if workspaceID != "" && s.workspaceRole(r.Context(), workspaceID) == "viewer" {
			writeErr(w, http.StatusForbidden, "E_FORBIDDEN", "viewer role is read-only")
			return
		}
		next.ServeHTTP(w, r)
	})
}
