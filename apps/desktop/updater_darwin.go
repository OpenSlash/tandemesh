package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const selfUpdateSupported = true

// swapScript waits for the app to quit, replaces the bundle and relaunches.
// If the new bundle cannot be moved in, the old one is put back.
const swapScript = `
pid="$1"; old="$2"; new="$3"; backup="$old.previous"
while kill -0 "$pid" 2>/dev/null; do sleep 0.2; done
rm -rf "$backup"
if mv "$old" "$backup"; then
  if mv "$new" "$old"; then rm -rf "$backup"; else mv "$backup" "$old"; fi
fi
open "$old"
`

// installUpdate unpacks a zipped .app, checks its code signature and hands
// the swap to a detached shell that outlives this process. The manifest
// signature already vouches for the package; when the running app carries a
// Developer ID, the update must also come from the same team.
func installUpdate(archive string) error {
	current, err := currentBundle()
	if err != nil {
		return err
	}
	team := signingTeam(current)
	if err := checkWritable(filepath.Dir(current)); err != nil {
		return fmt.Errorf("cannot replace %s: %w", current, err)
	}
	staging := filepath.Join(filepath.Dir(archive), "unpacked")
	if out, err := exec.Command("ditto", "-x", "-k", archive, staging).CombinedOutput(); err != nil {
		return fmt.Errorf("unpack update: %v: %s", err, out)
	}
	bundles, _ := filepath.Glob(filepath.Join(staging, "*.app"))
	if len(bundles) != 1 {
		return fmt.Errorf("update archive holds %d app bundles, want 1", len(bundles))
	}
	if err := verifyBundle(bundles[0], team); err != nil {
		return err
	}
	cmd := exec.Command("/bin/sh", "-c", swapScript, "tandemesh-update", strconv.Itoa(os.Getpid()), current, bundles[0])
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start update: %w", err)
	}
	return cmd.Process.Release()
}

// signingTeam reads the Developer ID team that signed bundle, or "" for an
// ad-hoc or unsigned build.
func signingTeam(bundle string) string {
	out, _ := exec.Command("codesign", "-dv", bundle).CombinedOutput()
	for _, line := range strings.Split(string(out), "\n") {
		if team, ok := strings.CutPrefix(line, "TeamIdentifier="); ok && team != "not set" {
			return team
		}
	}
	return ""
}

// currentBundle returns the .app this executable runs from.
func currentBundle() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if !strings.HasSuffix(bundle, ".app") {
		return "", errors.New("not running from an app bundle")
	}
	return bundle, nil
}

func checkWritable(dir string) error {
	probe, err := os.CreateTemp(dir, ".tandemesh-update-")
	if err != nil {
		return err
	}
	probe.Close()
	return os.Remove(probe.Name())
}

// verifyBundle checks that the bundle's signature is intact and, when team
// is set, that it was signed by that team.
func verifyBundle(bundle, team string) error {
	args := []string{"--verify", "--deep", "--strict"}
	if team != "" {
		args = append(args, "-R", `=anchor apple generic and certificate leaf[subject.OU] = "`+team+`"`)
	}
	out, err := exec.Command("codesign", append(args, bundle)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("update signature: %s", strings.TrimSpace(string(out)))
	}
	return nil
}
