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
- optional: Wails 3, NSIS and a supported ACP agent CLI for desktop/agent testing

Install the repository-local Git hook once:

```powershell
pwsh scripts/setup-git-hooks.ps1
```

This sets `core.hooksPath=.githooks` and enables Conventional Commit validation.

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

Desktop (Windows):

```powershell
cd apps/tandemesh-desktop
powershell -ExecutionPolicy Bypass -File scripts\build.ps1
```

Desktop packaging requires Wails 3 and NSIS:

```powershell
wails3 package
```

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
