<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="design/logo/kit/svg/tandemesh-lockup-horizontal-paper.svg">
    <img src="design/logo/kit/svg/tandemesh-lockup-horizontal-ink.svg" alt="Tandemesh" height="56">
  </picture>
</p>

<p align="center">
  <strong>Run coding agents on your own machines, driven by the tools your team already uses.</strong>
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-blue" alt="License: Apache-2.0"></a>
  <img src="https://img.shields.io/badge/go-1.27%2B-00ADD8" alt="Go 1.27+">
  <img src="https://img.shields.io/badge/status-early%20development-orange" alt="Status: early development">
</p>

---

Tandemesh is a **local-first agent execution plane**. It picks up work from GitHub, Jira, Linear
and other systems, hands it to coding agents such as Claude Code, Codex or Gemini CLI running on
your machines, and writes the results back as comments and draft pull requests.

Your issue tracker stays the source of truth. Tandemesh only records how work was executed: runs,
events, approvals and artifacts. Your code and credentials stay on the machines you control.

## Why Tandemesh

- **Local-first.** Agents run on your laptop, workstation or build box, with your toolchain and
  your repositories. No hosted sandbox required.
- **Bring your own agent.** Any [ACP](https://agentclientprotocol.com)-compatible agent works,
  including Claude Code, Codex, Gemini CLI, Cursor Agent, Qwen Code, OpenCode and DeepSeek.
- **Humans stay in control.** Risky tool calls wait for approval, every run is recorded, and
  Tandemesh opens draft PRs. It never merges.
- **Fits your workflow.** Trigger runs from issues, comments and webhooks; chain agents with a
  visual workflow editor; follow progress from desktop, web or phone.

## How it works

```text
 GitHub · Jira · Linear · Notion · Slack · ...
                     │  webhooks / polling
                     ▼
┌──────────────────────────────────────────┐      Desktop · Web · Mobile
│  tandemesh-plane   control plane         │◀──── HTTP + WebSocket
│  tasks · runs · approvals · workflows    │
└──────────────────────────────────────────┘
                     │  leases / events
                     ▼
┌──────────────────────────────────────────┐
│  tandemesh-agentd  on each machine       │
│  git worktrees · terminals · approvals   │
└──────────────────────────────────────────┘
                     │  ACP over stdio
                     ▼
     Claude Code · Codex · Gemini CLI · ...
```

The control plane schedules and records; it never executes agent code. Each machine runs a small
daemon that owns the workspace, the agent process, Git and the filesystem.

## Features

- **Sources and write-back:** GitHub, GitLab, Jira, Linear, Notion, Outline, Slack, Outlook,
  Feishu and Plane.
- **Agents:** built-in ACP backends, custom ACP commands, an ACP Registry browser and managed
  installers (binary, `npx`, `uvx`).
- **Workflows:** sequential and DAG steps, parallel branches, fan-out and merge, gates,
  conditions, retries, timeouts and child workflows.
- **Isolation:** one Git worktree per run, cleaned up automatically.
- **Approvals:** pause on sensitive tool calls and approve from any client.
- **Clients:**
  - **Web** control panel with a visual workflow editor.
  - **Desktop** app for Windows and macOS that runs everything locally, with tray and autostart.
  - **Mobile** app for Android, iOS and HarmonyOS to follow sessions, review diffs and approve on
    the go.
- **Storage:** SQLite by default, Postgres optional.

## Quick start

You need Go 1.27+, Node 20+ with pnpm, and at least one agent CLI with an ACP adapter (for example
Claude Code with `claude-agent-acp`).

```bash
git clone https://github.com/OpenSlash/tandemesh.git
cd tandemesh

# 1. control plane on :8080
go run ./cmd/tandemesh-plane

# 2. second terminal: register this machine as an execution host
go run ./cmd/tandemesh-agentd -name dev-1 -runner acp -workdir ~/tandemesh-work

# 3. third terminal: web UI
cd web && pnpm install && pnpm dev
```

Open <http://127.0.0.1:5173>, create a task and assign it to an agent.

Prefer a single app? The [desktop client](apps/tandemesh-desktop/README.md) starts the plane and
the daemon for you. Download it from [Releases](https://github.com/OpenSlash/tandemesh/releases/latest):
`Tandemesh.dmg` for macOS (Apple silicon and Intel) or `Tandemesh-Setup-amd64.exe` for Windows.
The builds are not notarized or code-signed yet, so the first launch needs one extra step:

- **macOS:** System Settings → Privacy & Security → **Open Anyway**, or
  `xattr -dr com.apple.quarantine /Applications/Tandemesh.app`.
- **Windows:** SmartScreen → **More info** → **Run anyway**.

After that the app updates itself.

## Configuration

Common settings (see [`.env.example`](.env.example) for all of them):

| Variable | Default | Purpose |
|---|---|---|
| `TANDEMESH_LISTEN` | `:8080` | HTTP listen address |
| `TANDEMESH_DB_PATH` | `tandemesh.db` | SQLite file or `postgres://` DSN |
| `TANDEMESH_JOIN_TOKEN` | `dev-join-token` | token machines use to register |
| `TANDEMESH_LOCAL_TOKEN` | empty | require a bearer token for browser and API traffic |
| `TANDEMESH_WORKSPACE` | `off` | `on` prepares a Git worktree per run |
| `TANDEMESH_DATA_DIR` | `~/.tandemesh` | repository cache and run workspaces |

> **Security:** run the plane on loopback or a trusted private network such as Tailscale, never
> directly on the internet, and change `TANDEMESH_JOIN_TOKEN` before adding other machines.
> See [SECURITY.md](SECURITY.md).

## Status

Tandemesh is in early development. The single-user, local workflow is usable today; team
deployment is in progress. See the [roadmap](ROADMAP.md).

By design, Tandemesh does **not** include accounts or seat management, a model gateway or billing,
or auto-merge.

## Contributing

Contributions are welcome. [CONTRIBUTING.md](CONTRIBUTING.md) covers setup, the repository layout,
builds, packaging and commit conventions.

## License

[Apache-2.0](LICENSE)
