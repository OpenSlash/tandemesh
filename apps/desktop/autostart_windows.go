//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

const autostartKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func isAutostartEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, autostartKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	value, _, err := key.GetStringValue("Tandemesh")
	return err == nil && value != ""
}

func setAutostart(enabled bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, autostartKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if !enabled {
		return key.DeleteValue("Tandemesh")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return key.SetStringValue("Tandemesh", `"`+exe+`"`)
}
