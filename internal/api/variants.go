package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type variantRequest struct {
	Agents []string          `json:"agents"`
	Labels map[string]string `json:"labels,omitempty"`
}

func (s *Server) createVariants(w http.ResponseWriter, r *http.Request) {
	origin, err := s.store.GetRun(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if origin == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "run not found")
		return
	}
	var req variantRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if len(req.Agents) == 0 {
		writeErr(w, http.StatusBadRequest, "E_AGENTS", "agents is required")
		return
	}

	root := origin.VariantOf
	if root == "" {
		root = origin.ID
	}
	now := time.Now().UTC()
	seen := map[string]bool{}
	items := make([]*domain.Run, 0, len(req.Agents))
	for _, agent := range req.Agents {
		if agent == "" || seen[agent] {
			continue
		}
		seen[agent] = true
		id := "run_" + ulid.Make().String()
		label := req.Labels[agent]
		if label == "" {
			label = agent
		}
		run := &domain.Run{
			ID:           id,
			WorkspaceID:  origin.WorkspaceID,
			Work:         origin.Work,
			Snapshot:     origin.Snapshot,
			AutomationID: origin.AutomationID,
			AgentID:      agent,
			Status:       domain.RunQueued,
			Priority:     origin.Priority,
			Repo:         origin.Repo,
			Base:         origin.Base,
			Branch:       "orca/" + id[len(id)-8:],
			Prompt:       origin.Prompt,
			VariantOf:    root,
			VariantLabel: label,
			TaskID:       origin.TaskID,
			CreatedAt:    now,
		}
		if err := s.store.CreateRun(r.Context(), run); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		_, _ = s.store.AppendEvent(r.Context(), id, domain.EventRunCreated, map[string]any{
			"variantOf": root,
			"agentId":   agent,
		})
		items = append(items, run)
	}
	writeJSON(w, http.StatusCreated, map[string]any{"items": items})
}

func (s *Server) listVariants(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	origin, err := s.store.GetRun(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if origin == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "run not found")
		return
	}
	root := origin.VariantOf
	if root == "" {
		root = origin.ID
	}
	items, err := s.store.ListRunVariants(r.Context(), root)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.Run{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "root": root})
}
