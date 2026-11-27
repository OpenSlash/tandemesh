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

// installUpdate unpacks a zipped .app, checks that it is signed by the same
// team as the running app and hands the swap to a detached shell that
// outlives this process.
func installUpdate(archive string) error {
	current, err := currentBundle()
	if err != nil {
		return err
	}
	team, err := signingTeam(current)
	if err != nil {
		return err
	}
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

// signingTeam reads the Developer ID team that signed bundle. Builds without
// one (ad-hoc or unsigned) have nothing to pin an update to, so they refuse.
func signingTeam(bundle string) (string, error) {
	out, _ := exec.Command("codesign", "-dv", bundle).CombinedOutput()
	for _, line := range strings.Split(string(out), "\n") {
		if team, ok := strings.CutPrefix(line, "TeamIdentifier="); ok && team != "not set" {
			return team, nil
		}
	}
	return "", errors.New("this copy is not signed with a Developer ID; download the new version manually")
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

// verifyBundle rejects bundles not signed by team, so a tampered feed or
// mirror cannot install someone else's code.
func verifyBundle(bundle, team string) error {
	requirement := `=anchor apple generic and certificate leaf[subject.OU] = "` + team + `"`
	out, err := exec.Command("codesign", "--verify", "--deep", "--strict", "-R", requirement, bundle).CombinedOutput()
	if err != nil {
		return fmt.Errorf("update signature: %s", strings.TrimSpace(string(out)))
	}
	return nil
}
