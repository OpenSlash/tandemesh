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
try {
    go build -o bin/tandemesh-desktop.exe .
} finally {
    Pop-Location
}

Write-Host "Built desktop with embedded web assets."
