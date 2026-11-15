-- Orca schema (SQLite dialect, also valid for Postgres with minor type changes).
-- Execution facts (runs, events, approvals) plus a task projection. The tasks
-- table is a cache of external work items, never the source of truth. Documents
-- and messages are not stored.

CREATE TABLE IF NOT EXISTS machines (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    labels_json   TEXT NOT NULL DEFAULT '[]',
    agents_json   TEXT NOT NULL DEFAULT '[]',
    runtimes_json TEXT NOT NULL DEFAULT '{}',
    concurrency   INTEGER NOT NULL DEFAULT 2,
    status        TEXT NOT NULL DEFAULT 'online',
    version       TEXT NOT NULL DEFAULT '',
    last_seen_ms  INTEGER NOT NULL DEFAULT 0,
    metrics_json  TEXT NOT NULL DEFAULT '{}',
    created_at_ms INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS machine_join_tokens (
    id            TEXT PRIMARY KEY,
    token_hash    TEXT NOT NULL UNIQUE,
    expires_at_ms INTEGER NOT NULL,
    used_at_ms    INTEGER,
    created_by    TEXT NOT NULL DEFAULT '',
    created_at_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS runs (
    id                TEXT PRIMARY KEY,
    workspace_id      TEXT NOT NULL DEFAULT 'ws_default',
    provider          TEXT NOT NULL,
    instance          TEXT NOT NULL,
    work_type         TEXT NOT NULL,
    work_id           TEXT NOT NULL,
    work_url          TEXT NOT NULL DEFAULT '',
    work_revision     TEXT NOT NULL DEFAULT '',
    snapshot_json     TEXT NOT NULL DEFAULT '{}',
    automation_id     TEXT NOT NULL DEFAULT '',
    agent_id          TEXT NOT NULL DEFAULT '',
    machine_id        TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL,
    priority          INTEGER NOT NULL DEFAULT 0,
    attempt           INTEGER NOT NULL DEFAULT 0,
    fence             INTEGER NOT NULL DEFAULT 0,
    repo              TEXT NOT NULL DEFAULT '',
    base              TEXT NOT NULL DEFAULT '',
    branch            TEXT NOT NULL DEFAULT '',
    prompt            TEXT NOT NULL DEFAULT '',
    task_id           TEXT NOT NULL DEFAULT '',
    role              TEXT NOT NULL DEFAULT '',
    session_id        TEXT NOT NULL DEFAULT '',
    input_tokens      INTEGER NOT NULL DEFAULT 0,
    output_tokens     INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    cache_write_tokens INTEGER NOT NULL DEFAULT 0,
    cost_usd          REAL NOT NULL DEFAULT 0,
    usage_estimated   INTEGER NOT NULL DEFAULT 0,
    model             TEXT NOT NULL DEFAULT '',
    variant_of        TEXT NOT NULL DEFAULT '',
    variant_label     TEXT NOT NULL DEFAULT '',
    error_code        TEXT NOT NULL DEFAULT '',
    error_message     TEXT NOT NULL DEFAULT '',
    created_at_ms     INTEGER NOT NULL,
    started_at_ms     INTEGER,
    ended_at_ms       INTEGER,
    lease_expires_ms  INTEGER
);

CREATE INDEX IF NOT EXISTS runs_status_idx ON runs(status, priority DESC, created_at_ms);
CREATE INDEX IF NOT EXISTS runs_work_idx ON runs(provider, instance, work_type, work_id);
CREATE INDEX IF NOT EXISTS runs_machine_idx ON runs(machine_id, status);

CREATE TABLE IF NOT EXISTS tasks (
    id            TEXT PRIMARY KEY,
    provider      TEXT NOT NULL,
    instance      TEXT NOT NULL,
    work_type     TEXT NOT NULL,
    work_id       TEXT NOT NULL,
    url           TEXT NOT NULL DEFAULT '',
    revision      TEXT NOT NULL DEFAULT '',
    title         TEXT NOT NULL DEFAULT '',
    body          TEXT NOT NULL DEFAULT '',
    labels        TEXT NOT NULL DEFAULT '[]',
    author        TEXT NOT NULL DEFAULT '',
    assignee      TEXT NOT NULL DEFAULT '',
    assignee_agent_id TEXT NOT NULL DEFAULT '',
    assignee_group_id TEXT NOT NULL DEFAULT '',
    assignee_member_id TEXT NOT NULL DEFAULT '',
    origin        TEXT NOT NULL DEFAULT 'external',
    status        TEXT NOT NULL DEFAULT 'open',
    platform_state TEXT NOT NULL DEFAULT '',
    platform_state_group TEXT NOT NULL DEFAULT '',
    execution_state TEXT NOT NULL DEFAULT 'ready',
    comments_synced_at_ms INTEGER NOT NULL DEFAULT 0,
    workspace_id  TEXT NOT NULL DEFAULT 'ws_default',
    inbox_state   TEXT NOT NULL DEFAULT 'inbox',
    inbox_read_at_ms INTEGER,
    iteration_id  TEXT NOT NULL DEFAULT '',
    done_at_ms    INTEGER,
    auto_dispatched_at_ms INTEGER,
    parent_id     TEXT NOT NULL DEFAULT '',
    created_at_ms INTEGER NOT NULL,
    updated_at_ms INTEGER NOT NULL,
    synced_at_ms  INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS tasks_ref ON tasks(provider, instance, work_type, work_id);
CREATE INDEX IF NOT EXISTS tasks_status_idx ON tasks(status, updated_at_ms);

CREATE TABLE IF NOT EXISTS task_inbox (
    task_id       TEXT NOT NULL,
    user_id       TEXT NOT NULL,
    state         TEXT NOT NULL DEFAULT 'inbox',
    read_at_ms    INTEGER,
    updated_at_ms INTEGER NOT NULL,
    PRIMARY KEY (task_id, user_id)
);

CREATE TABLE IF NOT EXISTS skills (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    body          TEXT NOT NULL DEFAULT '',
    version       TEXT NOT NULL DEFAULT '',
    source        TEXT NOT NULL DEFAULT 'manual',
    source_url    TEXT NOT NULL DEFAULT '',
    tags_json     TEXT NOT NULL DEFAULT '[]',
    enabled       INTEGER NOT NULL DEFAULT 1,
    checksum      TEXT NOT NULL DEFAULT '',
    created_at_ms INTEGER NOT NULL,
    updated_at_ms INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS skills_workspace_name_idx ON skills(workspace_id, name);

CREATE TABLE IF NOT EXISTS task_dependencies (
    task_id       TEXT NOT NULL,
    depends_on    TEXT NOT NULL,
    PRIMARY KEY (task_id, depends_on)
);

CREATE TABLE IF NOT EXISTS task_comments (
    id            TEXT PRIMARY KEY,
    task_id       TEXT NOT NULL,
    author        TEXT NOT NULL DEFAULT '',
    body          TEXT NOT NULL,
    created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS task_comments_idx ON task_comments(task_id, created_at_ms);

CREATE TABLE IF NOT EXISTS task_notes (
    id            TEXT PRIMARY KEY,
    task_id       TEXT NOT NULL,
    author        TEXT NOT NULL DEFAULT '',
    body          TEXT NOT NULL,
    created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS task_notes_idx ON task_notes(task_id, created_at_ms);

CREATE TABLE IF NOT EXISTS workspaces (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    slug          TEXT NOT NULL,
    created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS workspaces_slug_idx ON workspaces(slug);

CREATE TABLE IF NOT EXISTS groups (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    created_at_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id      TEXT NOT NULL,
    agent_id      TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (group_id, agent_id)
);

INSERT INTO workspaces (id, name, slug, created_at_ms) VALUES ('ws_default', 'Default', 'default', 0) ON CONFLICT DO NOTHING;
INSERT INTO groups (id, workspace_id, name, description, created_at_ms) VALUES ('grp_core', 'ws_default', 'Core team', '', 0) ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS iterations (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL,
    name          TEXT NOT NULL,
    start_date    TEXT NOT NULL DEFAULT '',
    end_date      TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'planned',
    created_at_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS saved_views (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL,
    name          TEXT NOT NULL,
    filters       TEXT NOT NULL DEFAULT '{}',
    created_at_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS task_events (
    id            TEXT PRIMARY KEY,
    task_id       TEXT NOT NULL,
    actor         TEXT NOT NULL DEFAULT '',
    type          TEXT NOT NULL,
    payload       TEXT NOT NULL DEFAULT '{}',
    created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS task_events_idx ON task_events(task_id, created_at_ms);

CREATE TABLE IF NOT EXISTS run_events (
    run_id     TEXT NOT NULL,
    seq        INTEGER NOT NULL,
    ts_ms      INTEGER NOT NULL,
    type       TEXT NOT NULL,
    payload    TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (run_id, seq)
);

CREATE TABLE IF NOT EXISTS idempotency (
    key        TEXT PRIMARY KEY,
    run_id     TEXT NOT NULL DEFAULT '',
    created_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_log (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    ts_ms     INTEGER NOT NULL,
    actor     TEXT NOT NULL DEFAULT '',
    action    TEXT NOT NULL,
    target    TEXT NOT NULL DEFAULT '',
    ip        TEXT NOT NULL DEFAULT '',
    meta      TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS agents (
    id              TEXT PRIMARY KEY,
    workspace_id    TEXT NOT NULL DEFAULT 'ws_default',
    machine_id      TEXT NOT NULL DEFAULT '',
    name            TEXT NOT NULL,
    instructions    TEXT NOT NULL DEFAULT '',
    backend         TEXT NOT NULL DEFAULT 'claude',
    role            TEXT NOT NULL DEFAULT '',
    capabilities_json TEXT NOT NULL DEFAULT '[]',
    model           TEXT NOT NULL DEFAULT '',
    skills_json     TEXT NOT NULL DEFAULT '[]',
    repos_json      TEXT NOT NULL DEFAULT '[]',
    tools_json      TEXT NOT NULL DEFAULT '{}',
    created_at_ms   INTEGER NOT NULL,
    updated_at_ms   INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS agent_workspaces (
    agent_id      TEXT NOT NULL,
    workspace_id  TEXT NOT NULL,
    created_at_ms INTEGER NOT NULL,
    PRIMARY KEY (agent_id, workspace_id)
);

INSERT INTO agent_workspaces (agent_id, workspace_id, created_at_ms)
    SELECT id, workspace_id, created_at_ms FROM agents WHERE workspace_id != ''
    ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS connections (
    id                     TEXT PRIMARY KEY,
    workspace_id           TEXT NOT NULL DEFAULT 'ws_default',
    provider               TEXT NOT NULL,
    name                   TEXT NOT NULL DEFAULT '',
    status                 TEXT NOT NULL DEFAULT 'pending',
    auth_method            TEXT NOT NULL DEFAULT 'access_token',
    sync_direction         TEXT NOT NULL DEFAULT 'read_write',
    instance               TEXT NOT NULL DEFAULT '',
    base_url               TEXT NOT NULL DEFAULT '',
    account_login          TEXT NOT NULL DEFAULT '',
    execution_mode         TEXT NOT NULL DEFAULT 'cloud',
    runner_machine_id      TEXT NOT NULL DEFAULT '',
    token_expires_at_ms    INTEGER,
    last_callback_at_ms    INTEGER,
    automation_id          TEXT NOT NULL DEFAULT '',
    enabled                INTEGER NOT NULL DEFAULT 1,
    created_at_ms          INTEGER NOT NULL,
    updated_at_ms          INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS connections_scope_idx
    ON connections(workspace_id, provider, base_url, instance);

CREATE TABLE IF NOT EXISTS assignment_rules (
    id             TEXT PRIMARY KEY,
    workspace_id   TEXT NOT NULL DEFAULT 'ws_default',
    name           TEXT NOT NULL DEFAULT '',
    provider       TEXT NOT NULL DEFAULT '',
    instance       TEXT NOT NULL DEFAULT '',
    label          TEXT NOT NULL DEFAULT '',
    author         TEXT NOT NULL DEFAULT '',
    assignee       TEXT NOT NULL DEFAULT '',
    agent_id       TEXT NOT NULL DEFAULT '',
    group_id       TEXT NOT NULL DEFAULT '',
    repo           TEXT NOT NULL DEFAULT '',
    auto_dispatch  INTEGER NOT NULL DEFAULT 0,
    policy         TEXT NOT NULL DEFAULT 'manual',
    enabled        INTEGER NOT NULL DEFAULT 1,
    priority       INTEGER NOT NULL DEFAULT 0,
    created_at_ms  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS assignment_rules_workspace_idx ON assignment_rules(workspace_id, enabled, priority);

CREATE INDEX IF NOT EXISTS connections_workspace_idx ON connections(workspace_id, provider);

CREATE TABLE IF NOT EXISTS resource_bindings (
    id             TEXT PRIMARY KEY,
    workspace_id   TEXT NOT NULL DEFAULT 'ws_default',
    connection_id  TEXT NOT NULL,
    provider       TEXT NOT NULL,
    external_id    TEXT NOT NULL,
    name           TEXT NOT NULL DEFAULT '',
    kind           TEXT NOT NULL DEFAULT 'resource',
    direction      TEXT NOT NULL DEFAULT 'read_write',
    account_id     TEXT NOT NULL DEFAULT '',
    settings_json  TEXT NOT NULL DEFAULT '{}',
    enabled        INTEGER NOT NULL DEFAULT 1,
    last_synced_at_ms INTEGER,
    last_error     TEXT NOT NULL DEFAULT '',
    created_at_ms  INTEGER NOT NULL,
    updated_at_ms  INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS resource_bindings_connection_external_idx
    ON resource_bindings(connection_id, external_id);
CREATE INDEX IF NOT EXISTS resource_bindings_workspace_idx
    ON resource_bindings(workspace_id, provider);

CREATE TABLE IF NOT EXISTS external_identities (
    id             TEXT PRIMARY KEY,
    workspace_id   TEXT NOT NULL DEFAULT 'ws_default',
    provider       TEXT NOT NULL,
    external_id    TEXT NOT NULL,
    login          TEXT NOT NULL DEFAULT '',
    email          TEXT NOT NULL DEFAULT '',
    member_id      TEXT NOT NULL,
    created_at_ms  INTEGER NOT NULL,
    updated_at_ms  INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS external_identities_key_idx
    ON external_identities(workspace_id, provider, external_id);
CREATE INDEX IF NOT EXISTS external_identities_member_idx
    ON external_identities(workspace_id, member_id);

CREATE TABLE IF NOT EXISTS connection_accounts (
    id                 TEXT PRIMARY KEY,
    workspace_id       TEXT NOT NULL DEFAULT 'ws_default',
    connection_id      TEXT NOT NULL,
    provider           TEXT NOT NULL,
    kind               TEXT NOT NULL DEFAULT 'user',
    member_id          TEXT NOT NULL DEFAULT '',
    login              TEXT NOT NULL DEFAULT '',
    email              TEXT NOT NULL DEFAULT '',
    external_id        TEXT NOT NULL DEFAULT '',
    credential_subject TEXT NOT NULL,
    is_default         INTEGER NOT NULL DEFAULT 0,
    created_at_ms      INTEGER NOT NULL,
    updated_at_ms      INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS connection_accounts_connection_idx
    ON connection_accounts(connection_id, is_default);

CREATE TABLE IF NOT EXISTS approvals (
    id              TEXT PRIMARY KEY,
    run_id          TEXT NOT NULL,
    tool            TEXT NOT NULL,
    command         TEXT NOT NULL DEFAULT '',
    risk            TEXT NOT NULL DEFAULT 'medium',
    status          TEXT NOT NULL DEFAULT 'pending',
    requested_at_ms INTEGER NOT NULL,
    expires_at_ms   INTEGER NOT NULL,
    decided_at_ms   INTEGER,
    decided_by      TEXT NOT NULL DEFAULT '',
    message_ref     TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS approvals_status_idx ON approvals(status, requested_at_ms);

CREATE TABLE IF NOT EXISTS outbox (
    id             TEXT PRIMARY KEY,
    run_id         TEXT NOT NULL,
    provider       TEXT NOT NULL,
    action         TEXT NOT NULL,
    connection_id  TEXT NOT NULL DEFAULT '',
    account_id     TEXT NOT NULL DEFAULT '',
    work_provider  TEXT NOT NULL DEFAULT '',
    work_instance  TEXT NOT NULL DEFAULT '',
    work_type      TEXT NOT NULL DEFAULT '',
    work_id        TEXT NOT NULL DEFAULT '',
    work_url       TEXT NOT NULL DEFAULT '',
    work_revision  TEXT NOT NULL DEFAULT '',
    markdown       TEXT NOT NULL DEFAULT '',
    url            TEXT NOT NULL DEFAULT '',
    kind           TEXT NOT NULL DEFAULT '',
    attempts       INTEGER NOT NULL DEFAULT 0,
    next_at_ms     INTEGER NOT NULL,
    done_at_ms     INTEGER,
    last_error     TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS outbox_once ON outbox(run_id, action);
CREATE INDEX IF NOT EXISTS outbox_due_idx ON outbox(done_at_ms, next_at_ms);

CREATE TABLE IF NOT EXISTS oauth_apps (
    provider      TEXT PRIMARY KEY,
    client_id     TEXT NOT NULL,
    updated_at_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS credentials (
    provider      TEXT NOT NULL,
    subject       TEXT NOT NULL,
    kind          TEXT NOT NULL DEFAULT 'token',
    ciphertext    TEXT NOT NULL,
    meta          TEXT NOT NULL DEFAULT '{}',
    expires_at_ms INTEGER,
    created_at_ms INTEGER NOT NULL,
    updated_at_ms INTEGER NOT NULL,
    PRIMARY KEY (provider, subject)
);

CREATE TABLE IF NOT EXISTS automations (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL DEFAULT 'ws_default',
    name          TEXT NOT NULL,
    schedule      TEXT NOT NULL DEFAULT '',
    provider      TEXT NOT NULL DEFAULT '',
    instance      TEXT NOT NULL DEFAULT '',
    trigger_kind  TEXT NOT NULL DEFAULT '',
    trigger_value TEXT NOT NULL DEFAULT '',
    writeback_json TEXT NOT NULL DEFAULT '{}',
    state_map     TEXT NOT NULL DEFAULT '{}',
    connection_id TEXT NOT NULL DEFAULT '',
    priority      INTEGER NOT NULL DEFAULT 0,
    agent_id      TEXT NOT NULL,
    repo          TEXT NOT NULL DEFAULT '',
    base          TEXT NOT NULL DEFAULT 'main',
    prompt        TEXT NOT NULL DEFAULT '',
    enabled       INTEGER NOT NULL DEFAULT 1,
    precheck      TEXT NOT NULL DEFAULT '',
    missed_grace_secs INTEGER NOT NULL DEFAULT 0,
    session_id    TEXT NOT NULL DEFAULT '',
    reuse_session INTEGER NOT NULL DEFAULT 0,
    last_status   TEXT NOT NULL DEFAULT '',
    last_error    TEXT NOT NULL DEFAULT '',
    next_run_ms   INTEGER,
    last_run_ms   INTEGER,
    created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS automations_due_idx ON automations(enabled, next_run_ms);


CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id             TEXT PRIMARY KEY,
    provider       TEXT NOT NULL,
    automation_id  TEXT NOT NULL,
    delivery       TEXT NOT NULL DEFAULT '',
    kind           TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'received',
    reason         TEXT NOT NULL DEFAULT '',
    run_id         TEXT NOT NULL DEFAULT '',
    body           TEXT NOT NULL DEFAULT '',
    attempts       INTEGER NOT NULL DEFAULT 0,
    retry_after_ms INTEGER NOT NULL DEFAULT 0,
    created_at_ms  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS webhook_deliveries_idx ON webhook_deliveries(automation_id, created_at_ms);

CREATE TABLE IF NOT EXISTS sync_cursors (
    connection_id     TEXT PRIMARY KEY,
    cursor_ms         INTEGER NOT NULL DEFAULT 0,
    last_synced_at_ms INTEGER NOT NULL DEFAULT 0,
    last_full_ms      INTEGER NOT NULL DEFAULT 0,
    last_error        TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS binding_sync_cursors (
    connection_id     TEXT NOT NULL,
    binding_id        TEXT NOT NULL,
    cursor_ms         INTEGER NOT NULL DEFAULT 0,
    last_synced_at_ms INTEGER NOT NULL DEFAULT 0,
    last_full_ms      INTEGER NOT NULL DEFAULT 0,
    last_error        TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (connection_id, binding_id)
);
CREATE TABLE IF NOT EXISTS external_comments (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT '',
    external_id TEXT NOT NULL DEFAULT '',
    author TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL,
    thread_id TEXT NOT NULL DEFAULT '',
    parent_external_id TEXT NOT NULL DEFAULT '',
    author_external_id TEXT NOT NULL DEFAULT '',
    author_member_id TEXT NOT NULL DEFAULT '',
    direction TEXT NOT NULL DEFAULT 'inbound',
    created_at_ms INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS external_comments_idx ON external_comments(task_id, created_at_ms);

CREATE TABLE IF NOT EXISTS webhook_debounce (
    key         TEXT PRIMARY KEY,
    seen_at_ms  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sync_locks (
    name          TEXT PRIMARY KEY,
    owner         TEXT NOT NULL,
    expires_at_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS notifications (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL DEFAULT '',
    type          TEXT NOT NULL DEFAULT '',
    title         TEXT NOT NULL DEFAULT '',
    body          TEXT NOT NULL DEFAULT '',
    link          TEXT NOT NULL DEFAULT '',
    read_at_ms    INTEGER NOT NULL DEFAULT 0,
    created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS notifications_workspace_idx ON notifications(workspace_id, read_at_ms, created_at_ms);

CREATE TABLE IF NOT EXISTS documents (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL DEFAULT '',
    title         TEXT NOT NULL DEFAULT '',
    body          TEXT NOT NULL DEFAULT '',
    tags          TEXT NOT NULL DEFAULT '[]',
    created_by    TEXT NOT NULL DEFAULT '',
    created_at_ms INTEGER NOT NULL DEFAULT 0,
    updated_at_ms INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS documents_workspace_idx ON documents(workspace_id, updated_at_ms);

CREATE TABLE IF NOT EXISTS audit_logs (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL DEFAULT '',
    actor         TEXT NOT NULL DEFAULT '',
    action        TEXT NOT NULL DEFAULT '',
    resource      TEXT NOT NULL DEFAULT '',
    detail        TEXT NOT NULL DEFAULT '',
    created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS audit_logs_workspace_idx ON audit_logs(workspace_id, created_at_ms);

CREATE TABLE IF NOT EXISTS dispatches (
    id             TEXT PRIMARY KEY,
    task_id        TEXT NOT NULL,
    workspace_id   TEXT NOT NULL DEFAULT '',
    run_id         TEXT NOT NULL DEFAULT '',
    agent_id       TEXT NOT NULL DEFAULT '',
    attempt        INTEGER NOT NULL DEFAULT 1,
    status         TEXT NOT NULL DEFAULT 'pending',
    outcome        TEXT NOT NULL DEFAULT '',
    result         TEXT NOT NULL DEFAULT '',
    files_modified TEXT NOT NULL DEFAULT '[]',
    failure_count  INTEGER NOT NULL DEFAULT 0,
    created_at_ms  INTEGER NOT NULL,
    updated_at_ms  INTEGER NOT NULL,
    ended_at_ms    INTEGER
);

CREATE INDEX IF NOT EXISTS dispatches_task_idx ON dispatches(task_id, created_at_ms);
CREATE INDEX IF NOT EXISTS dispatches_run_idx ON dispatches(run_id);

CREATE TABLE IF NOT EXISTS agent_messages (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL DEFAULT '',
    task_id       TEXT NOT NULL DEFAULT '',
    run_id        TEXT NOT NULL DEFAULT '',
    dispatch_id   TEXT NOT NULL DEFAULT '',
    sender        TEXT NOT NULL DEFAULT '',
    recipient     TEXT NOT NULL DEFAULT '',
    type          TEXT NOT NULL DEFAULT 'status',
    subject       TEXT NOT NULL DEFAULT '',
    body          TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'unread',
    created_at_ms INTEGER NOT NULL,
    handled_at_ms INTEGER
);

CREATE INDEX IF NOT EXISTS agent_messages_run_idx ON agent_messages(run_id, status, created_at_ms);

CREATE TABLE IF NOT EXISTS decision_gates (
    id            TEXT PRIMARY KEY,
    task_id       TEXT NOT NULL,
    dispatch_id   TEXT NOT NULL DEFAULT '',
    question      TEXT NOT NULL DEFAULT '',
    options       TEXT NOT NULL DEFAULT '[]',
    resolution    TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'open',
    created_at_ms INTEGER NOT NULL,
    resolved_at_ms INTEGER
);

CREATE INDEX IF NOT EXISTS decision_gates_task_idx ON decision_gates(task_id, status);

CREATE TABLE IF NOT EXISTS diff_annotations (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL DEFAULT '',
    task_id       TEXT NOT NULL DEFAULT '',
    run_id        TEXT NOT NULL DEFAULT '',
    file_path     TEXT NOT NULL DEFAULT '',
    line          INTEGER NOT NULL DEFAULT 0,
    side          TEXT NOT NULL DEFAULT 'new',
    body          TEXT NOT NULL DEFAULT '',
    author        TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'open',
    batch_id      TEXT NOT NULL DEFAULT '',
    created_at_ms INTEGER NOT NULL,
    sent_at_ms    INTEGER,
    resolved_at_ms INTEGER
);

CREATE INDEX IF NOT EXISTS diff_annotations_task_idx ON diff_annotations(task_id, status, created_at_ms);

CREATE TABLE IF NOT EXISTS worktrees (
    id                TEXT PRIMARY KEY,
    workspace_id      TEXT NOT NULL DEFAULT '',
    task_id           TEXT NOT NULL DEFAULT '',
    run_id            TEXT NOT NULL DEFAULT '',
    repo              TEXT NOT NULL DEFAULT '',
    path              TEXT NOT NULL DEFAULT '',
    branch            TEXT NOT NULL DEFAULT '',
    base_ref          TEXT NOT NULL DEFAULT '',
    start_ref         TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'creating',
    checkpoint_comment TEXT NOT NULL DEFAULT '',
    workspace_status  TEXT NOT NULL DEFAULT '',
    created_at_ms     INTEGER NOT NULL,
    updated_at_ms     INTEGER NOT NULL,
    archived_at_ms    INTEGER
);

CREATE UNIQUE INDEX IF NOT EXISTS worktrees_run_idx ON worktrees(run_id);
CREATE INDEX IF NOT EXISTS worktrees_task_idx ON worktrees(task_id, created_at_ms);

CREATE TABLE IF NOT EXISTS agent_sessions (
    id                  TEXT PRIMARY KEY,
    workspace_id        TEXT NOT NULL DEFAULT '',
    task_id             TEXT NOT NULL DEFAULT '',
    agent_id            TEXT NOT NULL DEFAULT '',
    backend             TEXT NOT NULL DEFAULT '',
    model               TEXT NOT NULL DEFAULT '',
    reasoning           TEXT NOT NULL DEFAULT '',
    mode                TEXT NOT NULL DEFAULT 'chat',
    title               TEXT NOT NULL DEFAULT '',
    repo                TEXT NOT NULL DEFAULT '',
    base                TEXT NOT NULL DEFAULT 'main',
    worktree_id         TEXT NOT NULL DEFAULT '',
    status              TEXT NOT NULL DEFAULT 'active',
    backend_session_id  TEXT NOT NULL DEFAULT '',
    last_run_id         TEXT NOT NULL DEFAULT '',
    created_by          TEXT NOT NULL DEFAULT '',
    created_at_ms       INTEGER NOT NULL,
    updated_at_ms       INTEGER NOT NULL,
    seed_context        TEXT NOT NULL DEFAULT '',
    closed_at_ms        INTEGER
);

CREATE INDEX IF NOT EXISTS agent_sessions_task_idx ON agent_sessions(task_id, updated_at_ms);

CREATE TABLE IF NOT EXISTS session_messages (
    id            TEXT PRIMARY KEY,
    session_id    TEXT NOT NULL,
    run_id        TEXT NOT NULL DEFAULT '',
    role          TEXT NOT NULL,
    kind          TEXT NOT NULL DEFAULT 'text',
    text          TEXT NOT NULL DEFAULT '',
    created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS session_messages_idx ON session_messages(session_id, created_at_ms, id);

CREATE TABLE IF NOT EXISTS session_comments (
    id            TEXT PRIMARY KEY,
    session_id    TEXT NOT NULL,
    file          TEXT NOT NULL DEFAULT '',
    line          INTEGER NOT NULL DEFAULT 0,
    side          TEXT NOT NULL DEFAULT 'new',
    body          TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'open',
    created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS session_comments_idx ON session_comments(session_id, status, created_at_ms);

CREATE TABLE IF NOT EXISTS terminal_sessions (
    id               TEXT PRIMARY KEY,
    workspace_id     TEXT NOT NULL DEFAULT 'ws_default',
    agent_session_id TEXT NOT NULL DEFAULT '',
    machine_id       TEXT NOT NULL DEFAULT '',
    shell            TEXT NOT NULL DEFAULT '',
    cwd              TEXT NOT NULL DEFAULT '',
    cols             INTEGER NOT NULL DEFAULT 120,
    rows             INTEGER NOT NULL DEFAULT 30,
    status           TEXT NOT NULL DEFAULT 'pending',
    exit_code        INTEGER,
    created_at_ms    INTEGER NOT NULL,
    last_seen_ms     INTEGER
);

CREATE INDEX IF NOT EXISTS terminal_sessions_pending_idx ON terminal_sessions(machine_id, status, created_at_ms);

CREATE TABLE IF NOT EXISTS terminal_events (
    id            TEXT PRIMARY KEY,
    terminal_id   TEXT NOT NULL,
    seq           INTEGER NOT NULL,
    data          TEXT NOT NULL DEFAULT '',
    created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS terminal_events_idx ON terminal_events(terminal_id, seq);

CREATE TABLE IF NOT EXISTS terminal_inputs (
    id            TEXT PRIMARY KEY,
    terminal_id   TEXT NOT NULL,
    seq           INTEGER NOT NULL,
    data          TEXT NOT NULL DEFAULT '',
    consumed      INTEGER NOT NULL DEFAULT 0,
    created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS terminal_inputs_idx ON terminal_inputs(terminal_id, consumed, seq);
CREATE TABLE IF NOT EXISTS worktree_cleanups (
    id              TEXT PRIMARY KEY,
    worktree_id     TEXT NOT NULL,
    run_id          TEXT NOT NULL DEFAULT '',
    path            TEXT NOT NULL DEFAULT '',
    repo            TEXT NOT NULL DEFAULT '',
    machine_id      TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending',
    attempts        INTEGER NOT NULL DEFAULT 0,
    error           TEXT NOT NULL DEFAULT '',
    created_at_ms   INTEGER NOT NULL,
    completed_at_ms INTEGER
);

CREATE INDEX IF NOT EXISTS worktree_cleanups_idx ON worktree_cleanups(status, machine_id, created_at_ms);
CREATE UNIQUE INDEX IF NOT EXISTS worktree_cleanups_pending_idx ON worktree_cleanups(worktree_id) WHERE status = 'pending';
CREATE TABLE IF NOT EXISTS git_jobs (
    id              TEXT PRIMARY KEY,
    session_id      TEXT NOT NULL DEFAULT '',
    worktree_id     TEXT NOT NULL DEFAULT '',
    machine_id      TEXT NOT NULL DEFAULT '',
    path            TEXT NOT NULL DEFAULT '',
    action          TEXT NOT NULL DEFAULT 'ship',
    message         TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending',
    result_json     TEXT NOT NULL DEFAULT '',
    error           TEXT NOT NULL DEFAULT '',
    created_at_ms   INTEGER NOT NULL,
    updated_at_ms   INTEGER NOT NULL,
    completed_at_ms INTEGER
);

CREATE INDEX IF NOT EXISTS git_jobs_pending_idx ON git_jobs(status, machine_id, created_at_ms);
CREATE UNIQUE INDEX IF NOT EXISTS git_jobs_session_pending_idx ON git_jobs(session_id) WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS agent_questions (
    id             TEXT PRIMARY KEY,
    session_id     TEXT NOT NULL,
    run_id         TEXT NOT NULL DEFAULT '',
    question       TEXT NOT NULL,
    options_json   TEXT NOT NULL DEFAULT '[]',
    status         TEXT NOT NULL DEFAULT 'pending',
    answer         TEXT NOT NULL DEFAULT '',
    created_at_ms  INTEGER NOT NULL,
    answered_at_ms INTEGER,
    expires_at_ms  INTEGER
);

CREATE INDEX IF NOT EXISTS agent_questions_pending_idx ON agent_questions(session_id, status, created_at_ms);

CREATE TABLE IF NOT EXISTS history_scans (
    id              TEXT PRIMARY KEY,
    machine_id      TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    error           TEXT NOT NULL DEFAULT '',
    requested_at_ms INTEGER NOT NULL,
    completed_at_ms INTEGER
);

CREATE INDEX IF NOT EXISTS history_scans_pending_idx ON history_scans(status, machine_id, requested_at_ms);

CREATE TABLE IF NOT EXISTS machine_history (
    machine_id      TEXT PRIMARY KEY,
    items_json      TEXT NOT NULL DEFAULT '[]',
    scanned_at_ms   INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS history_cache (
    machine_id      TEXT PRIMARY KEY,
    items_json      TEXT NOT NULL DEFAULT '[]',
    scanned_at_ms   INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS session_shares (
    token           TEXT PRIMARY KEY,
    session_id      TEXT NOT NULL,
    created_by      TEXT NOT NULL DEFAULT '',
    created_at_ms   INTEGER NOT NULL,
    expires_at_ms   INTEGER
);

CREATE INDEX IF NOT EXISTS session_shares_session_idx ON session_shares(session_id, created_at_ms);

CREATE TABLE IF NOT EXISTS task_pull_requests (
    id             TEXT PRIMARY KEY,
    task_id        TEXT NOT NULL,
    workspace_id   TEXT NOT NULL DEFAULT 'ws_default',
    provider       TEXT NOT NULL,
    repo           TEXT NOT NULL,
    number         INTEGER NOT NULL,
    url            TEXT NOT NULL DEFAULT '',
    title          TEXT NOT NULL DEFAULT '',
    state          TEXT NOT NULL DEFAULT '',
    branch         TEXT NOT NULL DEFAULT '',
    author         TEXT NOT NULL DEFAULT '',
    backlinked_at_ms INTEGER,
    created_at_ms  INTEGER NOT NULL,
    updated_at_ms  INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS task_pull_requests_key_idx
    ON task_pull_requests(task_id, provider, repo, number);
CREATE INDEX IF NOT EXISTS task_pull_requests_task_idx
    ON task_pull_requests(task_id);

CREATE TABLE IF NOT EXISTS member_identities (
    id                 TEXT PRIMARY KEY,
    workspace_id       TEXT NOT NULL DEFAULT 'ws_default',
    member_id          TEXT NOT NULL,
    provider           TEXT NOT NULL,
    external_id        TEXT NOT NULL,
    login              TEXT NOT NULL DEFAULT '',
    email              TEXT NOT NULL DEFAULT '',
    credential_subject TEXT NOT NULL DEFAULT '',
    created_at_ms      INTEGER NOT NULL,
    updated_at_ms      INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS member_identities_key_idx
    ON member_identities(workspace_id, provider, external_id);
CREATE INDEX IF NOT EXISTS member_identities_member_idx
    ON member_identities(workspace_id, member_id);

CREATE TABLE IF NOT EXISTS comment_threads (
    id             TEXT PRIMARY KEY,
    workspace_id   TEXT NOT NULL DEFAULT 'ws_default',
    task_id        TEXT NOT NULL,
    provider       TEXT NOT NULL,
    external_id    TEXT NOT NULL,
    source         TEXT NOT NULL DEFAULT '',
    resolved       INTEGER NOT NULL DEFAULT 0,
    created_at_ms  INTEGER NOT NULL,
    updated_at_ms  INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS comment_threads_key_idx
    ON comment_threads(task_id, provider, external_id);
CREATE INDEX IF NOT EXISTS comment_threads_task_idx
    ON comment_threads(task_id);

CREATE TABLE IF NOT EXISTS task_commits (
    id             TEXT PRIMARY KEY,
    task_id        TEXT NOT NULL,
    workspace_id   TEXT NOT NULL DEFAULT 'ws_default',
    provider       TEXT NOT NULL,
    repo           TEXT NOT NULL,
    sha            TEXT NOT NULL,
    url            TEXT NOT NULL DEFAULT '',
    message        TEXT NOT NULL DEFAULT '',
    author         TEXT NOT NULL DEFAULT '',
    created_at_ms  INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS task_commits_key_idx
    ON task_commits(task_id, provider, repo, sha);
CREATE INDEX IF NOT EXISTS task_commits_task_idx
    ON task_commits(task_id);

CREATE TABLE IF NOT EXISTS task_attachments (
    id             TEXT PRIMARY KEY,
    task_id        TEXT NOT NULL,
    provider       TEXT NOT NULL,
    external_id    TEXT NOT NULL,
    name           TEXT NOT NULL DEFAULT '',
    url            TEXT NOT NULL DEFAULT '',
    mime           TEXT NOT NULL DEFAULT '',
    size           INTEGER NOT NULL DEFAULT 0,
    author         TEXT NOT NULL DEFAULT '',
    source         TEXT NOT NULL DEFAULT 'issue',
    created_at_ms  INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS task_attachments_key_idx
    ON task_attachments(task_id, provider, external_id);
CREATE INDEX IF NOT EXISTS task_attachments_task_idx
    ON task_attachments(task_id);

CREATE TABLE IF NOT EXISTS import_batches (
    id             TEXT PRIMARY KEY,
    workspace_id   TEXT NOT NULL,
    provider       TEXT NOT NULL,
    connection_id  TEXT NOT NULL,
    query          TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'running',
    imported       INTEGER NOT NULL DEFAULT 0,
    next_start_at  INTEGER NOT NULL DEFAULT 0,
    limit_count    INTEGER NOT NULL DEFAULT 100,
    failed         INTEGER NOT NULL DEFAULT 0,
    error          TEXT NOT NULL DEFAULT '',
    created_at_ms  INTEGER NOT NULL,
    finished_at_ms INTEGER
);

CREATE TABLE IF NOT EXISTS import_items (
    batch_id   TEXT NOT NULL,
    task_id    TEXT NOT NULL,
    work_id    TEXT NOT NULL DEFAULT '',
    PRIMARY KEY(batch_id, task_id)
);
CREATE INDEX IF NOT EXISTS import_items_task_idx ON import_items(task_id);

CREATE TABLE IF NOT EXISTS task_fields (
    task_id   TEXT NOT NULL,
    provider  TEXT NOT NULL DEFAULT '',
    key       TEXT NOT NULL,
    value     TEXT NOT NULL DEFAULT '',
    PRIMARY KEY(task_id, key, value)
);
CREATE INDEX IF NOT EXISTS task_fields_task_idx ON task_fields(task_id);

CREATE TABLE IF NOT EXISTS oauth_nonces (
    nonce          TEXT PRIMARY KEY,
    expires_at_ms  INTEGER NOT NULL,
    used           INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS comment_deliveries (
    id             TEXT PRIMARY KEY,
    workspace_id   TEXT NOT NULL DEFAULT 'ws_default',
    task_id        TEXT NOT NULL,
    comment_id     TEXT NOT NULL,
    provider       TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'pending',
    external_id    TEXT NOT NULL DEFAULT '',
    error          TEXT NOT NULL DEFAULT '',
    attempts       INTEGER NOT NULL DEFAULT 0,
    updated_at_ms  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS comment_deliveries_task_idx ON comment_deliveries(task_id);
