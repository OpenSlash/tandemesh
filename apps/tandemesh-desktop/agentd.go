package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type localAgentd struct {
	cmd *exec.Cmd
}

func findAgentdBinary() string {
	if value := os.Getenv("TANDEMESH_AGENTD_BIN"); value != "" {
		return value
	}
	name := "tandemesh-agentd"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	candidates := []string{
		filepath.Join(filepath.Dir(os.Args[0]), name),
		filepath.Join("..", "..", "bin", name),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func dataDirPath() string {
	home, err := os.UserConfigDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, "Tandemesh")
}

func startAgentd(planeURL, dataDir, localToken string) *localAgentd {
	binary := findAgentdBinary()
	if binary == "" {
		log.Printf("tandemesh-agentd not found; desktop will run without a local execution host")
		return nil
	}
	workdir := filepath.Join(dataDir, "work")
	_ = os.MkdirAll(workdir, 0o755)
	cmd := exec.Command(binary,
		"-plane", planeURL,
		"-token", "dev-join-token",
		"-name", "desktop",
		"-agents", "claude,codex",
		"-runner", "acp",
		"-acp-command", "claude-agent-acp",
		"-claude-acp-command", "claude-agent-acp",
		"-codex-acp-command", "codex-acp",
		"-workdir", workdir,
	)
	// Passed through the environment so the token stays out of process listings.
	cmd.Env = append(os.Environ(), "TANDEMESH_LOCAL_TOKEN="+localToken)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Printf("start agentd: %v", err)
		return nil
	}
	if err := attachToJob(cmd); err != nil {
		log.Printf("attach agentd job: %v", err)
	}
	return &localAgentd{cmd: cmd}
}

func (a *localAgentd) Stop() {
	if a == nil || a.cmd == nil || a.cmd.Process == nil {
		return
	}
	_ = a.cmd.Process.Kill()
	_, _ = a.cmd.Process.Wait()
}
