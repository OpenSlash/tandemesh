-- Orca schema (SQLite dialect, also valid for Postgres with minor type changes).
-- Only execution facts are stored. There is deliberately no tasks, documents
-- or messages table: external systems remain the source of truth.

CREATE TABLE IF NOT EXISTS machines (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    labels_json   TEXT NOT NULL DEFAULT '[]',
    agents_json   TEXT NOT NULL DEFAULT '[]',
    concurrency   INTEGER NOT NULL DEFAULT 2,
    status        TEXT NOT NULL DEFAULT 'online',
    version       TEXT NOT NULL DEFAULT '',
    last_seen_ms  INTEGER NOT NULL DEFAULT 0,
    created_at_ms INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS runs (
    id                TEXT PRIMARY KEY,
    provider          TEXT NOT NULL,
    instance          TEXT NOT NULL,
    work_type         TEXT NOT NULL,
    work_id           TEXT NOT NULL,
    work_url          TEXT NOT NULL DEFAULT '',
    work_revision     TEXT NOT NULL DEFAULT '',
    snapshot_json     TEXT NOT NULL DEFAULT '{}',
    binding_id        TEXT NOT NULL DEFAULT '',
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
