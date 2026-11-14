$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$out = Join-Path $root 'bin\agents'
New-Item -ItemType Directory -Force -Path $out | Out-Null

$targets = @(
    @{ os = 'windows'; arch = 'amd64'; file = 'tandemesh-agentd-windows-amd64.exe' },
    @{ os = 'linux';   arch = 'amd64'; file = 'tandemesh-agentd-linux-amd64' },
    @{ os = 'linux';   arch = 'arm64'; file = 'tandemesh-agentd-linux-arm64' },
    @{ os = 'darwin';  arch = 'amd64'; file = 'tandemesh-agentd-darwin-amd64' },
    @{ os = 'darwin';  arch = 'arm64'; file = 'tandemesh-agentd-darwin-arm64' }
)

Push-Location $root
try {
    foreach ($target in $targets) {
        $env:GOOS = $target.os
        $env:GOARCH = $target.arch
        $env:CGO_ENABLED = '0'
        $dest = Join-Path $out $target.file
        Write-Host "Building $($target.os)/$($target.arch) -> $dest"
        & go build -trimpath -ldflags '-s -w' -o $dest ./cmd/tandemesh-agentd
        if ($LASTEXITCODE -ne 0) {
            throw "build failed for $($target.os)/$($target.arch)"
        }
    }
} finally {
    Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
    Pop-Location
}

Get-ChildItem $out | Select-Object Name, Length | Format-Table -AutoSize
