package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type createTerminalRequest struct {
	WorkspaceID    string `json:"workspaceId"`
	AgentSessionID string `json:"agentSessionId"`
	MachineID      string `json:"machineId"`
	Shell          string `json:"shell"`
	Cwd            string `json:"cwd"`
	Cols           int    `json:"cols"`
	Rows           int    `json:"rows"`
}

func (s *Server) listTerminalSessions(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID != "" && !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListTerminalSessions(r.Context(), workspaceID, 200)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.TerminalSession{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// terminalResumeCommand reopens the agent CLI behind an existing chat turn.
func terminalResumeCommand(sess *domain.AgentSession) string {
	backend := strings.ToLower(strings.TrimSpace(sess.Backend))
	id := strings.TrimSpace(sess.BackendSessionID)
	var exe, resume string
	switch {
	case strings.Contains(backend, "claude"):
		exe, resume = "claude", "--resume"
	case strings.Contains(backend, "codex"):
		exe, resume = "codex", "resume"
	case strings.Contains(backend, "gemini"):
		exe, resume = "gemini", "--resume"
	case strings.Contains(backend, "opencode"):
		exe, resume = "opencode", "-s"
	case strings.Contains(backend, "cursor"):
		exe, resume = "cursor-agent", "--resume"
	case backend != "":
		exe = backend
	}
	if exe == "" {
		return ""
	}
	if id == "" || resume == "" {
		return exe
	}
	return fmt.Sprintf("%s %s %s", exe, resume, id)
}

func (s *Server) createTerminalSession(w http.ResponseWriter, r *http.Request) {
	var req createTerminalRequest
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
	var agentSession *domain.AgentSession
	if req.AgentSessionID != "" {
		sess, _ := s.store.GetAgentSession(r.Context(), req.AgentSessionID)
		if sess == nil || sess.WorkspaceID != req.WorkspaceID {
			writeErr(w, http.StatusBadRequest, "E_SESSION", "agent session not found in this workspace")
			return
		}
		agentSession = sess
	}
	machineID := req.MachineID
	if machineID == "" {
		machines, err := s.store.ListMachines(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		for _, m := range machines {
			if m.Status == "online" {
				machineID = m.ID
				break
			}
		}
	}
	if machineID == "" {
		writeErr(w, http.StatusUnprocessableEntity, "E_MACHINE", "no online execution machine")
		return
	}
	shell := req.Shell
	cwd := req.Cwd
	if agentSession != nil {
		if cwd == "" && agentSession.WorktreeID != "" {
			if wt, err := s.store.GetWorktree(r.Context(), agentSession.WorktreeID); err == nil && wt != nil {
				cwd = wt.Path
			}
		}
	}
	t := &domain.TerminalSession{
		WorkspaceID:    req.WorkspaceID,
		AgentSessionID: req.AgentSessionID,
		MachineID:      machineID,
		Shell:          shell,
		Cwd:            cwd,
		Cols:           req.Cols,
		Rows:           req.Rows,
		Status:         "pending",
	}
	if err := s.store.CreateTerminalSession(r.Context(), t); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

type continueTerminalRequest struct {
	Title   string `json:"title"`
	AgentID string `json:"agentId"`
}

func (s *Server) continueTerminalChat(w http.ResponseWriter, r *http.Request) {
	terminal, err := s.store.GetTerminalSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || terminal == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "terminal session not found")
		return
	}
	if !s.requireWorkspace(w, r, terminal.WorkspaceID) {
		return
	}
	var req continueTerminalRequest
	if r.ContentLength > 0 {
		if err := decodeJSON(w, r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
			return
		}
	}
	var source *domain.AgentSession
	if terminal.AgentSessionID != "" {
		source, _ = s.store.GetAgentSession(r.Context(), terminal.AgentSessionID)
	}
	agentID := req.AgentID
	backend, model := "", ""
	repo, base := "", ""
	title := "Continue terminal"
	if source != nil {
		if agentID == "" {
			agentID = source.AgentID
		}
		backend, model = source.Backend, source.Model
		repo, base = source.Repo, source.Base
		title = "Continue: " + source.Title
	}
	if agentID == "" {
		agents, _ := s.store.ListAgents(r.Context(), terminal.WorkspaceID)
		if len(agents) > 0 {
			agentID = agents[0].ID
			backend, model = agents[0].Backend, agents[0].Model
		}
	}
	if agentID == "" {
		writeErr(w, http.StatusUnprocessableEntity, "E_AGENT", "no agent available in this workspace")
		return
	}
	if req.Title != "" {
		title = req.Title
	}
	events, _ := s.store.TerminalOutputSince(r.Context(), terminal.ID, 0)
	var transcript strings.Builder
	for _, event := range events {
		transcript.WriteString(event.Data)
	}
	text := transcript.String()
	if len(text) > 12000 {
		text = text[len(text)-12000:]
	}
	sess, err := s.store.CreateAgentSession(r.Context(), &domain.AgentSession{
		WorkspaceID: terminal.WorkspaceID,
		AgentID:     agentID,
		Backend:     backend,
		Model:       model,
		Mode:        "chat",
		Title:       title,
		Repo:        repo,
		Base:        base,
		Context:     "Continue from this terminal transcript:\n\n" + text,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	_, _ = s.store.AppendSessionMessage(r.Context(), &domain.SessionMessage{
		SessionID: sess.ID, Role: "system", Kind: "transcript",
		Text: fmt.Sprintf("Terminal transcript attached (%d characters).", len(text)),
	})
	writeJSON(w, http.StatusCreated, sess)
}

func (s *Server) getTerminalSession(w http.ResponseWriter, r *http.Request) {
	t, err := s.store.GetTerminalSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if t == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "terminal session not found")
		return
	}
	if !s.requireWorkspace(w, r, t.WorkspaceID) {
		return
	}
	writeJSON(w, http.StatusOK, t)
}

type terminalInputRequest struct {
	Data string `json:"data"`
}

func (s *Server) terminalInput(w http.ResponseWriter, r *http.Request) {
	t, err := s.store.GetTerminalSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || t == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "terminal session not found")
		return
	}
	if !s.requireWorkspace(w, r, t.WorkspaceID) {
		return
	}
	if t.Status == "exited" || t.Status == "closed" {
		writeErr(w, http.StatusGone, "E_TERMINAL_DONE", "terminal session has ended")
		return
	}
	var req terminalInputRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if err := s.store.AppendTerminalInput(r.Context(), t.ID, req.Data); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "queued"})
}

func (s *Server) terminalOutput(w http.ResponseWriter, r *http.Request) {
	t, err := s.store.GetTerminalSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || t == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "terminal session not found")
		return
	}
	if !s.requireWorkspace(w, r, t.WorkspaceID) {
		return
	}
	after, _ := strconv.Atoi(r.URL.Query().Get("after"))
	events, err := s.store.TerminalOutputSince(r.Context(), t.ID, after)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"terminal": t,
		"events":   events,
	})
}

func (s *Server) closeTerminalSession(w http.ResponseWriter, r *http.Request) {
	t, err := s.store.GetTerminalSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || t == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "terminal session not found")
		return
	}
	if !s.requireWorkspace(w, r, t.WorkspaceID) {
		return
	}
	if err := s.store.SetTerminalStatus(r.Context(), t.ID, "closed", t.ExitCode); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "closed"})
}

// ---- agent daemon endpoints ----

func (s *Server) agentdTerminals(w http.ResponseWriter, r *http.Request) {
	machineID := r.URL.Query().Get("machineId")
	items, err := s.store.PendingTerminalSessions(r.Context(), machineID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.TerminalSession{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) agentdTerminalInput(w http.ResponseWriter, r *http.Request) {
	t, err := s.store.GetTerminalSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || t == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "terminal session not found")
		return
	}
	if t.Status == "closed" {
		writeErr(w, http.StatusGone, "E_TERMINAL_DONE", "terminal session closed")
		return
	}
	inputs, err := s.store.TakeTerminalInputs(r.Context(), t.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if inputs == nil {
		inputs = []domain.TerminalInput{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"inputs": inputs})
}

type agentdTerminalOutputRequest struct {
	Data string `json:"data"`
}

func (s *Server) agentdTerminalOutput(w http.ResponseWriter, r *http.Request) {
	var req agentdTerminalOutputRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Data != "" {
		if _, err := s.store.AppendTerminalOutput(r.Context(), chi.URLParam(r, "id"), req.Data); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type agentdTerminalStateRequest struct {
	Status   string `json:"status"`
	ExitCode *int   `json:"exitCode"`
}

func (s *Server) agentdTerminalState(w http.ResponseWriter, r *http.Request) {
	var req agentdTerminalStateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "running"
	}
	if err := s.store.SetTerminalStatus(r.Context(), chi.URLParam(r, "id"), status, req.ExitCode); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}
