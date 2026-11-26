$ErrorActionPreference = 'Stop'

$desktop = Split-Path -Parent $PSScriptRoot
$root = Split-Path -Parent (Split-Path -Parent $desktop)

Push-Location (Join-Path $root 'web')
try {
    if ($env:PNPM_MJS) {
        $node = if ($env:NODE_BIN) { $env:NODE_BIN } else { 'node' }
        & $node $env:PNPM_MJS run build
    } else {
        pnpm run build
    }
} finally {
    Pop-Location
}

$webDist = Join-Path $root 'web\dist'
$target = Join-Path $desktop 'webdist'
if (-not (Test-Path (Join-Path $webDist 'index.html'))) {
    throw "web/dist/index.html not found"
}

Get-ChildItem $target -Force | Where-Object { $_.Name -ne '.keep' } | Remove-Item -Recurse -Force
Copy-Item (Join-Path $webDist '*') $target -Recurse -Force

Push-Location $desktop
$syso = Join-Path $desktop 'wails_windows_amd64.syso'
try {
    # Plain `go build` does not embed build/windows/icon.ico. Generate the
    # Windows resource file with the Wails CLI so Explorer/taskbar show the
    # brand icon, matching the macOS .app bundle.
    if ($env:OS -eq 'Windows_NT') {
        $wails3 = Get-Command wails3 -ErrorAction SilentlyContinue
        if (-not $wails3) {
            $candidate = Join-Path $env:USERPROFILE 'go\bin\wails3.exe'
            if (Test-Path $candidate) { $wails3 = Get-Item $candidate }
        }
        if ($wails3) {
            Push-Location (Join-Path $desktop 'build')
            try {
                & $wails3.Source generate syso -arch amd64 `
                    -icon windows/icon.ico `
                    -manifest windows/wails.exe.manifest `
                    -info windows/info.json `
                    -out ../wails_windows_amd64.syso
                if ($LASTEXITCODE -ne 0) { throw "wails3 generate syso failed" }
            } finally {
                Pop-Location
            }
        } else {
            Write-Warning "wails3 not found; Windows executable will use the default icon."
        }
    }

    go build -o bin/tandemesh-desktop.exe .
} finally {
    if (Test-Path $syso) { Remove-Item $syso -Force }
    Pop-Location
}

Write-Host "Built desktop with embedded web assets."
