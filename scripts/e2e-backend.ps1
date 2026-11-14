param(
    [int]$Port = 8080,
    [int]$TimeoutSeconds = 60,
    [switch]$Build
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$bin = Join-Path $root 'bin'
$planeExe = Join-Path $bin 'orca-plane.exe'
$agentExe = Join-Path $bin 'orca-agentd.exe'

if ($Build -or -not (Test-Path $planeExe) -or -not (Test-Path $agentExe)) {
    Push-Location $root
    try {
        & go build -o $planeExe ./cmd/orca-plane
        if ($LASTEXITCODE -ne 0) { throw 'build orca-plane failed' }
        & go build -o $agentExe ./cmd/orca-agentd
        if ($LASTEXITCODE -ne 0) { throw 'build orca-agentd failed' }
    } finally {
        Pop-Location
    }
}

$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$db = Join-Path $bin "e2e-$stamp.db"
$planeLog = Join-Path $bin "e2e-$stamp-plane.log"
$planeErr = Join-Path $bin "e2e-$stamp-plane.err"
$agentLog = Join-Path $bin "e2e-$stamp-agent.log"
$agentErr = Join-Path $bin "e2e-$stamp-agent.err"

$env:ORCA_LISTEN = ":$Port"
$env:ORCA_DB_PATH = $db
$env:ORCA_JOIN_TOKEN = 'e2e-token'
$env:ORCA_RUN_TIMEOUT = '3m'
$env:ORCA_TELEMETRY = 'off'

$base = "http://127.0.0.1:$Port"
$plane = $null
$agent = $null
try {
    $plane = Start-Process -FilePath $planeExe -WorkingDirectory $root `
        -RedirectStandardOutput $planeLog -RedirectStandardError $planeErr `
        -PassThru -WindowStyle Hidden

    $healthy = $false
    for ($i = 0; $i -lt 40; $i++) {
        Start-Sleep -Milliseconds 250
        try {
            $resp = Invoke-WebRequest "$base/healthz" -UseBasicParsing -TimeoutSec 2
            if ($resp.StatusCode -eq 200) { $healthy = $true; break }
        } catch {}
    }
    if (-not $healthy) { throw "control plane did not become healthy; see $planeErr" }

    $agent = Start-Process -FilePath $agentExe `
        -ArgumentList @('-plane', $base, '-token', 'e2e-token', '-name', 'e2e-1', '-agents', 'claude', '-runner', 'mock') `
        -WorkingDirectory $root -RedirectStandardOutput $agentLog -RedirectStandardError $agentErr `
        -PassThru -WindowStyle Hidden

    $registered = $false
    for ($i = 0; $i -lt 40; $i++) {
        Start-Sleep -Milliseconds 250
        try {
            $machines = Invoke-RestMethod "$base/api/v1/machines"
            if (($machines.items | Measure-Object).Count -gt 0) { $registered = $true; break }
        } catch {}
    }
    if (-not $registered) { throw "agent did not register; see $agentErr" }

    $body = @{
        work = @{ provider = 'manual'; instance = 'default'; type = 'task'; id = "E2E-$stamp" }
        prompt = "E2E backend $stamp"
        repo = 'acme/api'
        agentId = 'claude'
    } | ConvertTo-Json -Depth 5

    $run = Invoke-RestMethod -Method Post -Uri "$base/api/v1/runs" `
        -ContentType 'application/json' -Body $body
    Write-Host "run $($run.id) queued"

    $final = $null
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        Start-Sleep -Milliseconds 500
        $final = Invoke-RestMethod "$base/api/v1/runs/$($run.id)"
        if ($final.status -in @('succeeded', 'failed', 'cancelled')) { break }
    }

    if (-not $final -or $final.status -ne 'succeeded') {
        $status = if ($final) { $final.status } else { 'unknown' }
        $message = if ($final) { $final.errorMessage } else { '' }
        throw "run ended $status $message; see $agentLog"
    }

    Write-Host "PASS backend run=$($run.id) status=$($final.status) task=$($final.taskId)"
} catch {
    Write-Host "FAIL: $($_.Exception.Message)"
    foreach ($file in @($planeLog, $planeErr, $agentLog, $agentErr)) {
        if (Test-Path $file) {
            Write-Host "--- $file"
            Get-Content $file -ErrorAction SilentlyContinue | Select-Object -Last 20
        }
    }
    exit 1
} finally {
    if ($agent -and (Get-Process -Id $agent.Id -ErrorAction SilentlyContinue)) {
        Stop-Process -Id $agent.Id -Force -ErrorAction SilentlyContinue
    }
    if ($plane -and (Get-Process -Id $plane.Id -ErrorAction SilentlyContinue)) {
        Stop-Process -Id $plane.Id -Force -ErrorAction SilentlyContinue
    }
}
