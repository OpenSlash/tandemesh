package plane

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestPlaneReaderAndComment(t *testing.T) {
	var commentBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/users/me/":
			_, _ = w.Write([]byte(`{"id":"user-1","email":"alice@example.test"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/":
			_, _ = w.Write([]byte(`{"results":[{"slug":"acme"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/acme/projects/proj-1/work-items/":
			_, _ = w.Write([]byte(`[{"id":"issue-1","sequence_id":7,"name":"Fix it","description_stripped":"Body","permalink":"https://plane.test/acme/projects/proj-1/issues/7","updated_at":"2026-01-01T00:00:00Z","labels":[{"name":"bug"}],"assignee_details":[{"display_name":"alice"}]}]`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/acme/projects/proj-1/states/":
			_, _ = w.Write([]byte(`[{"id":"state-done","name":"Done","group":"completed"}]`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/acme/projects/proj-1/work-items/issue-1/comments/":
			_ = json.NewDecoder(r.Body).Decode(&commentBody)
			_, _ = w.Write([]byte(`{"id":42}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	a := &Adapter{BaseURL: srv.URL, AppURL: "https://plane.test", HTTPClient: srv.Client()}
	cred := integration.Credential{Provider: "plane", AccessToken: "token"}

	if err := a.Probe(context.Background(), cred); err != nil {
		t.Fatalf("probe: %v", err)
	}
	items, err := a.ListAssigned(context.Background(), cred, integration.ListOptions{Instance: "acme/proj-1"})
	if err != nil || len(items) != 1 {
		t.Fatalf("items = %+v err=%v", items, err)
	}
	item := items[0]
	if item.Ref.Provider != "plane" || item.Ref.ID != "issue-1" || item.Title != "Fix it" {
		t.Fatalf("item = %+v", item)
	}
	if len(item.Labels) != 1 || item.Labels[0] != "bug" || item.Assignee != "alice" {
		t.Fatalf("labels = %+v assignee=%q", item.Labels, item.Assignee)
	}

	states, err := a.ListStates(context.Background(), cred, "acme/proj-1")
	if err != nil || len(states) != 1 || states[0].ID != "state-done" {
		t.Fatalf("states = %+v err=%v", states, err)
	}

	ref, err := a.Comment(context.Background(), domain.WorkRef{
		Provider: "plane", Instance: "acme/proj-1", Type: "issue", ID: "issue-1",
	}, cred, "done")
	if err != nil || ref != "42" {
		t.Fatalf("comment = %q err=%v", ref, err)
	}
	if body, _ := commentBody["comment_stripped"].(string); !strings.HasPrefix(body, "done") {
		t.Fatalf("comment body = %+v", commentBody)
	}
}

func TestPlaneWebhook(t *testing.T) {
	secret := "s3cret"
	a := &Adapter{Secret: secret, AppURL: "https://plane.test"}
	body := []byte(`{"event":"issue","action":"updated","webhook_id":"wh-1","data":{"id":"issue-1","name":"Fix it","description_stripped":"Body","project":"proj-1","workspace_slug":"acme","permalink":"https://plane.test/acme/projects/proj-1/issues/7","updated_at":"2026-01-01T00:00:00Z","label_details":[{"name":"bug"}],"assignee_details":[{"display_name":"alice"}]},"activity":{"actor":{"display_name":"alice"}}}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))
	req := httptest.NewRequest(http.MethodPost, "/webhooks/plane/bnd", bytes.NewReader(body))
	req.Header.Set("X-Plane-Signature", sig)
	if err := a.Verify(req, body); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := a.VerifyWithSecret(req, body, secret); err != nil {
		t.Fatalf("verify with secret: %v", err)
	}
	if err := a.VerifyWithSecret(req, body, "wrong"); err == nil {
		t.Fatal("wrong secret accepted")
	}
	ev, err := a.Parse(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	work, snap, prompt, skip, err := a.Normalize(ev)
	if err != nil || skip != "" {
		t.Fatalf("normalize: %v %s", err, skip)
	}
	if work.Provider != "plane" || work.Instance != "acme/proj-1" || work.ID != "issue-1" {
		t.Fatalf("work = %+v", work)
	}
	if snap.Title != "Fix it" || len(snap.Labels) != 1 || snap.Author != "alice" {
		t.Fatalf("snap = %+v", snap)
	}
	if !strings.Contains(prompt, "Fix it") {
		t.Fatalf("prompt = %q", prompt)
	}
}
