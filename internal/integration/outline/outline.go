// Package outline implements the Outline wiki document source.
package outline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Adapter implements integration.Docs over the Outline API.
type Adapter struct {
	Token      string
	BaseURL    string // e.g. https://app.getoutline.com or a self-hosted instance
	HTTPClient *http.Client
}

// New builds the Outline integration.
func New(baseURL, token string) *integration.Integration {
	if baseURL == "" {
		baseURL = "https://app.getoutline.com"
	}
	a := &Adapter{Token: token, BaseURL: baseURL, HTTPClient: &http.Client{Timeout: 20 * time.Second}}
	return &integration.Integration{
		Manifest: integration.Manifest{
			ID:           "outline",
			Name:         "Outline",
			Kind:         integration.KindDoc,
			Auth:         integration.AuthToken,
			Capabilities: []integration.Capability{integration.CapSearch, integration.CapFetch},
			ConfigFields: []integration.ConfigField{
				{Key: "token", LabelKey: "providers.token", Type: "password", Required: true, Secret: true},
				{Key: "baseUrl", LabelKey: "providers.baseUrl", Type: "url", Required: true, Placeholder: "https://app.getoutline.com"},
			}},
		Docs:  a,
		Probe: a,
	}
}

// Search implements integration.Docs.
func (a *Adapter) Search(ctx context.Context, cred integration.Credential, query string, limit int) ([]integration.DocHit, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("outline: empty query")
	}
	if limit <= 0 || limit > 25 {
		limit = 10
	}
	var out struct {
		Data []struct {
			Document struct {
				ID    string `json:"id"`
				Title string `json:"title"`
				URL   string `json:"url"`
			} `json:"document"`
			Context string `json:"context"`
		} `json:"data"`
	}
	if err := a.post(ctx, cred, "/api/documents.search", map[string]any{"query": query, "limit": limit}, &out); err != nil {
		return nil, err
	}
	hits := make([]integration.DocHit, 0, len(out.Data))
	for _, item := range out.Data {
		hits = append(hits, integration.DocHit{
			ID:      item.Document.ID,
			Title:   item.Document.Title,
			URL:     a.absolute(item.Document.URL),
			Snippet: item.Context,
		})
	}
	return hits, nil
}

// Fetch implements integration.Docs by document id or URL.
func (a *Adapter) Fetch(ctx context.Context, cred integration.Credential, ref string) (integration.Document, error) {
	id := documentID(ref)
	if id == "" {
		return integration.Document{}, errors.New("outline: cannot resolve document id")
	}
	var out struct {
		Data struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			Text      string `json:"text"`
			URL       string `json:"url"`
			UpdatedAt string `json:"updatedAt"`
		} `json:"data"`
	}
	if err := a.post(ctx, cred, "/api/documents.info", map[string]any{"id": id}, &out); err != nil {
		return integration.Document{}, err
	}
	return integration.Document{
		Title:     out.Data.Title,
		URL:       a.absolute(out.Data.URL),
		Text:      strings.TrimSpace(out.Data.Text),
		UpdatedAt: out.Data.UpdatedAt,
	}, nil
}

func (a *Adapter) post(ctx context.Context, cred integration.Credential, path string, body any, out any) error {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL()+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	a.headers(req, cred)
	resp, err := a.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("outline: unauthorized")
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("outline: status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (a *Adapter) headers(req *http.Request, cred integration.Credential) {
	token := cred.AccessToken
	if token == "" {
		token = a.Token
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

func (a *Adapter) baseURL() string {
	if a.BaseURL != "" {
		return strings.TrimRight(a.BaseURL, "/")
	}
	return "https://app.getoutline.com"
}

func (a *Adapter) client() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return http.DefaultClient
}

func (a *Adapter) absolute(link string) string {
	if link == "" || strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return link
	}
	return a.baseURL() + "/" + strings.TrimLeft(link, "/")
}

// documentID accepts a raw id or an Outline document URL.
func documentID(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "outline:")
	if ref == "" {
		return ""
	}
	if u, err := url.Parse(ref); err == nil && u.Host != "" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 && parts[0] == "doc" {
			slug := parts[1]
			if idx := strings.LastIndex(slug, "-"); idx >= 0 && idx+1 < len(slug) {
				return slug[idx+1:]
			}
			return slug
		}
		return ""
	}
	return ref
}
