package api

import (
	"net/http"
	"sort"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type unifiedAutomation struct {
	ID           string     `json:"id"`
	Kind         string     `json:"kind"` // schedule | event
	Name         string     `json:"name"`
	Enabled      bool       `json:"enabled"`
	AgentID      string     `json:"agentId"`
	TriggerKind  string     `json:"triggerKind"`
	TriggerValue string     `json:"triggerValue,omitempty"`
	Provider     string     `json:"provider,omitempty"`
	Instance     string     `json:"instance,omitempty"`
	Schedule     string     `json:"schedule,omitempty"`
	Prompt       string     `json:"prompt,omitempty"`
	Repo         string     `json:"repo,omitempty"`
	NextRun      *time.Time `json:"nextRun,omitempty"`
	LastRun      *time.Time `json:"lastRun,omitempty"`
	LastStatus   string     `json:"lastStatus,omitempty"`
	LastError    string     `json:"lastError,omitempty"`
}

func automationView(a *domain.Automation) unifiedAutomation {
	kind := "event"
	if a.Schedule != "" {
		kind = "schedule"
	}
	return unifiedAutomation{
		ID: a.ID, Kind: kind, Name: a.Name, Enabled: a.Enabled, AgentID: a.AgentID,
		TriggerKind: a.TriggerKind, TriggerValue: a.TriggerValue,
		Provider: a.Provider, Instance: a.Instance, Schedule: a.Schedule,
		Prompt: a.Prompt, Repo: a.Repo,
		NextRun: a.NextRun, LastRun: a.LastRun, LastStatus: a.LastStatus, LastError: a.LastError,
	}
}

func (s *Server) listUnifiedAutomations(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	automations, err := s.store.ListAutomations(r.Context(), workspaceID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	items := make([]unifiedAutomation, 0, len(automations))
	for _, automation := range automations {
		items = append(items, automationView(automation))
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
