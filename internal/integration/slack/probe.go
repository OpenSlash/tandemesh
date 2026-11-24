package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Probe calls auth.test with the bot token.
func (a *Adapter) Probe(ctx context.Context, cred integration.Credential) error {
	if cred.AccessToken == "" {
		return fmt.Errorf("slack: missing token")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL()+"/auth.test", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	resp, err := a.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if !out.OK {
		return fmt.Errorf("slack probe: %s", out.Error)
	}
	return nil
}
