package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// ListStates returns Jira statuses available on the site.
func (a *Adapter) ListStates(ctx context.Context, cred integration.Credential, instance string) ([]integration.State, error) {
	base := strings.TrimRight(instance, "/")
	if base == "" {
		base = strings.TrimRight(cred.BaseURL, "/")
	}
	if base == "" {
		return nil, fmt.Errorf("jira: site URL is required")
	}
	resp, err := a.do(ctx, http.MethodGet, base+"/rest/api/2/status", cred, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jira states: status %d", resp.StatusCode)
	}
	var rows []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status struct {
			Key string `json:"key"`
		} `json:"statusCategory"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]integration.State, 0, len(rows))
	for _, row := range rows {
		out = append(out, integration.State{ID: row.ID, Name: row.Name, Group: row.Status.Key})
	}
	return out, nil
}
