package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Probe checks the token against the authenticated user endpoint.
func (a *Adapter) Probe(ctx context.Context, cred integration.Credential) error {
	if cred.AccessToken == "" {
		return fmt.Errorf("github: missing token")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURLFor(cred)+"/user", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := a.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github probe: status %d", resp.StatusCode)
	}
	return nil
}
