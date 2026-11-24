package agent

import (
	"context"
	"encoding/json"
	"net/http"
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
