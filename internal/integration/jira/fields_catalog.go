package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// ListFieldOptions returns Jira priority, component and fix version names.
func (a *Adapter) ListFieldOptions(ctx context.Context, cred integration.Credential, instance, project string) (map[string][]string, error) {
	base := strings.TrimRight(instance, "/")
	if base == "" {
		base = strings.TrimRight(cred.BaseURL, "/")
	}
	if base == "" {
		return nil, fmt.Errorf("jira: site URL is required")
	}
	options := map[string][]string{}
	priority, err := a.listNames(ctx, base+"/rest/api/2/priority", cred)
	if err != nil {
		return nil, err
	}
	options["priority"] = priority
	components, err := a.listNames(ctx, base+"/rest/api/2/project/"+url.PathEscape(project)+"/components", cred)
	if err != nil {
		return nil, err
	}
	options["components"] = components
	versions, err := a.listNames(ctx, base+"/rest/api/2/project/"+url.PathEscape(project)+"/versions", cred)
	if err != nil {
		return nil, err
	}
	options["fixVersions"] = versions
	return options, nil
}

func (a *Adapter) listNames(ctx context.Context, endpoint string, cred integration.Credential) ([]string, error) {
	resp, err := a.do(ctx, http.MethodGet, endpoint, cred, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jira fields: status %d", resp.StatusCode)
	}
	var rows []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Name != "" {
			out = append(out, row.Name)
		}
	}
	return out, nil
}
