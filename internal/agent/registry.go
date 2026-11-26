package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const ACPRegistryURL = "https://cdn.agentclientprotocol.com/registry/v1/latest/registry.json"

type ACPRegistryEntry struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Version      string         `json:"version"`
	Description  string         `json:"description,omitempty"`
	Repository   string         `json:"repository,omitempty"`
	Website      string         `json:"website,omitempty"`
	License      string         `json:"license,omitempty"`
	Icon         string         `json:"icon,omitempty"`
	Distribution map[string]any `json:"distribution,omitempty"`
}

type ACPRegistrySnapshot struct {
	Source    string             `json:"source"`
	FetchedAt time.Time          `json:"fetchedAt"`
	Items     []ACPRegistryEntry `json:"items"`
}

var registryCache struct {
	sync.Mutex
	snapshot *ACPRegistrySnapshot
}

// FetchACPRegistry reads the public ACP Registry. The result is cached for one
// hour; this is a read-only catalog and never installs an agent.
func FetchACPRegistry(ctx context.Context) (*ACPRegistrySnapshot, error) {
	registryCache.Lock()
	defer registryCache.Unlock()
	if registryCache.snapshot != nil && time.Since(registryCache.snapshot.FetchedAt) < time.Hour {
		return registryCache.snapshot, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ACPRegistryURL, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var payload struct {
		Agents []ACPRegistryEntry `json:"agents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	snapshot := &ACPRegistrySnapshot{Source: ACPRegistryURL, FetchedAt: time.Now().UTC(), Items: payload.Agents}
	registryCache.snapshot = snapshot
	return snapshot, nil
}

// maxIconBytes bounds a registry icon; the registry ships small SVGs.
const maxIconBytes = 64 << 10

// ErrIconNotFound means the registry lists no icon for the agent.
var ErrIconNotFound = errors.New("registry agent has no icon")

// unsafeSVG matches SVG features that could run script or load content when
// the file is served from the plane's own origin.
// A plain public DOCTYPE is harmless; entity declarations are not.
var unsafeSVG = regexp.MustCompile(`(?i)<script|<foreignobject|<!entity|javascript:|\son[a-z]+\s*=|href\s*=\s*["']?\s*(https?:|//|data:)`)

// ACPIcon returns a registry agent's SVG icon. Icons are cached under
// cacheDir on first use, so the UI keeps them offline; registry icons are
// versioned with the agent and rarely change, so the cache never expires.
func ACPIcon(ctx context.Context, id, cacheDir string) ([]byte, error) {
	path := filepath.Join(cacheDir, id+".svg")
	if raw, err := os.ReadFile(path); err == nil {
		return raw, nil
	}
	snapshot, err := FetchACPRegistry(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch registry: %w", err)
	}
	iconURL := ""
	for _, item := range snapshot.Items {
		if item.ID == id {
			iconURL = item.Icon
			break
		}
	}
	if iconURL == "" {
		return nil, ErrIconNotFound
	}
	if !strings.HasPrefix(iconURL, "https://") {
		return nil, fmt.Errorf("registry icon must use https")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, iconURL, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download icon: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download icon: %s", resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxIconBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read icon: %w", err)
	}
	if len(raw) > maxIconBytes {
		return nil, fmt.Errorf("registry icon exceeds %d bytes", maxIconBytes)
	}
	if !SafeSVG(raw) {
		return nil, fmt.Errorf("registry icon %q is not a plain SVG", id)
	}
	// A failed cache write only costs a refetch next time.
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		tmp := path + ".tmp"
		if os.WriteFile(tmp, raw, 0o644) == nil {
			_ = os.Rename(tmp, path)
		}
	}
	return raw, nil
}

// SafeSVG reports whether raw is an SVG without script, event handlers or
// external references.
func SafeSVG(raw []byte) bool {
	return strings.Contains(strings.ToLower(string(raw)), "<svg") && !unsafeSVG.Match(raw)
}
