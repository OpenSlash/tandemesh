package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// getAgentEnv returns configured environment variable names, never values.
func (s *Server) getAgentEnv(w http.ResponseWriter, r *http.Request) {
	agent, err := s.store.GetAgent(r.Context(), chi.URLParam(r, "id"))
	if err != nil || agent == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "agent not found")
		return
	}
	if !s.allowAgent(r.Context(), agent) {
		writeErr(w, http.StatusForbidden, "E_FORBIDDEN", "not a member of any assigned workspace")
		return
	}
	keys := []string{}
	if s.vault != nil {
		if cred, ok, err := s.vault.Get(r.Context(), "agent-env", agent.ID); err == nil && ok && cred.AccessToken != "" {
			var vars map[string]string
			if json.Unmarshal([]byte(cred.AccessToken), &vars) == nil {
				for key := range vars {
					keys = append(keys, key)
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": keys})
}

// putAgentEnv replaces the agent's environment map (encrypted at rest).
func (s *Server) putAgentEnv(w http.ResponseWriter, r *http.Request) {
	agent, err := s.store.GetAgent(r.Context(), chi.URLParam(r, "id"))
	if err != nil || agent == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "agent not found")
		return
	}
	if !s.allowAgent(r.Context(), agent) {
		writeErr(w, http.StatusForbidden, "E_FORBIDDEN", "not a member of any assigned workspace")
		return
	}
	var req struct {
		Vars map[string]string `json:"vars"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if s.vault == nil {
		writeErr(w, http.StatusServiceUnavailable, "E_VAULT", "credential vault is not configured")
		return
	}
	raw, _ := json.Marshal(req.Vars)
	if err := s.vault.Put(r.Context(), integration.Credential{
		Provider: "agent-env", Subject: agent.ID, AccessToken: string(raw),
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_VAULT", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": len(req.Vars)})
}
