package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

// listWork asks a task source for items assigned to the credential owner.
func (s *Server) listWork(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider is required")
		return
	}
	ad, ok := s.integrations.Get(provider)
	if !ok || ad.Reader == nil {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider does not support an inbox: "+provider)
		return
	}
	cred := s.providerCredential(r.Context(), provider)
	if cred.AccessToken == "" {
		writeErr(w, http.StatusBadRequest, "E_CREDENTIAL", "no credential configured for "+provider)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := ad.Reader.ListAssigned(r.Context(), cred, integration.ListOptions{
		Instance: r.URL.Query().Get("instance"),
		Limit:    limit,
	})
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_INBOX", err.Error())
		return
	}
	if items == nil {
		items = []integration.WorkItem{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "provider": provider})
}

type syncWorkRequest struct {
	Provider     string `json:"provider"`
	Instance     string `json:"instance"`
	ConnectionID string `json:"connectionId"`
	WorkspaceID  string `json:"workspaceId"`
	Limit        int    `json:"limit"`
}

// syncWork pulls assigned items and projects them as tasks.
func (s *Server) syncWork(w http.ResponseWriter, r *http.Request) {
	var req syncWorkRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Provider == "" {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider is required")
		return
	}
	workspaceID := req.WorkspaceID
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	ad, ok := s.integrations.Get(req.Provider)
	if !ok || ad.Reader == nil {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider does not support an inbox: "+req.Provider)
		return
	}
	connectionID := req.ConnectionID
	if connectionID == "" {
		if all, err := s.store.ListAllConnections(r.Context()); err == nil {
			for _, c := range all {
				if c.Provider != req.Provider {
					continue
				}
				if c.Instance == req.Instance {
					connectionID = c.ID
					break
				}
				if connectionID == "" {
					bindings, _ := s.store.ListBindings(r.Context(), workspaceID, c.ID)
					for _, binding := range bindings {
						if binding.ExternalID == req.Instance {
							connectionID = c.ID
							break
						}
					}
				}
			}
		}
	}
	cred := s.providerCredential(r.Context(), req.Provider, connectionID)
	if cred.AccessToken == "" {
		writeErr(w, http.StatusBadRequest, "E_CREDENTIAL", "no credential configured for "+req.Provider)
		return
	}
	items, err := ad.Reader.ListAssigned(r.Context(), cred, integration.ListOptions{
		Instance: req.Instance,
		Limit:    req.Limit,
	})
	if err != nil {
		if connectionID != "" && integration.IsAuthError(err) {
			_ = s.store.SetConnectionStatus(r.Context(), connectionID, domain.ConnectionNeedsReauth)
		}
		writeErr(w, http.StatusBadGateway, "E_INBOX", err.Error())
		return
	}
	synced := 0
	stateMap := map[string]string{}
	conflict := "platform"
	if connectionID != "" {
		if automations, err := s.store.ListAutomations(r.Context(), req.WorkspaceID); err == nil {
			for _, automation := range automations {
				if automation.ConnectionID == connectionID {
					stateMap = automation.StateMap
					conflict = conflictPolicy(automation)
					break
				}
			}
		}
	}
	for _, item := range items {
		if existing, err := s.store.GetTask(r.Context(), store.TaskID(item.Ref)); err == nil && existing != nil && existing.Revision == item.Ref.Revision && existing.PlatformState == item.Status && existing.PlatformStateGroup == item.StatusGroup {
			continue
		}
		snap := domain.Snapshot{
			Title: item.Title, Body: item.Body, Labels: item.Labels, Author: item.Author, Assignee: item.Assignee,
			Status: item.Status, StatusGroup: item.StatusGroup,
		}
		task, err := s.store.EnsureTaskIn(r.Context(), req.WorkspaceID, item.Ref, snap)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		if len(item.Attachments) > 0 {
			attachments := make([]*domain.TaskAttachment, 0, len(item.Attachments))
			for _, attachment := range item.Attachments {
				attachments = append(attachments, &domain.TaskAttachment{
					ID: "att_" + task.ID + "_" + attachment.ID, TaskID: task.ID, Provider: task.Provider,
					ExternalID: attachment.ID, Name: attachment.Name, URL: attachment.URL,
					Mime: attachment.Mime, Size: attachment.Size, Author: attachment.Author, Source: "issue",
				})
			}
			if err := s.store.ReplaceTaskAttachments(r.Context(), task.ID, task.Provider, "issue", attachments); err != nil {
				writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
				return
			}
		}
		s.applyPlatformState(r.Context(), task, snap, stateMap, conflict)
		s.applyAssignmentRules(r.Context(), task)
		synced++
	}
	writeJSON(w, http.StatusOK, map[string]any{"synced": synced, "provider": req.Provider})
}

type importWorkRequest struct {
	Provider    string   `json:"provider"`
	Instance    string   `json:"instance"`
	Type        string   `json:"type"`
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Revision    string   `json:"revision"`
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	Labels      []string `json:"labels"`
	Author      string   `json:"author"`
	Assignee    string   `json:"assignee"`
	WorkspaceID string   `json:"workspaceId"`
}

// importWork projects an inbox item into the task workspace.
func (s *Server) importWork(w http.ResponseWriter, r *http.Request) {
	var req importWorkRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if strings.TrimSpace(req.Provider) == "" || strings.TrimSpace(req.ID) == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "provider and id are required")
		return
	}
	if req.Type == "" {
		req.Type = "issue"
	}
	if req.WorkspaceID == "" {
		req.WorkspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, req.WorkspaceID) {
		return
	}
	if req.URL == "" {
		req.URL = "https://" + strings.TrimRight(req.Instance, "/")
	}
	work := domain.WorkRef{
		Provider: req.Provider, Instance: req.Instance, Type: req.Type,
		ID: req.ID, URL: req.URL, Revision: req.Revision,
	}
	task, err := s.store.EnsureTaskIn(r.Context(), req.WorkspaceID, work, domain.Snapshot{
		Title: req.Title, Body: req.Body, Labels: req.Labels, Author: req.Author, Assignee: req.Assignee,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.applyAssignmentRules(r.Context(), task)
	s.recordTaskEvent(r.Context(), task.ID, "task.imported", map[string]any{
		"provider": req.Provider, "workId": req.ID, "assignee": req.Assignee,
	})
	writeJSON(w, http.StatusCreated, task)
}
