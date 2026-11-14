package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Probe runs a minimal GraphQL query with the stored token.
func (a *Adapter) Probe(ctx context.Context, cred integration.Credential) error {
	if cred.AccessToken == "" {
		return fmt.Errorf("linear: missing token")
	}
	body, _ := json.Marshal(map[string]string{"query": "{ viewer { id } }"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", cred.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("linear probe: status %d", resp.StatusCode)
	}
	var out struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if len(out.Errors) > 0 {
		return fmt.Errorf("linear probe: %s", out.Errors[0].Message)
	}
	return nil
}
