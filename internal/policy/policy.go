// Package policy decides whether an event should start a run for an automation.
package policy

import (
	"encoding/json"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Rule is the JSON shape stored in automations.trigger_value when
// trigger_kind = "rule".
type Rule struct {
	Kinds        []string `json:"kinds,omitempty"`
	LabelsAll    []string `json:"labelsAll,omitempty"`
	LabelsAny    []string `json:"labelsAny,omitempty"`
	Assignee     string   `json:"assignee,omitempty"`
	TextContains string   `json:"textContains,omitempty"`
}

// Match reports whether the event satisfies the automation trigger.
func Match(b *domain.Automation, ev integration.Event) bool {
	if b == nil {
		return false
	}
	switch b.TriggerKind {
	case "", domain.TriggerAny:
		return true
	case domain.TriggerLabel:
		want := strings.ToLower(strings.TrimSpace(b.TriggerValue))
		if want == "" {
			return true
		}
		for _, label := range ev.Labels {
			if strings.ToLower(label) == want {
				return true
			}
		}
		return false
	case domain.TriggerAssign:
		return strings.EqualFold(ev.Kind, "assigned") || ev.Assignee != ""
	case domain.TriggerComment:
		command := strings.TrimSpace(b.TriggerValue)
		if command == "" {
			command = "/orca"
		}
		return strings.HasPrefix(strings.ToLower(strings.TrimSpace(ev.Text)), strings.ToLower(command))
	case "rule":
		return matchRule(b.TriggerValue, ev)
	}
	return true
}

func matchRule(raw string, ev integration.Event) bool {
	var r Rule
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return false
	}
	if len(r.Kinds) > 0 {
		found := false
		for _, k := range r.Kinds {
			if strings.EqualFold(k, ev.Kind) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(r.LabelsAll) > 0 && !hasAll(ev.Labels, r.LabelsAll) {
		return false
	}
	if len(r.LabelsAny) > 0 && !hasAny(ev.Labels, r.LabelsAny) {
		return false
	}
	if r.Assignee != "" && !strings.EqualFold(r.Assignee, ev.Assignee) {
		return false
	}
	if r.TextContains != "" && !strings.Contains(strings.ToLower(ev.Text), strings.ToLower(r.TextContains)) {
		return false
	}
	return true
}

func hasAll(have, want []string) bool {
	set := map[string]bool{}
	for _, h := range have {
		set[strings.ToLower(h)] = true
	}
	for _, w := range want {
		if !set[strings.ToLower(w)] {
			return false
		}
	}
	return true
}

func hasAny(have, want []string) bool {
	set := map[string]bool{}
	for _, h := range have {
		set[strings.ToLower(h)] = true
	}
	for _, w := range want {
		if set[strings.ToLower(w)] {
			return true
		}
	}
	return false
}
