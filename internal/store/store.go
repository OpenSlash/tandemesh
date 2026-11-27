// Package store persists execution facts.
//
// SQLite is the default; a postgres:// DSN switches to Postgres while keeping
// the same SQL (see internal/store/pgdriver for placeholder rebinding).
package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/OpenSlash/tandemesh/internal/domain"
	_ "github.com/OpenSlash/tandemesh/internal/store/pgdriver"
)

//go:embed schema.sql
var schemaFS embed.FS

// Store is a SQL-backed repository.
type Store struct {
	db       *sql.DB
	postgres bool
}

// Open opens (and creates if needed) the database. A postgres:// or
// postgresql:// DSN selects Postgres; anything else is a SQLite file.
func Open(path string) (*Store, error) {
	if isPostgres(path) {
		db, err := sql.Open("orca-postgres", path)
		if err != nil {
			return nil, fmt.Errorf("open postgres: %w", err)
		}
		db.SetMaxOpenConns(10)
		if err := db.Ping(); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("ping postgres: %w", err)
		}
		return &Store{db: db, postgres: true}, nil
	}

	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite: single writer keeps things simple.
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return &Store{db: db}, nil
}

func isPostgres(dsn string) bool {
	return strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://")
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// Migrate applies the embedded schema. It is idempotent and runs statements
// one by one so both SQLite and Postgres accept the script. Databases created
// by the earlier multi-tenant schema are first converted to the local-only
// workspace schema.
func (s *Store) Migrate(ctx context.Context) error {
	if err := s.RebuildLocalOnly(ctx); err != nil {
		return fmt.Errorf("migrate local schema: %w", err)
	}
	raw, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	schema := string(raw)
	if s.postgres {
		schema = strings.ReplaceAll(schema, "INTEGER PRIMARY KEY AUTOINCREMENT", "BIGSERIAL PRIMARY KEY")
	}
	for _, stmt := range splitStatements(schema) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	// Additive migrations for databases created before a column existed.
	for _, stmt := range []string{
		`ALTER TABLE runs ADD COLUMN variant_of TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE runs ADD COLUMN variant_label TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE runs ADD COLUMN task_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE runs ADD COLUMN role TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tasks ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'ws_default'`,
		`ALTER TABLE tasks ADD COLUMN inbox_state TEXT NOT NULL DEFAULT 'inbox'`,
		`ALTER TABLE tasks ADD COLUMN inbox_read_at_ms INTEGER`,
		`ALTER TABLE tasks ADD COLUMN iteration_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tasks ADD COLUMN done_at_ms INTEGER`,
		`ALTER TABLE tasks ADD COLUMN parent_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agents ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'ws_default'`,
		`ALTER TABLE automations ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'ws_default'`,
		`ALTER TABLE runs ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'ws_default'`,
		`ALTER TABLE approvals ADD COLUMN message_ref TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE machines ADD COLUMN runtimes_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE machines ADD COLUMN runtime_details_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE machines ADD COLUMN metrics_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE agents ADD COLUMN machine_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_sessions ADD COLUMN repo TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_sessions ADD COLUMN base TEXT NOT NULL DEFAULT 'main'`,
		`ALTER TABLE agent_sessions ADD COLUMN worktree_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_sessions ADD COLUMN model TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_sessions ADD COLUMN reasoning TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_sessions ADD COLUMN seed_context TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_sessions ADD COLUMN created_by TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_questions ADD COLUMN expires_at_ms INTEGER`,
		`ALTER TABLE runs ADD COLUMN input_tokens INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE runs ADD COLUMN output_tokens INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE runs ADD COLUMN cache_read_tokens INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE runs ADD COLUMN cache_write_tokens INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE runs ADD COLUMN cost_usd REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE runs ADD COLUMN usage_estimated INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE runs ADD COLUMN model TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE outbox ADD COLUMN connection_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE outbox ADD COLUMN account_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tasks ADD COLUMN assignee_agent_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tasks ADD COLUMN author TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tasks ADD COLUMN auto_dispatched_at_ms INTEGER`,
		`ALTER TABLE agents ADD COLUMN role TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agents ADD COLUMN capabilities_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE tasks ADD COLUMN assignee_group_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE assignment_rules ADD COLUMN group_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE assignment_rules ADD COLUMN policy TEXT NOT NULL DEFAULT 'manual'`,
		`UPDATE assignment_rules SET policy = 'conditional' WHERE auto_dispatch = 1`,
		`UPDATE tasks SET workspace_id = 'ws_default' WHERE workspace_id = ''`,
		`ALTER TABLE webhook_deliveries ADD COLUMN attempts INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE webhook_deliveries ADD COLUMN retry_after_ms INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tasks ADD COLUMN execution_state TEXT NOT NULL DEFAULT 'ready'`,
		`ALTER TABLE tasks ADD COLUMN assignee_member_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tasks ADD COLUMN origin TEXT NOT NULL DEFAULT 'external'`,
		`ALTER TABLE runs ADD COLUMN session_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE automations ADD COLUMN precheck TEXT NOT NULL DEFAULT ''''`,
		`ALTER TABLE automations ADD COLUMN missed_grace_secs INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE automations ADD COLUMN session_id TEXT NOT NULL DEFAULT ''''`,
		`ALTER TABLE automations ADD COLUMN reuse_session INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE automations ADD COLUMN last_status TEXT NOT NULL DEFAULT ''''`,
		`ALTER TABLE automations ADD COLUMN last_error TEXT NOT NULL DEFAULT ''''`,
		`ALTER TABLE automations ADD COLUMN provider TEXT NOT NULL DEFAULT ''''`,
		`ALTER TABLE automations ADD COLUMN instance TEXT NOT NULL DEFAULT ''''`,
		`ALTER TABLE automations ADD COLUMN trigger_kind TEXT NOT NULL DEFAULT ''''`,
		`ALTER TABLE automations ADD COLUMN trigger_value TEXT NOT NULL DEFAULT ''''`,
		`ALTER TABLE automations ADD COLUMN writeback_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE automations ADD COLUMN state_map TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE automations ADD COLUMN connection_id TEXT NOT NULL DEFAULT ''''`,
		`ALTER TABLE connections ADD COLUMN sync_direction TEXT NOT NULL DEFAULT 'read_write'`,
		`ALTER TABLE agents ADD COLUMN session_mode TEXT NOT NULL DEFAULT 'off'`,
		`ALTER TABLE workflow_runs ADD COLUMN definition_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE connections ADD COLUMN automation_id TEXT NOT NULL DEFAULT ''''`,
		`ALTER TABLE resource_bindings ADD COLUMN account_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE connection_accounts ADD COLUMN member_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE resource_bindings ADD COLUMN last_synced_at_ms INTEGER`,
		`ALTER TABLE resource_bindings ADD COLUMN last_error TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE automations ADD COLUMN priority INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE skills ADD COLUMN version TEXT NOT NULL DEFAULT ''''`,
		`ALTER TABLE skills ADD COLUMN source TEXT NOT NULL DEFAULT ''manual''`,
		`ALTER TABLE skills ADD COLUMN source_url TEXT NOT NULL DEFAULT ''''`,
		`ALTER TABLE skills ADD COLUMN tags_json TEXT NOT NULL DEFAULT ''[]''`,
		`ALTER TABLE skills ADD COLUMN enabled INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE skills ADD COLUMN checksum TEXT NOT NULL DEFAULT ''''`,
		`ALTER TABLE import_batches ADD COLUMN next_start_at INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE import_batches ADD COLUMN limit_count INTEGER NOT NULL DEFAULT 100`,
		`ALTER TABLE task_attachments ADD COLUMN source TEXT NOT NULL DEFAULT 'issue'`,
		`UPDATE tasks SET execution_state = 'pending' WHERE status != 'done' AND EXISTS (SELECT 1 FROM task_dependencies d JOIN tasks dep ON dep.id = d.depends_on WHERE d.task_id = tasks.id AND dep.status != 'done')`,
	} {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil && !isDuplicateColumn(err) {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	if err := s.migrateBindings(ctx); err != nil {
		return fmt.Errorf("migrate bindings: %w", err)
	}
	return nil
}

// migrateBindings folds the legacy bindings table into automations. New
// databases never create the table, so this is a no-op there.
func (s *Store) migrateBindings(ctx context.Context) error {
	var probe int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM bindings LIMIT 1`).Scan(&probe)
	if err != nil && err != sql.ErrNoRows {
		return nil
	}
	_, copyErr := s.db.ExecContext(ctx, `INSERT INTO automations
		(id, workspace_id, name, schedule, provider, instance, trigger_kind, trigger_value,
		 writeback_json, state_map, connection_id, priority, agent_id, repo, base, prompt, enabled,
		 precheck, missed_grace_secs, session_id, reuse_session, last_status, last_error, created_at_ms)
		SELECT id, workspace_id, name, '', provider, instance, trigger_kind, trigger_value,
		 writeback_json, state_map, connection_id, priority, agent_id, '', 'main', '', enabled,
		 '', 0, '', 0, '', '', created_at_ms
		FROM bindings
		WHERE 1 = 1
		ON CONFLICT(id) DO NOTHING`)
	if copyErr != nil {
		return copyErr
	}
	for _, stmt := range []string{
		`ALTER TABLE runs RENAME COLUMN binding_id TO automation_id`,
		`ALTER TABLE webhook_deliveries RENAME COLUMN binding_id TO automation_id`,
		`DROP TABLE bindings`,
	} {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "no such column") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "already exists") {
				continue
			}
			return err
		}
	}
	return nil
}

func isDuplicateColumn(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column") || strings.Contains(msg, "already exists")
}

// splitStatements splits a SQL script on semicolons while ignoring line
// comments and quoted strings, so comments may contain punctuation.
func splitStatements(schema string) []string {
	var (
		out     []string
		current strings.Builder
		quoted  bool
	)
	flush := func() {
		stmt := strings.TrimSpace(current.String())
		for strings.HasPrefix(stmt, "--") {
			if idx := strings.IndexByte(stmt, '\n'); idx >= 0 {
				stmt = strings.TrimSpace(stmt[idx+1:])
			} else {
				stmt = ""
			}
		}
		if stmt != "" {
			out = append(out, stmt)
		}
		current.Reset()
	}
	for i := 0; i < len(schema); i++ {
		ch := schema[i]
		if quoted {
			current.WriteByte(ch)
			if ch == '\'' {
				quoted = false
			}
			continue
		}
		if ch == '\'' {
			quoted = true
			current.WriteByte(ch)
			continue
		}
		if ch == '-' && i+1 < len(schema) && schema[i+1] == '-' {
			for i < len(schema) && schema[i] != '\n' {
				current.WriteByte(schema[i])
				i++
			}
			if i < len(schema) {
				current.WriteByte('\n')
			}
			continue
		}
		if ch == ';' {
			flush()
			continue
		}
		current.WriteByte(ch)
	}
	flush()
	return out
}

// DB exposes the underlying handle for tests and advanced queries.
func (s *Store) DB() *sql.DB { return s.db }

func ms(t time.Time) int64 { return t.UTC().UnixMilli() }

func fromMS(v int64) time.Time { return time.UnixMilli(v).UTC() }

func fromNull(v sql.NullInt64) *time.Time {
	if !v.Valid {
		return nil
	}
	t := fromMS(v.Int64)
	return &t
}

// CreateRun inserts a new run in queued state.
func (s *Store) CreateRun(ctx context.Context, r *domain.Run) error {
	if r.Status == "" {
		r.Status = domain.RunQueued
	}
	if r.WorkspaceID == "" {
		r.WorkspaceID = "ws_default"
	}
	snap, err := json.Marshal(r.Snapshot)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO runs (id, workspace_id, provider, instance, work_type, work_id, work_url, work_revision,
			snapshot_json, automation_id, agent_id, status, priority, attempt, fence, repo, base, branch, prompt, created_at_ms,
			variant_of, variant_label, task_id, role, session_id)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.ID, r.WorkspaceID, r.Work.Provider, r.Work.Instance, r.Work.Type, r.Work.ID, r.Work.URL, r.Work.Revision,
		string(snap), r.AutomationID, r.AgentID, string(r.Status), r.Priority, r.Attempt, r.Fence,
		r.Repo, r.Base, r.Branch, r.Prompt, ms(r.CreatedAt), r.VariantOf, r.VariantLabel, r.TaskID, r.Role, r.SessionID)
	if err != nil {
		return fmt.Errorf("insert run: %w", err)
	}
	return nil
}

const runColumns = `id, workspace_id, provider, instance, work_type, work_id, work_url, work_revision, snapshot_json,
	automation_id, agent_id, machine_id, status, priority, attempt, fence, repo, base, branch, prompt,
	error_code, error_message, created_at_ms, started_at_ms, ended_at_ms, lease_expires_ms,
	variant_of, variant_label, task_id, role, session_id,
	input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, cost_usd, usage_estimated, model`

func scanRun(sc interface{ Scan(...any) error }) (*domain.Run, error) {
	var (
		r       domain.Run
		snap    string
		status  string
		created int64
		started sql.NullInt64
		ended   sql.NullInt64
		lease   sql.NullInt64
	)
	err := sc.Scan(&r.ID, &r.WorkspaceID, &r.Work.Provider, &r.Work.Instance, &r.Work.Type, &r.Work.ID,
		&r.Work.URL, &r.Work.Revision, &snap, &r.AutomationID, &r.AgentID, &r.MachineID, &status,
		&r.Priority, &r.Attempt, &r.Fence, &r.Repo, &r.Base, &r.Branch, &r.Prompt,
		&r.ErrorCode, &r.ErrorMessage, &created, &started, &ended, &lease,
		&r.VariantOf, &r.VariantLabel, &r.TaskID, &r.Role, &r.SessionID,
		&r.InputTokens, &r.OutputTokens, &r.CacheReadTokens, &r.CacheWriteTokens, &r.CostUSD, &r.UsageEstimated, &r.Model)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(snap), &r.Snapshot)
	r.Status = domain.RunStatus(status)
	r.CreatedAt = fromMS(created)
	r.StartedAt = fromNull(started)
	r.EndedAt = fromNull(ended)
	r.LeaseExpiresAt = fromNull(lease)
	return &r, nil
}

// ListRunVariants returns the origin run and every variant of it.
func (s *Store) ListRunVariants(ctx context.Context, rootID string) ([]*domain.Run, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+runColumns+` FROM runs
		WHERE id = ? OR variant_of = ? ORDER BY created_at_ms ASC`, rootID, rootID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// TaskID derives a stable id for an external work item.
func TaskID(work domain.WorkRef) string {
	sum := sha256.Sum256([]byte(work.Provider + "|" + work.Instance + "|" + work.Type + "|" + work.ID))
	return "task_" + hex.EncodeToString(sum[:])[:24]
}

// EnsureTask upserts the projection of an external work item.
func (s *Store) EnsureTask(ctx context.Context, work domain.WorkRef, snap domain.Snapshot) (*domain.Task, error) {
	return s.EnsureTaskIn(ctx, "ws_default", work, snap)
}

// EnsureTaskIn upserts a task projection inside a workspace.
func (s *Store) EnsureTaskIn(ctx context.Context, workspaceID string, work domain.WorkRef, snap domain.Snapshot) (*domain.Task, error) {
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	id := TaskID(work)
	labels, _ := json.Marshal(snap.Labels)
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `INSERT INTO tasks
		(id, workspace_id, provider, instance, work_type, work_id, url, revision, title, body, labels, author, assignee, status, platform_state, platform_state_group,
		 created_at_ms, updated_at_ms, synced_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(provider, instance, work_type, work_id) DO UPDATE SET
			url = excluded.url, revision = excluded.revision, title = excluded.title, body = excluded.body,
			labels = excluded.labels, author = excluded.author, assignee = excluded.assignee,
			platform_state = excluded.platform_state, platform_state_group = excluded.platform_state_group,
			updated_at_ms = excluded.updated_at_ms, synced_at_ms = excluded.synced_at_ms`,
		id, workspaceID, work.Provider, work.Instance, work.Type, work.ID, work.URL, work.Revision,
		snap.Title, snap.Body, string(labels), snap.Author, snap.Assignee, domain.TaskOpen, snap.Status, snap.StatusGroup,
		ms(now), ms(now), ms(now))
	if err != nil {
		return nil, err
	}
	return s.GetTask(ctx, id)
}

// SetTaskPlatformState records the external state and optionally maps local status.
func (s *Store) SetTaskPlatformState(ctx context.Context, id, state, group, localStatus string) error {
	now := ms(time.Now())
	if localStatus == "" {
		_, err := s.db.ExecContext(ctx, `UPDATE tasks SET platform_state = ?, platform_state_group = ?, updated_at_ms = ? WHERE id = ?`, state, group, now, id)
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET platform_state = ?, platform_state_group = ?, status = ?, updated_at_ms = ? WHERE id = ?`, state, group, localStatus, now, id)
	return err
}

// SetTaskIteration assigns a task to an iteration (empty clears it).
func (s *Store) SetTaskIteration(ctx context.Context, id, iterationID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET iteration_id = ?, updated_at_ms = ? WHERE id = ?`,
		iterationID, ms(time.Now()), id)
	return err
}

// SetTaskOrigin marks a task as local or external.
func (s *Store) SetTaskOrigin(ctx context.Context, id, origin string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET origin = ?, updated_at_ms = ? WHERE id = ?`, origin, ms(time.Now()), id)
	return err
}

// SetTaskMemberAssignee assigns a workspace member and clears agent/group assignments.
func (s *Store) SetTaskMemberAssignee(ctx context.Context, id, memberID, login string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET assignee_member_id = ?, assignee = ?, assignee_agent_id = '', assignee_group_id = '', updated_at_ms = ? WHERE id = ?`,
		memberID, login, ms(time.Now()), id)
	return err
}

// SetTaskAssigneeGroup assigns a group and clears any agent assignment.
func (s *Store) SetTaskAssigneeGroup(ctx context.Context, id, groupID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET assignee_group_id = ?, assignee_agent_id = '', updated_at_ms = ? WHERE id = ?`,
		groupID, ms(time.Now()), id)
	return err
}

// SetTaskAssignee records the agent responsible for a task (empty clears it).
func (s *Store) SetTaskAssignee(ctx context.Context, id, agentID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET assignee_agent_id = ?, assignee_group_id = '', updated_at_ms = ? WHERE id = ?`,
		agentID, ms(time.Now()), id)
	return err
}

// SetTaskStatus updates the local projection status. The source system may
// overwrite it on the next sync.
func (s *Store) SetTaskStatus(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET status = ?, updated_at_ms = ? WHERE id = ?`,
		status, ms(time.Now()), id)
	return err
}

// SetTaskExecutionState updates the dispatch/DAG state without touching local status.
func (s *Store) SetTaskExecutionState(ctx context.Context, id, state string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET execution_state = ?, updated_at_ms = ? WHERE id = ?`,
		state, ms(time.Now()), id)
	return err
}

// GetTask returns one task projection or nil.
func (s *Store) GetTask(ctx context.Context, id string) (*domain.Task, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, provider, instance, work_type, work_id, url, revision,
		title, body, labels, author, assignee, assignee_agent_id, assignee_group_id, assignee_member_id, origin, status, platform_state, platform_state_group, execution_state, created_at_ms, updated_at_ms, synced_at_ms,
		inbox_state, inbox_read_at_ms, iteration_id, done_at_ms, parent_id, auto_dispatched_at_ms,
		(SELECT COUNT(*) FROM runs WHERE runs.task_id = tasks.id),
		(SELECT COUNT(*) FROM runs WHERE runs.task_id = tasks.id AND runs.status IN ('queued','leased','preparing','running','waiting_approval','finalizing')),
		(SELECT COUNT(*) FROM runs WHERE runs.task_id = tasks.id AND runs.status = 'failed'),
		(SELECT COUNT(*) FROM runs WHERE runs.task_id = tasks.id AND runs.status = 'succeeded'),
		(SELECT agent_id FROM runs WHERE runs.task_id = tasks.id AND runs.status IN ('queued','leased','preparing','running','waiting_approval') ORDER BY started_at_ms DESC LIMIT 1),
		(SELECT COUNT(*) FROM task_comments WHERE task_comments.task_id = tasks.id),
		(SELECT body FROM task_comments WHERE task_comments.task_id = tasks.id ORDER BY created_at_ms DESC LIMIT 1),
		(SELECT author FROM task_comments WHERE task_comments.task_id = tasks.id ORDER BY created_at_ms DESC LIMIT 1)
		FROM tasks WHERE id = ?`, id)
	return scanTask(row)
}

// TaskFilter narrows a task list.
type TaskFilter struct {
	Workspace  string
	Status     string
	Provider   string
	Instance   string
	Label      string
	Agent      string
	Inbox      string
	Assignee   string
	Iteration  string
	Parent     string
	Query      string
	Unassigned bool
	Limit      int
}

// ListTasks returns task projections, newest first.
func (s *Store) ListTasks(ctx context.Context, filter TaskFilter) ([]*domain.Task, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	q := `SELECT id, workspace_id, provider, instance, work_type, work_id, url, revision,
		title, body, labels, author, assignee, assignee_agent_id, assignee_group_id, assignee_member_id, origin, status, platform_state, platform_state_group, execution_state, created_at_ms, updated_at_ms, synced_at_ms,
		inbox_state, inbox_read_at_ms, iteration_id, done_at_ms, parent_id, auto_dispatched_at_ms,
		(SELECT COUNT(*) FROM runs WHERE runs.task_id = tasks.id),
		(SELECT COUNT(*) FROM runs WHERE runs.task_id = tasks.id AND runs.status IN ('queued','leased','preparing','running','waiting_approval','finalizing')),
		(SELECT COUNT(*) FROM runs WHERE runs.task_id = tasks.id AND runs.status = 'failed'),
		(SELECT COUNT(*) FROM runs WHERE runs.task_id = tasks.id AND runs.status = 'succeeded'),
		(SELECT agent_id FROM runs WHERE runs.task_id = tasks.id AND runs.status IN ('queued','leased','preparing','running','waiting_approval') ORDER BY started_at_ms DESC LIMIT 1),
		(SELECT COUNT(*) FROM task_comments WHERE task_comments.task_id = tasks.id),
		(SELECT body FROM task_comments WHERE task_comments.task_id = tasks.id ORDER BY created_at_ms DESC LIMIT 1),
		(SELECT author FROM task_comments WHERE task_comments.task_id = tasks.id ORDER BY created_at_ms DESC LIMIT 1)
		FROM tasks`
	// Tasks kept only for their execution history after their project was
	// unbound stay out of lists; they are still reachable by id.
	where := []string{"platform_state != '" + domain.PlatformStateUnbound + "'"}
	var args []any
	if filter.Unassigned {
		where = append(where, "(assignee_agent_id = '' AND assignee_group_id = '')")
	}
	if filter.Workspace != "" {
		where = append(where, "workspace_id = ?")
		args = append(args, filter.Workspace)
	}
	if filter.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Provider != "" {
		where = append(where, "provider = ?")
		args = append(args, filter.Provider)
	}
	if filter.Instance != "" {
		where = append(where, "instance = ?")
		args = append(args, filter.Instance)
	}
	if filter.Label != "" {
		where = append(where, "labels LIKE ?")
		args = append(args, "%\""+filter.Label+"\"%")
	}
	if filter.Agent != "" {
		where = append(where, "EXISTS (SELECT 1 FROM runs WHERE runs.task_id = tasks.id AND runs.agent_id = ?)")
		args = append(args, filter.Agent)
	}
	if filter.Inbox != "" {
		where = append(where, "inbox_state = ?")
		args = append(args, filter.Inbox)
	}
	if filter.Assignee != "" {
		where = append(where, "assignee = ?")
		args = append(args, filter.Assignee)
	}
	if filter.Iteration != "" {
		where = append(where, "iteration_id = ?")
		args = append(args, filter.Iteration)
	}
	if filter.Parent != "" {
		where = append(where, "parent_id = ?")
		args = append(args, filter.Parent)
	}
	if filter.Query != "" {
		where = append(where, "(title LIKE ? OR body LIKE ? OR work_id LIKE ?)")
		like := "%" + filter.Query + "%"
		args = append(args, like, like, like)
	}
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY updated_at_ms DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListRunsByTask returns every run started for a task.
func (s *Store) ListRunsByTask(ctx context.Context, taskID string) ([]*domain.Run, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+runColumns+` FROM runs WHERE task_id = ?
		ORDER BY created_at_ms ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AddTaskNote stores a local instruction for the agents working a task.
func (s *Store) AddTaskNote(ctx context.Context, n *domain.TaskNote) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO task_notes (id, task_id, author, body, created_at_ms)
		VALUES (?,?,?,?,?)`, n.ID, n.TaskID, n.Author, n.Body, ms(n.CreatedAt))
	return err
}

// ListTaskNotes returns notes in creation order.
func (s *Store) ListTaskNotes(ctx context.Context, taskID string) ([]*domain.TaskNote, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, task_id, author, body, created_at_ms
		FROM task_notes WHERE task_id = ? ORDER BY created_at_ms ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.TaskNote
	for rows.Next() {
		var (
			n       domain.TaskNote
			created int64
		)
		if err := rows.Scan(&n.ID, &n.TaskID, &n.Author, &n.Body, &created); err != nil {
			return nil, err
		}
		n.CreatedAt = fromMS(created)
		out = append(out, &n)
	}
	return out, rows.Err()
}

// ListWorkspaces returns all local workspaces.
func (s *Store) ListWorkspaces(ctx context.Context) ([]*domain.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, slug, created_at_ms FROM workspaces ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Workspace
	for rows.Next() {
		var (
			w       domain.Workspace
			created int64
		)
		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &created); err != nil {
			return nil, err
		}
		w.CreatedAt = fromMS(created)
		out = append(out, &w)
	}
	return out, rows.Err()
}

// CreateWorkspace inserts a local workspace.
func (s *Store) CreateWorkspace(ctx context.Context, w *domain.Workspace) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO workspaces (id, name, slug, created_at_ms)
		VALUES (?,?,?,?)`, w.ID, w.Name, w.Slug, ms(time.Now()))
	return err
}

// GetWorkspace returns one workspace or nil.
func (s *Store) GetWorkspace(ctx context.Context, id string) (*domain.Workspace, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, slug, created_at_ms FROM workspaces WHERE id = ?`, id)
	var (
		w       domain.Workspace
		created int64
	)
	err := row.Scan(&w.ID, &w.Name, &w.Slug, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	w.CreatedAt = fromMS(created)
	return &w, nil
}

// UpdateWorkspaceName renames a workspace and returns the updated row.
func (s *Store) UpdateWorkspaceName(ctx context.Context, id, name string) (*domain.Workspace, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE workspaces SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, nil
	}
	return s.GetWorkspace(ctx, id)
}

// ListGroups returns groups in a workspace.
func (s *Store) ListGroups(ctx context.Context, workspaceID string) ([]*domain.Group, error) {
	q := `SELECT id, workspace_id, name, description, created_at_ms FROM groups`
	var args []any
	if workspaceID != "" {
		q += " WHERE workspace_id = ?"
		args = append(args, workspaceID)
	}
	q += " ORDER BY created_at_ms"
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Group
	for rows.Next() {
		var (
			g       domain.Group
			created int64
		)
		if err := rows.Scan(&g.ID, &g.WorkspaceID, &g.Name, &g.Description, &created); err != nil {
			return nil, err
		}
		g.CreatedAt = fromMS(created)
		out = append(out, &g)
	}
	return out, rows.Err()
}

// GetGroup returns one group or nil.
func (s *Store) GetGroup(ctx context.Context, id string) (*domain.Group, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, name, description, created_at_ms
		FROM groups WHERE id = ?`, id)
	var (
		g       domain.Group
		created int64
	)
	err := row.Scan(&g.ID, &g.WorkspaceID, &g.Name, &g.Description, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	g.CreatedAt = fromMS(created)
	return &g, nil
}

// CreateGroup inserts a group.
func (s *Store) CreateGroup(ctx context.Context, g *domain.Group) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO groups (id, workspace_id, name, description, created_at_ms)
		VALUES (?,?,?,?,?)`, g.ID, g.WorkspaceID, g.Name, g.Description, ms(time.Now()))
	return err
}

// AddGroupMember upserts a member role.
func (s *Store) AddGroupMember(ctx context.Context, m *domain.GroupMember) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO group_members (group_id, agent_id, role) VALUES (?,?,?)
		ON CONFLICT(group_id, agent_id) DO UPDATE SET role = excluded.role`,
		m.GroupID, m.AgentID, m.Role)
	return err
}

// ListGroupMembers returns members of a group.
func (s *Store) ListGroupMembers(ctx context.Context, groupID string) ([]*domain.GroupMember, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT group_id, agent_id, role FROM group_members WHERE group_id = ? ORDER BY agent_id`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.GroupMember
	for rows.Next() {
		var m domain.GroupMember
		if err := rows.Scan(&m.GroupID, &m.AgentID, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

func scanTask(sc interface{ Scan(...any) error }) (*domain.Task, error) {
	var (
		t                        domain.Task
		labels                   string
		created, updated, synced int64
		readAt                   sql.NullInt64
		doneAt                   sql.NullInt64
		autoDispatched           sql.NullInt64
		activeAgent              sql.NullString
		lastComment              sql.NullString
		lastCommentBy            sql.NullString
	)
	err := sc.Scan(&t.ID, &t.WorkspaceID, &t.Provider, &t.Instance, &t.WorkType, &t.WorkID, &t.URL, &t.Revision,
		&t.Title, &t.Body, &labels, &t.Author, &t.Assignee, &t.AssigneeAgentID, &t.AssigneeGroupID, &t.AssigneeMemberID, &t.Origin, &t.Status, &t.PlatformState, &t.PlatformStateGroup, &t.ExecutionState, &created, &updated, &synced,
		&t.InboxState, &readAt, &t.IterationID, &doneAt, &t.ParentID, &autoDispatched,
		&t.RunCount, &t.ActiveRuns, &t.FailedRuns, &t.Succeeded,
		&activeAgent, &t.CommentCount, &lastComment, &lastCommentBy)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.ReadyToClose = t.RunCount > 0 && t.ActiveRuns == 0 && t.FailedRuns == 0 && t.Succeeded > 0
	_ = json.Unmarshal([]byte(labels), &t.Labels)
	t.CreatedAt = fromMS(created)
	t.UpdatedAt = fromMS(updated)
	t.SyncedAt = fromMS(synced)
	t.InboxReadAt = fromNull(readAt)
	t.DoneAt = fromNull(doneAt)
	t.AutoDispatchedAt = fromNull(autoDispatched)
	if activeAgent.Valid {
		t.ActiveAgentID = activeAgent.String
	}
	if lastComment.Valid {
		t.LastComment = lastComment.String
	}
	if lastCommentBy.Valid {
		t.LastCommentBy = lastCommentBy.String
	}
	return &t, nil
}

// GetRun returns one run.
func (s *Store) GetRun(ctx context.Context, id string) (*domain.Run, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+runColumns+` FROM runs WHERE id = ?`, id)
	r, err := scanRun(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

// ListRuns lists runs, optionally filtered by status.
func (s *Store) ListRuns(ctx context.Context, workspaceID, status string, limit int) ([]*domain.Run, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := `SELECT ` + runColumns + ` FROM runs`
	where := []string{}
	args := []any{}
	if workspaceID != "" {
		where = append(where, "workspace_id = ?")
		args = append(args, workspaceID)
	}
	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, " AND ")
	}
	q += ` ORDER BY priority DESC, created_at_ms ASC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AssignRun leases a queued run to a machine. It returns the new fence.
func (s *Store) AssignRun(ctx context.Context, id, machineID string, leaseUntil time.Time) (int64, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	var fence int64
	if err := tx.QueryRowContext(ctx, `SELECT status, fence FROM runs WHERE id = ?`, id).Scan(&status, &fence); err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	if domain.RunStatus(status) != domain.RunQueued {
		return 0, false, nil
	}
	fence++
	res, err := tx.ExecContext(ctx, `UPDATE runs
		SET machine_id = ?, status = ?, fence = ?, lease_expires_ms = ?,
		    started_at_ms = COALESCE(started_at_ms, ?)
		WHERE id = ? AND status = ? AND fence = ?`,
		machineID, string(domain.RunLeased), fence, ms(leaseUntil), ms(time.Now()), id, string(domain.RunQueued), fence-1)
	if err != nil {
		return 0, false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return 0, false, nil
	}
	if err := tx.Commit(); err != nil {
		return 0, false, err
	}
	return fence, true, nil
}

// UpdateStatus transitions a run and records an optional error.
func (s *Store) UpdateStatus(ctx context.Context, id string, from, to domain.RunStatus, code, message string) (bool, error) {
	var ended any
	if to.Terminal() {
		ended = ms(time.Now())
	}
	res, err := s.db.ExecContext(ctx, `UPDATE runs SET status = ?, error_code = ?, error_message = ?, ended_at_ms = COALESCE(?, ended_at_ms)
		WHERE id = ? AND status = ?`, string(to), code, message, ended, id, string(from))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ExpireLeases re-queues runs whose lease has expired, or fails them after the
// maximum number of attempts. It returns the affected runs.
func (s *Store) ExpireLeases(ctx context.Context, now time.Time, maxAttempts int) ([]*domain.Run, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+runColumns+` FROM runs
		WHERE status IN ('leased','preparing','running') AND lease_expires_ms IS NOT NULL AND lease_expires_ms < ?`, ms(now))
	if err != nil {
		return nil, err
	}
	var expired []*domain.Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		expired = append(expired, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, r := range expired {
		next := r.Attempt + 1
		if next >= maxAttempts {
			_, err = s.db.ExecContext(ctx, `UPDATE runs SET status = ?, error_code = ?, error_message = ?, ended_at_ms = ?
				WHERE id = ? AND status = ?`, string(domain.RunFailed), "E_ATTEMPTS", "lease expired too many times", ms(now), r.ID, string(r.Status))
		} else {
			_, err = s.db.ExecContext(ctx, `UPDATE runs SET status = ?, machine_id = '', attempt = ?, lease_expires_ms = NULL
				WHERE id = ? AND status = ?`, string(domain.RunQueued), next, r.ID, string(r.Status))
		}
		if err != nil {
			return nil, err
		}
	}
	return expired, nil
}

// AppendEvent appends one event, allocating the next sequence number.
func (s *Store) AppendEvent(ctx context.Context, runID, typ string, payload map[string]any) (domain.Event, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return domain.Event{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Event{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var seq int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq),0)+1 FROM run_events WHERE run_id = ?`, runID).Scan(&seq); err != nil {
		return domain.Event{}, err
	}
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO run_events (run_id, seq, ts_ms, type, payload) VALUES (?,?,?,?,?)`,
		runID, seq, ms(now), typ, string(raw)); err != nil {
		return domain.Event{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Event{}, err
	}
	ev := domain.Event{RunID: runID, Seq: seq, TS: now, Type: typ, Payload: payload}
	return ev, nil
}

// ListEvents returns events with seq greater than after.
func (s *Store) ListEvents(ctx context.Context, runID string, after int) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT run_id, seq, ts_ms, type, payload FROM run_events
		WHERE run_id = ? AND seq > ? ORDER BY seq ASC LIMIT 500`, runID, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Event
	for rows.Next() {
		var (
			ev  domain.Event
			ts  int64
			raw string
		)
		if err := rows.Scan(&ev.RunID, &ev.Seq, &ts, &ev.Type, &raw); err != nil {
			return nil, err
		}
		ev.TS = fromMS(ts)
		_ = json.Unmarshal([]byte(raw), &ev.Payload)
		out = append(out, ev)
	}
	return out, rows.Err()
}

// ListEventsTail returns the newest events in chronological order.
func (s *Store) ListEventsTail(ctx context.Context, runID string, limit int) ([]domain.Event, error) {
	return s.listEventsQuery(ctx, `SELECT run_id, seq, ts_ms, type, payload FROM run_events WHERE run_id = ? ORDER BY seq DESC LIMIT ?`, runID, limit)
}

// ListEventsBefore returns events older than a cursor in chronological order.
func (s *Store) ListEventsBefore(ctx context.Context, runID string, before int, limit int) ([]domain.Event, error) {
	return s.listEventsQuery(ctx, `SELECT run_id, seq, ts_ms, type, payload FROM run_events WHERE run_id = ? AND seq < ? ORDER BY seq DESC LIMIT ?`, runID, before, limit)
}

func (s *Store) listEventsQuery(ctx context.Context, query string, args ...any) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Event
	for rows.Next() {
		var (
			ev  domain.Event
			ts  int64
			raw string
		)
		if err := rows.Scan(&ev.RunID, &ev.Seq, &ts, &ev.Type, &raw); err != nil {
			return nil, err
		}
		ev.TS = fromMS(ts)
		_ = json.Unmarshal([]byte(raw), &ev.Payload)
		out = append(out, ev)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, rows.Err()
}

// TakeoverMachine moves agents to another machine and requeues active runs
// from the source in one transaction.
func (s *Store) TakeoverMachine(ctx context.Context, from, to string) (int64, int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()

	agentRes, err := tx.ExecContext(ctx, `UPDATE agents SET machine_id = ? WHERE machine_id = ?`, to, from)
	if err != nil {
		return 0, 0, err
	}
	agents, _ := agentRes.RowsAffected()

	runRes, err := tx.ExecContext(ctx, `UPDATE runs SET status = ?, machine_id = '', lease_expires_ms = NULL
		WHERE machine_id = ? AND status IN (?,?,?,?)`,
		string(domain.RunQueued), from, string(domain.RunLeased), string(domain.RunPreparing),
		string(domain.RunRunning), string(domain.RunWaitingApproval))
	if err != nil {
		return 0, 0, err
	}
	runs, _ := runRes.RowsAffected()
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return agents, runs, nil
}

// ReassignAgentsMachine moves agents bound to one machine to another.
func (s *Store) ReassignAgentsMachine(ctx context.Context, from, to string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE agents SET machine_id = ? WHERE machine_id = ?`, to, from)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// RequeueRunsFromMachine releases active runs so another machine can take them.
func (s *Store) RequeueRunsFromMachine(ctx context.Context, machineID string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE runs SET status = ?, machine_id = '', lease_expires_ms = NULL
		WHERE machine_id = ? AND status IN (?,?,?,?)`,
		string(domain.RunQueued), machineID, string(domain.RunLeased), string(domain.RunPreparing),
		string(domain.RunRunning), string(domain.RunWaitingApproval))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func parseMachineMetrics(raw string) *domain.MachineMetrics {
	var metrics domain.MachineMetrics
	if err := json.Unmarshal([]byte(raw), &metrics); err != nil || metrics.CollectedAt.IsZero() {
		return nil
	}
	return &metrics
}

const machineColumns = `id, name, labels_json, agents_json, runtimes_json, runtime_details_json, metrics_json, concurrency, status, version, last_seen_ms`

type rowScanner interface {
	Scan(dest ...any) error
}

// scanMachine reads one row selected with machineColumns.
func scanMachine(row rowScanner) (*domain.Machine, error) {
	var (
		m        domain.Machine
		labels   string
		agents   string
		runtimes string
		details  string
		metrics  string
		lastSee  int64
	)
	if err := row.Scan(&m.ID, &m.Name, &labels, &agents, &runtimes, &details, &metrics, &m.Concurrency, &m.Status, &m.Version, &lastSee); err != nil {
		return nil, err
	}
	// Columns default to valid JSON; a malformed value leaves the field empty.
	_ = json.Unmarshal([]byte(labels), &m.Labels)
	_ = json.Unmarshal([]byte(agents), &m.Agents)
	_ = json.Unmarshal([]byte(runtimes), &m.Runtimes)
	_ = json.Unmarshal([]byte(details), &m.RuntimeDetails)
	m.Metrics = parseMachineMetrics(metrics)
	m.LastSeen = fromMS(lastSee)
	return &m, nil
}

// GetMachineByName returns a machine by name, case-insensitively.
func (s *Store) GetMachineByName(ctx context.Context, name string) (*domain.Machine, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+machineColumns+` FROM machines WHERE LOWER(name) = LOWER(?) LIMIT 1`, name)
	m, err := scanMachine(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

// GetMachine returns one machine by ID or nil.
func (s *Store) GetMachine(ctx context.Context, id string) (*domain.Machine, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+machineColumns+` FROM machines WHERE id = ?`, id)
	m, err := scanMachine(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

// UpsertMachine registers or updates a machine and refreshes its heartbeat.
func (s *Store) UpsertMachine(ctx context.Context, m *domain.Machine) error {
	labels, _ := json.Marshal(m.Labels)
	agents, _ := json.Marshal(m.Agents)
	runtimes, _ := json.Marshal(m.Runtimes)
	details, _ := json.Marshal(m.RuntimeDetails)
	if m.RuntimeDetails == nil {
		details = []byte("[]")
	}
	metrics, _ := json.Marshal(m.Metrics)
	if m.Metrics == nil {
		metrics = []byte("{}")
	}
	now := ms(time.Now())
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO machines (id, name, labels_json, agents_json, runtimes_json, runtime_details_json, metrics_json, concurrency, status, version, last_seen_ms, created_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name, labels_json = excluded.labels_json, agents_json = excluded.agents_json,
			runtimes_json = excluded.runtimes_json, runtime_details_json = excluded.runtime_details_json,
			metrics_json = excluded.metrics_json,
			concurrency = excluded.concurrency, status = excluded.status, version = excluded.version,
			last_seen_ms = excluded.last_seen_ms`,
		m.ID, m.Name, string(labels), string(agents), string(runtimes), string(details), string(metrics), m.Concurrency, m.Status, m.Version, now, now)
	return err
}

// Heartbeat refreshes a machine's last-seen timestamp and optionally its runtimes.
func (s *Store) Heartbeat(ctx context.Context, id string, runtimes ...map[string]string) error {
	if len(runtimes) > 0 {
		raw, _ := json.Marshal(runtimes[0])
		_, err := s.db.ExecContext(ctx, `UPDATE machines SET last_seen_ms = ?, status = 'online', runtimes_json = ? WHERE id = ?`,
			ms(time.Now()), string(raw), id)
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE machines SET last_seen_ms = ?, status = 'online' WHERE id = ?`, ms(time.Now()), id)
	return err
}

// SetMachineRuntimes stores what a host can run, as reported by the host.
// A nil agents slice leaves the configured backends unchanged.
func (s *Store) SetMachineRuntimes(ctx context.Context, id string, runtimes map[string]string, details []domain.RuntimeStatus, agents []string) error {
	if runtimes == nil {
		runtimes = map[string]string{}
	}
	if details == nil {
		details = []domain.RuntimeStatus{}
	}
	rawRuntimes, _ := json.Marshal(runtimes)
	rawDetails, _ := json.Marshal(details)
	if agents == nil {
		_, err := s.db.ExecContext(ctx, `UPDATE machines SET runtimes_json = ?, runtime_details_json = ? WHERE id = ?`,
			string(rawRuntimes), string(rawDetails), id)
		return err
	}
	rawAgents, _ := json.Marshal(agents)
	_, err := s.db.ExecContext(ctx, `UPDATE machines SET runtimes_json = ?, runtime_details_json = ?, agents_json = ? WHERE id = ?`,
		string(rawRuntimes), string(rawDetails), string(rawAgents), id)
	return err
}

// SetMachineMetrics stores the latest resource sample for a machine.
func (s *Store) SetMachineMetrics(ctx context.Context, id string, metrics *domain.MachineMetrics) error {
	if metrics == nil {
		return nil
	}
	raw, _ := json.Marshal(metrics)
	_, err := s.db.ExecContext(ctx, `UPDATE machines SET metrics_json = ?, last_seen_ms = ?, status = 'online' WHERE id = ?`,
		string(raw), ms(time.Now()), id)
	return err
}

// ListMachines returns all machines ordered by name.
func (s *Store) ListMachines(ctx context.Context) ([]*domain.Machine, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+machineColumns+` FROM machines ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Machine
	for rows.Next() {
		m, err := scanMachine(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListStaleMachines returns online machines whose heartbeat is older than cutoff.
func (s *Store) ListStaleMachines(ctx context.Context, cutoff time.Time) ([]*domain.Machine, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+machineColumns+`
		FROM machines WHERE status = 'online' AND last_seen_ms < ? ORDER BY last_seen_ms`, ms(cutoff))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Machine
	for rows.Next() {
		m, err := scanMachine(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ErrMachineInUse means agents or connections still run on the machine.
var ErrMachineInUse = errors.New("machine still has agents or connections")

// DeleteMachine removes a host and the per-host queues and caches that only
// it would consume. Runs keep their machine id: they are execution history.
func (s *Store) DeleteMachine(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete machine: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var refs int
	if err := tx.QueryRowContext(ctx, `SELECT
		(SELECT COUNT(*) FROM agents WHERE machine_id = ?) +
		(SELECT COUNT(*) FROM connections WHERE runner_machine_id = ?)`, id, id).Scan(&refs); err != nil {
		return fmt.Errorf("count machine references: %w", err)
	}
	if refs > 0 {
		return ErrMachineInUse
	}
	for _, stmt := range []string{
		`DELETE FROM machine_jobs WHERE machine_id = ?`,
		`DELETE FROM history_scans WHERE machine_id = ?`,
		`DELETE FROM machine_history WHERE machine_id = ?`,
		`DELETE FROM history_cache WHERE machine_id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, stmt, id); err != nil {
			return fmt.Errorf("delete machine: %w", err)
		}
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM machines WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete machine: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

// SetMachineStatus updates a machine's scheduling status.
func (s *Store) SetMachineStatus(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE machines SET status = ? WHERE id = ?`, status, id)
	return err
}

// ClaimIdempotency records a key. It reports false when the key already exists.
func (s *Store) ClaimIdempotency(ctx context.Context, key, runID string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO idempotency (key, run_id, created_ms) VALUES (?,?,?)
		ON CONFLICT DO NOTHING`,
		key, runID, ms(time.Now()))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// Audit appends an audit record.
func (s *Store) Audit(ctx context.Context, actor, action, target, ip string, meta map[string]any) error {
	raw, _ := json.Marshal(meta)
	_, err := s.db.ExecContext(ctx, `INSERT INTO audit_log (ts_ms, actor, action, target, ip, meta) VALUES (?,?,?,?,?,?)`,
		ms(time.Now()), actor, action, target, ip, string(raw))
	return err
}

// CancelRun cancels a non-terminal run.
func (s *Store) CancelRun(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE runs SET status = ?, ended_at_ms = ?
		WHERE id = ? AND status NOT IN ('succeeded','failed','cancelled')`,
		string(domain.RunCancelled), ms(time.Now()), id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// CheckFence reports whether the run's current fence matches.
func (s *Store) CheckFence(ctx context.Context, id string, fence int64) (bool, error) {
	var current int64
	err := s.db.QueryRowContext(ctx, `SELECT fence FROM runs WHERE id = ?`, id).Scan(&current)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return current == fence, nil
}

// RenewLease extends a run lease while the daemon is still executing it.
func (s *Store) RenewLease(ctx context.Context, id, machineID string, fence int64, until time.Time) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE runs SET lease_expires_ms = ?
		WHERE id = ? AND machine_id = ? AND fence = ? AND status IN (?,?,?,?)`,
		ms(until), id, machineID, fence, string(domain.RunLeased), string(domain.RunPreparing),
		string(domain.RunRunning), string(domain.RunWaitingApproval))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// SetRunUsage records token usage for a run.
func (s *Store) SetRunUsage(ctx context.Context, id string, input, output, cacheRead, cacheWrite int, cost float64, estimated bool, model string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE runs SET input_tokens = ?, output_tokens = ?, cache_read_tokens = ?,
		cache_write_tokens = ?, cost_usd = ?, usage_estimated = ?, model = ? WHERE id = ?`,
		input, output, cacheRead, cacheWrite, cost, estimated, model, id)
	return err
}

// NextLease returns the oldest leased run for a machine, if any.
func (s *Store) NextLease(ctx context.Context, machineID string) (*domain.Run, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+runColumns+` FROM runs
		WHERE machine_id = ? AND status = ? ORDER BY created_at_ms ASC LIMIT 1`,
		machineID, string(domain.RunLeased))
	r, err := scanRun(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

// UpsertAgent creates or replaces an agent.
func (s *Store) UpsertAgent(ctx context.Context, a *domain.Agent) error {
	if a.WorkspaceID == "" {
		a.WorkspaceID = "ws_default"
	}
	skills, _ := json.Marshal(a.Skills)
	repos, _ := json.Marshal(a.Repos)
	tools, _ := json.Marshal(a.Tools)
	caps, _ := json.Marshal(a.Capabilities)
	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agents (id, workspace_id, machine_id, name, instructions, backend, role, capabilities_json, model, skills_json, repos_json, tools_json, session_mode, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			machine_id = excluded.machine_id,
			workspace_id = excluded.workspace_id,
			name = excluded.name, instructions = excluded.instructions, backend = excluded.backend,
			role = excluded.role, capabilities_json = excluded.capabilities_json,
			model = excluded.model, skills_json = excluded.skills_json, repos_json = excluded.repos_json,
			tools_json = excluded.tools_json, session_mode = excluded.session_mode, updated_at_ms = excluded.updated_at_ms`,
		a.ID, a.WorkspaceID, a.MachineID, a.Name, a.Instructions, a.Backend, a.Role, string(caps), a.Model, string(skills), string(repos), string(tools), a.SessionMode, ms(a.CreatedAt), ms(a.UpdatedAt))
	return err
}

// GetAgent returns one agent or nil.
func (s *Store) GetAgent(ctx context.Context, id string) (*domain.Agent, error) {
	var (
		a                   domain.Agent
		skills, repos, caps string
		tools               string
		created, upd        int64
	)
	err := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, machine_id, name, instructions, backend, role, capabilities_json, model, skills_json, repos_json, tools_json, session_mode, created_at_ms, updated_at_ms
		FROM agents WHERE id = ?`, id).
		Scan(&a.ID, &a.WorkspaceID, &a.MachineID, &a.Name, &a.Instructions, &a.Backend, &a.Role, &caps, &a.Model, &skills, &repos, &tools, &a.SessionMode, &created, &upd)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(skills), &a.Skills)
	_ = json.Unmarshal([]byte(repos), &a.Repos)
	_ = json.Unmarshal([]byte(tools), &a.Tools)
	_ = json.Unmarshal([]byte(caps), &a.Capabilities)
	a.CreatedAt = fromMS(created)
	a.UpdatedAt = fromMS(upd)
	if err := s.attachAgentWorkspaces(ctx, []*domain.Agent{&a}); err != nil {
		return nil, err
	}
	return &a, nil
}

// ListAgents returns agents in a workspace ordered by name.
func (s *Store) ListAgents(ctx context.Context, workspaceID string) ([]*domain.Agent, error) {
	q := `SELECT id, workspace_id, machine_id, name, instructions, backend, role, capabilities_json, model, skills_json, repos_json, tools_json, session_mode, created_at_ms, updated_at_ms FROM agents`
	args := []any{}
	if workspaceID != "" {
		q += ` WHERE workspace_id = ? OR EXISTS (SELECT 1 FROM agent_workspaces aw WHERE aw.agent_id = agents.id AND aw.workspace_id = ?)`
		args = append(args, workspaceID, workspaceID)
	}
	q += ` ORDER BY name`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Agent
	for rows.Next() {
		var (
			a                   domain.Agent
			skills, repos, caps string
			tools               string
			created, upd        int64
		)
		if err := rows.Scan(&a.ID, &a.WorkspaceID, &a.MachineID, &a.Name, &a.Instructions, &a.Backend, &a.Role, &caps, &a.Model, &skills, &repos, &tools, &a.SessionMode, &created, &upd); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(skills), &a.Skills)
		_ = json.Unmarshal([]byte(repos), &a.Repos)
		_ = json.Unmarshal([]byte(tools), &a.Tools)
		_ = json.Unmarshal([]byte(caps), &a.Capabilities)
		a.CreatedAt = fromMS(created)
		a.UpdatedAt = fromMS(upd)
		out = append(out, &a)
	}
	if err := s.attachAgentWorkspaces(ctx, out); err != nil {
		return nil, err
	}
	return out, rows.Err()
}

// SetAgentWorkspaces replaces the workspaces an agent can serve.
func (s *Store) SetAgentWorkspaces(ctx context.Context, agentID string, workspaceIDs []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM agent_workspaces WHERE agent_id = ?`, agentID); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, workspaceID := range workspaceIDs {
		if workspaceID == "" {
			continue
		}
		if _, ok := seen[workspaceID]; ok {
			continue
		}
		seen[workspaceID] = struct{}{}
		if _, err := tx.ExecContext(ctx, `INSERT INTO agent_workspaces (agent_id, workspace_id, created_at_ms)
			VALUES (?,?,?)`, agentID, workspaceID, ms(time.Now())); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) attachAgentWorkspaces(ctx context.Context, agents []*domain.Agent) error {
	if len(agents) == 0 {
		return nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT agent_id, workspace_id FROM agent_workspaces ORDER BY created_at_ms`)
	if err != nil {
		return err
	}
	defer rows.Close()
	byAgent := map[string][]string{}
	for rows.Next() {
		var agentID, workspaceID string
		if err := rows.Scan(&agentID, &workspaceID); err != nil {
			return err
		}
		byAgent[agentID] = append(byAgent[agentID], workspaceID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, agent := range agents {
		agent.WorkspaceIDs = byAgent[agent.ID]
		if len(agent.WorkspaceIDs) == 0 && agent.WorkspaceID != "" {
			agent.WorkspaceIDs = []string{agent.WorkspaceID}
		}
	}
	return nil
}

// DeleteAgent removes an agent and its workspace assignments.
func (s *Store) DeleteAgent(ctx context.Context, id string) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM agent_workspaces WHERE agent_id = ?`, id); err != nil {
		return false, err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM agents WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return n > 0, nil
}

// CreateApproval inserts a pending approval.
func (s *Store) CreateApproval(ctx context.Context, a *domain.Approval) error {
	if a.Status == "" {
		a.Status = domain.ApprovalPending
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO approvals
		(id, run_id, tool, command, risk, status, requested_at_ms, expires_at_ms, message_ref)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		a.ID, a.RunID, a.Tool, a.Command, a.Risk, a.Status, ms(a.RequestedAt), ms(a.ExpiresAt), a.MessageRef)
	return err
}

// SetApprovalMessageRef stores the notifier message reference for card updates.
func (s *Store) SetApprovalMessageRef(ctx context.Context, id, ref string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE approvals SET message_ref = ? WHERE id = ?`, ref, id)
	return err
}

// ListApprovals returns approvals, optionally filtered by status.
func (s *Store) ListApprovals(ctx context.Context, status, workspaceID string) ([]*domain.Approval, error) {
	q := `SELECT a.id, COALESCE(r.workspace_id, 'ws_default') AS workspace_id, a.run_id, a.tool, a.command, a.risk, a.status, a.requested_at_ms,
		a.expires_at_ms, a.decided_at_ms, a.decided_by, a.message_ref
		FROM approvals a LEFT JOIN runs r ON r.id = a.run_id`
	where := []string{}
	args := []any{}
	if status != "" {
		where = append(where, "a.status = ?")
		args = append(args, status)
	}
	if workspaceID != "" {
		where = append(where, "r.workspace_id = ?")
		args = append(args, workspaceID)
	}
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, " AND ")
	}
	q += ` ORDER BY a.requested_at_ms DESC LIMIT 200`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Approval
	for rows.Next() {
		var (
			a         domain.Approval
			requested int64
			expires   int64
			decided   sql.NullInt64
		)
		if err := rows.Scan(&a.ID, &a.WorkspaceID, &a.RunID, &a.Tool, &a.Command, &a.Risk, &a.Status, &requested, &expires, &decided, &a.DecidedBy, &a.MessageRef); err != nil {
			return nil, err
		}
		a.RequestedAt = fromMS(requested)
		a.ExpiresAt = fromMS(expires)
		a.DecidedAt = fromNull(decided)
		out = append(out, &a)
	}
	return out, rows.Err()
}

// GetApproval returns one approval or nil.
func (s *Store) GetApproval(ctx context.Context, id string) (*domain.Approval, error) {
	var (
		a         domain.Approval
		requested int64
		expires   int64
		decided   sql.NullInt64
	)
	err := s.db.QueryRowContext(ctx, `SELECT a.id, COALESCE(r.workspace_id, 'ws_default') AS workspace_id, a.run_id, a.tool, a.command, a.risk, a.status, a.requested_at_ms, a.expires_at_ms, a.decided_at_ms, a.decided_by, a.message_ref
		FROM approvals a LEFT JOIN runs r ON r.id = a.run_id WHERE a.id = ?`, id).
		Scan(&a.ID, &a.WorkspaceID, &a.RunID, &a.Tool, &a.Command, &a.Risk, &a.Status, &requested, &expires, &decided, &a.DecidedBy, &a.MessageRef)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.RequestedAt = fromMS(requested)
	a.ExpiresAt = fromMS(expires)
	a.DecidedAt = fromNull(decided)
	return &a, nil
}

// DecideApproval records a decision on a pending approval.
func (s *Store) DecideApproval(ctx context.Context, id, decision, by string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE approvals SET status = ?, decided_at_ms = ?, decided_by = ?
		WHERE id = ? AND status = ?`, decision, ms(time.Now()), by, id, domain.ApprovalPending)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ListExpiredApprovals returns pending approvals whose deadline has passed.
func (s *Store) ListExpiredApprovals(ctx context.Context, now time.Time) ([]*domain.Approval, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, run_id, tool, command, risk, status, requested_at_ms, expires_at_ms, decided_at_ms, decided_by, message_ref
		FROM approvals WHERE status = ? AND expires_at_ms <= ? ORDER BY expires_at_ms`,
		domain.ApprovalPending, ms(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Approval
	for rows.Next() {
		var (
			a         domain.Approval
			requested int64
			expires   int64
			decided   sql.NullInt64
		)
		if err := rows.Scan(&a.ID, &a.RunID, &a.Tool, &a.Command, &a.Risk, &a.Status, &requested, &expires, &decided, &a.DecidedBy, &a.MessageRef); err != nil {
			return nil, err
		}
		a.RequestedAt = fromMS(requested)
		a.ExpiresAt = fromMS(expires)
		a.DecidedAt = fromNull(decided)
		out = append(out, &a)
	}
	return out, rows.Err()
}

// ExpireApproval marks a pending approval as expired.
func (s *Store) ExpireApproval(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE approvals SET status = ?, decided_at_ms = ? WHERE id = ? AND status = ?`,
		domain.ApprovalExpired, ms(time.Now()), id, domain.ApprovalPending)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// EnqueueWriteback inserts a queued publication. Duplicate (run, action)
// entries are ignored so a retried finish cannot double-post.
func (s *Store) EnqueueWriteback(ctx context.Context, w *domain.Writeback) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO outbox
		(id, run_id, provider, connection_id, account_id, action, work_provider, work_instance, work_type, work_id, work_url, work_revision,
		 markdown, url, kind, attempts, next_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT DO NOTHING`,
		w.ID, w.RunID, w.Provider, w.ConnectionID, w.AccountID, w.Action,
		w.Work.Provider, w.Work.Instance, w.Work.Type, w.Work.ID, w.Work.URL, w.Work.Revision,
		w.Markdown, w.URL, w.Kind, w.Attempts, ms(w.NextAt))
	return err
}

// DueWritebacks returns queued publications whose next attempt is due.
func (s *Store) DueWritebacks(ctx context.Context, now time.Time, limit int) ([]*domain.Writeback, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, run_id, provider, connection_id, account_id, action,
		work_provider, work_instance, work_type, work_id, work_url, work_revision,
		markdown, url, kind, attempts, next_at_ms, done_at_ms, last_error
		FROM outbox WHERE done_at_ms IS NULL AND next_at_ms <= ?
		ORDER BY next_at_ms ASC LIMIT ?`, ms(now), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Writeback
	for rows.Next() {
		var (
			w      domain.Writeback
			nextAt int64
			doneAt sql.NullInt64
		)
		if err := rows.Scan(&w.ID, &w.RunID, &w.Provider, &w.ConnectionID, &w.AccountID, &w.Action,
			&w.Work.Provider, &w.Work.Instance, &w.Work.Type, &w.Work.ID, &w.Work.URL, &w.Work.Revision,
			&w.Markdown, &w.URL, &w.Kind, &w.Attempts, &nextAt, &doneAt, &w.LastError); err != nil {
			return nil, err
		}
		w.NextAt = fromMS(nextAt)
		w.DoneAt = fromNull(doneAt)
		out = append(out, &w)
	}
	return out, rows.Err()
}

// CompleteWriteback marks a publication as done.
func (s *Store) CompleteWriteback(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE outbox SET done_at_ms = ? WHERE id = ?`, ms(time.Now()), id)
	return err
}

// FailWriteback records a failed attempt. dead marks the row exhausted.
func (s *Store) FailWriteback(ctx context.Context, id string, attempts int, nextAt time.Time, lastErr string, dead bool) error {
	if dead {
		_, err := s.db.ExecContext(ctx, `UPDATE outbox SET attempts = ?, done_at_ms = ?, last_error = ? WHERE id = ?`,
			attempts, ms(time.Now()), lastErr, id)
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE outbox SET attempts = ?, next_at_ms = ?, last_error = ? WHERE id = ?`,
		attempts, ms(nextAt), lastErr, id)
	return err
}

// CredentialMeta describes a stored credential without exposing the secret.
type CredentialMeta struct {
	Provider  string     `json:"provider"`
	Subject   string     `json:"subject"`
	Kind      string     `json:"kind"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// PutCredential stores an encrypted credential, replacing any existing one.
func (s *Store) PutCredential(ctx context.Context, provider, subject, kind, ciphertext string, meta map[string]any, expires *time.Time) error {
	rawMeta, _ := json.Marshal(meta)
	now := ms(time.Now())
	var exp any
	if expires != nil {
		exp = ms(*expires)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO credentials
		(provider, subject, kind, ciphertext, meta, expires_at_ms, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?)
		ON CONFLICT(provider, subject) DO UPDATE SET
			kind = excluded.kind, ciphertext = excluded.ciphertext, meta = excluded.meta,
			expires_at_ms = excluded.expires_at_ms, updated_at_ms = excluded.updated_at_ms`,
		provider, subject, kind, ciphertext, string(rawMeta), exp, now, now)
	return err
}

// GetCredential returns the ciphertext and metadata, or ok=false.
func (s *Store) GetCredential(ctx context.Context, provider, subject string) (string, string, map[string]any, *time.Time, bool, error) {
	var (
		ciphertext, kind, meta string
		expires                sql.NullInt64
	)
	err := s.db.QueryRowContext(ctx, `SELECT ciphertext, kind, meta, expires_at_ms
		FROM credentials WHERE provider = ? AND subject = ?`, provider, subject).
		Scan(&ciphertext, &kind, &meta, &expires)
	if err == sql.ErrNoRows {
		return "", "", nil, nil, false, nil
	}
	if err != nil {
		return "", "", nil, nil, false, err
	}
	var metaMap map[string]any
	_ = json.Unmarshal([]byte(meta), &metaMap)
	return ciphertext, kind, metaMap, fromNull(expires), true, nil
}

// ListCredentialMeta returns all credential metadata, never secrets.
func (s *Store) ListCredentialMeta(ctx context.Context) ([]CredentialMeta, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT provider, subject, kind, expires_at_ms, updated_at_ms
		FROM credentials ORDER BY provider, subject`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CredentialMeta
	for rows.Next() {
		var (
			m       CredentialMeta
			exp     sql.NullInt64
			updated int64
		)
		if err := rows.Scan(&m.Provider, &m.Subject, &m.Kind, &exp, &updated); err != nil {
			return nil, err
		}
		m.ExpiresAt = fromNull(exp)
		m.UpdatedAt = fromMS(updated)
		out = append(out, m)
	}
	return out, rows.Err()
}

// DeleteCredential removes a credential.
func (s *Store) DeleteCredential(ctx context.Context, provider, subject string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM credentials WHERE provider = ? AND subject = ?`, provider, subject)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

const automationColumns = `id, workspace_id, name, schedule, provider, instance, trigger_kind, trigger_value,
	writeback_json, state_map, connection_id, priority, agent_id, repo, base, prompt, enabled, precheck,
	missed_grace_secs, session_id, reuse_session, last_status, last_error, next_run_ms, last_run_ms, created_at_ms`

// UpsertAutomation creates or replaces a scheduled or event automation.
func (s *Store) UpsertAutomation(ctx context.Context, a *domain.Automation) error {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	if a.WorkspaceID == "" {
		a.WorkspaceID = "ws_default"
	}
	enabled := 0
	if a.Enabled {
		enabled = 1
	}
	reuse := 0
	if a.ReuseSession {
		reuse = 1
	}
	writeback, _ := json.Marshal(a.Writeback)
	stateMap, _ := json.Marshal(a.StateMap)
	var next, last any
	if a.NextRun != nil {
		next = ms(*a.NextRun)
	}
	if a.LastRun != nil {
		last = ms(*a.LastRun)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO automations
		(id, workspace_id, name, schedule, provider, instance, trigger_kind, trigger_value,
		 writeback_json, state_map, connection_id, priority, agent_id, repo, base, prompt, enabled, precheck,
		 missed_grace_secs, session_id, reuse_session, last_status, last_error, next_run_ms, last_run_ms, created_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name, schedule = excluded.schedule,
			provider = excluded.provider, instance = excluded.instance,
			trigger_kind = excluded.trigger_kind, trigger_value = excluded.trigger_value,
			writeback_json = excluded.writeback_json, state_map = excluded.state_map,
			connection_id = excluded.connection_id, priority = excluded.priority,
			agent_id = excluded.agent_id, repo = excluded.repo, base = excluded.base, prompt = excluded.prompt,
			enabled = excluded.enabled, precheck = excluded.precheck, missed_grace_secs = excluded.missed_grace_secs,
			session_id = excluded.session_id, reuse_session = excluded.reuse_session,
			last_status = excluded.last_status, last_error = excluded.last_error,
			next_run_ms = excluded.next_run_ms, last_run_ms = excluded.last_run_ms`,
		a.ID, a.WorkspaceID, a.Name, a.Schedule, a.Provider, a.Instance, a.TriggerKind, a.TriggerValue,
		string(writeback), string(stateMap), a.ConnectionID, a.Priority,
		a.AgentID, a.Repo, a.Base, a.Prompt, enabled, a.Precheck,
		a.MissedGraceSeconds, a.SessionID, reuse, a.LastStatus, a.LastError, next, last, ms(a.CreatedAt))
	return err
}

// GetAutomation returns one automation or nil.
func (s *Store) GetAutomation(ctx context.Context, id string) (*domain.Automation, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+automationColumns+` FROM automations WHERE id = ?`, id)
	return scanAutomation(row)
}

// ListAutomations returns automations in a workspace.
func (s *Store) ListAutomations(ctx context.Context, workspaceID string) ([]*domain.Automation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+automationColumns+` FROM automations WHERE workspace_id = ? ORDER BY created_at_ms DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Automation
	for rows.Next() {
		a, err := scanAutomation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// DeleteAutomation removes an automation.
func (s *Store) DeleteAutomation(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM automations WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// DueAutomations returns enabled automations whose next run is due.
func (s *Store) DueAutomations(ctx context.Context, now time.Time) ([]*domain.Automation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+automationColumns+` FROM automations
		WHERE enabled = 1 AND next_run_ms IS NOT NULL AND next_run_ms <= ? ORDER BY next_run_ms`, ms(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Automation
	for rows.Next() {
		a, err := scanAutomation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// MarkAutomationRun records the last run, next scheduled time and outcome.
func (s *Store) MarkAutomationRun(ctx context.Context, id string, last, next time.Time, status, message string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE automations SET last_run_ms = ?, next_run_ms = ?, last_status = ?, last_error = ? WHERE id = ?`,
		ms(last), ms(next), status, message, id)
	return err
}

// SetAutomationSession stores a reusable session id.
func (s *Store) SetAutomationSession(ctx context.Context, id, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE automations SET session_id = ? WHERE id = ?`, sessionID, id)
	return err
}

func scanAutomation(sc interface{ Scan(...any) error }) (*domain.Automation, error) {
	var (
		a                   domain.Automation
		enabled             int
		reuse               int
		writeback, stateMap string
		next                sql.NullInt64
		last                sql.NullInt64
		created             int64
	)
	if err := sc.Scan(&a.ID, &a.WorkspaceID, &a.Name, &a.Schedule, &a.Provider, &a.Instance,
		&a.TriggerKind, &a.TriggerValue, &writeback, &stateMap, &a.ConnectionID, &a.Priority,
		&a.AgentID, &a.Repo, &a.Base, &a.Prompt, &enabled,
		&a.Precheck, &a.MissedGraceSeconds, &a.SessionID, &reuse, &a.LastStatus, &a.LastError, &next, &last, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	_ = json.Unmarshal([]byte(writeback), &a.Writeback)
	_ = json.Unmarshal([]byte(stateMap), &a.StateMap)
	a.Enabled = enabled == 1
	a.ReuseSession = reuse == 1
	a.NextRun = fromNull(next)
	a.LastRun = fromNull(last)
	a.CreatedAt = fromMS(created)
	return &a, nil
}

// CountRunsByStatus returns run counts grouped by status.
func (s *Store) CountRunsByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM runs GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		out[status] = n
	}
	return out, rows.Err()
}

// CountApprovals returns the number of approvals in a status.
func (s *Store) CountApprovals(ctx context.Context, status string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM approvals WHERE status = ?`, status).Scan(&n)
	return n, err
}

// CountOutboxPending returns queued publications.
func (s *Store) CountOutboxPending(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox WHERE done_at_ms IS NULL`).Scan(&n)
	return n, err
}

// CountOnlineMachines returns machines seen since the given time.
func (s *Store) CountOnlineMachines(ctx context.Context, since time.Time) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM machines WHERE status = 'online' AND last_seen_ms >= ?`, ms(since)).Scan(&n)
	return n, err
}
