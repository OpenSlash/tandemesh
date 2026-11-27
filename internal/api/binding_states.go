package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Per-binding state mapping lives in the binding's settings:
//
//	state.in.<state>  = local status a platform state maps to
//	state.out.<local> = platform state a local status writes back to
//	writebackStatus   = "true" to write local status changes back
//
// <state> is the platform state id, or its name for bindings that span
// several projects (a whole Plane workspace), whose ids differ per project.
const (
	stateInPrefix          = "state.in."
	stateOutPrefix         = "state.out."
	settingWritebackStatus = "writebackStatus"
)

// defaultStateGroups lists, per local status, the platform state groups a
// write-back picks from when no target state is configured.
var defaultStateGroups = map[string][]string{
	domain.TaskOpen:       {"unstarted", "backlog"},
	domain.TaskInProgress: {"started"},
	domain.TaskDone:       {"completed"},
	domain.TaskCancelled:  {"cancelled", "canceled"},
}

// withBindingStateMap layers the task's binding overrides over base.
func (s *Server) withBindingStateMap(ctx context.Context, task *domain.Task, base map[string]string) map[string]string {
	binding, err := s.store.FindBinding(ctx, task.Provider, task.Instance)
	if err != nil || binding == nil || len(binding.Settings) == 0 {
		return base
	}
	merged := map[string]string{}
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range binding.Settings {
		if state, ok := strings.CutPrefix(k, stateInPrefix); ok && v != "" {
			merged[state] = v
		}
	}
	return merged
}

// bindingStates lists the platform states of a binding's project for the
// mapping editor.
func (s *Server) bindingStates(w http.ResponseWriter, r *http.Request) {
	binding, err := s.store.GetBinding(r.Context(), chi.URLParam(r, "id"))
	if err != nil || binding == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "binding not found")
		return
	}
	if !s.requireWorkspace(w, r, binding.WorkspaceID) {
		return
	}
	ad, ok := s.integrations.Get(binding.Provider)
	if !ok || ad.States == nil {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider does not expose states")
		return
	}
	cred := s.accountCredential(r.Context(), binding.Provider, binding.AccountID, s.providerCredential(r.Context(), binding.Provider, binding.ConnectionID))
	states, err := ad.States.ListStates(r.Context(), cred, binding.ExternalID)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_STATES", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": states})
}

// resolveStateTarget turns a configured target (a state id or name, or
// nothing) into the state id for the task's own project. With no target it
// picks the first state in the local status's default group.
func (s *Server) resolveStateTarget(ctx context.Context, ad *integration.Integration, cred integration.Credential, task *domain.Task, target, status string) string {
	if ad.States == nil {
		return target
	}
	states, err := ad.States.ListStates(ctx, cred, task.Instance)
	if err != nil || len(states) == 0 {
		return target
	}
	if target != "" {
		for _, st := range states {
			if st.ID == target {
				return st.ID
			}
		}
		for _, st := range states {
			if strings.EqualFold(st.Name, target) {
				return st.ID
			}
		}
		return target
	}
	for _, group := range defaultStateGroups[status] {
		for _, st := range states {
			if strings.EqualFold(st.Group, group) {
				return st.ID
			}
		}
	}
	return ""
}
