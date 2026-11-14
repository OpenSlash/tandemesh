// Package history discovers agent CLI sessions already stored on this host.
package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry is one agent CLI conversation found on disk.
type Entry struct {
	ID           string    `json:"id"`
	Agent        string    `json:"agent"`
	Title        string    `json:"title"`
	Cwd          string    `json:"cwd"`
	Branch       string    `json:"branch,omitempty"`
	Model        string    `json:"model,omitempty"`
	Preview      string    `json:"preview,omitempty"`
	Path         string    `json:"path,omitempty"`
	MessageCount int       `json:"messageCount"`
	UpdatedAt    time.Time `json:"updatedAt"`
	MachineID    string    `json:"machineId,omitempty"`
	MachineName  string    `json:"machineName,omitempty"`
}

type claudeLine struct {
	Type      string    `json:"type"`
	SessionID string    `json:"sessionId"`
	Cwd       string    `json:"cwd"`
	GitBranch string    `json:"gitBranch"`
	Slug      string    `json:"slug"`
	Timestamp time.Time `json:"timestamp"`
	Sidechain bool      `json:"isSidechain"`
	Message   struct {
		Role    string `json:"role"`
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
}

// ScanClaude reads Claude Code sessions from <home>/.claude/projects.
func ScanClaude(home string, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	root := filepath.Join(home, ".claude", "projects")
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0)
	for _, project := range entries {
		if !project.IsDir() {
			continue
		}
		dir := filepath.Join(root, project.Name())
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".jsonl") {
				continue
			}
			entry, err := parseClaudeFile(filepath.Join(dir, file.Name()))
			if err != nil || entry.ID == "" {
				continue
			}
			out = append(out, entry)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func parseClaudeFile(path string) (Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return Entry{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Entry{}, err
	}
	entry := Entry{Agent: "claude", Path: path, UpdatedAt: info.ModTime()}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var line claudeLine
		if json.Unmarshal(scanner.Bytes(), &line) != nil {
			continue
		}
		if line.Sidechain {
			continue
		}
		if entry.ID == "" && line.SessionID != "" {
			entry.ID = line.SessionID
		}
		if line.Cwd != "" {
			entry.Cwd = line.Cwd
		}
		if line.GitBranch != "" {
			entry.Branch = line.GitBranch
		}
		if line.Slug != "" {
			entry.Title = line.Slug
		}
		if line.Message.Model != "" {
			entry.Model = line.Message.Model
		}
		if line.Timestamp.After(entry.UpdatedAt) {
			entry.UpdatedAt = line.Timestamp
		}
		text := claudeText(line)
		if line.Type != "user" || text == "" {
			continue
		}
		entry.MessageCount++
		if entry.Title == "" {
			entry.Title = preview(text, 80)
		}
		entry.Preview = preview(text, 300)
	}
	if err := scanner.Err(); err != nil {
		return Entry{}, err
	}
	if entry.ID == "" {
		entry.ID = strings.TrimSuffix(filepath.Base(path), ".jsonl")
	}
	if entry.Title == "" {
		entry.Title = fmt.Sprintf("Claude session %s", short(entry.ID))
	}
	return entry, nil
}

func claudeText(line claudeLine) string {
	for _, block := range line.Message.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			return strings.TrimSpace(block.Text)
		}
	}
	return ""
}

func preview(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "…"
}

func short(id string) string {
	if len(id) > 8 {
		return id[len(id)-8:]
	}
	return id
}
