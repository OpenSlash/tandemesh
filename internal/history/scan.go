package history

import "sort"

// Scan discovers all supported agent CLI sessions on this host.
func Scan(home string, limit int) ([]Entry, error) {
	claude, err := ScanClaude(home, limit)
	if err != nil {
		return nil, err
	}
	codex, err := ScanCodex(home, limit)
	if err != nil {
		return nil, err
	}
	opencode, err := ScanOpenCode(home, limit)
	if err != nil {
		return nil, err
	}
	gemini, err := ScanGemini(home, limit)
	if err != nil {
		return nil, err
	}
	cursor, err := ScanCursor(home, limit)
	if err != nil {
		return nil, err
	}
	cursorIDE, err := ScanCursorIDE(home, limit)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(claude)+len(codex)+len(opencode)+len(gemini)+len(cursor)+len(cursorIDE))
	out := make([]Entry, 0, len(claude)+len(codex)+len(opencode)+len(gemini)+len(cursor)+len(cursorIDE))
	for _, entry := range append(append(append(append(append(claude, codex...), opencode...), gemini...), cursor...), cursorIDE...) {
		key := entry.Agent + ":" + entry.ID
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
