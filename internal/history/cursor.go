package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ScanCursor reads Cursor Agent CLI JSONL transcripts.
func ScanCursor(home string, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	root := cursorRoot(home)
	projects := filepath.Join(root, "projects")
	if info, err := os.Stat(projects); err != nil || !info.IsDir() {
		return []Entry{}, nil
	}
	out := []Entry{}
	err := filepath.WalkDir(projects, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".jsonl") {
			return nil
		}
		if !strings.Contains(strings.ToLower(filepath.ToSlash(path)), "/agent-transcripts/") {
			return nil
		}
		entry, err := parseCursorFile(path)
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

func cursorRoot(home string) string {
	if value := strings.TrimSpace(os.Getenv("CURSOR_CONFIG_DIR")); value != "" {
		return value
	}
	return filepath.Join(home, ".cursor")
}

func parseCursorFile(path string) (Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return Entry{}, err
	}
	defer file.Close()
	entry := Entry{
		ID:    strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		Agent: "cursor",
		Path:  path,
		Cwd:   cursorChatCwd(cursorRootForPath(path), entryID(path)),
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var line struct {
			Role    string `json:"role"`
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
			Timestamp string `json:"timestamp"`
		}
		if json.Unmarshal(scanner.Bytes(), &line) != nil {
			continue
		}
		if line.Role != "user" && line.Role != "assistant" {
			continue
		}
		text, stamp := cursorText(line.Message.Content)
		if text == "" {
			continue
		}
		entry.MessageCount++
		if stamp != "" && entry.UpdatedAt.IsZero() {
			entry.UpdatedAt = parseHistoryString(stamp)
		}
		if entry.Title == "" && line.Role == "user" {
			entry.Title = truncatePreview(text)
		}
		if entry.Preview == "" {
			entry.Preview = truncatePreview(text)
		}
	}
	if entry.UpdatedAt.IsZero() {
		if info, err := file.Stat(); err == nil {
			entry.UpdatedAt = info.ModTime()
		}
	}
	if entry.Title == "" {
		entry.Title = "Cursor session"
	}
	return entry, scanner.Err()
}

func entryID(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func cursorRootForPath(path string) string {
	// path ends with <root>/projects/<encoded>/agent-transcripts/.../id.jsonl
	dir := filepath.Dir(path)
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		if filepath.Base(dir) == "projects" {
			return parent
		}
		dir = parent
	}
}

func cursorChatCwd(root, id string) string {
	if root == "" || id == "" {
		return ""
	}
	matches, _ := filepath.Glob(filepath.Join(root, "chats", "*", id, "meta.json"))
	for _, path := range matches {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var meta struct {
			Cwd string `json:"cwd"`
		}
		if json.Unmarshal(raw, &meta) == nil && meta.Cwd != "" {
			return meta.Cwd
		}
	}
	return ""
}

func cursorText(raw json.RawMessage) (string, string) {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return cleanCursorText(text)
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if part.Type == "" || part.Type == "text" {
				out = append(out, part.Text)
			}
		}
		return cleanCursorText(strings.Join(out, "\n"))
	}
	return "", ""
}

func cleanCursorText(text string) (string, string) {
	text = strings.TrimSpace(text)
	stamp := ""
	if strings.HasPrefix(text, "<timestamp>") {
		end := strings.Index(text, "</timestamp>")
		if end > 0 {
			stamp = strings.TrimSpace(text[len("<timestamp>"):end])
			text = strings.TrimSpace(text[end+len("</timestamp>"):])
		}
	}
	return text, stamp
}
