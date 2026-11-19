package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type workflowRequest struct {
	WorkspaceID string                `json:"workspaceId"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Nodes       []domain.WorkflowNode `json:"nodes"`
}

func (s *Server) listWorkflows(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListWorkflows(r.Context(), workspaceID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var req workflowRequest
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
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeErr(w, http.StatusBadRequest, "E_NAME", "name is required")
		return
	}
	if err := validateWorkflowNodes(req.Nodes); err != nil {
		writeErr(w, http.StatusBadRequest, "E_WORKFLOW", err.Error())
		return
	}
	wf := &domain.Workflow{WorkspaceID: req.WorkspaceID, Name: req.Name, Description: req.Description, Nodes: req.Nodes}
	if err := s.store.CreateWorkflow(r.Context(), wf); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, wf)
}

func (s *Server) getWorkflow(w http.ResponseWriter, r *http.Request) {
	wf, err := s.store.GetWorkflow(r.Context(), chi.URLParam(r, "id"))
	if err != nil || wf == nil {
		writeErr(w, http.StatusNotFound, "E_WORKFLOW", "workflow not found")
		return
	}
	if !s.requireWorkspace(w, r, wf.WorkspaceID) {
		return
	}
	writeJSON(w, http.StatusOK, wf)
}

func (s *Server) startWorkflow(w http.ResponseWriter, r *http.Request) {
	wf, err := s.store.GetWorkflow(r.Context(), chi.URLParam(r, "id"))
	if err != nil || wf == nil {
		writeErr(w, http.StatusNotFound, "E_WORKFLOW", "workflow not found")
		return
	}
	if !s.requireWorkspace(w, r, wf.WorkspaceID) {
		return
	}
	var req struct {
		TaskID string `json:"taskId"`
	}
	_ = decodeJSON(w, r, &req)
	if req.TaskID != "" {
		task, err := s.store.GetTask(r.Context(), req.TaskID)
		if err != nil || task == nil || task.WorkspaceID != wf.WorkspaceID {
			writeErr(w, http.StatusBadRequest, "E_TASK", "task not found in this workspace")
			return
		}
	} else {
		for _, node := range wf.Nodes {
			if node.Kind == "agent" {
				writeErr(w, http.StatusBadRequest, "E_TASK", "taskId is required for agent nodes")
				return
			}
		}
	}
	run := &domain.WorkflowRun{
		WorkflowID: wf.ID, WorkspaceID: wf.WorkspaceID, TaskID: req.TaskID,
		Status: domain.WorkflowRunning, Nodes: initialNodeRuns(wf.Nodes),
	}
	if len(wf.Nodes) > 0 {
		run.CurrentNode = wf.Nodes[0].ID
	}
	if err := s.store.CreateWorkflowRun(r.Context(), run); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	go s.executeWorkflow(run.ID, wf.Nodes[0].ID)
	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) getWorkflowRun(w http.ResponseWriter, r *http.Request) {
	run, err := s.store.GetWorkflowRun(r.Context(), chi.URLParam(r, "id"))
	if err != nil || run == nil {
		writeErr(w, http.StatusNotFound, "E_RUN", "workflow run not found")
		return
	}
	if !s.requireWorkspace(w, r, run.WorkspaceID) {
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) listWorkflowRuns(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListWorkflowRuns(r.Context(), workspaceID, r.URL.Query().Get("task"), 100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) gateWorkflowRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID string `json:"nodeId"`
		Action string `json:"action"`
		Target string `json:"target"`
		Note   string `json:"note"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	run, err := s.store.GetWorkflowRun(r.Context(), chi.URLParam(r, "id"))
	if err != nil || run == nil {
		writeErr(w, http.StatusNotFound, "E_RUN", "workflow run not found")
		return
	}
	if !s.requireWorkspace(w, r, run.WorkspaceID) {
		return
	}
	if run.Status != domain.WorkflowWaiting {
		writeErr(w, http.StatusConflict, "E_STATE", "workflow is not waiting")
		return
	}
	if req.NodeID == "" {
		req.NodeID = run.CurrentNode
	}
	node := findNodeRun(run, req.NodeID)
	if node == nil || node.Status != domain.WorkflowNodeWaiting {
		writeErr(w, http.StatusConflict, "E_NODE", "gate is not waiting")
		return
	}
	wf, _ := s.store.GetWorkflow(r.Context(), run.WorkflowID)
	if wf == nil {
		writeErr(w, http.StatusNotFound, "E_WORKFLOW", "workflow not found")
		return
	}
	def := findNode(wf.Nodes, req.NodeID)
	if def == nil {
		writeErr(w, http.StatusBadRequest, "E_NODE", "node not found")
		return
	}
	switch req.Action {
	case "approve":
		now := time.Now().UTC()
		node.Status = domain.WorkflowNodeSucceeded
		node.Output = strings.TrimSpace(node.Output + "\n" + req.Note)
		node.FinishedAt = &now
		run.Status = domain.WorkflowRunning
		next := def.Next
		if next == "" {
			run.Status = domain.WorkflowSucceeded
			run.CurrentNode = ""
		} else {
			run.CurrentNode = next
		}
		_ = s.store.UpdateWorkflowRun(r.Context(), run)
		if next != "" {
			go s.executeWorkflow(run.ID, next)
		}
	case "reject", "retry":
		target := strings.TrimSpace(req.Target)
		if target == "" && req.Action == "retry" {
			target = req.NodeID
		}
		if target == "" {
			writeErr(w, http.StatusBadRequest, "E_TARGET", "target node is required")
			return
		}
		if findNode(wf.Nodes, target) == nil {
			writeErr(w, http.StatusBadRequest, "E_TARGET", "target node not found")
			return
		}
		resetFrom(run, wf.Nodes, target)
		run.Status = domain.WorkflowRunning
		run.CurrentNode = target
		_ = s.store.UpdateWorkflowRun(r.Context(), run)
		go s.executeWorkflow(run.ID, target)
	default:
		writeErr(w, http.StatusBadRequest, "E_ACTION", "action must be approve, reject or retry")
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) downloadWorkflowArtifact(w http.ResponseWriter, r *http.Request) {
	run, err := s.store.GetWorkflowRun(r.Context(), chi.URLParam(r, "id"))
	if err != nil || run == nil {
		writeErr(w, http.StatusNotFound, "E_RUN", "workflow run not found")
		return
	}
	if !s.requireWorkspace(w, r, run.WorkspaceID) {
		return
	}
	nodeID := chi.URLParam(r, "nodeId")
	path := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	node := findNodeRun(run, nodeID)
	if node == nil {
		writeErr(w, http.StatusNotFound, "E_NODE", "node not found")
		return
	}
	for _, artifact := range node.Artifacts {
		if artifact.Path != path {
			continue
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+artifact.Name+`"`)
		_, _ = w.Write([]byte(artifact.Content))
		return
	}
	writeErr(w, http.StatusNotFound, "E_ARTIFACT", "artifact not found")
}
func (s *Server) retryWorkflowRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID string `json:"nodeId"`
	}
	_ = decodeJSON(w, r, &req)
	run, err := s.store.GetWorkflowRun(r.Context(), chi.URLParam(r, "id"))
	if err != nil || run == nil {
		writeErr(w, http.StatusNotFound, "E_RUN", "workflow run not found")
		return
	}
	if !s.requireWorkspace(w, r, run.WorkspaceID) {
		return
	}
	if run.Status == domain.WorkflowRunning {
		writeErr(w, http.StatusConflict, "E_STATE", "workflow is already running")
		return
	}
	wf, _ := s.store.GetWorkflow(r.Context(), run.WorkflowID)
	if wf == nil {
		writeErr(w, http.StatusNotFound, "E_WORKFLOW", "workflow not found")
		return
	}
	target := req.NodeID
	if target == "" {
		target = run.CurrentNode
	}
	if target == "" {
		writeErr(w, http.StatusBadRequest, "E_NODE", "nodeId is required")
		return
	}
	if findNode(wf.Nodes, target) == nil {
		writeErr(w, http.StatusBadRequest, "E_NODE", "node not found")
		return
	}
	resetFrom(run, wf.Nodes, target)
	run.Status = domain.WorkflowRunning
	run.CurrentNode = target
	run.Error = ""
	if err := s.store.UpdateWorkflowRun(r.Context(), run); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	go s.executeWorkflow(run.ID, target)
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) cancelWorkflowRun(w http.ResponseWriter, r *http.Request) {
	run, err := s.store.GetWorkflowRun(r.Context(), chi.URLParam(r, "id"))
	if err != nil || run == nil {
		writeErr(w, http.StatusNotFound, "E_RUN", "workflow run not found")
		return
	}
	if !s.requireWorkspace(w, r, run.WorkspaceID) {
		return
	}
	if run.Status == domain.WorkflowSucceeded || run.Status == domain.WorkflowFailed || run.Status == domain.WorkflowCancelled {
		writeErr(w, http.StatusConflict, "E_STATE", "workflow is already finished")
		return
	}
	s.cancelWorkflowTree(r.Context(), run, 0)
	writeJSON(w, http.StatusOK, run)
}
func (s *Server) cancelWorkflowTree(ctx context.Context, run *domain.WorkflowRun, depth int) {
	if run == nil || depth > 8 {
		return
	}
	run.Status = domain.WorkflowCancelled
	run.CurrentNode = ""
	_ = s.store.UpdateWorkflowRun(ctx, run)
	for _, node := range run.Nodes {
		if node.RunID == "" {
			continue
		}
		switch node.Kind {
		case "agent":
			_, _ = s.store.CancelRun(ctx, node.RunID)
		case "workflow":
			child, _ := s.store.GetWorkflowRun(ctx, node.RunID)
			if child != nil && child.Status != domain.WorkflowSucceeded && child.Status != domain.WorkflowFailed && child.Status != domain.WorkflowCancelled {
				s.cancelWorkflowTree(ctx, child, depth+1)
			}
		}
	}
}
func (s *Server) executeWorkflow(runID, start string) {
	s.workflowMu.Lock()
	defer s.workflowMu.Unlock()
	run, err := s.store.GetWorkflowRun(context.Background(), runID)
	if err != nil || run == nil {
		return
	}
	wf, _ := s.store.GetWorkflow(context.Background(), run.WorkflowID)
	if wf == nil {
		return
	}
	current := start
	for current != "" {
		def := findNode(wf.Nodes, current)
		if def == nil {
			run.Status = domain.WorkflowFailed
			run.Error = "node not found: " + current
			_ = s.store.UpdateWorkflowRun(context.Background(), run)
			return
		}
		node := findNodeRun(run, current)
		if node == nil {
			run.Status = domain.WorkflowFailed
			run.Error = "node state not found: " + current
			_ = s.store.UpdateWorkflowRun(context.Background(), run)
			return
		}
		now := time.Now().UTC()
		node.Status = domain.WorkflowNodeRunning
		node.StartedAt = &now
		run.CurrentNode = current
		run.Status = domain.WorkflowRunning
		_ = s.store.UpdateWorkflowRun(context.Background(), run)

		switch def.Kind {
		case "script":
			artifactDir := s.workflowArtifactDir(run.ID, current)
			attempts := def.Retries + 1
			if attempts < 1 {
				attempts = 1
			}
			var out string
			var err error
			for attempt := 0; attempt < attempts; attempt++ {
				out, err = runScript(def.Command, artifactDir)
				if err == nil {
					break
				}
			}
			finish := time.Now().UTC()
			node.FinishedAt = &finish
			node.Output = out
			node.Artifacts = collectArtifacts(artifactDir)
			if err != nil {
				node.Status = domain.WorkflowNodeFailed
				node.Error = err.Error()
				if def.OnFail != "" {
					current = def.OnFail
					run.Status = domain.WorkflowRunning
				} else {
					run.Status = domain.WorkflowFailed
					run.Error = err.Error()
					_ = s.store.UpdateWorkflowRun(context.Background(), run)
					return
				}
			} else {
				node.Status = domain.WorkflowNodeSucceeded
				current = def.Next
			}
		case "agent":
			task, err := s.store.GetTask(context.Background(), run.TaskID)
			if err != nil || task == nil {
				node.Status = domain.WorkflowNodeFailed
				node.Error = "task is required for agent nodes"
				run.Status = domain.WorkflowFailed
				run.Error = node.Error
				_ = s.store.UpdateWorkflowRun(context.Background(), run)
				return
			}
			prompt := strings.TrimSpace(def.Prompt)
			if prompt == "" {
				prompt, _ = s.taskPrompt(context.Background(), task, "")
			}
			role := def.Role
			if role == "" {
				role = "lead"
			}
			agentRun, err := s.createTaskRunModel(context.Background(), task, def.AgentID, role, "", "", prompt, def.Model)
			if err != nil {
				node.Status = domain.WorkflowNodeFailed
				node.Error = err.Error()
				run.Status = domain.WorkflowFailed
				run.Error = err.Error()
				_ = s.store.UpdateWorkflowRun(context.Background(), run)
				return
			}
			node.RunID = agentRun.ID
			node.Status = domain.WorkflowNodeRunning
			run.Status = domain.WorkflowWaiting
			_ = s.store.UpdateWorkflowRun(context.Background(), run)
			go s.watchWorkflowAgent(run.ID, current, agentRun.ID)
			return
		case "workflow":
			childWF, err := s.store.GetWorkflow(context.Background(), def.WorkflowID)
			if err != nil || childWF == nil || childWF.WorkspaceID != run.WorkspaceID {
				node.Status = domain.WorkflowNodeFailed
				node.Error = "child workflow not found"
				run.Status = domain.WorkflowFailed
				run.Error = node.Error
				_ = s.store.UpdateWorkflowRun(context.Background(), run)
				return
			}
			child := &domain.WorkflowRun{WorkflowID: childWF.ID, WorkspaceID: run.WorkspaceID, TaskID: run.TaskID, Status: domain.WorkflowRunning, Nodes: initialNodeRuns(childWF.Nodes)}
			if len(childWF.Nodes) > 0 {
				child.CurrentNode = childWF.Nodes[0].ID
			}
			if err := s.store.CreateWorkflowRun(context.Background(), child); err != nil {
				node.Status = domain.WorkflowNodeFailed
				node.Error = err.Error()
				run.Status = domain.WorkflowFailed
				run.Error = err.Error()
				_ = s.store.UpdateWorkflowRun(context.Background(), run)
				return
			}
			node.RunID = child.ID
			node.Status = domain.WorkflowNodeRunning
			run.Status = domain.WorkflowWaiting
			_ = s.store.UpdateWorkflowRun(context.Background(), run)
			go s.executeWorkflow(child.ID, childWF.Nodes[0].ID)
			go s.watchWorkflowChild(run.ID, current, child.ID)
			return
		case "condition":
			matched := conditionMatches(def.When, previousOutput(run, current))
			node.Status = domain.WorkflowNodeSucceeded
			node.Output = fmt.Sprintf("condition=%t", matched)
			if matched {
				current = def.Next
			} else {
				current = def.OnFail
			}
		case "gate":
			node.Status = domain.WorkflowNodeWaiting
			run.Status = domain.WorkflowWaiting
			_ = s.store.UpdateWorkflowRun(context.Background(), run)
			return
		default:
			node.Status = domain.WorkflowNodeFailed
			node.Error = "unsupported node kind: " + def.Kind
			run.Status = domain.WorkflowFailed
			_ = s.store.UpdateWorkflowRun(context.Background(), run)
			return
		}
		if current == "" {
			run.Status = domain.WorkflowSucceeded
			run.CurrentNode = ""
		}
		_ = s.store.UpdateWorkflowRun(context.Background(), run)
	}
}

func (s *Server) watchWorkflowAgent(runID, nodeID, agentRunID string) {
	deadline := time.Now().Add(30 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		agentRun, err := s.store.GetRun(context.Background(), agentRunID)
		if err != nil || agentRun == nil {
			continue
		}
		if agentRun.Status != domain.RunSucceeded && agentRun.Status != domain.RunFailed {
			continue
		}
		s.workflowMu.Lock()
		run, _ := s.store.GetWorkflowRun(context.Background(), runID)
		if run == nil {
			s.workflowMu.Unlock()
			return
		}
		wf, _ := s.store.GetWorkflow(context.Background(), run.WorkflowID)
		if wf == nil {
			s.workflowMu.Unlock()
			return
		}
		node := findNodeRun(run, nodeID)
		def := findNode(wf.Nodes, nodeID)
		if node == nil || def == nil {
			s.workflowMu.Unlock()
			return
		}
		now := time.Now().UTC()
		node.FinishedAt = &now
		next := def.Next
		if agentRun.Status == domain.RunSucceeded {
			node.Status = domain.WorkflowNodeSucceeded
			node.Output = "agent run " + agentRun.ID + " succeeded"
			node.Artifacts = s.agentRunArtifacts(context.Background(), agentRun.ID)
			run.CurrentNode = def.Next
			run.Status = domain.WorkflowRunning
		} else {
			node.Status = domain.WorkflowNodeFailed
			node.Error = "agent run failed"
			if def.OnFail != "" {
				next = def.OnFail
				run.Status = domain.WorkflowRunning
			} else {
				run.Status = domain.WorkflowFailed
				run.Error = node.Error
			}
		}
		if next == "" {
			next = def.Next
		}
		if run.Status == domain.WorkflowSucceeded || (run.Status == domain.WorkflowRunning && next == "") {
			run.Status = domain.WorkflowSucceeded
			run.CurrentNode = ""
		}
		_ = s.store.UpdateWorkflowRun(context.Background(), run)
		s.workflowMu.Unlock()
		if run.Status == domain.WorkflowRunning && next != "" {
			s.executeWorkflow(runID, next)
		}
		return
	}
}
func (s *Server) watchWorkflowChild(runID, nodeID, childID string) {
	deadline := time.Now().Add(2 * time.Hour)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		child, err := s.store.GetWorkflowRun(context.Background(), childID)
		if err != nil || child == nil {
			continue
		}
		if child.Status != domain.WorkflowSucceeded && child.Status != domain.WorkflowFailed && child.Status != domain.WorkflowCancelled {
			continue
		}
		s.workflowMu.Lock()
		run, _ := s.store.GetWorkflowRun(context.Background(), runID)
		if run == nil {
			s.workflowMu.Unlock()
			return
		}
		wf, _ := s.store.GetWorkflow(context.Background(), run.WorkflowID)
		if wf == nil {
			s.workflowMu.Unlock()
			return
		}
		node := findNodeRun(run, nodeID)
		def := findNode(wf.Nodes, nodeID)
		if node == nil || def == nil {
			s.workflowMu.Unlock()
			return
		}
		now := time.Now().UTC()
		node.FinishedAt = &now
		next := def.Next
		if child.Status == domain.WorkflowSucceeded {
			node.Status = domain.WorkflowNodeSucceeded
			node.Output = "child workflow " + child.ID + " succeeded"
			run.CurrentNode = next
			run.Status = domain.WorkflowRunning
		} else {
			node.Status = domain.WorkflowNodeFailed
			node.Error = "child workflow " + child.Status
			if def.OnFail != "" {
				next = def.OnFail
				run.Status = domain.WorkflowRunning
			} else {
				run.Status = domain.WorkflowFailed
				run.Error = node.Error
			}
		}
		if run.Status == domain.WorkflowRunning && next == "" {
			run.Status = domain.WorkflowSucceeded
			run.CurrentNode = ""
		}
		_ = s.store.UpdateWorkflowRun(context.Background(), run)
		s.workflowMu.Unlock()
		if run.Status == domain.WorkflowRunning && next != "" {
			s.executeWorkflow(runID, next)
		}
		return
	}
}
func previousOutput(run *domain.WorkflowRun, nodeID string) string {
	last := ""
	for _, node := range run.Nodes {
		if node.NodeID == nodeID {
			return last
		}
		if node.Status == domain.WorkflowNodeSucceeded {
			last = node.Output
		}
	}
	return last
}

func conditionMatches(expr, input string) bool {
	expr = strings.TrimSpace(expr)
	parts := strings.SplitN(expr, ":", 2)
	if len(parts) != 2 {
		return false
	}
	op := strings.ToLower(strings.TrimSpace(parts[0]))
	want := strings.TrimSpace(parts[1])
	got := strings.ToLower(input)
	wantLower := strings.ToLower(want)
	switch op {
	case "contains":
		return strings.Contains(got, wantLower)
	case "not_contains":
		return !strings.Contains(got, wantLower)
	case "equals":
		return strings.EqualFold(strings.TrimSpace(input), want)
	case "prefix":
		return strings.HasPrefix(got, wantLower)
	case "suffix":
		return strings.HasSuffix(got, wantLower)
	case "nonempty":
		return strings.TrimSpace(input) != ""
	default:
		return false
	}
}
func (s *Server) agentRunArtifacts(ctx context.Context, runID string) []domain.WorkflowArtifact {
	events, err := s.store.ListEventsTail(ctx, runID, 500)
	if err != nil {
		return nil
	}
	var out strings.Builder
	for _, event := range events {
		if event.Type != "agent.message.complete" {
			continue
		}
		if role, _ := event.Payload["role"].(string); role != "assistant" {
			continue
		}
		if text, _ := event.Payload["text"].(string); text != "" {
			if out.Len() > 0 {
				out.WriteString("\n\n")
			}
			out.WriteString(text)
		}
	}
	content := out.String()
	if content == "" {
		return nil
	}
	if len(content) > 64*1024 {
		content = content[len(content)-64*1024:]
	}
	return []domain.WorkflowArtifact{{Name: "agent-output.md", Path: "agent-output.md", Size: len(content), Content: content}}
}
func (s *Server) workflowArtifactDir(runID, nodeID string) string {
	dir := filepath.Join(filepath.Dir(s.cfg.DBPath), "artifacts", runID, nodeID)
	_ = os.MkdirAll(dir, 0o755)
	return dir
}
func collectArtifacts(dir string) []domain.WorkflowArtifact {
	var out []domain.WorkflowArtifact
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		info, infoErr := entry.Info()
		if infoErr != nil || info.Size() > 64*1024 {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		out = append(out, domain.WorkflowArtifact{Name: entry.Name(), Path: rel, Size: int(info.Size()), Content: string(raw)})
		return nil
	})
	return out
}
func runScript(command, artifactDir string) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("command is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	cmd.Env = append(os.Environ(), "TANDEMESH_ARTIFACT_DIR="+artifactDir)
	out, err := cmd.CombinedOutput()
	text := string(out)
	if len(text) > 64*1024 {
		text = text[len(text)-64*1024:]
	}
	if err != nil {
		return text, fmt.Errorf("%w: %s", err, strings.TrimSpace(text))
	}
	return text, nil
}

func validateWorkflowNodes(nodes []domain.WorkflowNode) error {
	if len(nodes) == 0 {
		return fmt.Errorf("at least one node is required")
	}
	seen := map[string]bool{}
	for _, node := range nodes {
		if strings.TrimSpace(node.ID) == "" {
			return fmt.Errorf("node id is required")
		}
		if seen[node.ID] {
			return fmt.Errorf("duplicate node id: %s", node.ID)
		}
		seen[node.ID] = true
		switch node.Kind {
		case "script":
			if strings.TrimSpace(node.Command) == "" {
				return fmt.Errorf("script node %s needs a command", node.ID)
			}
		case "gate":
		case "condition":
			if strings.TrimSpace(node.When) == "" {
				return fmt.Errorf("condition node %s needs a when expression", node.ID)
			}
		case "workflow":
			if strings.TrimSpace(node.WorkflowID) == "" {
				return fmt.Errorf("workflow node %s needs a workflowId", node.ID)
			}
		case "agent":
			if strings.TrimSpace(node.AgentID) == "" {
				return fmt.Errorf("agent node %s needs an agentId", node.ID)
			}
		default:
			return fmt.Errorf("unsupported node kind: %s", node.Kind)
		}
	}
	for _, node := range nodes {
		if node.Next != "" && !seen[node.Next] {
			return fmt.Errorf("node %s references missing next node %s", node.ID, node.Next)
		}
		if node.OnFail != "" && !seen[node.OnFail] {
			return fmt.Errorf("node %s references missing onFail node %s", node.ID, node.OnFail)
		}
	}
	return nil
}

func initialNodeRuns(nodes []domain.WorkflowNode) []domain.WorkflowNodeRun {
	out := make([]domain.WorkflowNodeRun, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, domain.WorkflowNodeRun{NodeID: node.ID, Kind: node.Kind, Name: node.Name, Status: domain.WorkflowNodePending})
	}
	return out
}

func findNode(nodes []domain.WorkflowNode, id string) *domain.WorkflowNode {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

func findNodeRun(run *domain.WorkflowRun, id string) *domain.WorkflowNodeRun {
	for i := range run.Nodes {
		if run.Nodes[i].NodeID == id {
			return &run.Nodes[i]
		}
	}
	return nil
}

func resetFrom(run *domain.WorkflowRun, nodes []domain.WorkflowNode, target string) {
	started := false
	for i := range run.Nodes {
		if run.Nodes[i].NodeID == target {
			started = true
		}
		if started {
			run.Nodes[i].Status = domain.WorkflowNodePending
			run.Nodes[i].Output = ""
			run.Nodes[i].Error = ""
			run.Nodes[i].RunID = ""
			run.Nodes[i].StartedAt = nil
			run.Nodes[i].FinishedAt = nil
		}
	}
	run.Error = ""
}
