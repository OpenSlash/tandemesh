package api

import (
	"hash/fnv"
	"strconv"
	"strings"
)

// commentAllowlist reads the optional commentUsers list from binding options.
func commentAllowlist(options map[string]any) []string {
	if options == nil {
		return nil
	}
	raw, ok := options["commentUsers"]
	if !ok {
		return nil
	}
	switch value := raw.(type) {
	case []string:
		return value
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if name, ok := item.(string); ok && strings.TrimSpace(name) != "" {
				out = append(out, name)
			}
		}
		return out
	case string:
		parts := strings.Split(value, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if name := strings.TrimSpace(part); name != "" {
				out = append(out, name)
			}
		}
		return out
	}
	return nil
}

func actorAllowed(allowed []string, actor string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, name := range allowed {
		if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(actor)) {
			return true
		}
	}
	return false
}

func shortHash(value string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(value))
	return strconv.FormatUint(h.Sum64(), 16)
}
