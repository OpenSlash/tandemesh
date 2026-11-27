package main

import (
	"fmt"
	"os/exec"
)

const selfUpdateSupported = true

// installUpdate runs the NSIS installer silently once this process has had
// a moment to exit.
func installUpdate(installer string) error {
	cmd := exec.Command("cmd.exe", "/c", "ping 127.0.0.1 -n 3 >nul & \""+installer+"\" /S /UPDATE")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start installer: %w", err)
	}
	return nil
}
