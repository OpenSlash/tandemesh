package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// CustomACPBackend is a user-supplied ACP launcher. It lets an operator add
// an agent without changing the built-in table.
type CustomACPBackend struct {
	ID      string            `json:"id"`
	Name    string            `json:"name,omitempty"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type customACPFile struct {
	Backends []CustomACPBackend `json:"backends"`
}

// LoadACPBackends reads a JSON file of custom ACP launchers.
func LoadACPBackends(path string) (map[string]ACPBackendSpec, error) {
	out := map[string]ACPBackendSpec{}
	if strings.TrimSpace(path) == "" {
		return out, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file customACPFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, err
	}
	for _, backend := range file.Backends {
		id := strings.TrimSpace(backend.ID)
		command := strings.TrimSpace(backend.Command)
		if id == "" || command == "" {
			return nil, fmt.Errorf("custom ACP backend requires id and command")
		}
		if _, exists := DefaultACPBackends[id]; exists {
			return nil, fmt.Errorf("custom ACP backend %q shadows a built-in backend", id)
		}
		env := make([]string, 0, len(backend.Env))
		keys := make([]string, 0, len(backend.Env))
		for key := range backend.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			env = append(env, key+"="+backend.Env[key])
		}
		out[id] = ACPBackendSpec{Command: command, Args: backend.Args, Env: env}
	}
	return out, nil
}
