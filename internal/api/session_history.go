package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/history"
)

const localHistoryMachine = "local"

// loadLocalHistory returns the indexed history, rescanning when forced or stale.
func (s *Server) loadLocalHistory(ctx context.Context, force bool) ([]history.Entry, error) {
	raw, scannedAt, err := s.store.GetHistoryCache(ctx, localHistoryMachine)
	if err == nil && raw != "" && !force && time.Since(scannedAt) < 5*time.Minute {
		var cached []history.Entry
		if json.Unmarshal([]byte(raw), &cached) == nil {
			return cached, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	items, err := history.Scan(home, 200)
	if err != nil {
		return nil, err
	}
	encoded, _ := json.Marshal(items)
	if err := s.store.SetHistoryCache(ctx, localHistoryMachine, string(encoded), time.Now().UTC()); err != nil {
		s.log.Warn("cache local history", "err", err)
	}
	return items, nil
}

func (s *Server) listSessionHistory(w http.ResponseWriter, r *http.Request) {
	items, err := s.loadLocalHistory(r.Context(), r.URL.Query().Get("refresh") == "1")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_HISTORY", err.Error())
		return
	}
	localKeys := make(map[string]bool, len(items))
	for _, item := range items {
		localKeys[item.Agent+":"+item.ID] = true
	}
	host, _ := os.Hostname()
	if r.URL.Query().Get("all") == "1" {
		if cached, err := s.store.MachineHistory(r.Context()); err == nil {
			names := map[string]string{}
			if machines, err := s.store.ListMachines(r.Context()); err == nil {
				for _, machine := range machines {
					names[machine.ID] = machine.Name
				}
			}
			for machineID, raw := range cached {
				if names[machineID] == host {
					continue
				}
				var remote []history.Entry
				if json.Unmarshal([]byte(raw), &remote) != nil {
					continue
				}
				kept := make([]history.Entry, 0, len(remote))
				for i := range remote {
					if localKeys[remote[i].Agent+":"+remote[i].ID] {
						continue
					}
					if remote[i].MachineID == "" {
						remote[i].MachineID = machineID
					}
					if remote[i].MachineName == "" {
						remote[i].MachineName = names[machineID]
					}
					kept = append(kept, remote[i])
				}
				items = append(items, kept...)
			}
			sort.SliceStable(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
		}
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("query")))
	filtered := make([]history.Entry, 0, len(items))
	for _, item := range items {
		if query != "" {
			haystack := strings.ToLower(item.Title + " " + item.Cwd + " " + item.Preview + " " + item.ID)
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": filtered, "host": host})
}

// scanSessionHistory queues a history scan on every online machine.
func (s *Server) scanSessionHistory(w http.ResponseWriter, r *http.Request) {
	local, err := s.loadLocalHistory(r.Context(), true)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_HISTORY", err.Error())
		return
	}
	machines, err := s.store.ListMachines(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	ids := make([]string, 0, len(machines))
	for _, machine := range machines {
		if machine.Status == "online" {
			ids = append(ids, machine.ID)
		}
	}
	count, err := s.store.CreateHistoryScans(r.Context(), ids)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]int{"queued": count, "local": len(local)})
}

func (s *Server) agentdHistoryScans(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.PendingHistoryScans(r.Context(), chi.URLParam(r, "machineId"), 5)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type historyScanResult struct {
	Items []history.Entry `json:"items"`
	Error string          `json:"error"`
}

func (s *Server) agentdHistoryScanResult(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineId")
	var req historyScanResult
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	status := "done"
	if req.Error != "" {
		status = "failed"
	}
	raw, _ := json.Marshal(req.Items)
	if err := s.store.FinishHistoryScan(r.Context(), chi.URLParam(r, "id"), machineID, status, string(raw), req.Error); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

type resumeHistoryRequest struct {
	WorkspaceID string `json:"workspaceId"`
	AgentID     string `json:"agentId"`
	SessionID   string `json:"sessionId"`
	Title       string `json:"title"`
	Model       string `json:"model"`
	Repo        string `json:"repo"`
	MachineID   string `json:"machineId"`
}

func (s *Server) resumeSessionHistory(w http.ResponseWriter, r *http.Request) {
	var req resumeHistoryRequest
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
	var entry *history.Entry
	if req.MachineID != "" {
		all, err := s.store.MachineHistory(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		var remote []history.Entry
		if raw, ok := all[req.MachineID]; ok {
			_ = json.Unmarshal([]byte(raw), &remote)
		}
		for i := range remote {
			if remote[i].ID == req.SessionID {
				entry = &remote[i]
				entry.MachineID = req.MachineID
				break
			}
		}
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_HOME", err.Error())
			return
		}
		items, err := history.Scan(home, 500)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_HISTORY", err.Error())
			return
		}
		for i := range items {
			if items[i].ID == req.SessionID {
				entry = &items[i]
				break
			}
		}
	}
	if entry == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "history session not found")
		return
	}
	agents, err := s.store.ListAgents(r.Context(), req.WorkspaceID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	var agent *domain.Agent
	for _, candidate := range agents {
		if req.MachineID != "" && candidate.MachineID != req.MachineID {
			continue
		}
		if req.AgentID != "" {
			if candidate.ID == req.AgentID {
				agent = candidate
				break
			}
			continue
		}
		if strings.Contains(strings.ToLower(candidate.Backend), entry.Agent) {
			agent = candidate
			break
		}
		if agent == nil {
			agent = candidate
		}
	}
	if agent == nil {
		writeErr(w, http.StatusUnprocessableEntity, "E_AGENT", "no agent is available on this history host")
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = entry.Title
	}
	repo := strings.TrimSpace(req.Repo)
	if repo == "" {
		repo = entry.Cwd
	}
	model := req.Model
	if model == "" {
		model = entry.Model
	}
	sess, err := s.store.CreateAgentSession(r.Context(), &domain.AgentSession{
		WorkspaceID:      req.WorkspaceID,
		AgentID:          agent.ID,
		Backend:          entry.Agent,
		Model:            model,
		Mode:             "chat",
		Title:            title,
		Repo:             repo,
		Base:             "main",
		BackendSessionID: entry.ID,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	_, _ = s.store.AppendSessionMessage(r.Context(), &domain.SessionMessage{
		SessionID: sess.ID, Role: "system", Kind: "resumed",
		Text: "Resumed from " + entry.Agent + " session " + entry.ID + ".",
	})
	writeJSON(w, http.StatusCreated, sess)
}
