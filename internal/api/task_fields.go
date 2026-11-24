package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

type taskFieldsRequest struct {
	Priority     string            `json:"priority"`
	Components   []string          `json:"components"`
	FixVersions  []string          `json:"fixVersions"`
	CustomFields map[string]string `json:"customFields"`
}

func (s *Server) updateTaskFields(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	var req taskFieldsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
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
	if !ok || ad.Work == nil {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider not registered")
		return
	}
	writer, ok := ad.Work.(integration.FieldWriter)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_UNSUPPORTED", "provider does not support field updates")
		return
	}
	cred := s.providerCredential(r.Context(), task.Provider, binding.ConnectionID)
	cred = s.accountCredential(r.Context(), task.Provider, binding.AccountID, cred)
	customIDs := map[string]string{}
	customFieldNames := map[string]string{}
	if len(req.CustomFields) > 0 {
		automation := s.automationForTask(r.Context(), task)
		if automation == nil || automation.Writeback == nil {
			writeErr(w, http.StatusBadRequest, "E_FIELDS", "custom field mapping is not configured")
			return
		}
		raw, _ := automation.Writeback["customFields"].(string)
		for id, name := range parseCustomFields(raw) {
			customFieldNames[id] = name
			customIDs[name] = id
		}
		for name, value := range req.CustomFields {
			id, ok := customIDs[name]
			if !ok {
				writeErr(w, http.StatusBadRequest, "E_FIELDS", "unknown custom field: "+name)
				return
			}
			customIDs[id] = value
		}
	}
	requested := map[string]string{}
	for key, value := range customIDs {
		if strings.HasPrefix(key, "customfield_") {
			requested[key] = value
		}
	}
	if len(req.CustomFields) > 0 {
		if metadata, ok := ad.Work.(integration.FieldMetadataReader); ok {
			items, err := metadata.ListFieldMetadata(r.Context(), cred, task.Instance, task.WorkID)
			if err == nil {
				byName := map[string]integration.FieldMetadata{}
				for _, item := range items {
					name := customFieldNames[item.ID]
					if name == "" {
						name = item.Name
					}
					byName[name] = item
				}
				for name, value := range req.CustomFields {
					meta, ok := byName[name]
					if !ok {
						continue
					}
					if value == "" && meta.Required {
						writeErr(w, http.StatusBadRequest, "E_FIELDS", name+" is required")
						return
					}
					if value == "" {
						continue
					}
					if meta.Type == "number" {
						if _, err := strconv.ParseFloat(value, 64); err != nil {
							writeErr(w, http.StatusBadRequest, "E_FIELDS", name+" must be a number")
							return
						}
					}
					if len(meta.Allowed) > 0 {
						valid := false
						for _, allowed := range meta.Allowed {
							if allowed == value {
								valid = true
								break
							}
						}
						if !valid {
							writeErr(w, http.StatusBadRequest, "E_FIELDS", name+" must be one of the allowed values")
							return
						}
					}
				}
			}
		}
	}
	ref := domain.WorkRef{
		Provider: task.Provider, Instance: task.Instance, Type: task.WorkType,
		ID: task.WorkID, URL: task.URL, Revision: task.Revision,
	}
	if err := writer.UpdateFields(r.Context(), ref, cred, integration.FieldUpdate{
		Priority: req.Priority, Components: req.Components, FixVersions: req.FixVersions, CustomFields: requested,
	}); err != nil {
		writeErr(w, http.StatusBadGateway, "E_FIELDS", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "updated"})
}

func (s *Server) taskFieldOptions(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	binding, _ := s.store.FindBinding(r.Context(), task.Provider, task.Instance)
	if binding == nil {
		writeErr(w, http.StatusBadRequest, "E_BINDING", "no binding for task instance")
		return
	}
	ad, ok := s.integrations.Get(task.Provider)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider not registered")
		return
	}
	catalog, ok := ad.Work.(integration.FieldCatalog)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_UNSUPPORTED", "provider does not expose field options")
		return
	}
	cred := s.providerCredential(r.Context(), task.Provider, binding.ConnectionID)
	cred = s.accountCredential(r.Context(), task.Provider, binding.AccountID, cred)
	project := task.WorkID
	if idx := strings.Index(project, "-"); idx > 0 {
		project = project[:idx]
	}
	options, err := catalog.ListFieldOptions(r.Context(), cred, task.Instance, project)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_FIELDS", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"options": options})
}

func (s *Server) taskFieldMetadata(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	binding, _ := s.store.FindBinding(r.Context(), task.Provider, task.Instance)
	if binding == nil {
		writeErr(w, http.StatusBadRequest, "E_BINDING", "no binding for task instance")
		return
	}
	ad, ok := s.integrations.Get(task.Provider)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider not registered")
		return
	}
	reader, ok := ad.Work.(integration.FieldMetadataReader)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_UNSUPPORTED", "provider does not expose field metadata")
		return
	}
	cred := s.providerCredential(r.Context(), task.Provider, binding.ConnectionID)
	cred = s.accountCredential(r.Context(), task.Provider, binding.AccountID, cred)
	items, err := reader.ListFieldMetadata(r.Context(), cred, task.Instance, task.WorkID)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_FIELDS", err.Error())
		return
	}
	names := map[string]string{}
	if automation := s.automationForTask(r.Context(), task); automation != nil && automation.Writeback != nil {
		if raw, ok := automation.Writeback["customFields"].(string); ok {
			for id, name := range parseCustomFields(raw) {
				names[id] = name
			}
		}
	}
	fields := map[string]integration.FieldMetadata{}
	for _, item := range items {
		name := names[item.ID]
		if name == "" {
			name = item.Name
		}
		fields[name] = item
	}
	writeJSON(w, http.StatusOK, map[string]any{"fields": fields})
}
