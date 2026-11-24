package policy

import (
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestMatchKinds(t *testing.T) {
	ev := integration.Event{Kind: "labeled", Labels: []string{"agent:fix"}, Assignee: "alice", Text: "/orca fix"}
	cases := []struct {
		name    string
		binding domain.Automation
		want    bool
	}{
		{"any", domain.Automation{TriggerKind: domain.TriggerAny}, true},
		{"label hit", domain.Automation{TriggerKind: domain.TriggerLabel, TriggerValue: "AGENT:FIX"}, true},
		{"label miss", domain.Automation{TriggerKind: domain.TriggerLabel, TriggerValue: "bug"}, false},
		{"assign", domain.Automation{TriggerKind: domain.TriggerAssign}, true},
		{"comment", domain.Automation{TriggerKind: domain.TriggerComment, TriggerValue: "/orca"}, true},
		{"comment miss", domain.Automation{TriggerKind: domain.TriggerComment, TriggerValue: "/deploy"}, false},
	}
	for _, c := range cases {
		if got := Match(&c.binding, ev); got != c.want {
			t.Fatalf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestMatchRule(t *testing.T) {
	ev := integration.Event{Kind: "labeled", Labels: []string{"agent:fix", "backend"}, Assignee: "alice", Text: "/orca fix login"}

	if !Match(&domain.Automation{TriggerKind: "rule", TriggerValue: `{"labelsAll":["agent:fix","backend"]}`}, ev) {
		t.Fatal("labelsAll should match")
	}
	if Match(&domain.Automation{TriggerKind: "rule", TriggerValue: `{"labelsAll":["agent:fix","frontend"]}`}, ev) {
		t.Fatal("missing label should not match")
	}
	if !Match(&domain.Automation{TriggerKind: "rule", TriggerValue: `{"labelsAny":["frontend","backend"],"kinds":["labeled"]}`}, ev) {
		t.Fatal("labelsAny + kinds should match")
	}
	if !Match(&domain.Automation{TriggerKind: "rule", TriggerValue: `{"textContains":"fix login","assignee":"alice"}`}, ev) {
		t.Fatal("text + assignee should match")
	}
	if Match(&domain.Automation{TriggerKind: "rule", TriggerValue: `not json`}, ev) {
		t.Fatal("invalid rule should not match")
	}
}
