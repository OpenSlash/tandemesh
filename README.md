# Tandemesh

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="design/logo/kit/svg/tandemesh-lockup-horizontal-paper.svg">
    <img src="design/logo/kit/svg/tandemesh-lockup-horizontal-ink.svg" alt="Tandemesh" height="48">
  </picture>
</p>

> **Internal codename: Orca.** The repository, Go module and public product name are Tandemesh.
> Orca remains only as an internal codename and as a compatibility fallback for legacy environment variables and headers.

**An agent execution plane.** Route work from Jira, Linear and GitHub to agents running on
many machines, and write the results back to the tools your team already uses.

> Tandemesh does not own tasks, documents or chat. External systems remain the source of truth;
> Tandemesh stores only execution facts: runs, events, machines, approvals and artifacts.

## Status (M0 skeleton)

See [ROADMAP.md](ROADMAP.md) for the full plan and current progress.

Working today:

- `tandemesh-plane`: SQLite or Postgres storage, run lifecycle, GitHub webhook verification/normalization/
  idempotency, machine registry, lease scheduling, SSE event stream, audit log
- `tandemesh-agentd`: registration, heartbeat, lease long-poll, mock execution, event reporting
- `tandemesh` CLI: `runs`, `machines`, `run`
- `web`: Vite + React run board

Not implemented yet (M1/M2): real ACP execution, WSS transport, approvals, write-back
outbox, Jira/Linear/Notion adapters, Postgres.

## Quick start

Requires Go 1.24+ (verified with 1.27.3). Node 20+ for the web UI.

```bash
# terminal 1 - control plane
go run ./cmd/tandemesh-plane

# terminal 2 - machine daemon (mock agent)
go run ./cmd/tandemesh-agentd -name dev-1 -agents claude,codex

# terminal 2 (alternative) - real ACP agent
go run ./cmd/tandemesh-agentd -name dev-1 -runner acp \
  -acp-command claude-code-acp -workdir /path/to/repo

# terminal 2 (recommended) - isolated git worktree per run
go run ./cmd/tandemesh-agentd -name dev-1 -workspace \
  -data-dir ~/.tandemesh -git git

# terminal 2 (push transport) - leases arrive over WebSocket
go run ./cmd/tandemesh-agentd -name dev-1 -transport ws

# terminal 3 - create and observe
go run ./cmd/tandemesh run --repo=acme/api --prompt="fix the flaky test"
go run ./cmd/tandemesh runs
```

Web board:

```bash
cd web
pnpm install
pnpm dev   # http://localhost:5173, /api proxied to :8080
```

### Console

React 19 + Vite + Tailwind v4 + shadcn-style primitives + TanStack Query, in a dark,
Linear-inspired layout.

- `/runs` — dense table with status filters, create-run dialog, live refresh
- `/runs/:id` — SSE timeline, run metadata, cancel
- `/approvals` — pending tool approvals with approve/deny

## Repository layout

```
cmd/
  tandemesh-plane/     control plane
  tandemesh-agentd/    machine daemon
  tandemesh/      CLI
internal/
  domain/         types and invariants
  config/         environment configuration
  store/          SQLite/Postgres persistence + schema
  api/            REST + SSE
  ingress/        webhook verification and normalization
  scheduler/      lease scheduling and expiry
api/openapi.yaml  API contract
migrations/       operational SQL (same source as internal/store/schema.sql)
web/              React control panel
deploy/           Dockerfile and compose
docs/             design docs (not published yet)
```

## Configuration

See [.env.example](.env.example). Core variables:

| Variable | Default | Purpose |
|---|---|---|
| `TANDEMESH_LISTEN` | `:8080` | HTTP listen address |
| `TANDEMESH_DB_PATH` | `tandemesh.db` | SQLite file or `postgres://` DSN |
| `TANDEMESH_JOIN_TOKEN` | `dev-join-token` | machine registration token (**change in production**) |
| `TANDEMESH_GITHUB_SECRET` | empty | GitHub webhook secret; empty disables verification (dev only) |
| `TANDEMESH_LEASE_TTL` | `30s` | lease duration |
| `TANDEMESH_MAX_ATTEMPTS` | `3` | max reschedules after lease loss |
| `TANDEMESH_WORKSPACE` | `off` | set to `on` to prepare a worktree per run |
| `TANDEMESH_DATA_DIR` | `~/.tandemesh` | repository cache and run workspaces |
| `TANDEMESH_TRANSPORT` | `http` | `ws` pushes leases over WebSocket; falls back to HTTP |

## Verification

```bash
go build ./...
go test ./...
go vet ./...
```

Verified locally:

- `go test ./...` passes (webhook signature and normalization tests)
- End to end: `tandemesh-plane` + `tandemesh-agentd` + `tandemesh run` moves a run from `queued` to
  `succeeded` in about two seconds
- `web`: `pnpm build` succeeds (about 225 KB JS / 2 KB CSS)

On hosts with Application Control, `go test` may be blocked from executing binaries in the
build cache. Compile the test binary into `bin/` and run it directly:

```bash
go test -c -o bin/agent.test.exe ./internal/agent
./bin/agent.test.exe -test.v
```

## Design decisions

- Go for the control plane, daemon and CLI: single binary, cross-platform process and PTY
  control, simple deployment.
- SQLite by default; set `TANDEMESH_DB_PATH=postgres://...` for team deployments.
- ACP for agent backends: one adapter covers Claude Code, Codex, OpenCode and others.
- SSE for the browser, outbound WSS for daemons (M1).
- Apache-2.0.

The full PRD, architecture and design specification are maintained locally in `docs/` and
will be published with the first public release.

## License

Apache-2.0. See [LICENSE](LICENSE).
