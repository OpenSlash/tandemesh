package main

import (
	"context"
	"log/slog"
	"sort"
	"sync"

	"github.com/OpenSlash/tandemesh/internal/agent"
	"github.com/OpenSlash/tandemesh/internal/domain"
)

// runtimeHost owns what this machine can run. Detection runs at startup and
// again when the plane asks for a check or after an ACP install; there is no
// background polling.
type runtimeHost struct {
	acpHome     string
	custom      map[string]agent.ACPBackendSpec
	overrides   map[string]agent.ACPBackendSpec
	fixedAgents []string
	// router receives the resolved launchers; nil unless runner=acp.
	router *agent.Router
	log    *slog.Logger

	mu       sync.Mutex
	runtimes map[string]string
	details  []domain.RuntimeStatus
	agents   []string
}

// refresh re-detects runtimes and swaps the router's backends.
func (h *runtimeHost) refresh(ctx context.Context) {
	installs, err := agent.ListACPInstalls(h.acpHome)
	if err != nil {
		h.log.Warn("list ACP registry installs", "err", err)
	}
	det := agent.DetectRuntimes(ctx, agent.DetectOptions{Installs: installs, Custom: h.custom, Overrides: h.overrides})

	runtimes := map[string]string{}
	for _, st := range det.Statuses {
		if st.Status == domain.RuntimeReady {
			version := st.Version
			if version == "" {
				version = "installed"
			}
			runtimes[st.ID] = version
		}
	}
	agents := h.fixedAgents
	if len(agents) == 0 {
		agents = make([]string, 0, len(runtimes))
		for id := range runtimes {
			agents = append(agents, id)
		}
		sort.Strings(agents)
	}
	if h.router != nil {
		byBackend := make(map[string]agent.Runner, len(det.Backends))
		for id, spec := range det.Backends {
			byBackend[id] = agent.NewACP(agent.ACPConfig{Command: spec.Command, Args: spec.Args, Env: spec.Env, Logger: h.log})
		}
		h.router.SetBackends(byBackend)
	}

	h.mu.Lock()
	h.runtimes, h.details, h.agents = runtimes, det.Statuses, agents
	h.mu.Unlock()
	h.log.Info("runtimes detected", "ready", len(runtimes), "known", len(det.Statuses))
}

// report returns the fields the plane stores for this host.
func (h *runtimeHost) report() map[string]any {
	h.mu.Lock()
	defer h.mu.Unlock()
	return map[string]any{"runtimes": h.runtimes, "runtimeDetails": h.details, "agents": h.agents}
}
