package jira

import (
	"encoding/json"
	"strings"
)

func adfToMarkdown(raw []byte) string {
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return ""
	}
	return strings.TrimSpace(adfNode(doc)) + "\n"
}

func adfNodes(value any) []any {
	list, _ := value.([]any)
	return list
}

func adfNode(node any) string {
	m, ok := node.(map[string]any)
	if !ok {
		return ""
	}
	typ, _ := m["type"].(string)
	children := adfNodes(m["content"])
	switch typ {
	case "doc":
		var b strings.Builder
		for _, child := range children {
			b.WriteString(adfNode(child))
		}
		return b.String()
	case "paragraph":
		var b strings.Builder
		for _, child := range children {
			b.WriteString(adfNode(child))
		}
		return b.String() + "\n\n"
	case "heading":
		level := 1
		if attrs, ok := m["attrs"].(map[string]any); ok {
			if value, ok := attrs["level"].(float64); ok && value > 0 {
				level = int(value)
			}
		}
		var b strings.Builder
		for _, child := range children {
			b.WriteString(adfNode(child))
		}
		return strings.Repeat("#", level) + " " + b.String() + "\n\n"
	case "text":
		text, _ := m["text"].(string)
		marks := adfNodes(m["marks"])
		for _, mark := range marks {
			mm, ok := mark.(map[string]any)
			if !ok {
				continue
			}
			switch mm["type"] {
			case "strong":
				text = "**" + text + "**"
			case "em":
				text = "*" + text + "*"
			case "code":
				text = "`" + text + "`"
			case "strike":
				text = "~~" + text + "~~"
			}
		}
		return text
	case "hardBreak":
		return "\n"
	case "bulletList", "orderedList":
		var b strings.Builder
		for i, child := range children {
			prefix := "- "
			if typ == "orderedList" {
				prefix = strings.TrimSpace(strings.Join([]string{string(rune('1' + i)), ". "}, ""))
			}
			item := strings.TrimSpace(adfNode(child))
			lines := strings.Split(item, "\n")
			b.WriteString(prefix + lines[0] + "\n")
			for _, line := range lines[1:] {
				if strings.TrimSpace(line) != "" {
					b.WriteString("  " + line + "\n")
				}
			}
		}
		return b.String() + "\n"
	case "listItem":
		var b strings.Builder
		for _, child := range children {
			b.WriteString(adfNode(child))
		}
		return b.String()
	case "codeBlock":
		var b strings.Builder
		for _, child := range children {
			if tm, ok := child.(map[string]any); ok {
				if text, ok := tm["text"].(string); ok {
					b.WriteString(text)
				}
			}
		}
		return "```\n" + b.String() + "\n```\n\n"
	case "blockquote":
		var b strings.Builder
		for _, child := range children {
			b.WriteString(adfNode(child))
		}
		quoted := strings.Split(strings.TrimSpace(b.String()), "\n")
		for i := range quoted {
			quoted[i] = "> " + quoted[i]
		}
		return strings.Join(quoted, "\n") + "\n\n"
	case "mention":
		if attrs, ok := m["attrs"].(map[string]any); ok {
			if text, ok := attrs["text"].(string); ok {
				return "@" + text
			}
		}
		return "@"
	default:
		var b strings.Builder
		for _, child := range children {
			b.WriteString(adfNode(child))
		}
		return b.String()
	}
}
