package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/skillmarket"
)

type marketSkill struct {
	skillmarket.Entry
	Installed bool `json:"installed"`
}

func (s *Server) skillMarketEntries(ctx context.Context) ([]skillmarket.Entry, error) {
	entries, err := skillmarket.Load(s.cfg.SkillMarketFile)
	if err != nil {
		return nil, err
	}
	var repos []string
	for _, repo := range strings.Split(s.cfg.SkillMarketRepos, ",") {
		if repo = strings.TrimSpace(repo); repo != "" {
			repos = append(repos, repo)
		}
	}
	if strings.TrimSpace(s.cfg.SkillMarketURL) != "" || len(repos) > 0 {
		remote, err := skillmarket.Fetch(ctx, s.cfg.SkillMarketURL, repos, s.cfg.GitHubToken)
		if err != nil {
			if s.log != nil {
				s.log.Warn("skill market source", "err", err)
			}
		} else {
			entries = append(entries, remote...)
		}
	}
	seen := map[string]struct{}{}
	out := make([]skillmarket.Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.Name == "" {
			continue
		}
		if _, ok := seen[entry.Name]; ok {
			continue
		}
		seen[entry.Name] = struct{}{}
		out = append(out, entry)
	}
	return out, nil
}

func (s *Server) listSkillMarket(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	entries, err := s.skillMarketEntries(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_MARKET", err.Error())
		return
	}
	entries = skillmarket.Search(entries, r.URL.Query().Get("q"))
	installed := map[string]bool{}
	if skills, err := s.store.ListSkills(r.Context(), workspaceID); err == nil {
		for _, skill := range skills {
			installed[skill.Name] = true
		}
	}
	items := make([]marketSkill, 0, len(entries))
	for _, entry := range entries {
		items = append(items, marketSkill{Entry: entry, Installed: installed[entry.Name]})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) installSkillMarket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	entries, err := s.skillMarketEntries(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_MARKET", err.Error())
		return
	}
	var entry *skillmarket.Entry
	for i := range entries {
		if entries[i].ID == id {
			entry = &entries[i]
			break
		}
	}
	if entry == nil {
		writeErr(w, http.StatusNotFound, "E_MARKET", "market skill not found")
		return
	}

	var req struct {
		WorkspaceID string   `json:"workspaceId"`
		AgentIDs    []string `json:"agentIds"`
	}
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

	sourceURL := entry.SourceURL
	if sourceURL == "" {
		sourceURL = "orca://skill-market/" + entry.ID
	}
	skill := &domain.Skill{
		ID:          "skl_" + ulid.Make().String(),
		WorkspaceID: req.WorkspaceID,
		Name:        entry.Name,
		Description: entry.Description,
		Version:     entry.Version,
		Source:      "market",
		SourceURL:   sourceURL,
		Tags:        entry.Tags,
		Enabled:     true,
		Body:        entry.Body,
	}
	if err := s.store.UpsertSkill(r.Context(), skill); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	created, _ := s.store.GetSkillByName(r.Context(), req.WorkspaceID, entry.Name)
	if created == nil {
		created = skill
	}

	attached := 0
	for _, agentID := range req.AgentIDs {
		agent, err := s.store.GetAgent(r.Context(), agentID)
		if err != nil || agent == nil || !s.allowAgent(r.Context(), agent) {
			continue
		}
		if !s.agentInWorkspace(agent, req.WorkspaceID) {
			continue
		}
		has := false
		for _, name := range agent.Skills {
			if name == entry.Name {
				has = true
				break
			}
		}
		if !has {
			agent.Skills = append(agent.Skills, entry.Name)
			if err := s.store.UpsertAgent(r.Context(), agent); err != nil {
				writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
				return
			}
		}
		attached++
	}

	writeJSON(w, http.StatusCreated, map[string]any{"skill": created, "agents": attached})
}
