# Starts orca-agentd with Claude Code pointed at DeepSeek's Anthropic API.
#
# Usage:
#   $env:DEEPSEEK_API_KEY = "sk-..."      # or write the key to .tmp\deepseek.key
#   .\scripts\agentd-deepseek.ps1 -Check  # print the resolved settings
#   .\scripts\agentd-deepseek.ps1         # start the daemon with the ACP runner
param(
  [string]$Plane = "http://127.0.0.1:8080",
  [string]$Name = $env:COMPUTERNAME,
  [string]$KeyFile = "",
  [int]$Concurrency = 1,
  [string]$Workdir = "",
  [switch]$Check
)

$ErrorActionPreference = "Stop"
$root = Split-Path $PSScriptRoot -Parent
if (-not $KeyFile) { $KeyFile = Join-Path $root ".tmp\deepseek.key" }

$key = $env:DEEPSEEK_API_KEY
if (-not $key -and (Test-Path $KeyFile)) {
  $key = (Get-Content $KeyFile -Raw).Trim()
}
if (-not $key) {
  throw "DeepSeek API key missing. Set DEEPSEEK_API_KEY or write it to $KeyFile."
}

$env:ANTHROPIC_BASE_URL = "https://api.deepseek.com/anthropic"
$env:ANTHROPIC_AUTH_TOKEN = $key
$env:ANTHROPIC_MODEL = "claude-sonnet-4-5"
$env:ANTHROPIC_DEFAULT_OPUS_MODEL = "claude-sonnet-4-5"
$env:ANTHROPIC_DEFAULT_SONNET_MODEL = "claude-sonnet-4-5"
$env:ANTHROPIC_DEFAULT_HAIKU_MODEL = "claude-haiku-4-5"
$env:CLAUDE_CODE_SUBAGENT_MODEL = "claude-sonnet-4-5"
$env:CLAUDE_CODE_EFFORT_LEVEL = "max"
$env:CLAUDE_CODE_AUTO_COMPACT_WINDOW = "786432"

# Claude Code on Windows needs a git-bash binary.
$gitBash = Join-Path $root ".tmp\PortableGit\bin\bash.exe"
if (-not (Test-Path $gitBash)) { $gitBash = "C:\Program Files\Git\bin\bash.exe" }
if (Test-Path $gitBash) { $env:CLAUDE_CODE_GIT_BASH_PATH = $gitBash }

# Local dev installs: keep node and the CLI wrappers reachable.
$nodeDir = "C:\Users\jairo\.dsh\dsh-runtimes\dsh-primary-runtime\dependencies\node\bin"
$cliBin = Join-Path $root ".tmp\claude-cli\node_modules\.bin"
foreach ($dir in @($nodeDir, $cliBin)) {
  if ((Test-Path $dir) -and ($env:PATH -notlike "*$dir*")) { $env:PATH = "$dir;$env:PATH" }
}

$agentd = Join-Path $root "bin\orca-agentd.exe"
if (-not (Test-Path $agentd)) {
  go build -o $agentd ./cmd/orca-agentd
}

$agentdArgs = @(
  "-plane", $Plane,
  "-name", $Name,
  "-runner", "acp",
  "-acp-command", "claude-code-acp",
  "-concurrency", "$Concurrency"
)
if ($Workdir) { $agentdArgs += @("-workdir", $Workdir) }

if ($Check) {
  Write-Output "plane=$Plane name=$Name runner=acp"
  Write-Output "ANTHROPIC_BASE_URL=$env:ANTHROPIC_BASE_URL"
  Write-Output "ANTHROPIC_MODEL=$env:ANTHROPIC_MODEL"
  Write-Output "CLAUDE_CODE_GIT_BASH_PATH=$env:CLAUDE_CODE_GIT_BASH_PATH"
  Write-Output "agentd=$agentd"
  exit 0
}

& $agentd @agentdArgs
