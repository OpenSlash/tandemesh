package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
	"github.com/robfig/cron/v3"

	"github.com/OpenSlash/tandemesh/internal/automation"
	"github.com/OpenSlash/tandemesh/internal/domain"
)

type automationRequest struct {
	WorkspaceID        string            `json:"workspaceId"`
	Kind               string            `json:"kind"` // schedule | event
	Name               string            `json:"name"`
	Schedule           string            `json:"schedule"`
	AgentID            string            `json:"agentId"`
	Repo               string            `json:"repo"`
	Base               string            `json:"base"`
	Prompt             string            `json:"prompt"`
	Enabled            *bool             `json:"enabled"`
	Provider           string            `json:"provider"`
	Instance           string            `json:"instance"`
	TriggerKind        string            `json:"triggerKind"`
	TriggerValue       string            `json:"triggerValue"`
	ConnectionID       string            `json:"connectionId"`
	Writeback          map[string]any    `json:"writeback"`
	StateMap           map[string]string `json:"stateMap"`
	Priority           int               `json:"priority"`
	Precheck           *string           `json:"precheck"`
	MissedGraceSeconds *int              `json:"missedGraceSeconds"`
	ReuseSession       *bool             `json:"reuseSession"`
}

func (s *Server) createAutomation(w http.ResponseWriter, r *http.Request) {
	var req automationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Kind == "event" {
		s.createEventAutomation(w, r, req)
		return
	}
	schedule, err := cron.ParseStandard(req.Schedule)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "E_SCHEDULE", "invalid cron schedule")
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if req.AgentID == "" {
		req.AgentID = "claude"
	}
	if req.WorkspaceID == "" {
		req.WorkspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, req.WorkspaceID) {
		return
	}
	if agent, err := s.store.GetAgent(r.Context(), req.AgentID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	} else if agent != nil && !s.agentInWorkspace(agent, req.WorkspaceID) {
		writeErr(w, http.StatusBadRequest, "E_AGENT", "agent is not in this workspace")
		return
	}
	now := time.Now().UTC()
	a := &domain.Automation{
		ID:                 "auto_" + ulid.Make().String(),
		WorkspaceID:        req.WorkspaceID,
		Name:               req.Name,
		Schedule:           req.Schedule,
		AgentID:            req.AgentID,
		Repo:               req.Repo,
		Base:               req.Base,
		Prompt:             req.Prompt,
		Enabled:            enabled,
		Precheck:           derefString(req.Precheck),
		MissedGraceSeconds: derefInt(req.MissedGraceSeconds),
		ReuseSession:       derefBool(req.ReuseSession),
		CreatedAt:          now,
	}
	if enabled {
		next := schedule.Next(now)
		a.NextRun = &next
	}
	if err := s.store.UpsertAutomation(r.Context(), a); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, automationView(a))
}

func (s *Server) getAutomation(w http.ResponseWriter, r *http.Request) {
	a, err := s.store.GetAutomation(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if a == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "automation not found")
		return
	}
	if !s.requireWorkspace(w, r, a.WorkspaceID) {
		return
	}
	writeJSON(w, http.StatusOK, automationView(a))
}

func (s *Server) updateAutomation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req automationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Kind == "event" {
		s.updateEventAutomation(w, r, id, req)
		return
	}
	a, err := s.store.GetAutomation(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if a == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "automation not found")
		return
	}
	if !s.requireWorkspace(w, r, a.WorkspaceID) {
		return
	}
	if req.Schedule != "" {
		if _, err := cron.ParseStandard(req.Schedule); err != nil {
			writeErr(w, http.StatusBadRequest, "E_SCHEDULE", "invalid cron schedule")
			return
		}
		a.Schedule = req.Schedule
	}
	if req.Name != "" {
		a.Name = req.Name
	}
	if req.AgentID != "" {
		a.AgentID = req.AgentID
	}
	a.Repo = req.Repo
	a.Base = req.Base
	a.Prompt = req.Prompt
	if req.Precheck != nil {
		a.Precheck = *req.Precheck
	}
	if req.MissedGraceSeconds != nil {
		a.MissedGraceSeconds = *req.MissedGraceSeconds
	}
	if req.ReuseSession != nil {
		a.ReuseSession = *req.ReuseSession
	}
	if req.Enabled != nil {
		a.Enabled = *req.Enabled
	}
	schedule, _ := cron.ParseStandard(a.Schedule)
	if a.Enabled {
		next := schedule.Next(time.Now().UTC())
		a.NextRun = &next
	} else {
		a.NextRun = nil
	}
	if err := s.store.UpsertAutomation(r.Context(), a); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, automationView(a))
}

func (s *Server) deleteAutomation(w http.ResponseWriter, r *http.Request) {
	a, err := s.store.GetAutomation(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if a == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "automation not found")
		return
	}
	if !s.requireWorkspace(w, r, a.WorkspaceID) {
		return
	}
	ok, err := s.store.DeleteAutomation(r.Context(), a.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "automation not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) runAutomation(w http.ResponseWriter, r *http.Request) {
	a, err := s.store.GetAutomation(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if a == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "automation not found")
		return
	}
	if !s.requireWorkspace(w, r, a.WorkspaceID) {
		return
	}
	schedule, err := cron.ParseStandard(a.Schedule)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "E_SCHEDULE", "invalid cron schedule")
		return
	}
	now := time.Now().UTC()
	run, err := automation.StartRun(r.Context(), s.store, a, now)
	if err != nil {
		var pre *automation.PrecheckError
		if errors.As(err, &pre) {
			_ = s.store.MarkAutomationRun(r.Context(), a.ID, now, schedule.Next(now), "precheck_failed", err.Error())
			writeErr(w, http.StatusUnprocessableEntity, "E_PRECHECK", err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	_ = s.store.MarkAutomationRun(r.Context(), a.ID, now, schedule.Next(now), "started", "")
	writeJSON(w, http.StatusAccepted, run)
}

func (s *Server) createEventAutomation(w http.ResponseWriter, r *http.Request, req automationRequest) {
	if req.Provider == "" {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "provider is required for event automations")
		return
	}
	if req.WorkspaceID == "" {
		req.WorkspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, req.WorkspaceID) {
		return
	}
	if req.AgentID == "" {
		req.AgentID = "claude"
	}
	if agent, err := s.store.GetAgent(r.Context(), req.AgentID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	} else if agent != nil && !s.agentInWorkspace(agent, req.WorkspaceID) {
		writeErr(w, http.StatusBadRequest, "E_AGENT", "agent is not in this workspace")
		return
	}
	if req.TriggerKind == "" {
		req.TriggerKind = domain.TriggerCreated
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	automation := &domain.Automation{
		ID: "auto_" + ulid.Make().String(), WorkspaceID: req.WorkspaceID,
		Name: req.Name, Provider: req.Provider, Instance: req.Instance,
		AgentID: req.AgentID, TriggerKind: req.TriggerKind, TriggerValue: req.TriggerValue,
		Writeback: req.Writeback, StateMap: req.StateMap, ConnectionID: req.ConnectionID,
		Priority: req.Priority, Enabled: enabled, Base: "main", Prompt: req.Prompt,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.UpsertAutomation(r.Context(), automation); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, automationView(automation))
}

func (s *Server) updateEventAutomation(w http.ResponseWriter, r *http.Request, id string, req automationRequest) {
	b, err := s.store.GetAutomation(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if b == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "automation not found")
		return
	}
	if !s.requireWorkspace(w, r, b.WorkspaceID) {
		return
	}
	if req.Name != "" {
		b.Name = req.Name
	}
	if req.Provider != "" {
		b.Provider = req.Provider
	}
	if req.Instance != "" {
		b.Instance = req.Instance
	}
	if req.AgentID != "" {
		if agent, err := s.store.GetAgent(r.Context(), req.AgentID); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		} else if agent != nil && !s.agentInWorkspace(agent, b.WorkspaceID) {
			writeErr(w, http.StatusBadRequest, "E_AGENT", "agent is not in this workspace")
			return
		}
		b.AgentID = req.AgentID
	}
	if req.TriggerKind != "" {
		b.TriggerKind = req.TriggerKind
	}
	if req.TriggerValue != "" {
		b.TriggerValue = req.TriggerValue
	}
	if req.Writeback != nil {
		if b.Writeback == nil {
			b.Writeback = map[string]any{}
		}
		for key, value := range req.Writeback {
			b.Writeback[key] = value
		}
	}
	if req.StateMap != nil {
		b.StateMap = req.StateMap
	}
	if req.Enabled != nil {
		b.Enabled = *req.Enabled
	}
	if err := s.store.UpsertAutomation(r.Context(), b); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, automationView(b))
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func derefBool(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}
