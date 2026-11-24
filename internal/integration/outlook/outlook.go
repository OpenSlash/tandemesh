// Package outlook implements the Microsoft Graph Outlook context source.
package outlook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Adapter implements integration.Docs over Microsoft Graph.
type Adapter struct {
	Token      string
	BaseURL    string // defaults to https://graph.microsoft.com
	HTTPClient *http.Client
}

// New builds the Outlook integration.
func New(token string) *integration.Integration {
	a := &Adapter{Token: token, BaseURL: "https://graph.microsoft.com", HTTPClient: &http.Client{Timeout: 20 * time.Second}}
	return &integration.Integration{
		Manifest: integration.Manifest{
			ID:           "outlook",
			Name:         "Outlook",
			Kind:         integration.KindEmail,
			Auth:         integration.AuthOAuth2,
			Capabilities: []integration.Capability{integration.CapSearch, integration.CapFetch},
			ConfigFields: []integration.ConfigField{
				{Key: "token", LabelKey: "providers.token", Type: "password", Required: true, Secret: true},
			}},
		Docs:  a,
		Probe: a,
	}
}

// Search implements integration.Docs.
func (a *Adapter) Search(ctx context.Context, cred integration.Credential, query string, limit int) ([]integration.DocHit, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("outlook: empty query")
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	params := url.Values{}
	params.Set("$search", `"`+query+`"`)
	params.Set("$top", fmt.Sprint(limit))
	params.Set("$select", "id,subject,webLink,bodyPreview,lastModifiedDateTime")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL()+"/v1.0/me/messages?"+params.Encode(), nil)
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
		return nil, errors.New("outlook: unauthorized")
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("outlook search: status %d", resp.StatusCode)
	}
	var out struct {
		Value []struct {
			ID          string `json:"id"`
			Subject     string `json:"subject"`
			WebLink     string `json:"webLink"`
			BodyPreview string `json:"bodyPreview"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	hits := make([]integration.DocHit, 0, len(out.Value))
	for _, m := range out.Value {
		hits = append(hits, integration.DocHit{
			ID:      m.ID,
			Title:   m.Subject,
			URL:     m.WebLink,
			Snippet: m.BodyPreview,
		})
	}
	return hits, nil
}

// Fetch implements integration.Docs by message id or Outlook web link.
func (a *Adapter) Fetch(ctx context.Context, cred integration.Credential, ref string) (integration.Document, error) {
	id := messageID(ref)
	if id == "" {
		return integration.Document{}, errors.New("outlook: cannot resolve message id")
	}
	params := url.Values{}
	params.Set("$select", "subject,body,webLink,lastModifiedDateTime,receivedDateTime")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL()+"/v1.0/me/messages/"+url.PathEscape(id)+"?"+params.Encode(), nil)
	if err != nil {
		return integration.Document{}, err
	}
	a.headers(req, cred)
	resp, err := a.client().Do(req)
	if err != nil {
		return integration.Document{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return integration.Document{}, fmt.Errorf("outlook message: status %d", resp.StatusCode)
	}
	var out struct {
		Subject string `json:"subject"`
		WebLink string `json:"webLink"`
		Body    struct {
			ContentType string `json:"contentType"`
			Content     string `json:"content"`
		} `json:"body"`
		LastModified string `json:"lastModifiedDateTime"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return integration.Document{}, err
	}
	text := stripHTML(out.Body.Content)
	return integration.Document{
		Title:     out.Subject,
		URL:       out.WebLink,
		Text:      text,
		UpdatedAt: out.LastModified,
	}, nil
}

var tagPattern = regexp.MustCompile(`(?s)<[^>]*>`)

func stripHTML(raw string) string {
	text := tagPattern.ReplaceAllString(raw, " ")
	text = html.UnescapeString(text)
	return strings.Join(strings.Fields(text), " ")
}

// messageID accepts a Graph id or a ref of the form outlook:<id>.
func messageID(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "outlook:")
	if ref == "" {
		return ""
	}
	// A raw Graph id is a long URL-safe base64 string; a web link is not.
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ""
	}
	return ref
}

func (a *Adapter) headers(req *http.Request, cred integration.Credential) {
	token := cred.AccessToken
	if token == "" {
		token = a.Token
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
}

func (a *Adapter) baseURL() string {
	if a.BaseURL != "" {
		return strings.TrimRight(a.BaseURL, "/")
	}
	return "https://graph.microsoft.com"
}

func (a *Adapter) client() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return http.DefaultClient
}
