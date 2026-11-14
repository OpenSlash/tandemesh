// Package pty starts an interactive shell attached to a pseudo terminal.
package pty

import (
	"io"
)

// Session is an interactive shell process.
type Session interface {
	io.Reader
	io.Writer
	Resize(cols, rows uint16) error
	Close() error
	Wait() (int, error)
	Pid() int
}
