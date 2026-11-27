package plane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestSplitInstance(t *testing.T) {
	tests := []struct {
		in            string
		slug, project string
		ok            bool
	}{
		{"acme", "acme", "", true},
		{"acme/", "acme", "", true},
		{"acme/proj-1", "acme", "proj-1", true},
		{"", "", "", false},
	}
	for _, tt := range tests {
		slug, project, err := splitInstance(tt.in)
		if (err == nil) != tt.ok || slug != tt.slug || project != tt.project {
			t.Errorf("splitInstance(%q) = %q, %q, %v", tt.in, slug, project, err)
		}
	}
	if _, _, err := splitProjectInstance("acme"); err == nil {
		t.Error("splitProjectInstance accepted a workspace-only instance")
	}
}

func TestWorkspaceInstanceCoversEveryProject(t *testing.T) {
	var webhook map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/workspaces/acme/projects/":
			_, _ = w.Write([]byte(`[{"id":"p1","name":"One","identifier":"ONE"},{"id":"p2","name":"Two","identifier":"TWO"}]`))
		case "/api/v1/workspaces/acme/projects/p1/work-items/":
			_, _ = w.Write([]byte(`[{"id":"i1","name":"First","updated_at":"2026-01-01T00:00:00Z"}]`))
		case "/api/v1/workspaces/acme/projects/p2/work-items/":
			_, _ = w.Write([]byte(`[{"id":"i2","name":"Second","updated_at":"2026-01-01T00:00:00Z"}]`))
		case "/api/v1/workspaces/acme/projects/p1/states/":
			_, _ = w.Write([]byte(`[{"id":"s1","name":"Todo","group":"unstarted"},{"id":"s2","name":"Done","group":"completed"}]`))
		case "/api/v1/workspaces/acme/projects/p2/states/":
			_, _ = w.Write([]byte(`[{"id":"s3","name":"Done","group":"completed"}]`))
		case "/api/v1/users/me/":
			_, _ = w.Write([]byte(`{"id":"u-1","display_name":"alice","email":"a@example.test"}`))
		case "/api/v1/workspaces/acme/webhooks/":
			_ = json.NewDecoder(r.Body).Decode(&webhook)
			_, _ = w.Write([]byte(`{"id":"wh1"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	a := &Adapter{BaseURL: srv.URL, AppURL: "https://plane.test", HTTPClient: srv.Client()}
	cred := integration.Credential{Provider: "plane", AccessToken: "token"}
	ctx := context.Background()

	items, err := a.ListAssigned(ctx, cred, integration.ListOptions{Instance: "acme"})
	if err != nil || len(items) != 2 {
		t.Fatalf("items = %+v err=%v", items, err)
	}
	if items[0].Ref.Instance != "acme/p1" || items[1].Ref.Instance != "acme/p2" {
		t.Fatalf("items must carry their project: %q %q", items[0].Ref.Instance, items[1].Ref.Instance)
	}

	states, err := a.ListStates(ctx, cred, "acme")
	if err != nil || len(states) != 2 || states[0].Name != "Todo" || states[1].Name != "Done" {
		t.Fatalf("states should merge by name: %+v err=%v", states, err)
	}

	if _, err := a.RegisterWebhook(ctx, cred, "acme", "https://hook.test", ""); err != nil {
		t.Fatal(err)
	}
	if _, scoped := webhook["project"]; scoped {
		t.Fatalf("workspace webhook should not be project-scoped: %+v", webhook)
	}

	account, err := a.CurrentAccount(ctx, cred)
	if err != nil || account.ID != "u-1" || account.Login != "alice" {
		t.Fatalf("account = %+v err=%v", account, err)
	}
}
