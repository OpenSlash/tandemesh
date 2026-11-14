# Tandemesh desktop (Wails v3)

This directory holds the Wails v3 shell. It is **build-tagged** so the default
`go build ./...` does not pull Wails or require a webview.

## Why a separate shell

- The core (`internal/app`, `internal/desktop`) never imports Wails.
- The shell only binds `desktop.Service` and serves the same React console.
- Web, CLI and desktop share one object graph and one UI codebase.

## Build

```bash
# 1) build the console
cd web && pnpm install && pnpm build

# 2) stage assets next to the shell (embedded at build time)
mkdir -p ../cmd/tandemesh-desktop/frontend
cp -r dist ../cmd/tandemesh-desktop/frontend/dist

# 3) fetch Wails v3 (alpha; pin a version for reproducibility)
cd .. && go get github.com/wailsapp/wails/v3@latest

# 4) build and run
go build -tags wails -o tandemesh-desktop ./cmd/tandemesh-desktop
./tandemesh-desktop
```

Platform notes:

- Windows needs WebView2 runtime (preinstalled on current Windows 10/11).
- macOS needs Xcode command line tools; sign and notarise before distributing.
- Linux needs WebKitGTK 4.1 development packages.

## What the shell binds

`desktop.Service` exposes:

- `ListRuns`, `GetRun`, `CreateRun`, `CancelRun`
- `ListApprovals`, `DecideApproval`
- `Providers`, `ListMachines`
- `Version`

These return plain structs, so Wails can marshal them directly. The shell does
not call the HTTP API; it uses the store and registry in process.

## Next steps

1. Add tray icon with pending-approval count.
2. Register `tandemesh://run/<id>` deep links and single-instance focus.
3. Package with `wails build` and wire the update feed.

Wails v3 is alpha: the `application.Options` / `Service` API may change between
releases. Keep this shell thin so upgrades stay mechanical.
