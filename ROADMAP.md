# Roadmap and progress

Status: **2026-10-30**

Legend: ✅ done · 🟡 partial · ⬜ planned · ⛔ blocked

## Snapshot

| Milestone | Theme | Done | Partial | Planned | Status |
|---|---|---|---|---|---|
| M0 | single-host loop | 12 | 2 | 0 | 🟡 core works, integration gaps |
| M1 | multi-machine and trust | 8 | 1 | 0 | 🟡 transport and trust landed |
| M2 | more sources and plugins | 11 | 0 | 0 | 🟡 |
| M3 | team deployment | 0 | 1 | 5 | ⬜ |

Progress is counted in tasks, not effort. A task is done when the acceptance criteria in
the design specification are met and covered by a test or a documented manual check.

## M0 — single-host loop

| # | Task | Status | Evidence | Remaining |
|---|---|---|---|---|
| 1 | domain types, schema, repository | ✅ | `internal/domain`, `internal/store` | — |
| 2 | REST API and OpenAPI contract | 🟡 | `internal/api`, `api/openapi.yaml` | auth, pagination |
| 3 | GitHub webhook (verify, normalize, dedupe) | ✅ | `internal/integration/github`, commit `156eb46` | — |
| 4 | policy engine and bindings | ✅ | `internal/policy`, comment + rule triggers | — |
| 5 | scheduler, leases, watchdog | ✅ | `internal/scheduler` | — |
| 6 | daemon transport | ✅ | `/ws` push, `internal/hub` | — |
| 7 | workspace and worktree per run | ✅ | `internal/workspace`, commit `41670b7` | retention policy |
| 8 | ACP client and one real backend | 🟡 | `internal/agent/acp.go`, commit `27b6e2a` | real CLI e2e |
| 9 | event stream, SSE, run board | ✅ | `internal/api`, `web/` | — |
| 10 | comment and link write-back | ✅ | `internal/writeback`, GitHub comment | attach/transition execution |
| 11 | Slack notifications and approvals | ✅ | `internal/integration/slack`, signed links | Feishu cards too |
| 12 | mock agent and e2e | ✅ | `internal/agent/mock.go` | — |
| 13 | compose, docs, quick start | ✅ | `deploy/`, `README.md` | — |
| 14 | task-first workspace | ✅ | inbox + sync, board, filters, teams, commands | — |
| 15 | organization, workspaces, groups | ✅ | multiple orgs, switcher, workspaces, members, auth | — |
| 16 | inbox states and my tasks | ✅ | inbox/later/cleared, read tracking, assignee view | — |
| 17 | icon rail and grouped sidebar | ✅ | rail, workspace switcher, unread badges | saved views |
| 18 | human task activity | ✅ | `task_events`, merged timeline | — |
| 19 | saved views | ✅ | `saved_views`, selector, rename/delete | — |
| 20 | iterations | ✅ | iterations, assignment, filter, counts, burndown | — |
| 21 | global search | ✅ | `/search`, ⌘K palette | — |
| 22 | task activity pagination | ✅ | limit/offset, load older | — |
| 23 | task comments | ✅ | `task_comments`, latest comment column | — |
| 25 | task relations | ✅ | parent/children, dependencies, cycle checks | — |
| 24 | group team launch | ✅ | `/tasks/{id}/team` with groupId | templates |
| 26 | workspace-scoped agents/bindings | ✅ | `workspace_id`, scoped lists, validation | — |
| 27 | workspace-scoped automations | ✅ | `workspace_id`, scoped lists, agent validation | — |
| 28 | skills | ✅ | workspace skills, CRUD, prompt injection | file sync |
| 29 | English/Chinese i18n | ✅ | i18n provider, shell/pages/settings translated | — |
| 30 | light/dark theme | ✅ | theme provider, Settings appearance | — |
| 31 | machine runtimes | ✅ | daemon CLI detection, machines page, agent availability warning | install flow |
| 32 | agent-machine binding | ✅ | agent.machine_id, scheduler affinity, UI selector | tags |
| 33 | runtime install guide | ✅ | `/runtimes`, install commands, ACP detection | one-click install |
| 34 | DeepSeek-backed ACP | ✅ | launcher, env mapping, real ACP run | usage/cost |
| 35 | Codex + DeepSeek | ✅ | codex CLI 0.166.0, custom provider verified | ACP adapter |
| 36 | run token usage | ✅ | transcript input tokens, estimated output, UI | true output/cost |
| 37 | machine reconnect | ✅ | stable machine identity by name | — |
| 38 | machine takeover | ✅ | reassign agents, requeue runs, UI | cross-org |
| 39 | end-to-end flow | ✅ | pull → dispatch → ACP → writeback verified | provider matrix |
| 40 | Plane integration | ✅ | cloud/self-hosted API, pull, comments, webhooks | — |
| 42 | Plane state mapping | ✅ | state picker, per-connection webhook secret | — |
| 41 | web connection wizard | ✅ | named connections, sync/delete, webhook URL | — |
| 43 | GitHub OAuth | ✅ | code flow, signed state, binding | app credentials |
| 44 | connection records | ✅ | status, per-connection credentials, reconnect, health loop | — |
| 45 | real Plane e2e | ✅ | AIVIDEO-76 run + comment writeback | labels/assignees |

## M1 — multi-machine and trust

| # | Task | Status | Evidence | Remaining |
|---|---|---|---|---|
| 1 | runner seam | ✅ | `internal/agent/runner.go`, commit `7897f3f` | — |
| 2 | ACP runner over stdio | ✅ | `internal/agent/acp.go` | terminal requests |
| 3 | per-run worktree | ✅ | `internal/workspace` | cleanup policy |
| 4 | WSS hub with resume | ✅ | `internal/hub`, `internal/api/ws.go` | — |
| 5 | fence enforcement on write paths | ✅ | `store.CheckFence`, event/finish endpoints | — |
| 6 | approvals (tool policy, signed callbacks) | ✅ | `internal/agent/acp.go`, expiry in scheduler | escalation |
| 7 | write-back outbox with conflict handling | ✅ | `internal/writeback`, attach/transition | revision check |
| 8 | audit trail and metrics | ✅ | `audit_log`, `/metrics` | OTel traces |
| 9 | encrypted credential vault | ✅ | `internal/credentials`, AES-256-GCM | OAuth refresh |

## M2 — more sources and plugins

| # | Task | Status |
|---|---|---|
| 1 | Jira task source (OAuth, webhooks, transitions) | ✅ | `internal/integration/jira` |
| 2 | Linear task source | ✅ | `internal/integration/linear`, webhook + comments |
| 3 | Notion context source (read-only, citations) | ✅ | `internal/integration/notion`, `/context/*` |
| 4 | Skills from git repositories | ✅ | `internal/skills`, prompt injection |
| 5 | MCP configuration injection | ✅ | `internal/agent` session/new, scheduler |
| 6 | gRPC plugin protocol | ✅ | `internal/plugin`, `pkg/plugin` |
| 7 | Slack chat + approval cards | ✅ | `internal/integration/slack`, signed links |
| 8 | Postgres adapter | ✅ | `internal/store/pgdriver`, `TANDEMESH_DB_PATH=postgres://` |
| 9 | Feishu notifier + approval cards | ✅ | `internal/integration/feishu` |
| 10 | Cron automations | ✅ | `internal/automation`, `/automations` |
| 11 | Outline wiki source + email kind | ✅ | `internal/integration/outline` |

## M3 — team deployment

| # | Task | Status |
|---|---|---|
| 1 | OIDC/SSO | ✅ | `/auth/*`, session cookie, dev header switch | — |
| 2 | multi-tenancy | ⬜ |
| 3 | run variants and comparison | 🟡 | `internal/api/variants.go`, run detail | side-by-side diff |
| 4 | hosted edition | ⬜ |
| 5 | cost reporting and budgets | ⬜ |
| 6 | Wails v3 desktop shell (service + shell scaffold) | 🟡 |

## Next two weeks

| Week | Work | Outcome |
|---|---|---|
| 1 | WSS hub with lease resume; fence checks on event and finish endpoints (done) | daemons survive restarts; stale writes rejected |
| 1 | bindings and agents API + console (done) | configure sources without editing code |
| 2 | approvals in Slack (done) | `git push` can require a human |
| 2 | write-back outbox + GitHub comments | run results land back on the issue |

## Constraints and risks

| Item | Impact | Mitigation |
|---|---|---|
| Application Control blocks rebuilt daemon binaries on the dev host | local daemon e2e must use `go run` or a signed binary | CI on Linux; document the workaround |
| No real ACP agent installed here | ACP runner covered by helper-process tests only | run `claude-code-acp` in CI or a dev container |
| SQLite single writer | contention above ~100 concurrent runs | batch event writes; Postgres at M2 |
| Connector maintenance | long-term cost | three official adapters, plugin SDK |

## How to update this file

- Change a status in the same pull request that changes the behaviour.
- Link the commit or file that proves the status.
- Do not mark a task done without a test or a documented manual check.
- Keep the snapshot table consistent with the per-milestone tables.
