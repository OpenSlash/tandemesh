package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// UpdateFields implements integration.FieldWriter.
func (a *Adapter) UpdateFields(ctx context.Context, work domain.WorkRef, cred integration.Credential, update integration.FieldUpdate) error {
	base := strings.TrimRight(work.Instance, "/")
	fields := map[string]any{}

	if update.Priority != "" {
		id, err := a.lookupID(ctx, base+"/rest/api/2/priority", cred, update.Priority)
		if err != nil {
			return err
		}
		fields["priority"] = map[string]string{"id": id}
	}
	if len(update.Components) > 0 {
		project := strings.SplitN(work.ID, "-", 2)[0]
		ids := make([]map[string]string, 0, len(update.Components))
		for _, name := range update.Components {
			id, err := a.lookupID(ctx, base+"/rest/api/2/project/"+url.PathEscape(project)+"/components", cred, name)
			if err != nil {
				return err
			}
			ids = append(ids, map[string]string{"id": id})
		}
		fields["components"] = ids
	}
	if len(update.FixVersions) > 0 {
		project := strings.SplitN(work.ID, "-", 2)[0]
		ids := make([]map[string]string, 0, len(update.FixVersions))
		for _, name := range update.FixVersions {
			id, err := a.lookupID(ctx, base+"/rest/api/2/project/"+url.PathEscape(project)+"/versions", cred, name)
			if err != nil {
				return err
			}
			ids = append(ids, map[string]string{"id": id})
		}
		fields["fixVersions"] = ids
	}
	for id, value := range update.CustomFields {
		fields[id] = coerceFieldValue(value)
	}
	if len(fields) == 0 {
		return nil
	}
	body, _ := json.Marshal(map[string]any{"fields": fields})
	resp, err := a.do(ctx, http.MethodPut, base+"/rest/api/2/issue/"+url.PathEscape(work.ID), cred, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("jira fields: status %d", resp.StatusCode)
	}
	return nil
}

func coerceFieldValue(value string) any {
	if number, err := strconv.ParseFloat(value, 64); err == nil {
		return number
	}
	if boolean, err := strconv.ParseBool(value); err == nil {
		return boolean
	}
	return value
}

func (a *Adapter) lookupID(ctx context.Context, endpoint string, cred integration.Credential, name string) (string, error) {
	resp, err := a.do(ctx, http.MethodGet, endpoint, cred, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("jira field lookup: status %d", resp.StatusCode)
	}
	var rows []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return "", err
	}
	for _, row := range rows {
		if strings.EqualFold(row.Name, name) {
			return row.ID, nil
		}
	}
	return "", fmt.Errorf("jira field %q not found", name)
}
