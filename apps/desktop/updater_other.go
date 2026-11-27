//go:build !darwin && !windows

package main

import "errors"

const selfUpdateSupported = false

func installUpdate(string) error {
	return errors.New("self-update is not supported on this platform")
}
