// Package notion implements the read-only Notion document source.
package notion

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

const notionVersion = "2022-06-28"

// Adapter implements integration.Docs.
type Adapter struct {
	Token      string
	BaseURL    string // defaults to https://api.notion.com
	HTTPClient *http.Client
}

// New builds the Notion integration.
func New(token string) *integration.Integration {
	a := &Adapter{Token: token, BaseURL: "https://api.notion.com", HTTPClient: &http.Client{Timeout: 20 * time.Second}}
	return &integration.Integration{
		Manifest: integration.Manifest{
			ID:           "notion",
			Name:         "Notion",
			Kind:         integration.KindDoc,
			Auth:         integration.AuthOAuth2,
			Capabilities: []integration.Capability{integration.CapSearch, integration.CapFetch},
			ConfigFields: []integration.ConfigField{
				{Key: "token", LabelKey: "providers.token", Type: "password", Required: true, Secret: true},
				{Key: "baseUrl", LabelKey: "providers.baseUrl", Type: "url", Placeholder: "https://api.notion.com"},
			}},
		Docs:  a,
		Probe: a,
	}
}

// Search implements integration.Docs.
func (a *Adapter) Search(ctx context.Context, cred integration.Credential, query string, limit int) ([]integration.DocHit, error) {
	if limit <= 0 || limit > 25 {
		limit = 10
	}
	body, _ := json.Marshal(map[string]any{"query": query, "page_size": limit})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURLFor(cred)+"/v1/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	a.headers(req, cred)
	resp, err := a.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("notion: unauthorized")
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("notion search: status %d", resp.StatusCode)
	}
	var out struct {
		Results []struct {
			Object     string         `json:"object"`
			ID         string         `json:"id"`
			URL        string         `json:"url"`
			Properties map[string]any `json:"properties"`
			Title      []struct {
				PlainText string `json:"plain_text"`
			} `json:"title"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	hits := make([]integration.DocHit, 0, len(out.Results))
	for _, r := range out.Results {
		title := titleOf(r.Properties)
		if title == "" && len(r.Title) > 0 {
			title = r.Title[0].PlainText
		}
		hits = append(hits, integration.DocHit{ID: r.ID, Title: title, URL: r.URL, Snippet: r.Object})
	}
	return hits, nil
}

// Fetch implements integration.Docs for a page URL or id.
func (a *Adapter) Fetch(ctx context.Context, cred integration.Credential, pageURL string) (integration.Document, error) {
	id := pageID(pageURL)
	if id == "" {
		return integration.Document{}, errors.New("notion: cannot resolve page id")
	}
	page, err := a.getPage(ctx, cred, id)
	if err != nil {
		return integration.Document{}, err
	}
	blocks, err := a.getBlocks(ctx, cred, id)
	if err != nil {
		return integration.Document{}, err
	}
	return integration.Document{
		Title:     page,
		URL:       pageURL,
		Text:      blocks,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (a *Adapter) getPage(ctx context.Context, cred integration.Credential, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURLFor(cred)+"/v1/pages/"+url.PathEscape(id), nil)
	if err != nil {
		return "", err
	}
	a.headers(req, cred)
	resp, err := a.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("notion page: status %d", resp.StatusCode)
	}
	var out struct {
		Properties map[string]any `json:"properties"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return titleOf(out.Properties), nil
}

func (a *Adapter) getBlocks(ctx context.Context, cred integration.Credential, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURLFor(cred)+"/v1/blocks/"+url.PathEscape(id)+"/children?page_size=100", nil)
	if err != nil {
		return "", err
	}
	a.headers(req, cred)
	resp, err := a.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("notion blocks: status %d", resp.StatusCode)
	}
	var out struct {
		Results []struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}

	var b strings.Builder
	for _, block := range out.Results {
		text := a.blockText(ctx, cred, block.ID, block.Type)
		if text != "" {
			b.WriteString(text)
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String()), nil
}

func (a *Adapter) blockText(ctx context.Context, cred integration.Credential, id, kind string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURLFor(cred)+"/v1/blocks/"+url.PathEscape(id), nil)
	if err != nil {
		return ""
	}
	a.headers(req, cred)
	resp, err := a.client().Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return ""
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return ""
	}
	return richText(raw[kind])
}

func (a *Adapter) headers(req *http.Request, cred integration.Credential) {
	token := cred.AccessToken
	if token == "" {
		token = a.Token
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", notionVersion)
	req.Header.Set("Content-Type", "application/json")
}

func (a *Adapter) baseURLFor(cred integration.Credential) string {
	if cred.BaseURL != "" {
		return strings.TrimRight(cred.BaseURL, "/")
	}
	return a.baseURL()
}

func (a *Adapter) baseURL() string {
	if a.BaseURL != "" {
		return strings.TrimRight(a.BaseURL, "/")
	}
	return "https://api.notion.com"
}

func (a *Adapter) client() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return http.DefaultClient
}

func titleOf(props map[string]any) string {
	for _, value := range props {
		obj, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if arr, ok := obj["title"].([]any); ok {
			return richTextArray(arr)
		}
	}
	return ""
}

func richText(node any) string {
	obj, ok := node.(map[string]any)
	if !ok {
		return ""
	}
	arr, ok := obj["rich_text"].([]any)
	if !ok {
		return ""
	}
	return richTextArray(arr)
}

func richTextArray(arr []any) string {
	var parts []string
	for _, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if text, ok := obj["plain_text"].(string); ok {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "")
}

func pageID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if u, err := url.Parse(value); err == nil && u.Host != "" {
		value = u.Path
	}
	value = strings.TrimSuffix(value, "/")
	parts := strings.Split(value, "-")
	last := parts[len(parts)-1]
	last = strings.ReplaceAll(last, "-", "")
	if len(last) >= 32 {
		return last[len(last)-32:]
	}
	return last
}
