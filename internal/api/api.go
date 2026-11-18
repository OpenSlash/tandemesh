// Package api exposes the Tandemesh control-plane HTTP API.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"sync"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/agent"
	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/credentials"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/githubapp"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/policy"
	"github.com/OpenSlash/tandemesh/internal/scheduler"
	"github.com/OpenSlash/tandemesh/internal/store"
	"github.com/OpenSlash/tandemesh/internal/tenancy"
)

// Version is the build version reported by /api/v1/version.
const Version = "0.1.0-dev"

// Server wires configuration, storage and scheduling into an HTTP handler.
type Server struct {
	cfg          config.Config
	store        *store.Store
	sched        *scheduler.Scheduler
	hub          *hub.Hub
	integrations *integration.Registry
	vault        *credentials.Vault
	githubApp    *githubapp.Client
	tenancy      tenancy.Resolver
	log          *slog.Logger
	debounceState
	reminderState
	metricsMu    sync.Mutex
	httpCounts   map[string]int
	importSlots  chan struct{}
	importConnMu sync.Mutex
	importConns  map[string]chan struct{}
	workflowMu   sync.Mutex
}

// ConfigureGitHubApp enables GitHub App installation authentication.
func (s *Server) ConfigureGitHubApp(app *githubapp.Client) { s.githubApp = app }

// New creates the API server.
func New(cfg config.Config, st *store.Store, sch *scheduler.Scheduler, h *hub.Hub, reg *integration.Registry, vault *credentials.Vault, log *slog.Logger) *Server {
	slots := 2
	if raw := os.Getenv("TANDEMESH_IMPORT_CONCURRENCY"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			slots = parsed
		}
	}
	if slots < 1 {
		slots = 1
	}
	if slots > 8 {
		slots = 8
	}
	var resolver tenancy.Resolver = tenancy.Local{Store: st, User: cfg.User}
	return &Server{cfg: cfg, tenancy: resolver, store: st, sched: sch, hub: h, integrations: reg, vault: vault, log: log, debounceState: debounceState{at: map[string]time.Time{}}, reminderState: reminderState{reminded: map[string]time.Time{}}, importSlots: make(chan struct{}, slots), importConns: map[string]chan struct{}{}}
}

// Router builds the HTTP routes.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(s.publicAuth)
	r.Use(s.identify)
	r.Use(s.enforceViewer)
	r.Use(s.accessLog)
	r.Use(s.auditRequests)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/metrics", s.metrics)
	r.Get("/install/agentd.sh", s.installAgentdShell)
	r.Get("/install/agentd.ps1", s.installAgentdPowerShell)
	r.Get("/download/agentd/*", s.downloadAgentd)
	r.Post("/auth/logout", s.authLogout)
	r.Get("/auth/me", s.authMe)
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.store.DB().PingContext(r.Context()); err != nil {
			writeErr(w, http.StatusServiceUnavailable, "E_DB", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"name":    "orca",
			"version": Version,
			"docs":    "/api/v1/runs",
		})
	})

	r.Get("/api/v1/capabilities", s.capabilities)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/usage/trend", s.usageTrend)
		r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"version": Version})
		})
		r.Get("/providers", s.listProviders)
		r.Post("/providers/{id}/probe", s.probeProvider)
		r.Get("/context/search", s.contextSearch)
		r.Get("/context/fetch", s.contextFetch)
		r.Get("/credentials", s.listCredentials)
		r.Put("/credentials", s.putCredential)
		r.Delete("/credentials/{provider}/{subject}", s.deleteCredential)

		r.Get("/runs", s.listRuns)
		r.Post("/runs", s.createRun)
		r.Get("/runs/{id}", s.getRun)
		r.Post("/runs/{id}/cancel", s.cancelRun)
		r.Post("/runs/{id}/retry", s.retryRun)
		r.Get("/runs/{id}/variants", s.listVariants)
		r.Post("/runs/{id}/variants", s.createVariants)

		r.Get("/tasks", s.listTasks)
		r.Post("/tasks", s.createLocalTask)
		r.Get("/tasks/{id}", s.getTask)
		r.Patch("/tasks/{id}", s.patchTask)
		r.Post("/tasks/{id}/assignee", s.assignTaskMember)
		r.Patch("/tasks/{id}/member-assignee", s.assignLocalMember)
		r.Patch("/tasks/{id}/fields", s.updateTaskFields)
		r.Get("/tasks/{id}/field-options", s.taskFieldOptions)
		r.Get("/tasks/{id}/field-metadata", s.taskFieldMetadata)
		r.Patch("/tasks/{id}/inbox", s.setTaskInbox)
		r.Post("/tasks/{id}/read", s.markTaskRead)
		r.Get("/tasks/{id}/activity", s.taskActivity)
		r.Get("/tasks/{id}/notes", s.listTaskNotes)
		r.Post("/tasks/{id}/notes", s.addTaskNote)
		r.Get("/tasks/{id}/comments", s.listTaskComments)
		r.Get("/tasks/{id}/comment-deliveries", s.listCommentDeliveries)
		r.Get("/tasks/{id}/platform-comments", s.listPlatformComments)
		r.Post("/tasks/{id}/platform-comments", s.addPlatformComment)
		r.Get("/tasks/{id}/comment-threads", s.listCommentThreads)
		r.Patch("/tasks/{id}/comment-threads/{threadId}", s.resolveCommentThread)
		r.Get("/tasks/{id}/pull-requests", s.listTaskPullRequests)
		r.Post("/tasks/{id}/pull-requests/sync", s.syncTaskPullRequests)
		r.Post("/tasks/{id}/pull-requests/{pr}/comments", s.addPullRequestComment)
		r.Get("/tasks/{id}/attachments", s.listTaskAttachments)
		r.Get("/tasks/{id}/attachments/{attachmentId}/content", s.downloadTaskAttachment)
		r.Get("/tasks/{id}/commits", s.listTaskCommits)
		r.Post("/tasks/{id}/commits/sync", s.syncTaskCommits)
		r.Delete("/pull-requests/{id}", s.deleteTaskPullRequest)
		r.Get("/documents", s.listDocuments)
		r.Get("/documents/{id}", s.getDocument)
		r.Post("/documents", s.createDocument)
		r.Put("/documents/{id}", s.updateDocument)
		r.Delete("/documents/{id}", s.deleteDocument)
		r.Get("/audit-logs", s.listAuditLogs)
		r.Get("/notifications", s.listNotifications)
		r.Post("/notifications/{id}/read", s.markNotificationRead)
		r.Post("/notifications/read-all", s.markAllNotificationsRead)
		r.Post("/tasks/{id}/comments", s.addTaskComment)
		r.Post("/tasks/{id}/runs", s.startTaskRun)
		r.Post("/tasks/{id}/summon", s.summonAgent)
		r.Post("/tasks/{id}/team", s.startTaskTeam)
		r.Post("/tasks/{id}/command", s.taskCommand)
		r.Get("/work", s.listWork)
		r.Post("/work/sync", s.syncWork)
		r.Post("/work/import", s.importWork)
		r.Post("/workspaces/{workspace}/imports/jira", s.createJiraImport)
		r.Get("/workspaces/{workspace}/imports", s.listImports)
		r.Get("/imports/runtime", s.importRuntime)
		r.Post("/imports/{id}/resume", s.resumeImport)
		r.Post("/imports/{id}/cancel", s.cancelImport)
		r.Post("/imports/{id}/rollback", s.rollbackImport)

		r.Get("/workspaces", s.listWorkspaces)
		r.Post("/workspaces", s.createWorkspace)
		r.Patch("/workspaces/{id}", s.updateWorkspace)
		r.Get("/workspaces/{id}/views", s.listSavedViews)
		r.Post("/workspaces/{id}/views", s.createSavedView)
		r.Delete("/workspaces/{id}/views/{view}", s.deleteSavedView)
		r.Patch("/workspaces/{id}/views/{view}", s.renameSavedView)
		r.Get("/workspaces/{id}/iterations", s.listIterations)
		r.Post("/workspaces/{id}/iterations", s.createIteration)
		r.Patch("/iterations/{id}", s.patchIteration)
		r.Get("/iterations/{id}/burndown", s.iterationBurndown)
		r.Delete("/iterations/{id}", s.deleteIteration)
		r.Patch("/tasks/{id}/assignee", s.setTaskAssignee)
		r.Get("/assignment-rules", s.listAssignmentRules)
		r.Post("/assignment-rules", s.createAssignmentRule)
		r.Delete("/assignment-rules/{id}", s.deleteAssignmentRule)
		r.Patch("/tasks/{id}/iteration", s.setTaskIteration)
		r.Patch("/tasks/{id}/parent", s.setTaskParent)
		r.Get("/tasks/{id}/children", s.listTaskChildren)
		r.Get("/tasks/{id}/dependencies", s.listTaskDependencies)
		r.Post("/tasks/{id}/dependencies", s.addTaskDependency)
		r.Delete("/tasks/{id}/dependencies/{dep}", s.removeTaskDependency)
		r.Get("/tasks/{id}/dispatches", s.listTaskDispatches)
		r.Post("/tasks/{id}/dispatches", s.createTaskDispatch)
		r.Get("/dispatches/{id}", s.getDispatch)
		r.Post("/dispatches/{id}/start", s.startDispatch)
		r.Post("/dispatches/{id}/complete", s.completeDispatch)
		r.Get("/tasks/{id}/gates", s.listGates)
		r.Post("/tasks/{id}/gates", s.createGate)
		r.Post("/gates/{id}/resolve", s.resolveGate)
		r.Get("/tasks/{id}/messages", s.listTaskMessages)
		r.Post("/tasks/{id}/messages", s.sendTaskMessage)
		r.Post("/tasks/{id}/messages/check", s.checkTaskMessage)
		r.Post("/messages/{id}/ack", s.ackTaskMessage)
		r.Get("/tasks/{id}/annotations", s.listAnnotations)
		r.Post("/tasks/{id}/annotations", s.createAnnotation)
		r.Post("/tasks/{id}/annotations/send", s.sendAnnotations)
		r.Post("/annotations/{id}/resolve", s.resolveAnnotation)
		r.Get("/repositories", s.listRepositories)
		r.Get("/worktrees", s.listWorktrees)
		r.Get("/worktrees/{id}", s.getWorktree)
		r.Post("/worktrees/{id}/cleanup", s.cleanupWorktree)
		r.Get("/worktrees/retention", s.worktreeRetention)
		r.Post("/worktrees/retention/apply", s.applyWorktreeRetention)
		r.Get("/agentd/worktree-cleanups", s.agentdWorktreeCleanups)
		r.Post("/agentd/worktree-cleanups/{id}", s.agentdWorktreeCleanupResult)
		r.Post("/worktrees/{id}/checkpoint", s.setWorktreeCheckpoint)
		r.Post("/worktrees/{id}/archive", s.archiveWorktree)
		r.Get("/runs/{id}/worktree", s.getRunWorktree)
		r.Post("/runs/{id}/worktree/checkpoint", s.checkpointRunWorktree)
		r.Get("/session-history", s.listSessionHistory)
		r.Post("/session-history/resume", s.resumeSessionHistory)
		r.Post("/session-history/scan", s.scanSessionHistory)
		r.Get("/agentd/history-scans/{machineId}", s.agentdHistoryScans)
		r.Post("/agentd/history-scans/{machineId}/{id}", s.agentdHistoryScanResult)
		r.Get("/sessions", s.listSessions)
		r.Post("/sessions", s.createSession)
		r.Get("/sessions/{id}", s.getSession)
		r.Patch("/sessions/{id}", s.renameSession)
		r.Post("/sessions/{id}/archive", s.archiveSession)
		r.Post("/sessions/{id}/share", s.createSessionShare)
		r.Delete("/sessions/{id}/share/{token}", s.revokeSessionShare)
		r.Get("/shared/sessions/{token}", s.getSharedSession)
		r.Post("/sessions/{id}/close", s.closeSession)
		r.Post("/sessions/{id}/stop", s.stopSession)
		r.Get("/sessions/{id}/runs", s.listSessionRuns)
		r.Post("/sessions/{id}/runs", s.createSessionRun)
		r.Post("/sessions/{id}/regenerate", s.regenerateSession)
		r.Get("/sessions/{id}/messages", s.sessionMessages)
		r.Patch("/sessions/{id}/messages/{messageId}", s.editSessionMessage)
		r.Get("/sessions/{id}/events", s.streamSessionEvents)
		r.Get("/sessions/{id}/diff", s.sessionDiff)
		r.Get("/sessions/{id}/comments", s.listSessionComments)
		r.Post("/sessions/{id}/comments", s.createSessionComment)
		r.Post("/sessions/{id}/comments/send", s.sendSessionComments)
		r.Get("/sessions/{id}/questions", s.listSessionQuestions)
		r.Post("/session-comments/{id}/resolve", s.resolveSessionComment)
		r.Patch("/session-comments/{id}", s.updateSessionComment)
		r.Delete("/session-comments/{id}", s.deleteSessionComment)
		r.Post("/questions/{id}/answer", s.answerAgentQuestion)
		r.Get("/sessions/{id}/worktree", s.getSessionWorktree)
		r.Post("/sessions/{id}/worktree/archive", s.archiveSessionWorktree)
		r.Post("/sessions/{id}/worktree/cleanup", s.cleanupSessionWorktree)
		r.Get("/sessions/{id}/git", s.getSessionGit)
		r.Post("/sessions/{id}/git/ship", s.shipSession)
		r.Post("/sessions/{id}/git/pr", s.createSessionPR)
		r.Get("/sessions/{id}/git/pr", s.getSessionPRStatus)
		r.Post("/sessions/{id}/git/pr/refresh", s.refreshSessionPR)
		r.Get("/sessions/{id}/git/jobs/{jobId}", s.getSessionGitJob)
		r.Get("/agentd/git-jobs", s.agentdGitJobs)
		r.Post("/agentd/git-jobs/{id}", s.agentdGitJobResult)
		r.Get("/terminal-sessions", s.listTerminalSessions)
		r.Post("/terminal-sessions", s.createTerminalSession)
		r.Get("/terminal-sessions/{id}", s.getTerminalSession)
		r.Post("/terminal-sessions/{id}/continue-chat", s.continueTerminalChat)
		r.Post("/terminal-sessions/{id}/input", s.terminalInput)
		r.Get("/terminal-sessions/{id}/output", s.terminalOutput)
		r.Post("/terminal-sessions/{id}/close", s.closeTerminalSession)
		r.Get("/agentd/terminals", s.agentdTerminals)
		r.Get("/agentd/terminals/{id}/input", s.agentdTerminalInput)
		r.Post("/agentd/terminals/{id}/output", s.agentdTerminalOutput)
		r.Post("/agentd/terminals/{id}/state", s.agentdTerminalState)
		r.Get("/groups", s.listGroups)
		r.Post("/groups", s.createGroup)
		r.Get("/groups/{id}", s.getGroup)
		r.Post("/groups/{id}/members", s.addGroupMember)
		r.Get("/me", s.me)
		r.Get("/search", s.search)
		r.Get("/runs/{id}/events", s.streamEvents)
		r.Get("/runs/{id}/transcript", s.runTranscript)
		r.Get("/runs/{id}/diff", s.runDiff)

		r.Get("/machines", s.listMachines)
		r.Get("/machines/{id}", s.getMachine)
		r.Get("/runtimes", s.listRuntimes)
		r.Get("/system/info", s.systemInfo)
		r.Post("/machines/join-token", s.createMachineJoinToken)
		r.Post("/machines/register", s.registerMachine)
		r.Post("/integrations/connect", s.connectProvider)
		r.Get("/integrations/github/oauth/status", s.githubOAuthStatus)
		r.Get("/integrations/github/oauth/start", s.githubOAuthStart)
		r.Get("/integrations/github/oauth/callback", s.githubOAuthCallback)
		r.Get("/integrations/github/app/status", s.githubAppStatus)
		r.Get("/integrations/github/app/start", s.githubAppStart)
		r.Get("/integrations/github/app/callback", s.githubAppCallback)
		r.Get("/integrations/oauth/status", s.oauthStatus)
		r.Get("/integrations/{provider}/oauth/config", s.getOAuthApp)
		r.Put("/integrations/{provider}/oauth/config", s.putOAuthApp)
		r.Delete("/integrations/{provider}/oauth/config", s.deleteOAuthApp)
		r.Get("/integrations/{provider}/oauth/start", s.oauthStart)
		r.Get("/integrations/{provider}/oauth/callback", s.oauthCallback)
		r.Post("/integrations/states", s.integrationStates)
		r.Post("/integrations/{provider}/catalog", s.integrationCatalog)
		r.Get("/connections", s.listConnections)
		r.Post("/connections/{id}/reconnect", s.reconnectConnection)
		r.Get("/connections/{id}/states", s.connectionStates)
		r.Get("/connections/{id}/users", s.connectionUsers)
		r.Get("/connections/{id}/sync-status", s.connectionSyncStatus)
		r.Get("/connections/{id}/webhook", s.connectionWebhookURL)
		r.Get("/workspaces/{workspace}/identities", s.listExternalIdentities)
		r.Get("/workspaces/{workspace}/member-identities", s.listMemberIdentities)
		r.Post("/workspaces/{workspace}/member-identities", s.createMemberIdentity)
		r.Delete("/member-identities/{id}", s.deleteMemberIdentity)
		r.Post("/workspaces/{workspace}/identities", s.createExternalIdentity)
		r.Patch("/identities/{id}", s.patchExternalIdentity)
		r.Delete("/identities/{id}", s.deleteExternalIdentity)
		r.Get("/connections/{id}/accounts", s.listConnectionAccounts)
		r.Post("/connections/{id}/accounts", s.createConnectionAccount)
		r.Patch("/accounts/{id}", s.patchConnectionAccount)
		r.Delete("/accounts/{id}", s.deleteConnectionAccount)
		r.Get("/connections/{id}/bindings", s.listConnectionBindings)
		r.Post("/connections/{id}/bindings", s.createConnectionBinding)
		r.Post("/connections/{id}/register-webhook", s.registerConnectionWebhook)
		r.Post("/connections/{id}/bindings/{bindingId}/register-webhook", s.registerBindingWebhook)
		r.Get("/webhook-deliveries/{automation}", s.listWebhookDeliveries)
		r.Post("/webhook-deliveries/{id}/replay", s.replayWebhookDelivery)
		r.Patch("/connections/{id}", s.patchConnection)
		r.Delete("/connections/{id}", s.deleteConnectionByID)
		r.Post("/connections/{id}/merge/{source}", s.mergeConnection)
		r.Patch("/bindings/{id}", s.patchBinding)
		r.Delete("/bindings/{id}", s.deleteBinding)
		r.Delete("/integrations/{provider}", s.deleteConnection)
		r.Post("/machines/{id}/takeover", s.takeoverMachine)
		r.Post("/machines/{id}/status", s.setMachineStatus)
		r.Post("/machines/{id}/heartbeat", s.heartbeat)

		r.Get("/agents", s.listAgents)
		r.Post("/agents", s.createAgent)
		r.Get("/agents/{id}", s.getAgent)
		r.Get("/agents/{id}/env", s.getAgentEnv)
		r.Put("/agents/{id}/env", s.putAgentEnv)
		r.Put("/agents/{id}/skills", s.setAgentSkills)
		r.Put("/agents/{id}", s.updateAgent)
		r.Delete("/agents/{id}", s.deleteAgent)

		r.Get("/skills", s.listSkills)
		r.Post("/skills", s.createSkill)
		r.Post("/skills/install", s.installSkill)
		r.Get("/skills/export", s.exportSkills)
		r.Get("/skills/market", s.listSkillMarket)
		r.Post("/skills/market/{id}/install", s.installSkillMarket)
		r.Get("/skills/{id}", s.getSkill)
		r.Put("/skills/{id}", s.updateSkill)
		r.Delete("/skills/{id}", s.deleteSkill)
		r.Get("/skills/{id}/raw", s.rawSkill)
		r.Post("/skills/{id}/enabled", s.setSkillEnabled)

		r.Get("/approvals", s.listApprovals)
		r.Post("/approvals", s.createApproval)
		r.Get("/approvals/{id}", s.getApproval)
		r.Get("/approvals/{id}/decide", s.approvalCallback)
		r.Post("/approvals/{id}/decision", s.decideApproval)

		r.Get("/automations", s.listUnifiedAutomations)
		r.Post("/automations", s.createAutomation)
		r.Get("/automations/{id}", s.getAutomation)
		r.Put("/automations/{id}", s.updateAutomation)
		r.Delete("/automations/{id}", s.deleteAutomation)
		r.Post("/automations/{id}/run", s.runAutomation)

		r.Get("/workflows", s.listWorkflows)
		r.Post("/workflows", s.createWorkflow)
		r.Get("/workflows/{id}", s.getWorkflow)
		r.Post("/workflows/{id}/runs", s.startWorkflow)
		r.Get("/workflow-runs", s.listWorkflowRuns)
		r.Get("/workflow-runs/{id}", s.getWorkflowRun)
		r.Post("/workflow-runs/{id}/gate", s.gateWorkflowRun)
		r.Post("/workflow-runs/{id}/retry", s.retryWorkflowRun)
		r.Post("/workflow-runs/{id}/cancel", s.cancelWorkflowRun)

		r.Get("/agentd/lease", s.lease)
		r.Post("/agentd/runs/{id}/renew", s.renewLease)
		r.Post("/agentd/runs/{id}/start", s.startRun)
		r.Post("/agentd/runs/{id}/events", s.appendRunEvent)
		r.Post("/agentd/runs/{id}/questions", s.createAgentdQuestion)
		r.Get("/agentd/questions/{id}", s.getAgentdQuestion)
		r.Post("/agentd/runs/{id}/finish", s.finishRun)
		r.Post("/agentd/runs/{id}/usage", s.runUsage)
	})

	r.Get("/oauth/result", s.oauthResult)
	r.Post("/webhooks/slack/events", s.slackEvents)
	r.Post("/webhooks/github/app", s.githubAppWebhook)
	r.Post("/webhooks/{provider}/{automation}", s.webhook)
	r.Get("/ws", s.websocket)
	return r
}

func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		if requestID := middleware.GetReqID(r.Context()); requestID != "" {
			w.Header().Set("X-Request-Id", requestID)
		}
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		key := r.Method + ":" + strconv.Itoa(ww.Status())
		s.metricsMu.Lock()
		if s.httpCounts == nil {
			s.httpCounts = map[string]int{}
		}
		s.httpCounts[key]++
		s.metricsMu.Unlock()
		s.log.Info("http", "method", r.Method, "path", r.URL.Path, "status", ww.Status(), "ms", time.Since(start).Milliseconds(), "requestId", middleware.GetReqID(r.Context()))
	})
}

// ---- runs ----

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	workspaceID := r.URL.Query().Get("workspace")
	all := workspaceID == "all"
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !all && !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	lookup := workspaceID
	if all {
		lookup = ""
	}
	runs, err := s.store.ListRuns(r.Context(), lookup, status, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if all {
		filtered := make([]*domain.Run, 0, len(runs))
		for _, run := range runs {
			if s.allowWorkspace(r.Context(), run.WorkspaceID) {
				filtered = append(filtered, run)
			}
		}
		runs = filtered
	}
	if runs == nil {
		runs = []*domain.Run{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": runs})
}

type createRunRequest struct {
	Work        domain.WorkRef `json:"work"`
	WorkspaceID string         `json:"workspaceId"`
	AgentID     string         `json:"agentId"`
	Repo        string         `json:"repo"`
	Base        string         `json:"base"`
	Branch      string         `json:"branch"`
	Prompt      string         `json:"prompt"`
	Priority    int            `json:"priority"`
}

func (s *Server) createRun(w http.ResponseWriter, r *http.Request) {
	var req createRunRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Work.Provider == "" || req.Work.ID == "" {
		writeErr(w, http.StatusBadRequest, "E_WORK", "work.provider and work.id are required")
		return
	}
	if req.AgentID == "" {
		req.AgentID = "claude"
	}
	if req.WorkspaceID == "" {
		req.WorkspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, req.WorkspaceID) {
		return
	}
	if req.Base == "" {
		req.Base = "main"
	}
	id := "run_" + ulid.Make().String()
	if req.Branch == "" {
		req.Branch = "ap/" + id[len(id)-8:]
	}
	title := req.Work.URL
	if title == "" {
		title = strings.TrimSpace(strings.SplitN(req.Prompt, "\n", 2)[0])
	}
	if title == "" {
		title = req.Work.Provider + " " + req.Work.ID
	}
	run := &domain.Run{
		ID:          id,
		WorkspaceID: req.WorkspaceID,
		Work:        req.Work,
		Snapshot:    domain.Snapshot{Title: title, Body: req.Prompt},
		AgentID:     req.AgentID,
		Status:      domain.RunQueued,
		Priority:    req.Priority,
		Repo:        req.Repo,
		Base:        req.Base,
		Branch:      req.Branch,
		Prompt:      req.Prompt,
		CreatedAt:   time.Now().UTC(),
	}
	if task, err := s.store.EnsureTaskIn(r.Context(), run.WorkspaceID, run.Work, run.Snapshot); err == nil && task != nil {
		run.TaskID = task.ID
	}
	if err := s.store.CreateRun(r.Context(), run); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if run.TaskID != "" {
		_ = s.store.SetTaskStatus(r.Context(), run.TaskID, domain.TaskInProgress)
	}
	_, _ = s.store.AppendEvent(r.Context(), run.ID, domain.EventRunCreated, map[string]any{
		"provider": run.Work.Provider,
		"url":      run.Work.URL,
		"agent":    run.AgentID,
	})
	s.log.Info("run created", "run", run.ID)
	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	run, err := s.store.GetRun(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if run == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "run not found")
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) cancelRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	run, err := s.store.GetRun(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if run == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "run not found")
		return
	}
	ok, err := s.store.CancelRun(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusConflict, "E_STATE", "run is not cancellable")
		return
	}
	if run.MachineID != "" {
		s.hub.Send(run.MachineID, hub.Message{Type: "run.cancel", RunID: id, Fence: run.Fence})
	}
	_, _ = s.store.AppendEvent(r.Context(), id, "run.cancelled", map[string]any{"by": "api"})
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (s *Server) streamEvents(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	run, err := s.store.GetRun(r.Context(), id)
	if err != nil || run == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "run not found")
		return
	}
	after, _ := strconv.Atoi(r.URL.Query().Get("after"))

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "E_STREAM", "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(10 * time.Minute)
	defer deadline.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-deadline.C:
			fmt.Fprint(w, "event: timeout\ndata: {}\n\n")
			flusher.Flush()
			return
		case <-ticker.C:
			events, err := s.store.ListEvents(r.Context(), id, after)
			if err != nil {
				return
			}
			for _, ev := range events {
				raw, _ := json.Marshal(ev)
				fmt.Fprintf(w, "id: %d\ndata: %s\n\n", ev.Seq, raw)
				after = ev.Seq
			}
			if len(events) > 0 {
				flusher.Flush()
			}
			if run.Status.Terminal() && len(events) == 0 {
				fmt.Fprint(w, "event: end\ndata: {}\n\n")
				flusher.Flush()
				return
			}
		}
	}
}

func (s *Server) listRuntimes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": agent.RuntimeCatalog})
}

// ---- machines ----

func (s *Server) listMachines(w http.ResponseWriter, r *http.Request) {
	machines, err := s.store.ListMachines(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if machines == nil {
		machines = []*domain.Machine{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": machines})
}

func (s *Server) getMachine(w http.ResponseWriter, r *http.Request) {
	machine, err := s.store.GetMachine(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if machine == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "machine not found")
		return
	}
	writeJSON(w, http.StatusOK, machine)
}

type registerMachineRequest struct {
	Token       string                 `json:"token"`
	Name        string                 `json:"name"`
	Labels      []string               `json:"labels"`
	Agents      []string               `json:"agents"`
	Runtimes    map[string]string      `json:"runtimes"`
	Concurrency int                    `json:"concurrency"`
	Version     string                 `json:"version"`
	Metrics     *domain.MachineMetrics `json:"metrics,omitempty"`
}

func (s *Server) setMachineStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	switch req.Status {
	case "online", "offline", "draining":
	default:
		writeErr(w, http.StatusBadRequest, "E_STATUS", "status must be online, offline or draining")
		return
	}
	if err := s.store.SetMachineStatus(r.Context(), chi.URLParam(r, "id"), req.Status); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}
func (s *Server) takeoverMachine(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetID string `json:"targetId"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	source := chi.URLParam(r, "id")
	if req.TargetID == "" || req.TargetID == source {
		writeErr(w, http.StatusBadRequest, "E_TARGET", "targetId must be another machine")
		return
	}
	machines, err := s.store.ListMachines(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	var target *domain.Machine
	for _, m := range machines {
		if m.ID == req.TargetID {
			target = m
			break
		}
	}
	if target == nil {
		writeErr(w, http.StatusBadRequest, "E_TARGET", "target machine not found")
		return
	}
	if target.Status != "online" || time.Since(target.LastSeen) > 45*time.Second {
		writeErr(w, http.StatusBadRequest, "E_TARGET_OFFLINE", "target machine must be online")
		return
	}

	agents, err := s.store.ListAgents(r.Context(), "")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	for _, agent := range agents {
		if agent.MachineID != source {
			continue
		}
		if !machineAcceptsBackend(target, agent.Backend) {
			writeErr(w, http.StatusBadRequest, "E_RUNTIME", "target machine does not support backend "+agent.Backend)
			return
		}
	}
	agentCount, runCount, err := s.store.TakeoverMachine(r.Context(), source, req.TargetID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": agentCount, "runs": runCount})
}

func machineAcceptsBackend(m *domain.Machine, backend string) bool {
	if backend == "" {
		return true
	}
	if len(m.Runtimes) == 0 && len(m.Agents) == 0 {
		return true
	}
	if _, ok := m.Runtimes[backend]; ok {
		return true
	}
	for _, agent := range m.Agents {
		if agent == backend {
			return true
		}
	}
	return false
}

func (s *Server) registerMachine(w http.ResponseWriter, r *http.Request) {
	var req registerMachineRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if !s.authorizeMachineToken(r.Context(), req.Token) {
		writeErr(w, http.StatusUnauthorized, "E_JOIN_TOKEN", "invalid join token")
		return
	}
	if req.Name == "" {
		req.Name = "machine"
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 2
	}
	machineID := "mch_" + ulid.Make().String()
	if existing, err := s.store.GetMachineByName(r.Context(), req.Name); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	} else if existing != nil {
		machineID = existing.ID
	}
	m := &domain.Machine{
		ID:          machineID,
		Name:        req.Name,
		Labels:      req.Labels,
		Agents:      req.Agents,
		Runtimes:    req.Runtimes,
		Concurrency: req.Concurrency,
		Status:      "online",
		Version:     req.Version,
		LastSeen:    time.Now().UTC(),
		Metrics:     req.Metrics,
	}
	if err := s.store.UpsertMachine(r.Context(), m); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.log.Info("machine registered", "machine", m.ID, "name", m.Name)
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) heartbeat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Runtimes map[string]string      `json:"runtimes"`
		Metrics  *domain.MachineMetrics `json:"metrics,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if len(req.Runtimes) > 0 {
		if err := s.store.Heartbeat(r.Context(), chi.URLParam(r, "id"), req.Runtimes); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
	} else if err := s.store.Heartbeat(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if err := s.store.SetMachineMetrics(r.Context(), chi.URLParam(r, "id"), req.Metrics); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---- agent daemon ----

func (s *Server) lease(w http.ResponseWriter, r *http.Request) {
	machineID := r.URL.Query().Get("machineId")
	if machineID == "" {
		writeErr(w, http.StatusBadRequest, "E_MACHINE", "machineId is required")
		return
	}
	wait := 25 * time.Second
	if v := r.URL.Query().Get("wait"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 && d <= 30*time.Second {
			wait = d
		}
	}
	deadline := time.Now().Add(wait)
	for {
		run, err := s.store.NextLease(r.Context(), machineID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		if run != nil {
			if agent, err := s.store.GetAgent(r.Context(), run.AgentID); err == nil && agent != nil {
				run.Backend = agent.Backend
			}
			if run.SessionID != "" {
				if sess, err := s.store.GetAgentSession(r.Context(), run.SessionID); err == nil && sess != nil {
					run.SessionRecordID = run.SessionID
					run.SessionID = sess.BackendSessionID
					if sess.WorktreeID != "" {
						if wt, err := s.store.GetWorktree(r.Context(), sess.WorktreeID); err == nil && wt != nil {
							run.WorktreePath = wt.Path
						}
					}
				}
			}
			if s.vault != nil {
				if cred, ok, err := s.vault.Get(r.Context(), "agent-env", run.AgentID); err == nil && ok && cred.AccessToken != "" {
					_ = json.Unmarshal([]byte(cred.AccessToken), &run.AgentEnv)
				}
			}
			writeJSON(w, http.StatusOK, run)
			return
		}
		if time.Now().After(deadline) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (s *Server) renewLease(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MachineID string `json:"machineId"`
		Fence     int64  `json:"fence"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	ok, err := s.store.RenewLease(r.Context(), chi.URLParam(r, "id"), req.MachineID, req.Fence, time.Now().Add(s.cfg.LeaseTTL))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusConflict, "E_LEASE", "lease not held")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) startRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ok, err := s.store.UpdateStatus(r.Context(), id, domain.RunLeased, domain.RunRunning, "", "")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusConflict, "E_STATE", "run is not leased")
		return
	}
	_, _ = s.store.AppendEvent(r.Context(), id, domain.EventRunStarted, map[string]any{})
	writeJSON(w, http.StatusOK, map[string]string{"status": "running"})
}

type eventRequest struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
	Fence   int64          `json:"fence"`
}

func (s *Server) appendRunEvent(w http.ResponseWriter, r *http.Request) {
	var req eventRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Type == "" {
		writeErr(w, http.StatusBadRequest, "E_EVENT", "type is required")
		return
	}
	id := chi.URLParam(r, "id")
	if req.Fence > 0 {
		ok, err := s.store.CheckFence(r.Context(), id, req.Fence)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusConflict, "E_FENCE", "stale fence")
			return
		}
	}
	if req.Type == "workspace.ready" {
		path, _ := req.Payload["path"].(string)
		branch, _ := req.Payload["branch"].(string)
		if err := s.store.MarkWorktreeReady(r.Context(), id, path, branch); err != nil {
			s.log.Warn("record worktree", "run", id, "err", err)
		}
	}
	ev, err := s.store.AppendEvent(r.Context(), id, req.Type, req.Payload)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.recordSessionEvent(r.Context(), id, &ev)
	s.recordAgentQuestion(r.Context(), id, &ev)
	writeJSON(w, http.StatusCreated, ev)
}

type finishRequest struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Fence   int64  `json:"fence"`
}

func (s *Server) runUsage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InputTokens      int     `json:"inputTokens"`
		OutputTokens     int     `json:"outputTokens"`
		CacheReadTokens  int     `json:"cacheReadTokens"`
		CacheWriteTokens int     `json:"cacheWriteTokens"`
		CostUSD          float64 `json:"costUsd"`
		Estimated        bool    `json:"estimated"`
		Model            string  `json:"model"`
		Fence            int64   `json:"fence"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if err := s.store.SetRunUsage(r.Context(), chi.URLParam(r, "id"), req.InputTokens, req.OutputTokens, req.CacheReadTokens, req.CacheWriteTokens, req.CostUSD, req.Estimated, req.Model); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) finishRun(w http.ResponseWriter, r *http.Request) {
	var req finishRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	status := domain.RunStatus(req.Status)
	if status != domain.RunSucceeded && status != domain.RunFailed {
		writeErr(w, http.StatusBadRequest, "E_STATUS", "status must be succeeded or failed")
		return
	}
	id := chi.URLParam(r, "id")
	if req.Fence > 0 {
		ok, err := s.store.CheckFence(r.Context(), id, req.Fence)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusConflict, "E_FENCE", "stale fence")
			return
		}
	}
	ok, err := s.store.UpdateStatus(r.Context(), id, domain.RunRunning, status, req.Code, req.Message)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusConflict, "E_STATE", "run is not running")
		return
	}
	_, _ = s.store.AppendEvent(r.Context(), id, domain.EventRunFinished, map[string]any{
		"status": string(status),
		"code":   req.Code,
	})
	if run, err := s.store.GetRun(r.Context(), id); err == nil && run != nil {
		s.enqueueWritebacks(r.Context(), run)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": string(status)})
}

// ---- webhooks ----

func (s *Server) webhook(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	automationID := chi.URLParam(r, "automation")

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", "body too large")
		return
	}

	automation, err := s.store.GetAutomation(r.Context(), automationID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if automation == nil {
		writeErr(w, http.StatusNotFound, "E_AUTOMATION", "automation not found")
		return
	}
	if automation.Provider != provider {
		writeErr(w, http.StatusBadRequest, "E_AUTOMATION", "automation provider does not match the path")
		return
	}
	if !automation.Enabled {
		writeJSON(w, http.StatusOK, map[string]string{"skipped": "SKIP_DISABLED"})
		return
	}

	ad, ok := s.integrations.Get(automation.Provider)
	if !ok || ad.Webhook == nil {
		writeErr(w, http.StatusBadRequest, "E_PROVIDER", "integration does not support webhooks: "+automation.Provider)
		return
	}
	if !isReplayRequest(r) && !appWebhookVerified(r.Context()) {
		if err := s.verifyWebhook(r.Context(), automation.Provider, automation.ConnectionID, ad.Webhook, r, body); err != nil {
			s.log.Warn("bad webhook signature", "provider", automation.Provider, "automation", automationID, "ip", r.RemoteAddr)
			writeErr(w, http.StatusUnauthorized, "E_SIG", "signature verification failed")
			return
		}
	}
	ev, err := ad.Webhook.Parse(body)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "E_PAYLOAD", err.Error())
		return
	}
	if ev.Kind == "pipeline" {
		deliveryID := "whd_" + ulid.Make().String()
		_ = s.store.CreateWebhookDelivery(r.Context(), &domain.WebhookDelivery{
			ID: deliveryID, Provider: automation.Provider, AutomationID: automation.ID,
			Delivery: ev.Delivery, Kind: ev.Kind, Body: string(body),
		})
		linked := s.linkGitLabPipeline(r.Context(), body, automation.WorkspaceID)
		queued := s.enqueuePipelineFollowUp(r.Context(), body, automation)
		_ = s.store.UpdateWebhookDelivery(r.Context(), deliveryID, "accepted", "", "")
		writeJSON(w, http.StatusAccepted, map[string]any{"linked": linked, "queued": queued})
		return
	}
	if ev.Kind == "merge_request" {
		deliveryID := "whd_" + ulid.Make().String()
		_ = s.store.CreateWebhookDelivery(r.Context(), &domain.WebhookDelivery{
			ID: deliveryID, Provider: automation.Provider, AutomationID: automation.ID,
			Delivery: ev.Delivery, Kind: ev.Kind, Body: string(body),
		})
		linked := s.linkGitLabMergeRequest(r.Context(), body, automation.WorkspaceID)
		_ = s.store.UpdateWebhookDelivery(r.Context(), deliveryID, "accepted", "", "")
		writeJSON(w, http.StatusAccepted, map[string]any{"linked": linked})
		return
	}
	delivery := r.Header.Get("X-GitHub-Delivery")
	if delivery == "" {
		delivery = ev.Delivery
	}
	if replayID := headerWithLegacy(r, "X-Tandemesh-Replay", "X-Orca-Replay"); replayID != "" {
		delivery = "replay:" + replayID + ":" + ulid.Make().String()
	}
	deliveryID := "whd_" + ulid.Make().String()
	status, reason, runID := "error", "", ""
	_ = s.store.CreateWebhookDelivery(r.Context(), &domain.WebhookDelivery{
		ID: deliveryID, Provider: automation.Provider, AutomationID: automation.ID, Delivery: delivery, Kind: ev.Kind, Body: string(body),
	})
	defer func() { _ = s.store.UpdateWebhookDelivery(context.Background(), deliveryID, status, reason, runID) }()
	work, snap, prompt, skip, err := ad.Webhook.Normalize(ev)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "E_PAYLOAD", err.Error())
		return
	}
	if work.Instance == "" {
		work.Instance = automation.Instance
	}
	if skip != "" {
		status = "dropped"
		reason = skip
		writeJSON(w, http.StatusOK, map[string]string{"skipped": skip})
		return
	}
	if ev.Kind == "comment" && automation.TriggerKind != domain.TriggerComment {
		status = "dropped"
		reason = "SKIP_COMMENT"
		writeJSON(w, http.StatusOK, map[string]string{"skipped": "SKIP_COMMENT"})
		return
	}
	if automation.Instance != "" && !strings.EqualFold(automation.Instance, work.Instance) {
		status = "dropped"
		reason = "SKIP_INSTANCE"
		writeJSON(w, http.StatusOK, map[string]string{"skipped": "SKIP_INSTANCE"})
		return
	}
	if automation.TriggerKind == "created" && ev.Kind != "created" && ev.Kind != "opened" {
		status = "dropped"
		reason = "SKIP_CREATED_ONLY"
		writeJSON(w, http.StatusOK, map[string]string{"skipped": "SKIP_CREATED_ONLY"})
		return
	}
	if automation.TriggerKind == "updated" && ev.Kind != "updated" && ev.Kind != "edited" && ev.Kind != "status" {
		status = "dropped"
		reason = "SKIP_UPDATED_ONLY"
		writeJSON(w, http.StatusOK, map[string]string{"skipped": "SKIP_UPDATED_ONLY"})
		return
	}
	if !policy.Match(automation, ev) {
		status = "dropped"
		reason = "SKIP_TRIGGER"
		writeJSON(w, http.StatusOK, map[string]string{"skipped": "SKIP_TRIGGER"})
		return
	}
	if ev.Kind == "comment" {
		if !actorAllowed(commentAllowlist(automation.Writeback), ev.Actor) {
			status = "dropped"
			reason = "SKIP_ACTOR"
			writeJSON(w, http.StatusOK, map[string]string{"skipped": "SKIP_ACTOR"})
			return
		}
		if strings.Contains(ev.Text, "<!-- orca:") || strings.Contains(ev.Text, integration.OutboundCommentMarker) {
			status = "dropped"
			reason = "SKIP_ECHO"
			writeJSON(w, http.StatusOK, map[string]string{"skipped": "SKIP_ECHO"})
			return
		}
		if automation.TriggerValue != "" && !strings.Contains(ev.Text, automation.TriggerValue) {
			status = "dropped"
			reason = "SKIP_COMMAND"
			writeJSON(w, http.StatusOK, map[string]string{"skipped": "SKIP_COMMAND"})
			return
		}
		if text := strings.TrimSpace(ev.Text); text != "" {
			prompt = "评论：" + text
		}
	}
	if s.claimBurst(r.Context(), automation.Provider+":"+work.Type+":"+work.Instance+":"+work.ID) {
		status = "dropped"
		reason = "DEBOUNCED"
		writeJSON(w, http.StatusOK, map[string]string{"skipped": "DEBOUNCED"})
		return
	}
	agentID := automation.AgentID
	if agentID == "" {
		agentID = "claude"
	}

	key := automation.Provider + ":" + delivery
	if ev.Kind == "comment" {
		key += ":" + shortHash(ev.Text)
	}
	claimed, err := s.store.ClaimIdempotency(r.Context(), key, "")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !claimed {
		status = "deduped"
		writeJSON(w, http.StatusOK, map[string]string{"deduped": "true"})
		return
	}

	id := "run_" + ulid.Make().String()
	run := &domain.Run{
		ID:           id,
		WorkspaceID:  automation.WorkspaceID,
		Work:         work,
		Snapshot:     snap,
		AutomationID: automation.ID,
		AgentID:      agentID,
		Status:       domain.RunQueued,
		Priority:     automation.Priority,
		Repo:         work.Instance,
		Base:         "main",
		Branch:       "orca/" + id[len(id)-8:],
		Prompt:       prompt,
		CreatedAt:    time.Now().UTC(),
	}
	degraded, degradeReason := false, ""
	if task, err := s.store.EnsureTask(r.Context(), work, snap); err == nil && task != nil {
		if len(ev.Attachments) > 0 || strings.HasPrefix(ev.Kind, "jira:issue_") {
			attachments := make([]*domain.TaskAttachment, 0, len(ev.Attachments))
			for _, attachment := range ev.Attachments {
				attachments = append(attachments, &domain.TaskAttachment{
					ID: "att_" + task.ID + "_" + attachment.ID, TaskID: task.ID, Provider: task.Provider,
					ExternalID: attachment.ID, Name: attachment.Name, URL: attachment.URL,
					Mime: attachment.Mime, Size: attachment.Size, Author: attachment.Author,
				})
			}
			if err := s.store.ReplaceTaskAttachments(r.Context(), task.ID, task.Provider, "issue", attachments); err != nil {
				degraded, degradeReason = true, err.Error()
			}
		}
		if err := s.store.ReplaceTaskFields(r.Context(), task.ID, task.Provider, ev.Fields); err != nil {
			degraded, degradeReason = true, err.Error()
		}
		s.applyPlatformState(r.Context(), task, snap, automation.StateMap, conflictPolicy(automation))
		if auto := s.applyAssignmentRules(r.Context(), task); auto != nil {
			status = "accepted"
			runID = auto.ID
			writeJSON(w, http.StatusAccepted, map[string]string{"runId": auto.ID})
			return
		}
		if task.AssigneeAgentID != "" {
			agentID = task.AssigneeAgentID
			run.AgentID = agentID
		}
		run.TaskID = task.ID
	}
	if err := s.store.CreateRun(r.Context(), run); err != nil {
		status = "error"
		reason = err.Error()
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if run.TaskID != "" {
		_ = s.store.SetTaskStatus(r.Context(), run.TaskID, domain.TaskInProgress)
	}
	_, _ = s.store.AppendEvent(r.Context(), id, domain.EventRunCreated, map[string]any{
		"provider": automation.Provider,
		"url":      work.URL,
		"actor":    ev.Actor,
	})
	_ = s.store.Audit(r.Context(), ev.Actor, "webhook."+automation.Provider, id, r.RemoteAddr, map[string]any{
		"automation": automationID,
		"work":       work.ID,
	})
	status = "accepted"
	if degraded {
		status = "degraded"
		reason = degradeReason
	}
	runID = id
	writeJSON(w, http.StatusAccepted, map[string]string{"runId": id, "delivery": status})
}

// ---- helpers ----

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, errCode, msg string) {
	writeJSON(w, code, map[string]any{
		"error": map[string]string{"code": errCode, "message": msg},
	})
}
