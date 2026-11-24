package history

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

// ScanCursorIDE reads Cursor IDE composers from state.vscdb stores.
func ScanCursorIDE(home string, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	stores := cursorStateStores(home)
	byID := map[string]Entry{}
	for _, path := range stores {
		entries, err := scanCursorState(path)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if _, exists := byID[entry.ID]; !exists {
				byID[entry.ID] = entry
			}
		}
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

func cursorStateStores(home string) []string {
	roots := []string{}
	if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
		roots = append(roots, filepath.Join(appData, "Cursor", "User"))
	}
	if config := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); config != "" {
		roots = append(roots, filepath.Join(config, "Cursor", "User"))
	}
	roots = append(roots,
		filepath.Join(home, ".config", "Cursor", "User"),
		filepath.Join(home, "Library", "Application Support", "Cursor", "User"),
	)
	seen := map[string]bool{}
	out := []string{}
	for _, root := range roots {
		if root == "" || seen[root] {
			continue
		}
		seen[root] = true
		candidates := []string{filepath.Join(root, "globalStorage", "state.vscdb")}
		if matches, err := filepath.Glob(filepath.Join(root, "workspaceStorage", "*", "state.vscdb")); err == nil {
			candidates = append(candidates, matches...)
		}
		for _, path := range candidates {
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				out = append(out, path)
			}
		}
	}
	return out
}

func scanCursorState(path string) ([]Entry, error) {
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT key, value FROM cursorDiskKV WHERE key LIKE 'composerData:%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Entry{}
	for rows.Next() {
		var (
			key string
			raw []byte
		)
		if rows.Scan(&key, &raw) != nil {
			continue
		}
		var composer struct {
			ComposerID    string `json:"composerId"`
			Name          string `json:"name"`
			CreatedAt     int64  `json:"createdAt"`
			LastUpdatedAt int64  `json:"lastUpdatedAt"`
		}
		if json.Unmarshal(raw, &composer) != nil {
			continue
		}
		id := composer.ComposerID
		if id == "" {
			id = strings.TrimPrefix(key, "composerData:")
		}
		if id == "" {
			continue
		}
		entry := Entry{
			ID:        id,
			Agent:     "cursor",
			Title:     composer.Name,
			Path:      path,
			UpdatedAt: unixHistoryTime(composer.LastUpdatedAt),
		}
		if entry.UpdatedAt.IsZero() {
			entry.UpdatedAt = unixHistoryTime(composer.CreatedAt)
		}
		_ = db.QueryRow(`SELECT COUNT(*) FROM cursorDiskKV WHERE key LIKE ?`, "bubbleId:"+id+":%").Scan(&entry.MessageCount)
		entry.Preview, entry.Cwd = cursorBubblePreview(db, id)
		if entry.Title == "" {
			entry.Title = entry.Preview
		}
		if entry.Title == "" {
			entry.Title = "Cursor composer"
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

func cursorBubblePreview(db *sql.DB, composerID string) (string, string) {
	rows, err := db.Query(`SELECT value FROM cursorDiskKV WHERE key LIKE ? ORDER BY key LIMIT 12`, "bubbleId:"+composerID+":%")
	if err != nil {
		return "", ""
	}
	defer rows.Close()
	preview, cwd := "", ""
	for rows.Next() {
		var raw []byte
		if rows.Scan(&raw) != nil {
			continue
		}
		var bubble struct {
			Type                int    `json:"type"`
			Text                string `json:"text"`
			RawText             string `json:"rawText"`
			WorkspaceProjectDir string `json:"workspaceProjectDir"`
		}
		if json.Unmarshal(raw, &bubble) != nil {
			continue
		}
		if bubble.WorkspaceProjectDir != "" && cwd == "" {
			cwd = bubble.WorkspaceProjectDir
		}
		text := strings.TrimSpace(bubble.Text)
		if text == "" {
			text = strings.TrimSpace(bubble.RawText)
		}
		if preview == "" && bubble.Type == 1 && text != "" {
			preview = truncatePreview(text)
		}
	}
	return preview, cwd
}
