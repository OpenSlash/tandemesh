package notion

import (
	"context"
	"fmt"
	"net/http"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Probe reads the bot user.
func (a *Adapter) Probe(ctx context.Context, cred integration.Credential) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURLFor(cred)+"/v1/users/me", nil)
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
		return fmt.Errorf("notion probe: status %d", resp.StatusCode)
	}
	return nil
}
