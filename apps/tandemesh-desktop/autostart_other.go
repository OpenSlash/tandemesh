//go:build !windows

package main

func isAutostartEnabled() bool { return false }

func setAutostart(enabled bool) error { return nil }
