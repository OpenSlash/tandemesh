package plane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

type catalogRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Identifier string `json:"identifier"`
}

type catalogPage struct {
	Results []catalogRow `json:"results"`
}

func decodeCatalog(raw []byte) ([]catalogRow, error) {
	var rows []catalogRow
	if json.Unmarshal(raw, &rows) == nil {
		return rows, nil
	}
	var page catalogPage
	if err := json.Unmarshal(raw, &page); err != nil {
		return nil, err
	}
	return page.Results, nil
}

// ListWorkspaces implements integration.CatalogReader.
func (a *Adapter) ListWorkspaces(ctx context.Context, cred integration.Credential) ([]integration.CatalogItem, error) {
	raw, status, err := a.do(ctx, "GET", a.apiAt(cred, "/workspaces/"), cred, nil)
	if err != nil {
		if status == 404 {
			return []integration.CatalogItem{}, nil
		}
		return nil, err
	}
	rows, err := decodeCatalog(raw)
	if err != nil {
		return nil, err
	}
	out := make([]integration.CatalogItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, integration.CatalogItem{ID: row.Slug, Name: row.Name, Slug: row.Slug})
	}
	return out, nil
}

// ListProjects implements integration.CatalogReader.
func (a *Adapter) ListProjects(ctx context.Context, cred integration.Credential, workspace string) ([]integration.CatalogItem, error) {
	if workspace == "" {
		return nil, fmt.Errorf("plane: workspace is required")
	}
	path := fmt.Sprintf("/workspaces/%s/projects/", url.PathEscape(workspace))
	raw, _, err := a.do(ctx, "GET", a.apiAt(cred, path), cred, nil)
	if err != nil {
		return nil, err
	}
	rows, err := decodeCatalog(raw)
	if err != nil {
		return nil, err
	}
	out := make([]integration.CatalogItem, 0, len(rows))
	for _, row := range rows {
		name := row.Name
		if row.Identifier != "" {
			name = row.Identifier + " · " + row.Name
		}
		out = append(out, integration.CatalogItem{ID: row.ID, Name: name, Slug: row.Identifier})
	}
	return out, nil
}
