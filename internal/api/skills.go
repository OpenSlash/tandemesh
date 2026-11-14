package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	skillslib "github.com/OpenSlash/tandemesh/internal/skills"
)

type skillRequest struct {
	WorkspaceID string   `json:"workspaceId"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Source      string   `json:"source"`
	SourceURL   string   `json:"sourceUrl"`
	Tags        []string `json:"tags"`
	Enabled     *bool    `json:"enabled"`
	Body        string   `json:"body"`
}

type skillInstallRequest struct {
	WorkspaceID string   `json:"workspaceId"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Source      string   `json:"source"`
	SourceURL   string   `json:"sourceUrl"`
	Tags        []string `json:"tags"`
	Enabled     *bool    `json:"enabled"`
	Body        string   `json:"body"`
}

func (s *Server) listSkills(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListSkills(r.Context(), workspaceID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.Skill{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getSkill(w http.ResponseWriter, r *http.Request) {
	skill, err := s.store.GetSkill(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if skill == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "skill not found")
		return
	}
	if !s.requireWorkspace(w, r, skill.WorkspaceID) {
		return
	}
	writeJSON(w, http.StatusOK, skill)
}

func (s *Server) createSkill(w http.ResponseWriter, r *http.Request) {
	var req skillRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeErr(w, http.StatusBadRequest, "E_NAME", "name is required")
		return
	}
	if req.WorkspaceID == "" {
		req.WorkspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, req.WorkspaceID) {
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	source := req.Source
	if source == "" {
		source = "manual"
	}
	skill := &domain.Skill{
		ID: "skl_" + ulid.Make().String(), WorkspaceID: req.WorkspaceID,
		Name: strings.TrimSpace(req.Name), Description: req.Description, Version: req.Version,
		Source: source, SourceURL: req.SourceURL, Tags: req.Tags, Enabled: enabled, Body: req.Body,
	}
	if err := s.store.UpsertSkill(r.Context(), skill); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	created, _ := s.store.GetSkillByName(r.Context(), skill.WorkspaceID, skill.Name)
	if created == nil {
		created = skill
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) updateSkill(w http.ResponseWriter, r *http.Request) {
	skill, err := s.store.GetSkill(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if skill == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "skill not found")
		return
	}
	if !s.requireWorkspace(w, r, skill.WorkspaceID) {
		return
	}
	var req skillRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Name != "" {
		skill.Name = strings.TrimSpace(req.Name)
	}
	skill.Description = req.Description
	skill.Body = req.Body
	if req.Version != "" {
		skill.Version = req.Version
	}
	if req.Source != "" {
		skill.Source = req.Source
	}
	if req.SourceURL != "" {
		skill.SourceURL = req.SourceURL
	}
	if req.Tags != nil {
		skill.Tags = req.Tags
	}
	if req.Enabled != nil {
		skill.Enabled = *req.Enabled
	}
	if err := s.store.UpsertSkill(r.Context(), skill); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	updated, _ := s.store.GetSkillByName(r.Context(), skill.WorkspaceID, skill.Name)
	if updated == nil {
		updated = skill
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteSkill(w http.ResponseWriter, r *http.Request) {
	skill, err := s.store.GetSkill(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if skill == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "skill not found")
		return
	}
	if !s.requireWorkspace(w, r, skill.WorkspaceID) {
		return
	}
	ok, err := s.store.DeleteSkill(r.Context(), skill.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "skill not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) setSkillEnabled(w http.ResponseWriter, r *http.Request) {
	skill, err := s.store.GetSkill(r.Context(), chi.URLParam(r, "id"))
	if err != nil || skill == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "skill not found")
		return
	}
	if !s.requireWorkspace(w, r, skill.WorkspaceID) {
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	updated, err := s.store.SetSkillEnabled(r.Context(), skill.ID, req.Enabled)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) installSkill(w http.ResponseWriter, r *http.Request) {
	var req skillInstallRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.WorkspaceID == "" {
		req.WorkspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, req.WorkspaceID) {
		return
	}
	raw := []byte(req.Body)
	source := req.Source
	if source == "" {
		source = "manual"
	}
	if strings.TrimSpace(req.SourceURL) != "" {
		if !strings.HasPrefix(req.SourceURL, "http://") && !strings.HasPrefix(req.SourceURL, "https://") {
			writeErr(w, http.StatusBadRequest, "E_URL", "sourceUrl must be http(s)")
			return
		}
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Get(req.SourceURL)
		if err != nil {
			writeErr(w, http.StatusBadGateway, "E_FETCH", err.Error())
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			writeErr(w, http.StatusBadGateway, "E_FETCH", "source returned "+resp.Status)
			return
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			writeErr(w, http.StatusBadGateway, "E_FETCH", err.Error())
			return
		}
		raw = data
		source = "url"
	}
	if len(raw) == 0 {
		writeErr(w, http.StatusBadRequest, "E_BODY", "body or sourceUrl is required")
		return
	}
	name := strings.TrimSpace(req.Name)
	description := ""
	body := strings.TrimSpace(string(raw))
	version := req.Version
	tags := req.Tags
	if parsed, err := skillslib.Parse(raw); err == nil {
		if name == "" {
			name = parsed.Name
		}
		if description == "" {
			description = parsed.Description
		}
		if parsed.Body != "" {
			body = parsed.Body
		}
		if version == "" {
			version = parsed.Version
		}
		if len(tags) == 0 {
			tags = parsed.Tags
		}
	}
	if name == "" {
		writeErr(w, http.StatusBadRequest, "E_NAME", "name is required when the document has no frontmatter")
		return
	}
	sum := sha256.Sum256(raw)
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	skill := &domain.Skill{
		ID: "skl_" + ulid.Make().String(), WorkspaceID: req.WorkspaceID, Name: name,
		Description: description, Version: version, Source: source, SourceURL: req.SourceURL,
		Tags: tags, Enabled: enabled, Checksum: hex.EncodeToString(sum[:]), Body: body,
	}
	if err := s.store.UpsertSkill(r.Context(), skill); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	installed, _ := s.store.GetSkillByName(r.Context(), skill.WorkspaceID, skill.Name)
	if installed == nil {
		installed = skill
	}
	writeJSON(w, http.StatusCreated, installed)
}

func (s *Server) rawSkill(w http.ResponseWriter, r *http.Request) {
	skill, err := s.store.GetSkill(r.Context(), chi.URLParam(r, "id"))
	if err != nil || skill == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "skill not found")
		return
	}
	if !s.requireWorkspace(w, r, skill.WorkspaceID) {
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = w.Write([]byte(skillslib.Render(skillslib.Skill{
		Name: skill.Name, Description: skill.Description, Version: skill.Version, Tags: skill.Tags, Body: skill.Body,
	})))
}

func (s *Server) exportSkills(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListSkills(r.Context(), workspaceID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, skill := range items {
		out = append(out, map[string]any{
			"id": skill.ID, "name": skill.Name, "version": skill.Version, "enabled": skill.Enabled,
			"markdown": skillslib.Render(skillslib.Skill{
				Name: skill.Name, Description: skill.Description, Version: skill.Version, Tags: skill.Tags, Body: skill.Body,
			}),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// skillPrompt builds the system prefix for an agent: its instructions and
// the skills it references.
func (s *Server) skillPrompt(ctx context.Context, workspaceID, agentID string) string {
	if agentID == "" {
		return ""
	}
	agent, err := s.store.GetAgent(ctx, agentID)
	if err != nil || agent == nil {
		return ""
	}
	parts := []string{}
	if instructions := strings.TrimSpace(agent.Instructions); instructions != "" {
		parts = append(parts, "Agent instructions:\n"+instructions)
	}
	var items []skillslib.Skill
	for _, name := range agent.Skills {
		skill, err := s.store.GetSkillByName(ctx, workspaceID, name)
		if err != nil || skill == nil || !skill.Enabled {
			continue
		}
		items = append(items, skillslib.Skill{
			Name: skill.Name, Description: skill.Description, Body: skill.Body,
		})
	}
	if prefix := skillslib.Prompt(items, 32*1024); prefix != "" {
		parts = append(parts, prefix)
	}
	return strings.Join(parts, "\n\n")
}
