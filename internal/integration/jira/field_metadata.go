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

// ListFieldMetadata implements integration.FieldMetadataReader.
func (a *Adapter) ListFieldMetadata(ctx context.Context, cred integration.Credential, instance, issue string) ([]integration.FieldMetadata, error) {
	base := strings.TrimRight(instance, "/")
	resp, err := a.do(ctx, http.MethodGet, base+"/rest/api/2/issue/"+url.PathEscape(issue)+"/editmeta", cred, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jira editmeta: status %d", resp.StatusCode)
	}
	var payload struct {
		Fields map[string]struct {
			Name     string `json:"name"`
			Required bool   `json:"required"`
			Schema   struct {
				Type string `json:"type"`
			} `json:"schema"`
			AllowedValues []struct {
				Value string `json:"value"`
				Name  string `json:"name"`
			} `json:"allowedValues"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]integration.FieldMetadata, 0, len(payload.Fields))
	for id, field := range payload.Fields {
		meta := integration.FieldMetadata{ID: id, Name: field.Name, Type: field.Schema.Type, Required: field.Required}
		for _, allowed := range field.AllowedValues {
			if allowed.Value != "" {
				meta.Allowed = append(meta.Allowed, allowed.Value)
			} else if allowed.Name != "" {
				meta.Allowed = append(meta.Allowed, allowed.Name)
			}
		}
		out = append(out, meta)
	}
	return out, nil
}
