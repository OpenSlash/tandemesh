//go:build !windows

package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const pathMarker = "__TANDEMESH_PATH__"

// inheritLoginShellPath gives the sidecars the PATH a terminal would have.
// Apps started from Finder or the Dock get launchd's minimal PATH, which
// hides agent CLIs installed by npm, Homebrew or a version manager.
func inheritLoginShellPath() {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Interactive and login, so both .zprofile and .zshrc (nvm, fnm, ...) run.
	cmd := exec.CommandContext(ctx, shell, "-ilc", `printf '`+pathMarker+`%s`+pathMarker+`' "$PATH"`)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("read login shell PATH: %v", err)
		return
	}
	parts := strings.Split(string(out), pathMarker)
	if len(parts) < 3 || parts[1] == "" {
		log.Printf("read login shell PATH: no PATH in shell output")
		return
	}
	if err := os.Setenv("PATH", mergePathLists(parts[1], os.Getenv("PATH"))); err != nil {
		log.Printf("set PATH: %v", err)
	}
}
