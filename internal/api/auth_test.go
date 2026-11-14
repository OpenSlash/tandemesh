package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestWorkspaceAuthorization(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{User: "alice"}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	ws := postJSON[domain.Workspace](t, ts.URL+"/api/v1/workspaces", map[string]string{"name": "Private", "slug": "private"})

	// The local account can list; a different identity gets 403.
	if status := getStatus(t, ts.URL+"/api/v1/tasks?workspace="+ws.ID, "alice"); status != http.StatusOK {
		t.Fatalf("owner status = %d", status)
	}
	if status := getStatus(t, ts.URL+"/api/v1/tasks?workspace="+ws.ID, "charlie"); status != http.StatusForbidden {
		t.Fatalf("stranger status = %d", status)
	}

	// Stranger cannot import into the workspace.
	body, _ := json.Marshal(map[string]any{
		"provider": "github", "instance": "acme/x", "type": "issue", "id": "1",
		"title": "x", "workspaceId": ws.ID,
	})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/work/import", bytesReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Orca-User", "charlie")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("import status = %d", resp.StatusCode)
	}

	// The workspace list hides spaces the user cannot access.
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/v1/workspaces", nil)
	req.Header.Set("X-Orca-User", "charlie")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var listed struct {
		Items []domain.Workspace `json:"items"`
	}
	if err := json.Unmarshal(raw, &listed); err != nil {
		t.Fatal(err)
	}
	for _, item := range listed.Items {
		if item.ID == ws.ID {
			t.Fatalf("private workspace leaked to charlie: %+v", listed.Items)
		}
	}
}

func getStatus(t *testing.T, url, user string) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-Orca-User", user)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }
