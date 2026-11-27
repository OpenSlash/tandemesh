package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

const selfUpdateSupported = true

// installUpdate runs the NSIS installer silently into the current install
// directory once this process has had a moment to exit; the installer
// relaunches the app when it finishes. The manifest signature already
// vouches for the installer.
func installUpdate(installer string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// NSIS wants /D last and unquoted, even when the path has spaces.
	line := fmt.Sprintf(`ping 127.0.0.1 -n 3 >nul & "%s" /S /D=%s`, installer, filepath.Dir(exe))
	cmd := exec.Command("cmd.exe")
	// Pass the line verbatim; Go's argument quoting would wrap the /D path
	// in quotes, which NSIS rejects.
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: "cmd.exe /c " + line}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start installer: %w", err)
	}
	return cmd.Process.Release()
}
