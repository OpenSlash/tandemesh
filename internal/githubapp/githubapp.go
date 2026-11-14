// Package githubapp implements the GitHub App JWT and installation-token
// lifecycle used by the GitHub App connection mode.
package githubapp

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client calls the GitHub App API.
type Client struct {
	AppID      string
	Slug       string
	PrivateKey string
	BaseURL    string
	HTTPClient *http.Client

	mu     sync.Mutex
	tokens map[int64]cachedToken
}

type cachedToken struct {
	Token     string
	ExpiresAt time.Time
}

// Installation is one GitHub App installation.
type Installation struct {
	ID            int64  `json:"id"`
	AccountLogin  string `json:"accountLogin"`
	AccountType   string `json:"accountType"`
	RepositorySel string `json:"repositorySelection"`
	HTMLURL       string `json:"htmlUrl"`
}

// Repository is one repository available to an installation.
type Repository struct {
	FullName string `json:"fullName"`
	Private  bool   `json:"private"`
	HTMLURL  string `json:"htmlUrl"`
}

// New builds a GitHub App client. An empty private key or app id disables it.
func New(appID, slug, privateKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	return &Client{
		AppID: appID, Slug: slug, PrivateKey: strings.ReplaceAll(privateKey, `\n`, "\n"),
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 20 * time.Second},
		tokens:     map[int64]cachedToken{},
	}
}

// Enabled reports whether the app credentials are complete.
func (c *Client) Enabled() bool {
	return c != nil && c.AppID != "" && c.PrivateKey != "" && c.Slug != ""
}

// InstallURL is the browser entry point for installing the app.
func (c *Client) InstallURL(state string) string {
	return "https://github.com/apps/" + url.PathEscape(c.Slug) + "/installations/new?" + url.Values{"state": {state}}.Encode()
}

func (c *Client) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) jwt() (string, error) {
	block, _ := pem.Decode([]byte(c.PrivateKey))
	if block == nil {
		return "", fmt.Errorf("github app: invalid private key PEM")
	}
	var key *rsa.PrivateKey
	if parsed, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		key = parsed
	} else {
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("github app: parse private key: %w", err)
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("github app: private key is not RSA")
		}
		key = rsaKey
	}
	now := time.Now().Unix()
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	payload, _ := json.Marshal(map[string]any{"iat": now - 60, "exp": now + 540, "iss": c.AppID})
	encoded := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(encoded))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return encoded + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func (c *Client) request(ctx context.Context, method, path string, token string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("github app: status %d %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// Installations lists installations visible to the app.
func (c *Client) Installations(ctx context.Context) ([]Installation, error) {
	token, err := c.jwt()
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID      int64 `json:"id"`
		Account struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"account"`
		RepositorySelection string `json:"repository_selection"`
		HTMLURL             string `json:"html_url"`
	}
	if err := c.request(ctx, http.MethodGet, "/app/installations", token, nil, &rows); err != nil {
		return nil, err
	}
	out := make([]Installation, 0, len(rows))
	for _, row := range rows {
		out = append(out, Installation{
			ID: row.ID, AccountLogin: row.Account.Login, AccountType: row.Account.Type,
			RepositorySel: row.RepositorySelection, HTMLURL: row.HTMLURL,
		})
	}
	return out, nil
}

// InstallationToken returns a cached installation token, minting one if needed.
func (c *Client) InstallationToken(ctx context.Context, installationID int64) (string, time.Time, error) {
	c.mu.Lock()
	cached, ok := c.tokens[installationID]
	c.mu.Unlock()
	if ok && cached.Token != "" && time.Until(cached.ExpiresAt) > time.Minute {
		return cached.Token, cached.ExpiresAt, nil
	}
	token, err := c.jwt()
	if err != nil {
		return "", time.Time{}, err
	}
	var out struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := c.request(ctx, http.MethodPost, fmt.Sprintf("/app/installations/%d/access_tokens", installationID), token, nil, &out); err != nil {
		return "", time.Time{}, err
	}
	if out.Token == "" {
		return "", time.Time{}, fmt.Errorf("github app: empty installation token")
	}
	expires, _ := time.Parse(time.RFC3339, out.ExpiresAt)
	if expires.IsZero() {
		expires = time.Now().Add(50 * time.Minute)
	}
	c.mu.Lock()
	c.tokens[installationID] = cachedToken{Token: out.Token, ExpiresAt: expires}
	c.mu.Unlock()
	return out.Token, expires, nil
}

// Repositories lists repositories available to an installation token.
func (c *Client) Repositories(ctx context.Context, installationToken string) ([]Repository, error) {
	var out struct {
		Repositories []struct {
			FullName string `json:"full_name"`
			Private  bool   `json:"private"`
			HTMLURL  string `json:"html_url"`
		} `json:"repositories"`
	}
	if err := c.request(ctx, http.MethodGet, "/installation/repositories?per_page=100", installationToken, nil, &out); err != nil {
		return nil, err
	}
	repos := make([]Repository, 0, len(out.Repositories))
	for _, repo := range out.Repositories {
		repos = append(repos, Repository{FullName: repo.FullName, Private: repo.Private, HTMLURL: repo.HTMLURL})
	}
	return repos, nil
}
