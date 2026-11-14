// Package plane implements the Plane task-source integration (cloud and self-hosted).
package plane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Adapter implements the Plane reader, writer and probe slots.
type Adapter struct {
	Secret     string
	HTTPClient *http.Client
	BaseURL    string // API base, e.g. https://api.plane.so
	AppURL     string // browser base, e.g. https://app.plane.so
}

// New builds the Plane integration.
func New() *integration.Integration {
	base := os.Getenv("ORCA_PLANE_API_URL")
	if base == "" {
		base = "https://api.plane.so"
	}
	app := os.Getenv("ORCA_PLANE_APP_URL")
	if app == "" {
		app = "https://app.plane.so"
	}
	a := &Adapter{
		Secret:     os.Getenv("ORCA_PLANE_WEBHOOK_SECRET"),
		HTTPClient: &http.Client{Timeout: 20 * time.Second},
		BaseURL:    strings.TrimRight(base, "/"),
		AppURL:     strings.TrimRight(app, "/"),
	}
	return &integration.Integration{
		Manifest: integration.Manifest{
			ID:           "plane",
			Name:         "Plane",
			Kind:         integration.KindTask,
			Auth:         integration.AuthToken,
			Capabilities: []integration.Capability{integration.CapWebhook, integration.CapComment, integration.CapTransition},
			ConfigFields: []integration.ConfigField{
				{Key: "token", LabelKey: "providers.token", Type: "password", Required: true, Secret: true},
				{Key: "baseUrl", LabelKey: "providers.baseUrl", Type: "url", Required: true, Placeholder: "https://plane.example.com"},
				{Key: "instance", LabelKey: "providers.workspaceProject", Type: "text", Required: true, Placeholder: "workspace/project-uuid"},
				{Key: "webhookSecret", LabelKey: "providers.webhookSecret", Type: "password", Secret: true},
			},
		},
		Work:      a,
		Webhook:   a,
		Registrar: a,
		Reader:    a,
		Probe:     a,
		States:    a,
	}
}

func (a *Adapter) client() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return http.DefaultClient
}

func (a *Adapter) api(path string) string {
	return a.BaseURL + "/api/v1" + path
}

func (a *Adapter) apiAt(cred integration.Credential, path string) string {
	base := a.BaseURL
	if cred.BaseURL != "" {
		base = strings.TrimRight(cred.BaseURL, "/")
	}
	return base + "/api/v1" + path
}

// splitInstance accepts "workspace/project" (project may be a UUID or identifier).
func splitInstance(instance string) (string, string, error) {
	parts := strings.SplitN(strings.Trim(instance, "/"), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("plane: instance must be workspace/project")
	}
	return parts[0], parts[1], nil
}

func (a *Adapter) do(ctx context.Context, method, url string, cred integration.Credential, body any) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, 0, err
	}
	if cred.AccessToken != "" {
		req.Header.Set("X-API-Key", cred.AccessToken)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.client().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if resp.StatusCode >= 300 {
		return raw, resp.StatusCode, fmt.Errorf("plane: status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return raw, resp.StatusCode, nil
}

// issue is the subset of Plane work-item fields Orca projects.
type issue struct {
	ID              string          `json:"id"`
	SequenceID      int             `json:"sequence_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description_stripped"`
	DescriptionHTML string          `json:"description_html"`
	Project         string          `json:"project"`
	Workspace       string          `json:"workspace"`
	Permalink       string          `json:"permalink"`
	UpdatedAt       string          `json:"updated_at"`
	CreatedAt       string          `json:"created_at"`
	CreatedBy       string          `json:"created_by"`
	Labels          []any           `json:"labels"`
	LabelDetails    []label         `json:"label_details"`
	Assignees       []any           `json:"assignees"`
	AssigneeDetails []assignee      `json:"assignee_details"`
	State           json.RawMessage `json:"state"`
}

type label struct {
	Name string `json:"name"`
}

type assignee struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

func labelNames(row issue) []string {
	out := []string{}
	for _, l := range row.LabelDetails {
		if l.Name != "" {
			out = append(out, l.Name)
		}
	}
	if len(out) == 0 {
		for _, raw := range row.Labels {
			switch v := raw.(type) {
			case string:
				out = append(out, v)
			case map[string]any:
				if name, ok := v["name"].(string); ok {
					out = append(out, name)
				}
			}
		}
	}
	return out
}

func assigneeName(row issue) string {
	for _, a := range row.AssigneeDetails {
		if a.DisplayName != "" {
			return a.DisplayName
		}
		if a.Email != "" {
			return a.Email
		}
	}
	for _, raw := range row.Assignees {
		if m, ok := raw.(map[string]any); ok {
			if name, ok := m["display_name"].(string); ok {
				return name
			}
		}
	}
	return ""
}

func (a *Adapter) issueURL(slug, project string, row issue) string {
	if row.Permalink != "" {
		return row.Permalink
	}
	if a.AppURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/projects/%s/issues/%s", a.AppURL, slug, project, row.ID)
}
