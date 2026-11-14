package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) downloadAgentd(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/download/agentd/"), "/")
	if len(parts) != 2 {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "unknown agentd platform")
		return
	}
	osName, arch := parts[0], parts[1]
	switch osName {
	case "windows", "darwin", "linux":
	default:
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "unsupported operating system")
		return
	}
	switch arch {
	case "amd64", "arm64":
	default:
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "unsupported architecture")
		return
	}
	name := fmt.Sprintf("orca-agentd-%s-%s", osName, arch)
	if osName == "windows" {
		name += ".exe"
	}
	path := filepath.Join(s.cfg.AgentdDir, name)
	if _, err := os.Stat(path); err != nil {
		writeErr(w, http.StatusNotFound, "E_AGENTD", "agentd binary is not published for "+osName+"/"+arch)
		return
	}
	http.ServeFile(w, r, path)
}

func (s *Server) installAgentdShell(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeErr(w, http.StatusBadRequest, "E_TOKEN", "token is required")
		return
	}
	base := publicBase(s, r)
	script := fmt.Sprintf(`#!/bin/sh
set -eu
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$OS" in
  linux) ;;
  darwin) ;;
  *) echo "unsupported operating system: $OS" >&2; exit 1 ;;
esac
case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) echo "unsupported architecture: $ARCH" >&2; exit 1 ;;
esac
DEST="$HOME/.local/bin/orca-agentd"
mkdir -p "$(dirname "$DEST")"
echo "Downloading orca-agentd for $OS/$ARCH..."
curl -fsSL %s/download/agentd/$OS/$ARCH -o "$DEST"
chmod +x "$DEST"
echo "Starting orca-agentd..."
exec "$DEST" -plane %s -token %s -name "$(hostname)" -labels "os=$OS,arch=$ARCH"
`, shellQuote(base), shellQuote(base), shellQuote(token))
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	_, _ = w.Write([]byte(script))
}

func (s *Server) installAgentdPowerShell(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeErr(w, http.StatusBadRequest, "E_TOKEN", "token is required")
		return
	}
	base := publicBase(s, r)
	script := fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$dest = Join-Path $env:LOCALAPPDATA 'Orca\orca-agentd.exe'
New-Item -ItemType Directory -Force -Path (Split-Path $dest) | Out-Null
Write-Host "Downloading orca-agentd for windows/amd64..."
Invoke-WebRequest -UseBasicParsing '%s/download/agentd/windows/amd64' -OutFile $dest
Write-Host "Starting orca-agentd..."
& $dest -plane '%s' -token '%s' -name $env:COMPUTERNAME -labels 'os=windows,arch=amd64'
`, powerShellQuote(base), powerShellQuote(base), powerShellQuote(token))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(script))
}

func publicBase(s *Server, r *http.Request) string {
	base := strings.TrimRight(s.cfg.BaseURL, "/")
	if base == "" || strings.Contains(base, "localhost") || strings.Contains(base, "127.0.0.1") {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		base = scheme + "://" + r.Host
	}
	return base
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func powerShellQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
