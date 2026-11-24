# Install the repository-local Git hooks.
$ErrorActionPreference = 'Stop'

$repo = Split-Path -Parent $PSScriptRoot
Push-Location $repo
try {
    git config core.hooksPath .githooks
    Write-Host "Git hooks installed: core.hooksPath=.githooks"
} finally {
    Pop-Location
}
