//go:build !windows

package main

import "os/exec"

func attachToJob(cmd *exec.Cmd) error { return nil }
