package api

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) auditRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			return
		}
		if ww.Status() >= 400 || r.URL.Path == "/metrics" {
			return
		}
		workspaceID := r.URL.Query().Get("workspaceId")
		if workspaceID == "" {
			workspaceID = r.URL.Query().Get("workspace")
		}
		if workspaceID == "" {
			workspaceID = headerWithLegacy(r, "X-Tandemesh-Workspace", "X-Orca-Workspace")
		}
		if workspaceID == "" {
			return
		}
		entry := &domain.AuditLog{
			ID:          "aud_" + ulid.Make().String(),
			WorkspaceID: workspaceID,
			Actor:       userOf(r),
			Action:      r.Method,
			Resource:    r.URL.Path,
			Detail:      r.URL.RawQuery,
		}
		if err := s.store.CreateAuditLog(r.Context(), entry); err != nil {
			s.log.Warn("audit log", "err", err)
		}
	})
}

func (s *Server) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")
	if workspaceID == "" {
		writeErr(w, http.StatusBadRequest, "E_WORKSPACE", "workspaceId is required")
		return
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListAuditLogs(r.Context(), workspaceID, 200)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"auditLogs": items})
}
