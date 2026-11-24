# Tandemesh Desktop (Wails 3)

This is the Wails 3 shell. The web application remains in `../../web` and is not duplicated here.

## Behavior

```text
1. Start tandemesh-plane on a free localhost port
2. Use %APPDATA%/Tandemesh/tandemesh.db
3. Serve ../../web/dist through a local HTTP server
4. Proxy /api, /auth and /ws to the plane
5. Open the local URL in the Wails window
```

Build the web UI first:

```powershell
cd D:\project\orca\web
pnpm run build
```

Then:

```powershell
cd apps\tandemesh-desktop
go run .
```

## Environment

```text
TANDEMESH_PLANE_BIN    path to tandemesh-plane
TANDEMESH_MANAGE_PLANE 0 disables automatic plane startup
TANDEMESH_PLANE_URL    reuse an already running plane
TANDEMESH_WEB_DIST     path to web/dist
TANDEMESH_WEB_URL      dev-server mode; skips bundled web
```

## Lifecycle
`n- Single instance lock on loopback port 47831.
- Starts tandemesh-plane and tandemesh-agentd; both are killed on normal exit.
- Set TANDEMESH_AGENTD_BIN to override the agentd path.
`n## Next

- Start/stop agentd with the same lifecycle.
- Bundle web/dist into the executable for release.
- Add single-instance and tray behavior.
- Wire `wails3 build` packaging.
