// Package automation starts runs on a cron schedule.
package automation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/robfig/cron/v3"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/store"
)

// PrecheckError is returned when an automation precheck command fails.
type PrecheckError struct {
	Output string
	Err    error
}

func (e *PrecheckError) Error() string {
	if e.Output != "" {
		return fmt.Sprintf("precheck failed: %v: %s", e.Err, e.Output)
	}
	return fmt.Sprintf("precheck failed: %v", e.Err)
}

func (e *PrecheckError) Unwrap() error { return e.Err }

// Runner polls due automations and starts runs.
type Runner struct {
	Store *store.Store
	Log   *slog.Logger
	Now   func() time.Time
}

// New creates a runner.
func New(st *store.Store, log *slog.Logger) *Runner {
	if log == nil {
		log = slog.Default()
	}
	return &Runner{Store: st, Log: log, Now: time.Now}
}

// Run polls every 30 seconds until the context ends.
func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n, err := r.Tick(ctx); err != nil {
				r.Log.Warn("automation tick", "err", err)
			} else if n > 0 {
				r.Log.Info("automations started", "count", n)
			}
		}
	}
}

// Tick starts every due automation.
func (r *Runner) Tick(ctx context.Context) (int, error) {
	now := r.Now().UTC()
	due, err := r.Store.DueAutomations(ctx, now)
	if err != nil {
		return 0, err
	}
	started := 0
	for _, a := range due {
		schedule, err := cron.ParseStandard(a.Schedule)
		if err != nil {
			r.Log.Warn("invalid automation schedule", "automation", a.ID, "err", err)
			_ = r.Store.MarkAutomationRun(ctx, a.ID, now, now.Add(24*time.Hour), "invalid_schedule", err.Error())
			continue
		}
		if a.NextRun != nil && a.MissedGraceSeconds > 0 {
			grace := time.Duration(a.MissedGraceSeconds) * time.Second
			if now.Sub(*a.NextRun) > grace {
				next := schedule.Next(now)
				r.Log.Warn("automation missed grace window", "automation", a.ID, "due", a.NextRun)
				_ = r.Store.MarkAutomationRun(ctx, a.ID, now, next, "skipped_missed", "missed grace window")
				continue
			}
		}
		if _, err := StartRun(ctx, r.Store, a, now); err != nil {
			var pre *PrecheckError
			if errors.As(err, &pre) {
				r.Log.Warn("automation precheck failed", "automation", a.ID, "err", err)
				_ = r.Store.MarkAutomationRun(ctx, a.ID, now, schedule.Next(now), "precheck_failed", err.Error())
				continue
			}
			r.Log.Warn("start automation", "automation", a.ID, "err", err)
			_ = r.Store.MarkAutomationRun(ctx, a.ID, now, schedule.Next(now), "start_failed", err.Error())
			continue
		}
		if err := r.Store.MarkAutomationRun(ctx, a.ID, now, schedule.Next(now), "started", ""); err != nil {
			r.Log.Warn("update automation", "automation", a.ID, "err", err)
		}
		started++
	}
	return started, nil
}

// RunPrecheck executes the optional precheck command.
func RunPrecheck(ctx context.Context, a *domain.Automation) error {
	script := strings.TrimSpace(a.Precheck)
	if script == "" {
		return nil
	}
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(runCtx, "cmd", "/C", script)
	} else {
		cmd = exec.CommandContext(runCtx, "sh", "-c", script)
	}
	cmd.Env = append(os.Environ(), "TANDEMESH_AUTOMATION_ID="+a.ID, "TANDEMESH_AUTOMATION_NAME="+a.Name)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	text := strings.TrimSpace(string(out))
	if len(text) > 2000 {
		text = text[:2000]
	}
	return &PrecheckError{Output: text, Err: err}
}

// StartRun creates a run for an automation.
func StartRun(ctx context.Context, st *store.Store, a *domain.Automation, now time.Time) (*domain.Run, error) {
	if err := RunPrecheck(ctx, a); err != nil {
		return nil, err
	}
	id := "run_" + ulid.Make().String()
	base := a.Base
	if base == "" {
		base = "main"
	}
	sessionID := ""
	if a.ReuseSession {
		sessionID = a.SessionID
		if sessionID == "" {
			sess, err := st.CreateAgentSession(ctx, &domain.AgentSession{
				WorkspaceID: a.WorkspaceID,
				AgentID:     a.AgentID,
				Mode:        "chat",
				Title:       a.Name,
			})
			if err != nil {
				return nil, err
			}
			sessionID = sess.ID
			if err := st.SetAutomationSession(ctx, a.ID, sessionID); err != nil {
				return nil, err
			}
			a.SessionID = sessionID
		}
	}
	run := &domain.Run{
		ID:          id,
		WorkspaceID: a.WorkspaceID,
		Work:        domain.WorkRef{Provider: "automation", Instance: "schedule", Type: "schedule", ID: a.ID},
		Snapshot: domain.Snapshot{
			Title: a.Name,
		},
		AgentID:   a.AgentID,
		Status:    domain.RunQueued,
		Repo:      a.Repo,
		Base:      base,
		Branch:    "orca/" + id[len(id)-8:],
		Prompt:    a.Prompt,
		SessionID: sessionID,
		CreatedAt: now,
	}
	if a.ReuseSession {
		// A reused session keeps a stable working directory; worktrees are per run.
		run.Repo = ""
		run.Branch = ""
	}
	if task, err := st.EnsureTask(ctx, run.Work, run.Snapshot); err == nil && task != nil {
		run.TaskID = task.ID
	}
	if err := st.CreateRun(ctx, run); err != nil {
		return nil, err
	}
	if run.TaskID != "" {
		_ = st.SetTaskStatus(ctx, run.TaskID, domain.TaskInProgress)
	}
	if sessionID != "" {
		_ = st.RecordSessionRun(ctx, sessionID, "", run.ID)
	}
	_, _ = st.AppendEvent(ctx, id, domain.EventRunCreated, map[string]any{
		"provider":   "automation",
		"automation": a.ID,
		"taskId":     run.TaskID,
		"sessionId":  sessionID,
	})
	return run, nil
}
