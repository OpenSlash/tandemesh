package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

type assignTaskRequest struct {
	MemberID string `json:"memberId"`
}

// assignTaskMember assigns a task to a member's provider identity.
func (s *Server) assignTaskMember(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	var req assignTaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	req.MemberID = strings.TrimSpace(req.MemberID)
	if req.MemberID == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "memberId is required")
		return
	}
	identities, err := s.store.ListMemberIdentities(r.Context(), task.WorkspaceID, req.MemberID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	var identity *domain.MemberIdentity
	for _, item := range identities {
		if item.Provider == task.Provider {
			identity = item
			break
		}
	}
	if identity == nil {
		writeErr(w, http.StatusBadRequest, "E_IDENTITY", "member has no identity for this provider")
		return
	}
	binding, _ := s.store.FindBinding(r.Context(), task.Provider, task.Instance)
	if binding == nil {
		writeErr(w, http.StatusBadRequest, "E_BINDING", "no binding for task instance")
		return
	}
	if binding.Direction == "read" {
		writeErr(w, http.StatusConflict, "E_READ_ONLY", "binding is read-only")
		return
	}
	ad, ok := s.integrations.Get(task.Provider)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider not registered")
		return
	}
	writer, ok := ad.Work.(integration.AssigneeWriter)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_UNSUPPORTED", "provider does not support assignment")
		return
	}
	cred := s.providerCredential(r.Context(), task.Provider, binding.ConnectionID)
	if personal, ok := s.memberCredentialFor(r.Context(), task.WorkspaceID, task.Provider, req.MemberID); ok {
		cred = personal
	} else {
		cred = s.accountCredential(r.Context(), task.Provider, binding.AccountID, cred)
	}
	ref := domain.WorkRef{
		Provider: task.Provider, Instance: task.Instance, Type: task.WorkType,
		ID: task.WorkID, URL: task.URL, Revision: task.Revision,
	}
	if err := writer.Assign(r.Context(), ref, cred, identity.ExternalID); err != nil {
		writeErr(w, http.StatusBadGateway, "E_ASSIGN", err.Error())
		return
	}
	assignee := identity.Login
	if assignee == "" {
		assignee = identity.ExternalID
	}
	if err := s.store.SetTaskExternalAssignee(r.Context(), task.ID, assignee); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.recordTaskEvent(r.Context(), task.ID, "task.assignee", map[string]any{
		"memberId": req.MemberID, "provider": task.Provider, "assignee": assignee,
	})
	writeJSON(w, http.StatusOK, map[string]any{"assignee": assignee, "identity": identity})
}
