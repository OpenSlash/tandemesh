package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ScanCodex reads Codex CLI rollout sessions from <home>/.codex/sessions.
func ScanCodex(home string, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	root := filepath.Join(home, ".codex", "sessions")
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return []Entry{}, nil
	}
	out := make([]Entry, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}
		entry, err := parseCodexFile(path)
		if err != nil || entry.ID == "" {
			return nil
		}
		out = append(out, entry)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func parseCodexFile(path string) (Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return Entry{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Entry{}, err
	}
	entry := Entry{Agent: "codex", Path: path, UpdatedAt: info.ModTime()}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		var raw map[string]any
		if json.Unmarshal(scanner.Bytes(), &raw) != nil {
			continue
		}
		if ts, ok := raw["timestamp"].(string); ok {
			if parsed, err := time.Parse(time.RFC3339Nano, ts); err == nil && parsed.After(entry.UpdatedAt) {
				entry.UpdatedAt = parsed
			}
		}
		payload, _ := raw["payload"].(map[string]any)
		if payload == nil {
			continue
		}
		kind, _ := raw["type"].(string)
		switch kind {
		case "session_meta":
			if entry.ID == "" {
				entry.ID = stringValue(payload["id"])
			}
			if cwd := stringValue(payload["cwd"]); cwd != "" {
				entry.Cwd = cwd
			}
			if model := stringValue(payload["model"]); model != "" {
				entry.Model = model
			}
			if git, ok := payload["git"].(map[string]any); ok {
				if branch := stringValue(git["branch"]); branch != "" {
					entry.Branch = branch
				}
			}
		case "turn_context":
			if cwd := stringValue(payload["cwd"]); cwd != "" {
				entry.Cwd = cwd
			}
			if model := stringValue(payload["model"]); model != "" {
				entry.Model = model
			}
		case "response_item":
			if stringValue(payload["type"]) != "message" || stringValue(payload["role"]) != "user" {
				continue
			}
			text := codexContentText(payload["content"])
			if text == "" {
				continue
			}
			entry.MessageCount++
			if entry.Title == "" {
				entry.Title = preview(text, 80)
			}
			entry.Preview = preview(text, 300)
		case "event_msg":
			if stringValue(payload["type"]) != "user_message" {
				continue
			}
			text := stringValue(payload["message"])
			if text == "" {
				continue
			}
			if entry.Title == "" {
				entry.Title = preview(text, 80)
			}
			entry.Preview = preview(text, 300)
		}
	}
	if err := scanner.Err(); err != nil {
		return Entry{}, err
	}
	if entry.ID == "" {
		entry.ID = strings.TrimSuffix(filepath.Base(path), ".jsonl")
	}
	if entry.Title == "" {
		entry.Title = "Codex session " + short(entry.ID)
	}
	return entry, nil
}

func codexContentText(content any) string {
	blocks, _ := content.([]any)
	for _, raw := range blocks {
		block, _ := raw.(map[string]any)
		kind := stringValue(block["type"])
		if kind != "input_text" && kind != "text" {
			continue
		}
		if text := stringValue(block["text"]); strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func stringValue(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}
