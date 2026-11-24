package history

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// ScanOpenCode reads sessions from the local OpenCode SQLite store.
func ScanOpenCode(home string, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	path := opencodeDBPath(home)
	if path == "" {
		return []Entry{}, nil
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return []Entry{}, nil
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	tables := map[string]bool{}
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table'`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			tables[name] = true
		}
	}
	rows.Close()

	sessionTable := ""
	switch {
	case tables["session_v2"]:
		sessionTable = "session_v2"
	case tables["session"]:
		sessionTable = "session"
	default:
		return []Entry{}, nil
	}
	columns := tableColumns(db, sessionTable)
	titleCol := firstColumn(columns, "title", "slug", "id")
	dirCol := firstColumn(columns, "directory", "path", "workspace_id")
	timeCol := firstColumn(columns, "time_updated", "time_created", "updated_at")
	if timeCol == "" {
		return []Entry{}, nil
	}
	messageTable := "message"
	if tables["session_message"] {
		messageTable = "session_message"
	}

	query := fmt.Sprintf("SELECT id, %s, %s, %s FROM %s ORDER BY %s DESC LIMIT ?",
		titleCol, dirCol, timeCol, sessionTable, timeCol)
	rows, err = db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Entry{}
	for rows.Next() {
		var (
			id, title, dir string
			rawTime        any
		)
		if err := rows.Scan(&id, &title, &dir, &rawTime); err != nil || id == "" {
			continue
		}
		entry := Entry{
			ID:        id,
			Agent:     "opencode",
			Title:     firstNonEmpty(title, "OpenCode session"),
			Cwd:       dir,
			Path:      path,
			UpdatedAt: parseHistoryTime(rawTime),
		}
		_ = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE session_id = ?", messageTable), id).Scan(&entry.MessageCount)
		entry.Preview = openCodePreview(db, messageTable, id)
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}

func opencodeDBPath(home string) string {
	if override := strings.TrimSpace(os.Getenv("OPENCODE_DB")); override != "" {
		if filepath.IsAbs(override) {
			return override
		}
	}
	if data := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); data != "" {
		candidate := filepath.Join(data, "opencode", "opencode.db")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	candidate := filepath.Join(home, ".local", "share", "opencode", "opencode.db")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return candidate
}

func tableColumns(db *sql.DB, table string) map[string]bool {
	out := map[string]bool{}
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dflt    any
			pk      int
		)
		if rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk) == nil {
			out[name] = true
		}
	}
	return out
}

func firstColumn(columns map[string]bool, names ...string) string {
	for _, name := range names {
		if columns[name] {
			return name
		}
	}
	return "id"
}

func openCodePreview(db *sql.DB, table, sessionID string) string {
	query := fmt.Sprintf("SELECT data FROM %s WHERE session_id = ?", table)
	if table == "session_message" {
		query += " AND type = 'user' ORDER BY seq LIMIT 1"
	} else {
		query += " ORDER BY time_created LIMIT 5"
	}
	rows, err := db.Query(query, sessionID)
	if err != nil {
		return ""
	}
	defer rows.Close()
	for rows.Next() {
		var data string
		if rows.Scan(&data) != nil {
			continue
		}
		var payload struct {
			Role    string `json:"role"`
			Text    string `json:"text"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if json.Unmarshal([]byte(data), &payload) != nil {
			continue
		}
		if payload.Role != "" && payload.Role != "user" {
			continue
		}
		if text := strings.TrimSpace(payload.Text); text != "" {
			return truncatePreview(text)
		}
		for _, block := range payload.Content {
			if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
				return truncatePreview(block.Text)
			}
		}
	}
	return ""
}

func parseHistoryTime(raw any) time.Time {
	switch value := raw.(type) {
	case int64:
		return unixHistoryTime(value)
	case float64:
		return unixHistoryTime(int64(value))
	case []byte:
		return parseHistoryString(string(value))
	case string:
		return parseHistoryString(value)
	default:
		return time.Time{}
	}
}

func parseHistoryString(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return unixHistoryTime(n)
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t
	}
	return time.Time{}
}

func unixHistoryTime(value int64) time.Time {
	if value > 1_000_000_000_000 {
		return time.UnixMilli(value).UTC()
	}
	if value > 1_000_000_000 {
		return time.Unix(value, 0).UTC()
	}
	return time.Time{}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func truncatePreview(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 240 {
		return value[:240]
	}
	return value
}
