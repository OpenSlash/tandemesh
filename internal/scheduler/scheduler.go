// Package scheduler matches queued runs to online machines, grants leases and
// re-queues runs whose lease expired.
package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/agent"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

// Scheduler is a simple pull-based scheduler loop.
type Scheduler struct {
	store           *store.Store
	hub             *hub.Hub
	leaseTTL        time.Duration
	maxAttempts     int
	maxPerWorkspace int
	maxPerAgent     int
	log             *slog.Logger
}

// New creates a scheduler.
func New(st *store.Store, h *hub.Hub, leaseTTL time.Duration, maxAttempts int, log *slog.Logger) *Scheduler {
	if leaseTTL <= 0 {
		leaseTTL = 30 * time.Second
	}
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	return &Scheduler{store: st, hub: h, leaseTTL: leaseTTL, maxAttempts: maxAttempts, log: log}
}

// SetLimits configures optional active-run quotas. Zero means unlimited.
func (s *Scheduler) SetLimits(perWorkspace, perAgent int) {
	s.maxPerWorkspace = perWorkspace
	s.maxPerAgent = perAgent
}

// Run blocks until ctx is cancelled, ticking every 500 ms.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Tick(ctx)
		}
	}
}

// Tick performs one scheduling pass. It is exported for tests.
func (s *Scheduler) Tick(ctx context.Context) {
	now := time.Now().UTC()
	s.expireApprovals(ctx, now)
	if err := s.store.ExpireAgentQuestions(ctx, now); err != nil {
		s.log.Warn("expire questions", "err", err)
	}
	s.markStaleMachines(ctx, now)

	expired, err := s.store.ExpireLeases(ctx, now, s.maxAttempts)
	if err != nil {
		s.log.Error("expire leases", "err", err)
	}
	for _, r := range expired {
		s.log.Warn("lease expired", "run", r.ID, "attempt", r.Attempt)
		_, _ = s.store.AppendEvent(ctx, r.ID, domain.EventRunLeaseExpired, map[string]any{
			"attempt": r.Attempt,
			"machine": r.MachineID,
		})
	}

	queued, err := s.store.ListRuns(ctx, "", string(domain.RunQueued), 100)
	if err != nil {
		s.log.Error("list queued", "err", err)
		return
	}
	if len(queued) == 0 {
		return
	}
	machines, err := s.store.ListMachines(ctx)
	if err != nil {
		s.log.Error("list machines", "err", err)
		return
	}

	active, err := s.store.ListRuns(ctx, "", "", 500)
	if err != nil {
		s.log.Error("list runs", "err", err)
		return
	}
	busy := map[string]int{}
	byWorkspace := map[string]int{}
	byAgent := map[string]int{}
	busySession := map[string]int{}
	for _, r := range active {
		switch r.Status {
		case domain.RunLeased, domain.RunPreparing, domain.RunRunning, domain.RunWaitingApproval:
			if r.MachineID != "" {
				busy[r.MachineID]++
			}
			byWorkspace[r.WorkspaceID]++
			if r.AgentID != "" {
				byAgent[r.AgentID]++
			}
			if r.SessionID != "" {
				busySession[r.SessionID]++
			}
		}
	}

	for _, run := range queued {
		if s.maxPerWorkspace > 0 && byWorkspace[run.WorkspaceID] >= s.maxPerWorkspace {
			continue
		}
		if s.maxPerAgent > 0 && run.AgentID != "" && byAgent[run.AgentID] >= s.maxPerAgent {
			continue
		}
		if run.SessionID != "" && busySession[run.SessionID] > 0 {
			continue
		}
		machineID := ""
		backend := run.AgentID
		var nodeLabels []string
		if agent, err := s.store.GetAgent(ctx, run.AgentID); err == nil && agent != nil {
			machineID = agent.MachineID
			if agent.Backend != "" {
				backend = agent.Backend
			}
			nodeLabels = nodeSelectors(agent.Capabilities)
		}
		m := pickMachine(machines, busy, backend, machineID, nodeLabels)
		if m == nil {
			continue // no machine for this run; try the next one
		}
		fence, ok, err := s.store.AssignRun(ctx, run.ID, m.ID, now.Add(s.leaseTTL))
		if err != nil {
			s.log.Error("assign run", "run", run.ID, "err", err)
			continue
		}
		if !ok {
			continue // lost the race
		}
		busy[m.ID]++
		byWorkspace[run.WorkspaceID]++
		if run.AgentID != "" {
			byAgent[run.AgentID]++
		}
		if run.SessionID != "" {
			busySession[run.SessionID]++
		}
		_, _ = s.store.AppendEvent(ctx, run.ID, domain.EventRunLeased, map[string]any{
			"machine": m.ID,
			"fence":   fence,
		})
		s.log.Info("run leased", "run", run.ID, "machine", m.ID, "fence", fence)

		if s.hub != nil {
			askTools := []string{}
			skillNames := []string{}
			model := run.Model
			var mcpServers []agent.MCPServer
			if run.AgentID != "" {
				if a, err := s.store.GetAgent(ctx, run.AgentID); err == nil && a != nil {
					askTools = extractAskTools(a.Tools)
					skillNames = a.Skills
					if model == "" {
						model = a.Model
					}
					mcpServers = extractMCPServers(a.Tools)
				}
			}
			sessionID := ""
			worktreePath := ""
			reasoning := ""
			if run.SessionID != "" {
				if sess, err := s.store.GetAgentSession(ctx, run.SessionID); err == nil && sess != nil {
					sessionID = sess.BackendSessionID
					reasoning = sess.Reasoning
					if sess.WorktreeID != "" {
						if wt, err := s.store.GetWorktree(ctx, sess.WorktreeID); err == nil && wt != nil {
							worktreePath = wt.Path
						}
					}
				}
			}
			dispatchID := ""
			if dispatch, err := s.store.ActiveDispatchByRun(ctx, run.ID); err == nil && dispatch != nil {
				dispatchID = dispatch.ID
				_ = s.store.StartDispatch(ctx, dispatch.ID)
			}
			payload, _ := json.Marshal(map[string]any{
				"repo":            run.Repo,
				"taskId":          run.TaskID,
				"dispatchId":      dispatchID,
				"sessionId":       sessionID,
				"sessionRecordId": run.SessionID,
				"worktreePath":    worktreePath,
				"base":            run.Base,
				"branch":          run.Branch,
				"prompt":          run.Prompt,
				"agentId":         run.AgentID,
				"askTools":        askTools,
				"skills":          skillNames,
				"model":           model,
				"reasoning":       reasoning,
				"mcpServers":      mcpServers,
			})
			s.hub.Send(m.ID, hub.Message{
				Type:    "lease.offer",
				RunID:   run.ID,
				Fence:   fence,
				Payload: payload,
			})
		}
	}
}

// expireApprovals fails approvals whose deadline passed and tells the machine.
func (s *Scheduler) expireApprovals(ctx context.Context, now time.Time) {
	items, err := s.store.ListExpiredApprovals(ctx, now)
	if err != nil {
		s.log.Error("list expired approvals", "err", err)
		return
	}
	for _, a := range items {
		ok, err := s.store.ExpireApproval(ctx, a.ID)
		if err != nil || !ok {
			continue
		}
		_, _ = s.store.AppendEvent(ctx, a.RunID, "approval.expired", map[string]any{
			"approvalId": a.ID,
			"tool":       a.Tool,
		})
		run, err := s.store.GetRun(ctx, a.RunID)
		if err != nil || run == nil || run.MachineID == "" {
			continue
		}
		payload, _ := json.Marshal(map[string]string{"approvalId": a.ID, "decision": "expired"})
		s.hub.Send(run.MachineID, hub.Message{Type: "approval.decision", RunID: run.ID, Payload: payload})
		s.log.Warn("approval expired", "approval", a.ID, "run", a.RunID)
	}
}

// extractMCPServers reads the agent's MCP configuration: {"mcp":[{name,...}]}.
func extractMCPServers(tools map[string]any) []agent.MCPServer {
	if tools == nil {
		return nil
	}
	raw, ok := tools["mcp"]
	if !ok {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var out []agent.MCPServer
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}

// extractAskTools reads the agent's approval policy: {"approval":{"ask":[...]}}.
func extractAskTools(tools map[string]any) []string {
	if tools == nil {
		return nil
	}
	raw, ok := tools["approval"].(map[string]any)
	if !ok {
		return nil
	}
	list, ok := raw["ask"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func pickMachine(machines []*domain.Machine, busy map[string]int, agentID, preferred string, required []string) *domain.Machine {
	var best *domain.Machine
	bestLoad := 0.0
	bestBusy := 0
	for _, m := range machines {
		if m.Status != "online" {
			continue
		}
		if preferred != "" && m.ID != preferred {
			continue
		}
		if time.Since(m.LastSeen) > 45*time.Second {
			continue
		}
		capacity := m.Concurrency
		if capacity <= 0 {
			capacity = 1
		}
		load := busy[m.ID]
		if load >= capacity {
			continue
		}
		if !machineSupports(m, agentID) {
			continue
		}
		if !hasAllLabels(m.Labels, required) {
			continue
		}
		ratio := float64(load) / float64(capacity)
		if best == nil || ratio < bestLoad || (ratio == bestLoad && load < bestBusy) {
			best = m
			bestLoad = ratio
			bestBusy = load
		}
	}
	return best
}

// markStaleMachines marks machines that stopped heartbeating as offline and
// re-queues their active runs so another healthy node can pick them up.
func (s *Scheduler) markStaleMachines(ctx context.Context, now time.Time) {
	cutoff := now.Add(-45 * time.Second)
	stale, err := s.store.ListStaleMachines(ctx, cutoff)
	if err != nil {
		s.log.Error("list stale machines", "err", err)
		return
	}
	for _, m := range stale {
		if err := s.store.SetMachineStatus(ctx, m.ID, "offline"); err != nil {
			s.log.Error("mark machine offline", "machine", m.ID, "err", err)
			continue
		}
		if n, err := s.store.RequeueRunsFromMachine(ctx, m.ID); err == nil && n > 0 {
			s.log.Warn("requeued runs from stale machine", "machine", m.ID, "count", n)
		}
	}
}

// nodeSelectors extracts node label constraints from agent capabilities.
// Only capabilities prefixed with "node:" participate in scheduling.
func nodeSelectors(capabilities []string) []string {
	var out []string
	for _, c := range capabilities {
		if strings.HasPrefix(c, "node:") {
			if label := strings.TrimSpace(strings.TrimPrefix(c, "node:")); label != "" {
				out = append(out, label)
			}
		}
	}
	return out
}

// hasAllLabels reports whether every required label exists on the machine.
func hasAllLabels(available, required []string) bool {
	if len(required) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(available))
	for _, l := range available {
		set[l] = struct{}{}
	}
	for _, r := range required {
		if _, ok := set[r]; !ok {
			return false
		}
	}
	return true
}
func hasAgents(available []string, wanted string) bool {
	if wanted == "" || len(available) == 0 {
		return true
	}
	for _, a := range available {
		if a == wanted {
			return true
		}
	}
	return false
}

// machineSupports checks declared backends first, then detected runtimes.
func machineSupports(m *domain.Machine, wanted string) bool {
	if wanted == "" {
		return true
	}
	if hasAgents(m.Agents, wanted) && len(m.Agents) > 0 {
		return true
	}
	if _, ok := m.Runtimes[wanted]; ok {
		return true
	}
	return len(m.Agents) == 0 && len(m.Runtimes) == 0
}
