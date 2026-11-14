//go:build windows

package pty

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	gopty "github.com/aymanbagabas/go-pty"
)

type winSession struct {
	p    gopty.Pty
	cmd  *gopty.Cmd
	once sync.Once
}

// Start launches command inside a Windows ConPTY.
func Start(command, cwd string, cols, rows uint16) (Session, error) {
	p, err := gopty.New()
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		fields = []string{"cmd.exe"}
	}
	exe := strings.Trim(fields[0], `"`)
	cmd := p.Command(exe, fields[1:]...)
	cmd.Dir = cwd
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		_ = p.Close()
		return nil, err
	}
	if cols > 0 && rows > 0 {
		_ = p.Resize(int(cols), int(rows))
	}
	return &winSession{p: p, cmd: cmd}, nil
}

func (s *winSession) Read(p []byte) (int, error)  { return s.p.Read(p) }
func (s *winSession) Write(p []byte) (int, error) { return s.p.Write(p) }
func (s *winSession) Pid() int                    { return s.cmd.Process.Pid }

func (s *winSession) Resize(cols, rows uint16) error {
	return s.p.Resize(int(cols), int(rows))
}

func (s *winSession) Wait() (int, error) {
	err := s.cmd.Wait()
	if state := s.cmd.ProcessState; state != nil {
		return state.ExitCode(), err
	}
	return 1, err
}

func (s *winSession) Close() error {
	s.once.Do(func() {
		if s.cmd.Process != nil {
			_ = exec.Command("taskkill", "/PID", strconv.Itoa(s.cmd.Process.Pid), "/T", "/F").Run()
		}
	})
	return nil
}
