package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListWorkspaces(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	visible := make([]*domain.Workspace, 0, len(items))
	for _, item := range items {
		if s.allowWorkspace(r.Context(), item.ID) {
			visible = append(visible, item)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": visible})
}

type workspaceRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (s *Server) createWorkspace(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r, "") {
		return
	}
	var req workspaceRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Name == "" {
		writeErr(w, http.StatusBadRequest, "E_NAME", "name is required")
		return
	}
	if req.Slug == "" {
		req.Slug = slugify(req.Name)
	}
	ws := &domain.Workspace{ID: "ws_" + ulid.Make().String(), Name: req.Name, Slug: req.Slug}
	if err := s.store.CreateWorkspace(r.Context(), ws); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, ws)
}

func (s *Server) updateWorkspace(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !s.requireAdmin(w, r, id) {
		return
	}
	var req workspaceRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeErr(w, http.StatusBadRequest, "E_NAME", "name is required")
		return
	}
	if len(name) > 80 {
		writeErr(w, http.StatusBadRequest, "E_NAME", "name is too long")
		return
	}
	ws, err := s.store.UpdateWorkspaceName(r.Context(), id, name)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if ws == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "workspace not found")
		return
	}
	writeJSON(w, http.StatusOK, ws)
}

func (s *Server) listGroups(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListGroups(r.Context(), workspaceID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.Group{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type groupRequest struct {
	WorkspaceID string `json:"workspaceId"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s *Server) createGroup(w http.ResponseWriter, r *http.Request) {
	var req groupRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Name == "" {
		writeErr(w, http.StatusBadRequest, "E_NAME", "name is required")
		return
	}
	if req.WorkspaceID == "" {
		req.WorkspaceID = "ws_default"
	}
	group := &domain.Group{
		ID: "grp_" + ulid.Make().String(), WorkspaceID: req.WorkspaceID,
		Name: req.Name, Description: req.Description,
	}
	if err := s.store.CreateGroup(r.Context(), group); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

func (s *Server) getGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	groups, err := s.store.ListGroups(r.Context(), "")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	var found *domain.Group
	for _, g := range groups {
		if g.ID == id {
			found = g
			break
		}
	}
	if found == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "group not found")
		return
	}
	members, err := s.store.ListGroupMembers(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"group": found, "members": members})
}

type memberRequest struct {
	AgentID string `json:"agentId"`
	Role    string `json:"role"`
}

func (s *Server) addGroupMember(w http.ResponseWriter, r *http.Request) {
	var req memberRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.AgentID == "" {
		writeErr(w, http.StatusBadRequest, "E_AGENT", "agentId is required")
		return
	}
	member := &domain.GroupMember{GroupID: chi.URLParam(r, "id"), AgentID: req.AgentID, Role: req.Role}
	if err := s.store.AddGroupMember(r.Context(), member); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, member)
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
