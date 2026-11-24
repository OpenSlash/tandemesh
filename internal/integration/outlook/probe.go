package outlook

import (
	"context"
	"fmt"
	"net/http"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Probe reads the signed-in user profile.
func (a *Adapter) Probe(ctx context.Context, cred integration.Credential) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL()+"/v1.0/me?$select=id", nil)
	if err != nil {
		return err
	}
	a.headers(req, cred)
	resp, err := a.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("outlook probe: status %d", resp.StatusCode)
	}
	return nil
}
