# Tandemesh

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="design/logo/kit/svg/tandemesh-lockup-horizontal-paper.svg">
    <img src="design/logo/kit/svg/tandemesh-lockup-horizontal-ink.svg" alt="Tandemesh" height="48">
  </picture>
</p>

> **Internal codename: Orca.** The repository, Go module and public product name are Tandemesh.
> Orca remains only as an internal codename and as a compatibility fallback for legacy environment
> variables and headers.

Tandemesh is a **local-first agent execution plane**. It routes tasks from external systems to
coding agents running on local machines, records execution facts, and writes results back to the
tools a team already uses.

External systems remain the source of truth for tasks, documents and conversations. Tandemesh stores
execution state: runs, events, machines, agents, approvals, sessions and artifacts.

## Status

Tandemesh is under active development. The local single-user workflow is usable; the project is not
yet a hosted multi-tenant service.

Working today:

- `tandemesh-plane`: control plane with SQLite (Postgres optional), tasks, runs, agents, machines,
  sessions, approvals, workflows, artifacts, audit and usage APIs.
- `tandemesh-agentd`: machine daemon that registers with the plane, leases runs, executes ACP agents,
  streams events, handles approvals and reports usage.
- ACP execution: Claude Code, Codex, DeepSeek Harness, Gemini CLI, Cursor Agent, Qwen Code and
  OpenCode, plus custom ACP commands.
- ACP Registry browser and managed binary / `npx` / `uvx` installers.
- Workflow engine: linear steps and DAG `needs`, parallel branches, fanout, merge (including merge
  agent), gates, conditions, retries, timeouts, child workflows and `onFail`.
- Web control panel: React 19, Vite, TanStack Query, React Flow workflow editor.
- Desktop: Wails 3 shell, local plane/agentd lifecycle, tray, close-to-tray, autostart, local token,
  LAN mode, NSIS installer and installer-based update checks.
- Mobile: React Native is planned; the local plane already exposes the HTTP API it will use.

Not included in the self-hosted core:

- accounts, organizations or seat management
- model gateway or billing
- auto-merge

## Architecture

```text
Desktop / Web / Mobile
        |
        | HTTP + WebSocket
        v
  tandemesh-plane  (control plane, SQLite/Postgres)
        |
        | lease / events
        v
  tandemesh-agentd (execution host)
        |
        | ACP over stdio
        v
  Claude Code / Codex / DeepSeek / Gemini / ...
```

The control plane never executes agent code. The daemon owns the local workspace, agent process,
terminal, Git and filesystem.

## Quick start

Requirements:

- Go 1.27+
- Node 20+ and pnpm for the web UI
- at least one supported agent CLI for real execution

Start the control plane:

```bash
go run ./cmd/tandemesh-plane
```

Start a local execution host:

```bash
go run ./cmd/tandemesh-agentd \
  -name dev-1 \
  -runner acp \
  -claude-acp-command claude-agent-acp \
  -codex-acp-command codex-acp \
  -workdir /path/to/workspace
```

Start the web UI:

```bash
cd web
pnpm install
pnpm dev
```

Open `http://127.0.0.1:5173`.

### Desktop

Build the web bundle first, then the Wails app:

```powershell
cd web
pnpm run build

cd ..\apps\tandemesh-desktop
powershell -ExecutionPolicy Bypass -File scripts\build.ps1
```

For a development shell against the Vite server:

```powershell
$env:TANDEMESH_WEB_URL = "http://127.0.0.1:5173"
go run .
```

The desktop app can start `tandemesh-plane` and `tandemesh-agentd` itself. It writes data to
`%APPDATA%/Tandemesh` on Windows. Set `TANDEMESH_MANAGE_PLANE=0` to attach to an already-running plane.

### Windows installer

With Wails 3 and NSIS installed:

```powershell
cd apps\tandemesh-desktop
wails3 package
```

The installer includes the desktop shell, plane and agentd. The build is currently unsigned.

## Configuration

See `.env.example` for the full list. Common variables:

| Variable | Default | Purpose |
|---|---|---|
| `TANDEMESH_LISTEN` | `:8080` | HTTP listen address |
| `TANDEMESH_DB_PATH` | `tandemesh.db` | SQLite file or `postgres://` DSN |
| `TANDEMESH_AUTH` | `none` | local auth mode |
| `TANDEMESH_LOCAL_TOKEN` | empty | require a bearer token for browser/API traffic |
| `TANDEMESH_JOIN_TOKEN` | `dev-join-token` | machine registration token |
| `TANDEMESH_WORKSPACE` | `off` | prepare a git worktree per run |
| `TANDEMESH_DATA_DIR` | `~/.tandemesh` | repository cache and run workspaces |
| `TANDEMESH_TRANSPORT` | `http` | `ws` pushes leases over WebSocket |
| `TANDEMESH_LAN` | `0` | desktop: listen on `0.0.0.0` instead of loopback |
| `TANDEMESH_UPDATE_URL` | empty | desktop update manifest URL |

### Local security

The default desktop mode binds to `127.0.0.1`, generates a per-install token and injects it through
the local proxy. LAN mode is opt-in:

```powershell
$env:TANDEMESH_LAN = "1"
```

Remote browsers must open the desktop-provided URL with `?token=...` once; the plane sets an
HttpOnly cookie for subsequent API and WebSocket traffic.

Do not expose the plane directly to the public internet. Use a trusted private network such as
Tailscale, or a future relay.

## Data

Default desktop data directory on Windows:

```text
%APPDATA%/Tandemesh/
  tandemesh.db
  tandemesh.db-wal
  tandemesh.db-shm
  local.token
  work/
```

SQLite is the default. Back it up by stopping the plane and copying the database together with its
`-wal` and `-shm` files, or use SQLite's online backup API.

## Testing

```bash
go build ./...
go test ./...
go vet ./...
```

For web changes:

```bash
cd web
pnpm run build
pnpm exec tsc --noEmit
```

## Repository layout

```text
cmd/tandemesh-plane       control plane
cmd/tandemesh-agentd      execution host
cmd/tandemesh             CLI
internal/domain           types and invariants
internal/agent            ACP runner and ACP registry
internal/store            SQLite/Postgres persistence
internal/api              REST, SSE and WebSocket handlers
internal/scheduler        lease scheduling
internal/workspace        repository cache and worktrees
web/                      React control panel
apps/tandemesh-desktop    Wails 3 desktop shell
api/openapi.yaml          API contract
docs/                     architecture and design documents
```

## Documentation

- [AGENTS.md](AGENTS.md) — repository rules for contributors and AI agents
- [ROADMAP.md](ROADMAP.md) — planned milestones
- [docs/](docs/) — architecture, workflow, desktop, mobile and integration design
- [docs/workflows.md](docs/workflows.md) — workflow engine reference

All project documentation is written in English.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Commits follow Conventional Commits and the repository
provides a local `commit-msg` hook.

## Security

See [SECURITY.md](SECURITY.md) for reporting vulnerabilities and the current local security model.

## License

Apache-2.0. See [LICENSE](LICENSE).
