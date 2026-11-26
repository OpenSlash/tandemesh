// Command tandemesh-agentd runs on each execution machine.
//
// M0 keeps the transport deliberately simple: the daemon registers over HTTP,
// heartbeats, long-polls for a lease and reports events. The WSS transport and
// real ACP execution land in M1.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/agent"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/plugin"
	"github.com/OpenSlash/tandemesh/internal/skills"
	"github.com/OpenSlash/tandemesh/internal/workspace"
)

type machine struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Labels      []string `json:"labels"`
	Agents      []string `json:"agents"`
	Concurrency int      `json:"concurrency"`
	Status      string   `json:"status"`
}

type run struct {
	ID              string            `json:"id"`
	Status          string            `json:"status"`
	Repo            string            `json:"repo"`
	Base            string            `json:"base"`
	Branch          string            `json:"branch"`
	Prompt          string            `json:"prompt"`
	SessionID       string            `json:"sessionId"`
	SessionRecordID string            `json:"sessionRecordId"`
	WorktreePath    string            `json:"worktreePath"`
	AgentID         string            `json:"agentId"`
	Backend         string            `json:"backend"`
	AskTools        []string          `json:"askTools"`
	Skills          []string          `json:"skills"`
	MCPServers      []agent.MCPServer `json:"mcpServers"`
	Model           string            `json:"model,omitempty"`
	Reasoning       string            `json:"reasoning,omitempty"`
	AgentEnv        map[string]string `json:"agentEnv"`
	Fence           int64             `json:"fence"`
}

func main() {
	plane := flag.String("plane", env("TANDEMESH_PLANE_URL", "http://127.0.0.1:8080"), "control plane URL")
	token := flag.String("token", env("TANDEMESH_JOIN_TOKEN", "dev-join-token"), "join token")
	name := flag.String("name", hostname(), "machine name")
	labels := flag.String("labels", "os=windows", "comma separated labels")
	agents := flag.String("agents", "", "comma separated agent backends (empty = auto-detect)")
	concurrency := flag.Int("concurrency", 2, "parallel slots")
	runnerMode := flag.String("runner", env("TANDEMESH_RUNNER", "mock"), "runner backend: mock|acp|grpc")
	acpCommand := flag.String("acp-command", env("TANDEMESH_ACP_COMMAND", "claude-agent-acp"), "ACP agent command")
	acpArgs := flag.String("acp-args", env("TANDEMESH_ACP_ARGS", ""), "space separated ACP arguments")
	claudeACPCommand := flag.String("claude-acp-command", env("TANDEMESH_CLAUDE_ACP_COMMAND", ""), "Claude ACP command (defaults to -acp-command)")
	claudeACPArgs := flag.String("claude-acp-args", env("TANDEMESH_CLAUDE_ACP_ARGS", ""), "space separated Claude ACP arguments")
	codexACPCommand := flag.String("codex-acp-command", env("TANDEMESH_CODEX_ACP_COMMAND", ""), "Codex ACP command")
	codexACPArgs := flag.String("codex-acp-args", env("TANDEMESH_CODEX_ACP_ARGS", ""), "space separated Codex ACP arguments")
	grpcAddr := flag.String("grpc-addr", env("TANDEMESH_GRPC_ADDR", "127.0.0.1:9090"), "gRPC plugin address (runner=grpc)")
	workdir := flag.String("workdir", env("TANDEMESH_WORKDIR", ""), "working directory for runs")
	acpBackendsFile := flag.String("acp-backends", env("TANDEMESH_ACP_BACKENDS_FILE", ""), "JSON file of custom ACP backends")
	transport := flag.String("transport", env("TANDEMESH_TRANSPORT", "http"), "transport: http|ws")
	useWorkspace := flag.Bool("workspace", os.Getenv("TANDEMESH_WORKSPACE") == "on", "prepare a git worktree per run")
	dataDir := flag.String("data-dir", env("TANDEMESH_DATA_DIR", ""), "workspace data directory")
	gitPath := flag.String("git", env("TANDEMESH_GIT", ""), "git executable")
	retain := flag.Duration("retain", 7*24*time.Hour, "workspace retention")
	localToken := flag.String("local-token", os.Getenv("TANDEMESH_LOCAL_TOKEN"), "plane local token (desktop-managed planes)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	client := newPlaneClient(*plane, *localToken, 35*time.Second)

	customACP, err := agent.LoadACPBackends(*acpBackendsFile)
	if err != nil {
		log.Error("load custom ACP backends", "err", err)
		os.Exit(1)
	}
	runtimes := detectRuntimes()
	for id := range customACP {
		if _, ok := runtimes[id]; !ok {
			runtimes[id] = "configured"
		}
	}
	agentList := split(*agents)
	for id := range customACP {
		found := false
		for _, existing := range agentList {
			if existing == id {
				found = true
			}
		}
		if !found {
			agentList = append(agentList, id)
		}
	}
	if len(agentList) == 0 {
		for name := range runtimes {
			agentList = append(agentList, name)
		}
		sort.Strings(agentList)
	}
	m, err := register(client, *plane, *token, *name, split(*labels), agentList, runtimes, *concurrency, collectMetrics(*dataDir))
	if err != nil {
		log.Error("register", "err", err)
		os.Exit(1)
	}
	log.Info("registered", "machine", m.ID, "name", m.Name, "runtimes", len(runtimes))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go heartbeat(ctx, client, *plane, m.ID, runtimes, *dataDir, log)
	go terminalLoop(ctx, client, *plane, m.ID, *workdir, log)
	go gitJobLoop(ctx, client, *plane, m.ID, *dataDir, *gitPath, log)
	go historyScanLoop(ctx, client, *plane, m.ID, log)
	runner := newRunner(*runnerMode, *acpCommand, *acpArgs, *grpcAddr,
		*claudeACPCommand, *claudeACPArgs, *codexACPCommand, *codexACPArgs, customACP, log)

	var mgr *workspace.Manager
	if *useWorkspace {
		mgr = workspace.New(*dataDir, *gitPath, *retain, log)
		go func() {
			if n, err := mgr.GC(context.Background(), time.Now()); err != nil {
				log.Warn("workspace gc", "err", err)
			} else if n > 0 {
				log.Info("workspace gc", "removed", n)
			}
		}()
	}
	go worktreeCleanupLoop(ctx, client, *plane, m.ID, mgr, log)
	go workspaceGCLoop(ctx, mgr, log)

	if *transport == "ws" {
		if err := wsLoop(ctx, *plane, *token, m.ID, runner, *workdir, mgr, log); err != nil {
			log.Warn("websocket transport ended; falling back to http", "err", err)
			leaseLoop(ctx, client, *plane, m.ID, runner, *workdir, mgr, log)
		}
	} else {
		leaseLoop(ctx, client, *plane, m.ID, runner, *workdir, mgr, log)
	}
	log.Info("agentd stopped")
}

func newRunner(mode, command, args, grpcAddr, claudeCommand, claudeArgs, codexCommand, codexArgs string, custom map[string]agent.ACPBackendSpec, log *slog.Logger) agent.Runner {
	switch mode {
	case "acp":
		defaultRunner := agent.NewACP(agent.ACPConfig{
			Command: command,
			Args:    strings.Fields(args),
			Logger:  log,
		})
		byBackend := map[string]agent.Runner{}
		for backend, spec := range agent.DefaultACPBackends {
			byBackend[backend] = agent.NewACP(agent.ACPConfig{
				Command: spec.Command, Args: spec.Args, Env: spec.Env, Logger: log,
			})
		}
		for backend, spec := range custom {
			byBackend[backend] = agent.NewACP(agent.ACPConfig{
				Command: spec.Command, Args: spec.Args, Env: spec.Env, Logger: log,
			})
		}
		if claudeCommand != "" {
			byBackend["claude"] = agent.NewACP(agent.ACPConfig{
				Command: claudeCommand, Args: strings.Fields(claudeArgs), Logger: log,
			})
		}
		if codexCommand != "" {
			byBackend["codex"] = agent.NewACP(agent.ACPConfig{
				Command: codexCommand, Args: strings.Fields(codexArgs), Logger: log,
			})
		}
		if len(byBackend) > 0 {
			return &agent.Router{Default: defaultRunner, ByBackend: byBackend}
		}
		return defaultRunner
	case "grpc":
		runner, err := plugin.Dial(grpcAddr)
		if err != nil {
			log.Error("dial grpc plugin", "addr", grpcAddr, "err", err)
			os.Exit(1)
		}
		return runner
	}
	return agent.NewMock(500 * time.Millisecond)
}

func register(c *http.Client, plane, token, name string, labels, agents []string, runtimes map[string]string, concurrency int, metrics *domain.MachineMetrics) (*machine, error) {
	body, _ := json.Marshal(map[string]any{
		"token":       token,
		"name":        name,
		"labels":      labels,
		"agents":      agents,
		"runtimes":    runtimes,
		"concurrency": concurrency,
		"version":     "0.1.0-dev",
		"metrics":     metrics,
	})
	resp, err := c.Post(plane+"/api/v1/machines/register", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, raw)
	}
	var m machine
	return &m, json.NewDecoder(resp.Body).Decode(&m)
}

func heartbeat(ctx context.Context, c *http.Client, plane, machineID string, runtimes map[string]string, dataDir string, log *slog.Logger) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			raw, _ := json.Marshal(map[string]any{"runtimes": runtimes, "metrics": collectMetrics(dataDir)})
			req, _ := http.NewRequestWithContext(ctx, http.MethodPost, plane+"/api/v1/machines/"+machineID+"/heartbeat", bytes.NewReader(raw))
			req.Header.Set("Content-Type", "application/json")
			resp, err := c.Do(req)
			if err != nil {
				log.Warn("heartbeat", "err", err)
				continue
			}
			resp.Body.Close()
		}
	}
}

// detectRuntimes looks for known agent CLIs and records their versions.
func detectRuntimes() map[string]string {
	out := map[string]string{}
	names := make([]string, 0, len(agent.RuntimeCatalog)+2)
	for _, runtime := range agent.RuntimeCatalog {
		names = append(names, runtime.ID)
	}
	names = append(names, "claude-code-acp", "trae")
	for _, name := range names {
		binary := name
		for _, runtime := range agent.RuntimeCatalog {
			if runtime.ID == name && runtime.Command != "" {
				binary = runtime.Command
			}
		}
		path, err := exec.LookPath(binary)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		raw, _ := exec.CommandContext(ctx, path, "--version").Output()
		cancel()
		version := strings.TrimSpace(strings.SplitN(string(raw), "\n", 2)[0])
		if version == "" {
			version = "installed"
		}
		out[name] = version
	}
	return out
}

func backendOf(r *run) string {
	if r.Backend != "" {
		return r.Backend
	}
	return r.AgentID
}

func renewLoop(ctx context.Context, c *http.Client, plane, machineID string, r *run, stop <-chan struct{}, log *slog.Logger) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-ticker.C:
			raw, _ := json.Marshal(map[string]any{"machineId": machineID, "fence": r.Fence})
			req, _ := http.NewRequestWithContext(ctx, http.MethodPost, plane+"/api/v1/agentd/runs/"+r.ID+"/renew", bytes.NewReader(raw))
			req.Header.Set("Content-Type", "application/json")
			resp, err := c.Do(req)
			if err != nil {
				log.Warn("renew lease", "run", r.ID, "err", err)
				continue
			}
			resp.Body.Close()
			if resp.StatusCode >= 300 {
				log.Warn("renew lease", "run", r.ID, "status", resp.StatusCode)
			}
		}
	}
}

func leaseLoop(ctx context.Context, c *http.Client, plane, machineID string, runner agent.Runner, workdir string, mgr *workspace.Manager, log *slog.Logger) {
	events := &httpSink{c: c, plane: plane}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, plane+"/api/v1/agentd/lease?machineId="+machineID+"&wait=25s", nil)
		resp, err := c.Do(req)
		if err != nil {
			log.Warn("lease poll", "err", err)
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.StatusCode == http.StatusNoContent {
			resp.Body.Close()
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		var r run
		err = json.NewDecoder(resp.Body).Decode(&r)
		resp.Body.Close()
		if err != nil {
			log.Warn("decode lease", "err", err)
			continue
		}
		execute(ctx, events, &r, runner, workdir, mgr, c, plane, machineID, log)
	}
}

func execute(ctx context.Context, out sink, r *run, runner agent.Runner, workdir string, mgr *workspace.Manager, c *http.Client, plane, machineID string, log *slog.Logger) {
	log.Info("run leased", "run", r.ID, "repo", r.Repo, "branch", r.Branch)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctx = runCtx
	go watchCancel(ctx, c, plane, r.ID, cancel, log)
	out.Started(ctx, r.ID, r.Fence)
	stopRenew := make(chan struct{})
	go renewLoop(ctx, c, plane, machineID, r, stopRenew, log)
	defer close(stopRenew)

	runDir := workdir
	reusedWorktree := false
	if r.WorktreePath != "" {
		if info, err := os.Stat(r.WorktreePath); err == nil && info.IsDir() {
			runDir = r.WorktreePath
			reusedWorktree = true
			out.Event(ctx, r.ID, r.Fence, "workspace.ready", map[string]any{
				"path":      r.WorktreePath,
				"branch":    r.Branch,
				"machineId": machineID,
			})
		} else {
			out.Event(ctx, r.ID, r.Fence, "run.error", map[string]any{
				"code":    "E_WORKSPACE",
				"message": "selected worktree is not available on this machine",
			})
			out.Finish(ctx, r.ID, r.Fence, "failed", "E_WORKSPACE", "selected worktree is not available on this machine")
			return
		}
	}
	if !reusedWorktree && r.Repo == "" {
		key := r.SessionRecordID
		if key == "" {
			key = r.ID
		}
		runDir = filepath.Join(workdir, "sessions", key)
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			log.Warn("create session dir", "run", r.ID, "err", err)
		}
	}
	if !reusedWorktree && mgr != nil && r.Repo != "" {
		var ws *workspace.Workspace
		var err error
		if r.SessionRecordID != "" {
			ws, err = mgr.PrepareSession(ctx, workspace.Request{
				RunID:  r.SessionRecordID,
				Repo:   r.Repo,
				Base:   r.Base,
				Branch: r.Branch,
			})
		} else {
			ws, err = mgr.Prepare(ctx, workspace.Request{
				RunID:  r.ID,
				Repo:   r.Repo,
				Base:   r.Base,
				Branch: r.Branch,
			})
		}
		if err != nil {
			log.Error("prepare workspace", "run", r.ID, "err", err)
			out.Event(ctx, r.ID, r.Fence, "run.error", map[string]any{
				"code":    "E_WORKSPACE",
				"message": err.Error(),
			})
			out.Finish(ctx, r.ID, r.Fence, "failed", "E_WORKSPACE", err.Error())
			return
		}
		runDir = ws.Dir
		out.Event(ctx, r.ID, r.Fence, "workspace.ready", map[string]any{
			"path":      ws.Dir,
			"branch":    ws.Branch,
			"machineId": machineID,
		})
	}

	prompt := r.Prompt
	if len(r.Skills) > 0 {
		dirs := []string{}
		if runDir != "" {
			dirs = append(dirs, filepath.Join(runDir, ".orca", "skills"), filepath.Join(runDir, ".agents", "skills"))
		}
		if home, err := os.UserHomeDir(); err == nil {
			dirs = append(dirs, filepath.Join(home, ".orca", "skills"))
		}
		loaded, err := skills.Discover(dirs, r.Skills, 32*1024)
		if err != nil {
			log.Warn("load skills", "run", r.ID, "err", err)
		}
		if len(loaded) > 0 {
			prompt = skills.Prompt(loaded, 32*1024) + "\n\n" + prompt
			names := make([]string, 0, len(loaded))
			for _, s := range loaded {
				names = append(names, s.Name)
			}
			out.Event(ctx, r.ID, r.Fence, "skills.loaded", map[string]any{"skills": names})
		}
	}

	spec := agent.Spec{
		RunID:      r.ID,
		SessionID:  r.SessionID,
		Repo:       r.Repo,
		Branch:     r.Branch,
		Workdir:    runDir,
		Prompt:     prompt,
		Backend:    backendOf(r),
		Model:      r.Model,
		Reasoning:  r.Reasoning,
		Env:        r.AgentEnv,
		Timeout:    30 * time.Minute,
		AskTools:   r.AskTools,
		MCPServers: r.MCPServers,
	}
	if len(r.AskTools) > 0 {
		spec.Approve = func(ctx context.Context, tool, command, risk string) (bool, error) {
			decision, err := out.RequestApproval(ctx, r.ID, r.Fence, tool, command, risk)
			if err != nil {
				return false, err
			}
			return decision == "approved", nil
		}
	}
	spec.Ask = func(ctx context.Context, question string, options []string) (string, error) {
		return out.RequestQuestion(ctx, r.ID, r.Fence, question, options)
	}
	emit := func(ev agent.Event) {
		out.Event(ctx, r.ID, r.Fence, ev.Type, ev.Payload)
	}
	res, err := runner.Run(ctx, spec, emit)
	if err != nil {
		res = agent.Result{Status: "failed", Code: "E_RUNNER", Message: err.Error()}
	}
	if res.Usage != nil {
		post(ctx, c, plane+"/api/v1/agentd/runs/"+r.ID+"/usage", map[string]any{
			"inputTokens": res.Usage.InputTokens, "outputTokens": res.Usage.OutputTokens,
			"cacheReadTokens": res.Usage.CacheReadTokens, "cacheWriteTokens": res.Usage.CacheWriteTokens,
			"costUsd": res.Usage.CostUSD, "estimated": res.Usage.Estimated, "model": res.Usage.Model, "fence": r.Fence,
		})
	}
	if diff, truncated := captureDiff(ctx, runDir, r.Base); diff != "" {
		out.Event(ctx, r.ID, r.Fence, "workspace.diff", map[string]any{
			"diff": diff, "truncated": truncated,
		})
	}
	out.Finish(ctx, r.ID, r.Fence, res.Status, res.Code, res.Message)
	log.Info("run finished", "run", r.ID, "status", res.Status)
}

func watchCancel(ctx context.Context, c *http.Client, plane, runID string, cancel context.CancelFunc, log *slog.Logger) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, plane+"/api/v1/agentd/runs/"+runID+"/status", nil)
			resp, err := c.Do(req)
			if err != nil {
				continue
			}
			var body struct {
				Status string `json:"status"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&body)
			resp.Body.Close()
			if body.Status == "cancelled" {
				log.Info("run cancelled remotely", "run", runID)
				cancel()
				return
			}
		}
	}
}
func event(ctx context.Context, c *http.Client, plane, runID string, fence int64, typ string, payload map[string]any) {
	post(ctx, c, plane+"/api/v1/agentd/runs/"+runID+"/events", map[string]any{
		"type":    typ,
		"payload": payload,
		"fence":   fence,
	})
}

func post(ctx context.Context, c *http.Client, url string, body any) {
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func split(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "machine"
	}
	return h
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
