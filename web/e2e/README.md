# Web E2E

`smoke.mjs` runs the real Vite app in headless Edge/Chromium and mocks the Orca HTTP API at the browser boundary.

It verifies:

- task list renders
- task drawer opens
- task detail route and body render
- notifications page
- machines page and labels
- no horizontal overflow at 820px and 390px

Run:

```bash
cd web
pnpm e2e
```

The test uses `playwright-core` with the installed Edge browser. Override the browser path with `EDGE_PATH` if needed.

## Live full-stack E2E

`live.mjs` creates a real run through the control plane, waits for the mock agent to finish it, then drives the browser against the real API.

Prerequisites:

```bash
# terminal 1
D:\project\orca\bin\tandemesh-plane.exe

# terminal 2
D:\project\orca\bin\tandemesh-agentd.exe -name e2e-1 -agents claude -runner mock
```

Run:

```bash
cd web
pnpm e2e:live
```

The backend E2E can be launched and cleaned up automatically from the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\e2e-backend.ps1
```

## Why the API is mocked

On hosts with Application Control, Go binaries built in the workspace cannot be executed. The web E2E therefore isolates the frontend from the control-plane process. Real control-plane E2E lives in `internal/api/*_e2e_test.go` and runs with `go test ./internal/api` on a host that allows test binaries.

## Not covered

- real agent execution
- SSE/WebSocket timing
- authentication redirect
- file upload
- cross-browser (only Edge/Chromium)
