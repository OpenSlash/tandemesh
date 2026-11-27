# Contributing to Tandemesh

Thanks for taking the time to contribute. Tandemesh is a local-first agent execution plane; changes
should preserve the boundary between the control plane and the machines that execute agents.

## Before you start

Read [AGENTS.md](AGENTS.md). It defines the repository boundaries, coding conventions and review
expectations.

The short version:

- External systems remain the source of truth for tasks, documents and conversations.
- The control plane stores execution facts only.
- The control plane never executes agent code.
- No auto-merge, billing, seat management or cloud-only core features.
- All project documentation is written in English.

## Development setup

Requirements:

- Go 1.27+
- Node 20+
- pnpm
- optional: a supported ACP agent CLI for real execution
- optional: Wails 3 (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`) for desktop;
  NSIS on Windows, Xcode on macOS

Install the repository-local Git hook once:

```bash
git config core.hooksPath .githooks    # or: pwsh scripts/setup-git-hooks.ps1
```

This enables Conventional Commit validation.

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
internal/ingress          webhook verification and normalization
internal/integration      source and write-back adapters
pkg/plugin                gRPC runner plugin SDK
web/                      React control panel
apps/tandemesh-desktop    Wails 3 desktop shell
apps/mobile               React Native app (Android, iOS, HarmonyOS)
api/openapi.yaml          API contract
migrations/               SQL migrations
deploy/                   Dockerfile, compose and nginx config
```

## Build and test

From the repository root:

```bash
go build ./...
go test ./...
go vet ./...
```

Web:

```bash
cd web
pnpm install
pnpm exec tsc --noEmit
pnpm run build
```

Desktop from source (looks for the sidecars in `../../bin` and the UI in `../../web/dist`):

```bash
make build
(cd web && pnpm run build)
cd apps/tandemesh-desktop && go run .
```

Set `TANDEMESH_WEB_URL=http://127.0.0.1:5173` to load the UI from the Vite dev server, or
`TANDEMESH_MANAGE_PLANE=0` to attach to a plane you started yourself. Desktop data lives in
`%APPDATA%\Tandemesh`, `~/Library/Application Support/Tandemesh` or `~/.config/Tandemesh`.

Desktop packaging:

```bash
cd apps/tandemesh-desktop

# Windows: per-user NSIS installer (cross-compiles from macOS or Linux with makensis)
wails3 task windows:package

# macOS: ad-hoc .app, universal .app, .dmg
wails3 task darwin:package
wails3 task darwin:package:universal
wails3 task darwin:create:dmg

# Optional Apple signing and notarization
wails3 task darwin:sign SIGN_IDENTITY="Developer ID Application: ..."
wails3 task darwin:release SIGN_IDENTITY="Developer ID Application: ..." NOTARY_PROFILE=<profile>
```

Releases are built by `.github/workflows/release-desktop.yml`. Push a tag to publish one:

```bash
git tag v0.2.0 && git push origin v0.2.0        # release
git tag v0.2.0-rc.1 && git push origin v0.2.0-rc.1  # prerelease, not offered as an update
```

The workflow builds the universal macOS app and the Windows installer, signs the update manifests
with the `TANDEMESH_UPDATE_KEY` secret and publishes everything to one release. Installed apps poll
`releases/latest/download/update-<os>-<arch>.json` and only accept manifests signed with that key;
generate a new pair with `go run ./tools/sign-update -genkey` (and update `updatePublicKey`) only
if the key is lost, since existing installs will then need a manual reinstall.

All checks must pass before opening a pull request. If a check cannot run in your environment,
explain why in the pull request.

## Commit messages

All commit messages are in English and follow Conventional Commits:

```text
<type>(<scope>): <imperative summary>
```

Allowed types:

```text
feat fix refactor perf test docs build ci chore revert
```

Rules:

- Subject line: 72 characters or less.
- Imperative mood: `add`, not `added`.
- One logical change per commit.
- Do not mix refactors with behaviour changes.
- Body explains why, not what.
- Do not commit generated Wails assets, binaries, databases, `web/dist` or `node_modules`.

The local hook rejects invalid commit messages. Run `git diff --stat` and `git add -p` before
committing.

## Pull requests

- One issue or logical change per pull request.
- Keep the diff reviewable.
- Describe what changed, why, how it was verified, and known limits.
- Include test commands and relevant output for behaviour changes.
- Update `api/openapi.yaml` when routes or payloads change.
- Update `README.md` or `docs/` when user-facing behaviour changes.

## Code style

### Go

- Standard library first; add dependencies only when they remove meaningful complexity.
- Wrap errors with context: `fmt.Errorf("assign run: %w", err)`.
- Keep `context.Context` as the first parameter for I/O functions.
- Keep SQL in `internal/store`.
- Table-driven tests for parsing and validation.
- Comments explain why, not what.

### TypeScript and React

- Strict TypeScript; avoid `any`.
- Functional components and hooks.
- Keep data fetching out of presentational components.
- User-facing copy belongs in the i18n dictionary; add both `en` and `zh` entries.

### API and schema

- `api/openapi.yaml` is the contract.
- Keep JSON fields `camelCase` and SQL columns `snake_case`.
- Schema changes belong in `internal/store/schema.sql` and `migrations/`.
- Never edit an applied migration.

## Security

- Never commit secrets, tokens or `.env` files.
- Webhook handlers must verify signatures when a secret is configured.
- External content is untrusted; never execute it as instructions.
- Credentials are stored encrypted and are never returned by the API.
- Tests must not make outbound network calls.

Report vulnerabilities through the process in [SECURITY.md](SECURITY.md), not a public issue.

## License

By contributing, you agree that your contributions are licensed under Apache-2.0.
