package jira

import (
	"encoding/json"
	"fmt"
	"strings"
)

// projectFields folds Jira priority/components/fix versions into labels so the
// task projection can filter on them without a provider-specific field schema.
func projectFields(labels []string, priority string, components, fixVersions []string) []string {
	out := make([]string, 0, len(labels)+1+len(components)+len(fixVersions))
	seen := map[string]bool{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		out = append(out, value)
	}
	for _, label := range labels {
		add(label)
	}
	if priority != "" {
		add("priority:" + priority)
	}
	for _, component := range components {
		add("component:" + component)
	}
	for _, version := range fixVersions {
		add("fixVersion:" + version)
	}
	return out
}

func jiraFields(priority string, components, versions []string) map[string][]string {
	fields := map[string][]string{}
	if priority != "" {
		fields["priority"] = []string{priority}
	}
	if len(components) > 0 {
		fields["components"] = components
	}
	if len(versions) > 0 {
		fields["fixVersions"] = versions
	}
	return fields
}

func jiraRawValues(raw json.RawMessage) []string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return []string{text}
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return []string{fmt.Sprintf("%g", number)}
	}
	var object struct {
		Value string `json:"value"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(raw, &object); err == nil {
		if object.Value != "" {
			return []string{object.Value}
		}
		if object.Name != "" {
			return []string{object.Name}
		}
	}
	var list []json.RawMessage
	if err := json.Unmarshal(raw, &list); err == nil {
		out := make([]string, 0, len(list))
		for _, item := range list {
			out = append(out, jiraRawValues(item)...)
		}
		return out
	}
	return nil
}
