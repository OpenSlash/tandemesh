package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type geminiMessage struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	Content   json.RawMessage `json:"content"`
}

type geminiSession struct {
	SessionID   string          `json:"sessionId"`
	StartTime   string          `json:"startTime"`
	LastUpdated string          `json:"lastUpdated"`
	Messages    []geminiMessage `json:"messages"`
}

// ScanGemini reads Gemini CLI chats from <home>/.gemini/tmp.
func ScanGemini(home string, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	root := filepath.Join(home, ".gemini", "tmp")
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return []Entry{}, nil
	}
	byID := map[string]Entry{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if !strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".jsonl") {
			return nil
		}
		if !strings.Contains(strings.ToLower(filepath.ToSlash(path)), "/chats/") {
			return nil
		}
		entry, err := parseGeminiFile(path)
		if err != nil || entry.ID == "" {
			return nil
		}
		existing, ok := byID[entry.ID]
		if !ok || (strings.HasSuffix(name, ".jsonl") && !strings.HasSuffix(strings.ToLower(existing.Path), ".jsonl")) {
			byID[entry.ID] = entry
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(byID))
	for _, entry := range byID {
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func parseGeminiFile(path string) (Entry, error) {
	if strings.HasSuffix(strings.ToLower(path), ".jsonl") {
		return parseGeminiJSONL(path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Entry{}, err
	}
	var session geminiSession
	if err := json.Unmarshal(raw, &session); err != nil {
		return Entry{}, err
	}
	return geminiEntry(path, session), nil
}

func parseGeminiJSONL(path string) (Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return Entry{}, err
	}
	defer file.Close()
	session := geminiSession{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var line struct {
			geminiSession
			SessionID   string          `json:"sessionId"`
			StartTime   string          `json:"startTime"`
			LastUpdated string          `json:"lastUpdated"`
			Type        string          `json:"type"`
			Timestamp   string          `json:"timestamp"`
			Content     json.RawMessage `json:"content"`
			Set         struct {
				LastUpdated string `json:"lastUpdated"`
			} `json:"$set"`
		}
		if json.Unmarshal(scanner.Bytes(), &line) != nil {
			continue
		}
		if line.SessionID != "" {
			session.SessionID = line.SessionID
		}
		if line.StartTime != "" {
			session.StartTime = line.StartTime
		}
		if line.LastUpdated != "" {
			session.LastUpdated = line.LastUpdated
		}
		if line.Set.LastUpdated != "" {
			session.LastUpdated = line.Set.LastUpdated
		}
		if line.Type != "" {
			session.Messages = append(session.Messages, geminiMessage{
				Type: line.Type, Timestamp: line.Timestamp, Content: line.Content,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return Entry{}, err
	}
	return geminiEntry(path, session), nil
}

func geminiEntry(path string, session geminiSession) Entry {
	entry := Entry{
		ID:    session.SessionID,
		Agent: "gemini",
		Path:  path,
		Cwd:   geminiProjectRoot(path),
	}
	entry.UpdatedAt = parseHistoryString(firstNonEmpty(session.LastUpdated, session.StartTime))
	for _, message := range session.Messages {
		if message.Type != "user" && message.Type != "gemini" && message.Type != "model" {
			continue
		}
		text := geminiText(message.Content)
		if text == "" {
			continue
		}
		entry.MessageCount++
		if entry.Title == "" && message.Type == "user" {
			entry.Title = truncatePreview(text)
		}
		if entry.Preview == "" {
			entry.Preview = truncatePreview(text)
		}
	}
	if entry.Title == "" {
		entry.Title = "Gemini session"
	}
	return entry
}

func geminiText(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return strings.TrimSpace(text)
	}
	var parts []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if strings.TrimSpace(part.Text) != "" {
				out = append(out, part.Text)
			}
		}
		return strings.TrimSpace(strings.Join(out, "\n"))
	}
	return ""
}

func geminiProjectRoot(path string) string {
	root := filepath.Join(filepath.Dir(path), ".project_root")
	if raw, err := os.ReadFile(root); err == nil {
		return strings.TrimSpace(string(raw))
	}
	return ""
}
