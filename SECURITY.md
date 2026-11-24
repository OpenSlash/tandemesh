# Security policy

## Supported versions

Tandemesh is currently pre-1.0. Security fixes target the latest release and `main`.

## Reporting a vulnerability

Please do not open a public issue for a security problem.

Report privately through GitHub's **Security → Advisories → Report a vulnerability** flow for the
`OpenSlash/tandemesh` repository. Include:

- affected version or commit
- a description of the issue
- reproduction steps or proof of concept
- impact and any known mitigations

You will receive an acknowledgement as soon as possible. Please allow time for a fix before public
disclosure.

## Current security model

Tandemesh is designed for a trusted local machine or a trusted private network.

- The default plane listens on loopback only.
- The desktop launcher binds a random loopback port and generates a per-install token in
  `%APPDATA%/Tandemesh/local.token`.
- The desktop proxy injects the token; browser/API traffic is rejected without it.
- LAN mode is opt-in (`TANDEMESH_LAN=1`) and still requires the token.
- Agentd registration uses a join token.
- Workflow script nodes execute on the control-plane host and are not sandboxed.
- External ticket, document and message content is untrusted and must never be executed as
  instructions.
- Agent credentials are stored encrypted and are not returned by the API.

## Known limitations

- The project is pre-1.0 and not yet a hardened multi-tenant service.
- Workflow scripts run with the privileges of the plane process.
- The NSIS installer and auto-update flow are currently unsigned.
- The plane must not be exposed directly to the public internet.
- Postgres support exists, but SQLite is the default local deployment.

## Hardening guidance

- Keep the plane on loopback or a private network such as Tailscale.
- Use a strong, non-default join token.
- Configure webhook secrets for GitHub, Slack, Linear and Jira.
- Restrict who can create or edit workflow script nodes.
- Back up the database and token file together.
- Review third-party ACP adapters and Registry agents before installing them.

## Dependency and supply-chain notes

ACP agents are launched as external processes. The adapters
`claude-agent-acp`, `codex-acp`, `opencode acp` and similar are third-party components and should be
version-pinned in production. The ACP Registry is a read-only catalog; installing an agent does not
grant it access beyond the environment in which it is launched.
