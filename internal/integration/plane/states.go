package plane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// ListStates returns the project's workflow states.
func (a *Adapter) ListStates(ctx context.Context, cred integration.Credential, instance string) ([]integration.State, error) {
	slug, project, err := splitInstance(instance)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/workspaces/%s/projects/%s/states/", url.PathEscape(slug), url.PathEscape(project))
	raw, _, err := a.do(ctx, "GET", a.apiAt(cred, path), cred, nil)
	if err != nil {
		return nil, err
	}
	var list []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Group string `json:"group"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		var page struct {
			Results []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Group string `json:"group"`
			} `json:"results"`
		}
		if err := json.Unmarshal(raw, &page); err != nil {
			return nil, fmt.Errorf("plane: decode states: %w", err)
		}
		list = page.Results
	}
	out := make([]integration.State, 0, len(list))
	for _, item := range list {
		out = append(out, integration.State{ID: item.ID, Name: item.Name, Group: item.Group})
	}
	return out, nil
}
