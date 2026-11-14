package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// OAuthClient is the refresh configuration for a provider.
type OAuthClient struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scope        string
}

// OAuthConfig maps provider ids to refresh clients.
type OAuthConfig struct {
	Clients map[string]OAuthClient
}

func defaultTokenURL(provider string) string {
	switch provider {
	case "linear":
		return "https://api.linear.app/oauth/token"
	case "jira":
		return "https://auth.atlassian.com/oauth/token"
	case "outlook":
		return "https://login.microsoftonline.com/common/oauth2/v2.0/token"
	}
	return ""
}

// Ensure returns a usable credential, refreshing it shortly before expiry.
// It reports false when no credential is stored.
func (v *Vault) Ensure(ctx context.Context, provider, subject string) (integration.Credential, bool, error) {
	cred, ok, err := v.Get(ctx, provider, subject)
	if err != nil || !ok {
		return cred, ok, err
	}
	if cred.ExpiresAt.IsZero() || time.Until(cred.ExpiresAt) > time.Minute {
		return cred, true, nil
	}
	client, configured := v.OAuth.Clients[provider]
	if !configured || cred.RefreshToken == "" {
		return cred, true, nil
	}
	refreshed, err := v.refresh(ctx, provider, cred, client)
	if err != nil {
		v.log.Warn("oauth refresh failed", "provider", provider, "err", err)
		return cred, true, err
	}
	if err := v.Put(ctx, refreshed); err != nil {
		return cred, true, err
	}
	return refreshed, true, nil
}

func (v *Vault) refresh(ctx context.Context, provider string, cred integration.Credential, client OAuthClient) (integration.Credential, error) {
	tokenURL := client.TokenURL
	if tokenURL == "" {
		tokenURL = defaultTokenURL(provider)
	}
	if tokenURL == "" {
		return cred, fmt.Errorf("no token endpoint for %s", provider)
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", cred.RefreshToken)
	form.Set("client_id", client.ClientID)
	form.Set("client_secret", client.ClientSecret)
	if client.Scope != "" {
		form.Set("scope", client.Scope)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return cred, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	httpClient := v.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return cred, err
	}
	defer resp.Body.Close()

	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Error        string `json:"error"`
		Description  string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return cred, err
	}
	if resp.StatusCode >= 300 || out.AccessToken == "" {
		return cred, fmt.Errorf("oauth refresh: status %d %s %s", resp.StatusCode, out.Error, out.Description)
	}

	cred.AccessToken = out.AccessToken
	if out.RefreshToken != "" {
		cred.RefreshToken = out.RefreshToken
	}
	if out.ExpiresIn > 0 {
		cred.ExpiresAt = time.Now().UTC().Add(time.Duration(out.ExpiresIn) * time.Second)
	}
	return cred, nil
}
