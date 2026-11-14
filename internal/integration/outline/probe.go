package outline

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Probe calls auth.info with the API token.
func (a *Adapter) Probe(ctx context.Context, cred integration.Credential) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL()+"/api/auth.info", bytes.NewReader([]byte(`{}`)))
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
		return fmt.Errorf("outline probe: status %d", resp.StatusCode)
	}
	return nil
}
