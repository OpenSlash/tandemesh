package jira

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Probe checks the Jira site and credentials with the current-user endpoint.
func (a *Adapter) Probe(ctx context.Context, cred integration.Credential) error {
	base := strings.TrimRight(cred.BaseURL, "/")
	if base == "" && cred.Fields != nil {
		base = strings.TrimRight(cred.Fields["instance"], "/")
	}
	if base == "" {
		return fmt.Errorf("jira: site URL is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/rest/api/3/myself", nil)
	if err != nil {
		return err
	}
	cred.BaseURL = base
	account := cred.Account
	if account == "" {
		account = cred.Subject
	}
	if cred.Fields["auth"] != "oauth" && account != "" && strings.Contains(account, "@") {
		req.SetBasicAuth(account, cred.AccessToken)
	} else {
		req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := a.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("jira probe: status %d", resp.StatusCode)
	}
	return nil
}
