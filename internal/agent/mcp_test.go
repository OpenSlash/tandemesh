package agent

import "testing"

func TestMCPServersPayload(t *testing.T) {
	servers := []MCPServer{
		{Name: "fs", Command: "npx", Args: []string{"-y", "server-fs"}, Env: map[string]string{"ROOT": "/tmp"}},
		{Name: "web", URL: "https://mcp.example.com/sse"},
		{Name: ""},
	}
	out := mcpServersPayload(servers)
	if len(out) != 2 {
		t.Fatalf("payload = %+v", out)
	}
	first, ok := out[0].(map[string]any)
	if !ok || first["name"] != "fs" || first["command"] != "npx" {
		t.Fatalf("first = %+v", out[0])
	}
	if _, ok := first["env"]; !ok {
		t.Fatalf("env missing: %+v", first)
	}
	second := out[1].(map[string]any)
	if second["url"] != "https://mcp.example.com/sse" {
		t.Fatalf("second = %+v", second)
	}
	if _, ok := second["command"]; ok {
		t.Fatalf("url server should not carry command: %+v", second)
	}
}
