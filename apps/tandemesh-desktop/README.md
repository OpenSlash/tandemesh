# Tandemesh Desktop

A Wails 3 app for Windows and macOS that runs Tandemesh entirely on your machine. It reuses the web
console from `../../web`; there is no separate desktop UI.

## What it does

1. Starts `tandemesh-plane` on a free loopback port and `tandemesh-agentd` as the local execution
   host.
2. Generates a per-install access token and passes it to both.
3. Serves the web console and proxies `/api`, `/auth` and `/ws` to the plane, so the UI stays
   same-origin.
4. Keeps running in the tray when the window is closed; optional start at login.
5. Allows one instance at a time and stops the plane and daemon when it quits.

## Run from source

From the repository root:

```bash
make build                    # bin/tandemesh-plane, bin/tandemesh-agentd
(cd web && pnpm install && pnpm run build)
cd apps/tandemesh-desktop && go run .
```

Packaging (Windows installer, signed and notarized macOS `.dmg`) is described in
[CONTRIBUTING.md](../../CONTRIBUTING.md#build-and-test).

## Environment

| Variable | Purpose |
|---|---|
| `TANDEMESH_MANAGE_PLANE=0` | do not start a plane; attach to `TANDEMESH_PLANE_URL` |
| `TANDEMESH_PLANE_URL` | URL of an already-running plane |
| `TANDEMESH_PLANE_BIN`, `TANDEMESH_AGENTD_BIN` | override the sidecar paths |
| `TANDEMESH_WEB_URL` | load the UI from a dev server instead of the bundled build |
| `TANDEMESH_WEB_DIST` | serve the UI from this directory |
| `TANDEMESH_LAN=1` | listen on all interfaces for other devices on a trusted network |
| `TANDEMESH_UPDATE_URL` | update manifest URL instead of the latest GitHub release |

Release builds check the latest GitHub release at launch and every six hours; the tray menu shows
the version and has **Check for updates**. Manifests must carry a valid signature from the release
key.

With LAN mode on, open `http://<this-host>:<port>/?token=...` once from the other device; the
plane then sets an HttpOnly cookie. The exact URL is printed to the log.

## Data

| OS | Path |
|---|---|
| Windows | `%APPDATA%\Tandemesh` |
| macOS | `~/Library/Application Support/Tandemesh` |
| Linux | `~/.config/Tandemesh` |

The directory holds `tandemesh.db` (with `-wal` and `-shm`), `local.token` and `work/`. Stop the
app before copying the database for a backup.
