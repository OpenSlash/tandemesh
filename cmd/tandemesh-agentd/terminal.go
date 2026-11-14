package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/OpenSlash/tandemesh/internal/pty"
)

type terminalLease struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	Shell       string `json:"shell"`
	Cwd         string `json:"cwd"`
	Cols        int    `json:"cols"`
	Rows        int    `json:"rows"`
}

type terminalInputChunk struct {
	Seq  int    `json:"seq"`
	Data string `json:"data"`
}

func terminalLoop(ctx context.Context, c *http.Client, plane, machineID, workdir string, log *slog.Logger) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var active sync.Map
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, plane+"/api/v1/agentd/terminals?machineId="+machineID, nil)
			resp, err := c.Do(req)
			if err != nil {
				continue
			}
			var body struct {
				Items []terminalLease `json:"items"`
			}
			err = json.NewDecoder(resp.Body).Decode(&body)
			resp.Body.Close()
			if err != nil {
				continue
			}
			for _, t := range body.Items {
				if _, loaded := active.LoadOrStore(t.ID, true); loaded {
					continue
				}
				go func(t terminalLease) {
					defer active.Delete(t.ID)
					runTerminal(ctx, c, plane, machineID, workdir, t, log)
				}(t)
			}
		}
	}
}

func runTerminal(ctx context.Context, c *http.Client, plane, machineID, workdir string, t terminalLease, log *slog.Logger) {
	shell := strings.TrimSpace(t.Shell)
	if shell == "" {
		shell = defaultShell()
	}
	cwd := strings.TrimSpace(t.Cwd)
	if cwd == "" {
		cwd = workdir
	}
	if cwd != "" {
		if st, err := os.Stat(cwd); err != nil || !st.IsDir() {
			cwd = ""
		}
	}
	sess, err := pty.Start(terminalCommand(shell), cwd, uint16(t.Cols), uint16(t.Rows))
	if err != nil {
		log.Warn("terminal start", "terminal", t.ID, "err", err)
		terminalState(ctx, c, plane, t.ID, "exited", intPtr(1))
		return
	}
	log.Info("terminal started", "terminal", t.ID, "shell", shell, "cwd", cwd)
	terminalState(ctx, c, plane, t.ID, "running", nil)

	outputDone := make(chan struct{})
	go func() {
		defer close(outputDone)
		buf := make([]byte, 32*1024)
		for {
			n, err := sess.Read(buf)
			if n > 0 {
				terminalOutput(ctx, c, plane, t.ID, string(buf[:n]))
			}
			if err != nil {
				return
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				_ = sess.Close()
				return
			case <-outputDone:
				return
			case <-ticker.C:
				inputs, closed, err := terminalInputs(ctx, c, plane, t.ID)
				if err != nil || closed {
					_ = sess.Close()
					return
				}
				for _, in := range inputs {
					if _, err := sess.Write([]byte(in.Data)); err != nil {
						_ = sess.Close()
						return
					}
				}
			}
		}
	}()

	code, waitErr := sess.Wait()
	<-outputDone
	if waitErr != nil {
		log.Warn("terminal wait", "terminal", t.ID, "err", waitErr)
	}
	terminalState(ctx, c, plane, t.ID, "exited", &code)
}

func terminalCommand(shell string) string {
	fields := strings.Fields(shell)
	if len(fields) == 0 {
		return shell
	}
	base := strings.Trim(fields[0], `"`)
	if strings.HasSuffix(strings.ToLower(base), ".exe") {
		return shell
	}
	if path, err := exec.LookPath(base); err == nil && strings.HasSuffix(strings.ToLower(path), ".exe") {
		return shell
	}
	cmdPath, err := exec.LookPath("cmd.exe")
	if err != nil {
		return shell
	}
	return `"` + cmdPath + `" /k ` + shell
}
func defaultShell() string {
	if runtime.GOOS == "windows" {
		for _, name := range []string{"cmd.exe", "pwsh.exe", "powershell.exe"} {
			if path, err := exec.LookPath(name); err == nil {
				if name == "cmd.exe" {
					return path + " /k chcp 65001>nul"
				}
				return path
			}
		}
		return "cmd.exe"
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh
	}
	return "/bin/sh"
}

func terminalInputs(ctx context.Context, c *http.Client, plane, id string) ([]terminalInputChunk, bool, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, plane+"/api/v1/agentd/terminals/"+id+"/input", nil)
	resp, err := c.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusGone {
		return nil, true, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("status %d", resp.StatusCode)
	}
	var body struct {
		Inputs []terminalInputChunk `json:"inputs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, false, err
	}
	return body.Inputs, false, nil
}

func terminalOutput(ctx context.Context, c *http.Client, plane, id, data string) {
	raw, _ := json.Marshal(map[string]any{"data": data})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, plane+"/api/v1/agentd/terminals/"+id+"/output", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func terminalState(ctx context.Context, c *http.Client, plane, id, status string, code *int) {
	raw, _ := json.Marshal(map[string]any{"status": status, "exitCode": code})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, plane+"/api/v1/agentd/terminals/"+id+"/state", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func intPtr(v int) *int { return &v }
