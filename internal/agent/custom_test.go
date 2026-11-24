package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadACPBackends(t *testing.T) {
	path := filepath.Join(t.TempDir(), "acp.json")
	body := `{"backends":[{"id":"local-agent","command":"npx","args":["-y","local-acp"],"env":{"B":"2","A":"1"}}]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadACPBackends(path)
	if err != nil {
		t.Fatal(err)
	}
	spec, ok := got["local-agent"]
	if !ok || spec.Command != "npx" || len(spec.Args) != 2 {
		t.Fatalf("spec = %+v", spec)
	}
	if len(spec.Env) != 2 || spec.Env[0] != "A=1" || spec.Env[1] != "B=2" {
		t.Fatalf("env = %+v", spec.Env)
	}
}
