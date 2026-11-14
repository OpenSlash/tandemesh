//go:build !windows

package pty

import (
	"io"
	"os"
	"os/exec"
	"sync"
)

type pipeSession struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	once   sync.Once
}

// Start launches a shell using pipes. Non-Windows builds do not provide a
// kernel pseudo terminal here; most interactive commands still work.
func Start(command, cwd string, cols, rows uint16) (Session, error) {
	cmd := exec.Command("sh", "-c", command)
	if cwd != "" {
		cmd.Dir = cwd
	}
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &pipeSession{cmd: cmd, stdin: stdin, stdout: stdout}, nil
}

func (s *pipeSession) Read(p []byte) (int, error)     { return s.stdout.Read(p) }
func (s *pipeSession) Write(p []byte) (int, error)    { return s.stdin.Write(p) }
func (s *pipeSession) Resize(cols, rows uint16) error { return nil }
func (s *pipeSession) Pid() int                       { return s.cmd.Process.Pid }

func (s *pipeSession) Wait() (int, error) {
	err := s.cmd.Wait()
	if err == nil {
		return 0, nil
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), nil
	}
	return 1, err
}

func (s *pipeSession) Close() error {
	var err error
	s.once.Do(func() {
		_ = s.stdin.Close()
		if s.cmd.Process != nil {
			err = s.cmd.Process.Kill()
		}
	})
	return err
}
