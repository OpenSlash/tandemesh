package agent

import "testing"

func TestShouldAsk(t *testing.T) {
	ask := []string{"push", "Edit"}
	cases := map[string]bool{
		"Run git push origin main": true,
		"Edit src/app.ts":          true,
		"Read the README":          false,
		"":                         false,
	}
	for tool, want := range cases {
		if got := ShouldAsk(tool, ask); got != want {
			t.Fatalf("ShouldAsk(%q) = %v, want %v", tool, got, want)
		}
	}
	if ShouldAsk("git push", nil) {
		t.Fatal("empty policy should not ask")
	}
}

func TestRiskFor(t *testing.T) {
	cases := map[string]string{
		"git push":     "high",
		"rm -rf build": "high",
		"edit app.ts":  "medium",
	}
	for tool, want := range cases {
		if got := RiskFor(tool); got != want {
			t.Fatalf("RiskFor(%q) = %q, want %q", tool, got, want)
		}
	}
}

func TestPermissionOutcome(t *testing.T) {
	params := []byte(`{"options":[
		{"optionId":"allow_once","kind":"allow_once"},
		{"optionId":"reject_once","kind":"reject_once"}
	]}`)

	allow := permissionOutcome(params, true)
	if allow["outcome"].(map[string]any)["optionId"] != "allow_once" {
		t.Fatalf("allow outcome = %+v", allow)
	}
	deny := permissionOutcome(params, false)
	if deny["outcome"].(map[string]any)["optionId"] != "reject_once" {
		t.Fatalf("deny outcome = %+v", deny)
	}

	empty := permissionOutcome(nil, false)
	if empty["outcome"].(map[string]any)["outcome"] != "cancelled" {
		t.Fatalf("empty deny outcome = %+v", empty)
	}
}
