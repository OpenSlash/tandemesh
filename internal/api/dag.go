package api

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func errorf(format string, args ...any) error { return fmt.Errorf(format, args...) }
func ensureConditionNeeds(nodes []domain.WorkflowNode) []domain.WorkflowNode {
	byID := make(map[string]int, len(nodes))
	for i := range nodes {
		byID[nodes[i].ID] = i
	}
	for _, node := range nodes {
		if node.Kind != "condition" {
			continue
		}
		for _, target := range []string{node.Next, node.OnFail} {
			if target == "" {
				continue
			}
			index, ok := byID[target]
			if !ok {
				continue
			}
			found := false
			for _, need := range nodes[index].Needs {
				if need == node.ID {
					found = true
				}
			}
			if !found {
				nodes[index].Needs = append(nodes[index].Needs, node.ID)
			}
		}
	}
	return nodes
}
func (s *Server) lockWorkflowRun(runID string) func() {
	value, _ := s.workflowLocks.LoadOrStore(runID, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}
func (s *Server) executeDAG(runID string) {
	for {
		unlock := s.lockWorkflowRun(runID)
		run, _ := s.store.GetWorkflowRun(context.Background(), runID)
		if run == nil || run.Status == domain.WorkflowSucceeded || run.Status == domain.WorkflowFailed || run.Status == domain.WorkflowCancelled {
			unlock()
			return
		}
		wf := s.workflowForRun(context.Background(), run)
		if wf == nil {
			unlock()
			return
		}
		wf.Nodes = ensureConditionNeeds(wf.Nodes)
		ready := make([]string, 0)
		pending, waiting := 0, 0
		for _, node := range wf.Nodes {
			state := findNodeRun(run, node.ID)
			if state == nil {
				continue
			}
			switch state.Status {
			case domain.WorkflowNodeWaiting:
				waiting++
				continue
			case domain.WorkflowNodePending:
				skipped := false
				for _, need := range node.Needs {
					parent := findNodeRun(run, need)
					if parent != nil && parent.Status == domain.WorkflowNodeSkipped {
						state.Status = domain.WorkflowNodeSkipped
						skipped = true
						break
					}
				}
				if skipped {
					continue
				}
				pending++
			default:
				continue
			}
			ok := true
			skipped := false
			for _, need := range node.Needs {
				parent := findNodeRun(run, need)
				if parent == nil {
					ok = false
					break
				}
				if parent.Status == domain.WorkflowNodeSucceeded {
					continue
				}
				if parent.Status == domain.WorkflowNodeFailed {
					parentDef := findNode(wf.Nodes, need)
					if parentDef != nil && parentDef.OnFail == node.ID {
						continue
					}
					state.Status = domain.WorkflowNodeSkipped
					skipped = true
					break
				}
				ok = false
				break
			}
			if skipped {
				continue
			}
			if ok {
				ready = append(ready, node.ID)
			}
		}
		if len(ready) == 0 {
			if pending > 0 {
				run.Status = domain.WorkflowFailed
				run.Error = "workflow dependencies cannot make progress"
			} else if waiting > 0 {
				run.Status = domain.WorkflowWaiting
			} else {
				run.Status = domain.WorkflowSucceeded
				run.CurrentNode = ""
			}
			_ = s.store.UpdateWorkflowRun(context.Background(), run)
			unlock()
			return
		}
		for _, id := range ready {
			node := findNodeRun(run, id)
			node.Status = domain.WorkflowNodeRunning
			now := time.Now().UTC()
			node.StartedAt = &now
		}
		run.CurrentNode = ready[0]
		run.Status = domain.WorkflowRunning
		_ = s.store.UpdateWorkflowRun(context.Background(), run)
		unlock()

		var wg sync.WaitGroup
		for _, id := range ready {
			wg.Add(1)
			go func(nodeID string) {
				defer wg.Done()
				s.runDAGNode(runID, nodeID)
			}(id)
		}
		wg.Wait()
	}
}

func (s *Server) runDAGNode(runID, nodeID string) {
	run, _ := s.store.GetWorkflowRun(context.Background(), runID)
	if run == nil {
		return
	}
	wf := s.workflowForRun(context.Background(), run)
	if wf == nil {
		return
	}
	def := findNode(wf.Nodes, nodeID)
	node := findNodeRun(run, nodeID)
	if def == nil || node == nil {
		return
	}
	switch def.Kind {
	case "script":
		artifactDir := s.workflowArtifactDir(runID, nodeID)
		timeout := 10 * time.Minute
		if def.TimeoutSeconds > 0 {
			timeout = time.Duration(def.TimeoutSeconds) * time.Second
		}
		attempts := def.Retries + 1
		if attempts < 1 {
			attempts = 1
		}
		var out string
		var err error
		for attempt := 0; attempt < attempts; attempt++ {
			if attempt > 0 && def.RetryDelaySeconds > 0 {
				time.Sleep(time.Duration(def.RetryDelaySeconds) * time.Second)
			}
			out, err = runScript(def.Command, artifactDir, timeout)
			if err == nil {
				break
			}
		}
		artifacts := s.persistArtifacts(context.Background(), runID, nodeID, collectArtifacts(artifactDir))
		if err != nil && def.OnFail != "" {
			s.finishDAGNodeContinue(runID, nodeID, out, artifacts, err.Error())
		} else {
			s.finishDAGNode(runID, nodeID, out, artifacts, err)
		}
	case "merge":
		s.finishDAGNode(runID, nodeID, mergeInput(run, def), nil, nil)
	case "condition":
		source := mergeInput(run, def)
		matched := conditionMatches(def.When, source)
		selected, unselected := def.OnFail, def.Next
		if matched {
			selected, unselected = def.Next, def.OnFail
		}
		_ = selected
		unlock := s.lockWorkflowRun(runID)
		if current, _ := s.store.GetWorkflowRun(context.Background(), runID); current != nil {
			if state := findNodeRun(current, nodeID); state != nil {
				state.Status = domain.WorkflowNodeSucceeded
				state.Output = "condition=" + fmt.Sprintf("%t", matched)
			}
			if unselected != "" {
				if state := findNodeRun(current, unselected); state != nil && state.Status == domain.WorkflowNodePending {
					state.Status = domain.WorkflowNodeSkipped
				}
			}
			_ = s.store.UpdateWorkflowRun(context.Background(), current)
		}
		unlock()
		return
	case "parallel":
		artifactDir := s.workflowArtifactDir(runID, nodeID)
		timeout := 10 * time.Minute
		if def.TimeoutSeconds > 0 {
			timeout = time.Duration(def.TimeoutSeconds) * time.Second
		}
		type result struct {
			index int
			out   string
			err   error
		}
		ch := make(chan result, len(def.Commands))
		for i, command := range def.Commands {
			go func(index int, cmd string) {
				out, err := runScript(cmd, artifactDir, timeout)
				ch <- result{index: index, out: out, err: err}
			}(i, command)
		}
		outputs := make([]string, len(def.Commands))
		var firstErr error
		for range def.Commands {
			item := <-ch
			outputs[item.index] = item.out
			if item.err != nil && firstErr == nil {
				firstErr = item.err
			}
		}
		artifacts := s.persistArtifacts(context.Background(), runID, nodeID, collectArtifacts(artifactDir))
		if firstErr != nil {
			s.finishDAGFailure(runID, nodeID, def, strings.Join(outputs, "\n"), artifacts, firstErr)
		} else {
			s.finishDAGNode(runID, nodeID, strings.Join(outputs, "\n"), artifacts, nil)
		}
	case "gate":
		unlock := s.lockWorkflowRun(runID)
		current, _ := s.store.GetWorkflowRun(context.Background(), runID)
		if current != nil {
			state := findNodeRun(current, nodeID)
			if state != nil {
				state.Status = domain.WorkflowNodeWaiting
			}
			current.Status = domain.WorkflowWaiting
			current.CurrentNode = nodeID
			_ = s.store.UpdateWorkflowRun(context.Background(), current)
		}
		unlock()
	case "agent":
		s.runDAGAgent(runID, nodeID, def)
	case "fanout":
		s.runDAGFanout(runID, nodeID, def)
	case "workflow":
		s.runDAGChildWorkflow(runID, nodeID, def)
	default:
		s.finishDAGNode(runID, nodeID, "", nil, errorf("unsupported DAG node kind: %s", def.Kind))
	}
}

func (s *Server) runDAGChildWorkflow(runID, nodeID string, def *domain.WorkflowNode) {
	parent, _ := s.store.GetWorkflowRun(context.Background(), runID)
	if parent == nil {
		return
	}
	childWF, err := s.store.GetWorkflow(context.Background(), def.WorkflowID)
	if err != nil || childWF == nil || childWF.WorkspaceID != parent.WorkspaceID {
		s.finishDAGFailure(runID, nodeID, def, "", nil, errorf("child workflow not found"))
		return
	}
	child := &domain.WorkflowRun{
		WorkflowID: childWF.ID, WorkspaceID: parent.WorkspaceID, TaskID: parent.TaskID,
		Status: domain.WorkflowRunning, Nodes: initialNodeRuns(childWF.Nodes),
	}
	if len(childWF.Nodes) > 0 {
		child.CurrentNode = childWF.Nodes[0].ID
	}
	if err := s.store.CreateWorkflowRun(context.Background(), child); err != nil {
		s.finishDAGFailure(runID, nodeID, def, "", nil, err)
		return
	}
	unlock := s.lockWorkflowRun(runID)
	if current, _ := s.store.GetWorkflowRun(context.Background(), runID); current != nil {
		if state := findNodeRun(current, nodeID); state != nil {
			state.RunID = child.ID
		}
		_ = s.store.UpdateWorkflowRun(context.Background(), current)
	}
	unlock()
	go s.executeWorkflow(child.ID, childWF.Nodes[0].ID)

	deadline := time.Now().Add(2 * time.Hour)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		current, _ := s.store.GetWorkflowRun(context.Background(), child.ID)
		if current == nil {
			continue
		}
		if current.Status == domain.WorkflowSucceeded {
			s.finishDAGNode(runID, nodeID, "child workflow "+child.ID+" succeeded", nil, nil)
			return
		}
		if current.Status == domain.WorkflowFailed || current.Status == domain.WorkflowCancelled {
			s.finishDAGNode(runID, nodeID, "", nil, errorf("child workflow %s", current.Status))
			return
		}
	}
	s.finishDAGFailure(runID, nodeID, def, "", nil, errorf("child workflow timed out"))
}
func (s *Server) finishDAGFailure(runID, nodeID string, def *domain.WorkflowNode, output string, artifacts []domain.WorkflowArtifact, err error) {
	if def != nil && def.OnFail != "" {
		s.finishDAGNodeContinue(runID, nodeID, output, artifacts, err.Error())
		return
	}
	s.finishDAGNode(runID, nodeID, output, artifacts, err)
}
func (s *Server) runDAGAgent(runID, nodeID string, def *domain.WorkflowNode) {
	run, _ := s.store.GetWorkflowRun(context.Background(), runID)
	task, _ := s.store.GetTask(context.Background(), run.TaskID)
	if task == nil {
		s.finishDAGFailure(runID, nodeID, def, "", nil, errorf("task is required for agent nodes"))
		return
	}
	prompt := def.Prompt
	if prompt == "" {
		prompt, _ = s.taskPrompt(context.Background(), task, "")
	}
	role := def.Role
	if role == "" {
		role = "lead"
	}
	agentRun, err := s.createTaskRunModel(context.Background(), task, def.AgentID, role, "", "", prompt, def.Model)
	if err != nil {
		s.finishDAGFailure(runID, nodeID, def, "", nil, err)
		return
	}
	unlock := s.lockWorkflowRun(runID)
	if current, _ := s.store.GetWorkflowRun(context.Background(), runID); current != nil {
		if state := findNodeRun(current, nodeID); state != nil {
			state.RunID = agentRun.ID
		}
		_ = s.store.UpdateWorkflowRun(context.Background(), current)
	}
	unlock()

	timeout := 30 * time.Minute
	if def.TimeoutSeconds > 0 {
		timeout = time.Duration(def.TimeoutSeconds) * time.Second
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		child, _ := s.store.GetRun(context.Background(), agentRun.ID)
		if child == nil {
			continue
		}
		if child.Status == domain.RunSucceeded {
			artifacts := s.persistArtifacts(context.Background(), runID, nodeID, s.agentRunArtifacts(context.Background(), agentRun.ID))
			s.finishDAGNode(runID, nodeID, "agent run "+agentRun.ID+" succeeded", artifacts, nil)
			return
		}
		if child.Status == domain.RunFailed || child.Status == domain.RunCancelled {
			s.finishDAGNode(runID, nodeID, "", nil, errorf("agent run %s", child.Status))
			return
		}
	}
	_, _ = s.store.CancelRun(context.Background(), agentRun.ID)
	s.finishDAGFailure(runID, nodeID, def, "", nil, errorf("agent node timed out"))
}

func (s *Server) runDAGFanout(runID, nodeID string, def *domain.WorkflowNode) {
	run, _ := s.store.GetWorkflowRun(context.Background(), runID)
	task, _ := s.store.GetTask(context.Background(), run.TaskID)
	if task == nil {
		s.finishDAGFailure(runID, nodeID, def, "", nil, errorf("task is required for fanout nodes"))
		return
	}
	agentIDs := append([]string{}, def.AgentIDs...)
	for _, spec := range def.Agents {
		agentIDs = append(agentIDs, spec.AgentID)
	}
	if len(agentIDs) == 0 {
		s.finishDAGFailure(runID, nodeID, def, "", nil, errorf("fanout needs agents"))
		return
	}
	prompt, _ := s.taskPrompt(context.Background(), task, def.Prompt)
	runIDs := make([]string, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		child, err := s.createTaskRunModel(context.Background(), task, agentID, "lead", "", "", prompt, def.Model)
		if err != nil {
			s.finishDAGFailure(runID, nodeID, def, "", nil, err)
			return
		}
		runIDs = append(runIDs, child.ID)
	}
	unlock := s.lockWorkflowRun(runID)
	if current, _ := s.store.GetWorkflowRun(context.Background(), runID); current != nil {
		if state := findNodeRun(current, nodeID); state != nil {
			state.ChildRunIDs = runIDs
		}
		_ = s.store.UpdateWorkflowRun(context.Background(), current)
	}
	unlock()

	deadline := time.Now().Add(30 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		done, failed := 0, 0
		for _, id := range runIDs {
			child, _ := s.store.GetRun(context.Background(), id)
			if child == nil {
				continue
			}
			if child.Status == domain.RunSucceeded {
				done++
			} else if child.Status == domain.RunFailed || child.Status == domain.RunCancelled {
				done++
				failed++
			}
		}
		if done < len(runIDs) {
			continue
		}
		if failed > 0 {
			s.finishDAGFailure(runID, nodeID, def, "", nil, errorf("one or more fanout agents failed"))
			return
		}
		var artifacts []domain.WorkflowArtifact
		for i, id := range runIDs {
			for _, artifact := range s.agentRunArtifacts(context.Background(), id) {
				artifact.Name = "agent-" + string(rune('1'+i)) + "-" + artifact.Name
				artifact.Path = artifact.Name
				artifacts = append(artifacts, artifact)
			}
		}
		artifacts = s.persistArtifacts(context.Background(), runID, nodeID, artifacts)
		s.finishDAGNode(runID, nodeID, "agents completed", artifacts, nil)
		return
	}
	s.finishDAGFailure(runID, nodeID, def, "", nil, errorf("fanout timed out"))
}
func (s *Server) finishDAGNodeContinue(runID, nodeID, output string, artifacts []domain.WorkflowArtifact, message string) {
	unlock := s.lockWorkflowRun(runID)
	defer unlock()
	run, _ := s.store.GetWorkflowRun(context.Background(), runID)
	if run == nil {
		return
	}
	node := findNodeRun(run, nodeID)
	if node == nil {
		return
	}
	now := time.Now().UTC()
	node.Status = domain.WorkflowNodeFailed
	node.Error = message
	node.Output = output
	node.FinishedAt = &now
	if artifacts != nil {
		node.Artifacts = artifacts
	}
	run.Status = domain.WorkflowRunning
	_ = s.store.UpdateWorkflowRun(context.Background(), run)
}
