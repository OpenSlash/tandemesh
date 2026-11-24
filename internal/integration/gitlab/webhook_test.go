package gitlab

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestWebhookIssue(t *testing.T) {
	body := []byte(`{"object_kind":"issue","user":{"name":"Alice"},"project":{"path_with_namespace":"acme/api"},"object_attributes":{"iid":7,"title":"Fix","description":"body","state":"opened","url":"https://gitlab.test/acme/api/-/issues/7","updated_at":"2026-01-01T00:00:00Z","action":"open"},"labels":["bug"]}`)
	a := &Adapter{Secret: "s3cret"}
	req, _ := http.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Gitlab-Token", "s3cret")
	if err := a.Verify(req, body); err != nil {
		t.Fatal(err)
	}
	ev, err := a.Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	work, snap, _, skip, err := a.Normalize(ev)
	if err != nil || skip != "" {
		t.Fatalf("skip = %s, err = %v", skip, err)
	}
	if work.Instance != "acme/api" || work.ID != "7" || snap.Title != "Fix" {
		t.Fatalf("work = %+v, snap = %+v", work, snap)
	}
}

func TestWebhookNote(t *testing.T) {
	body := []byte(`{"object_kind":"note","user":{"name":"Bob"},"project":{"path_with_namespace":"acme/api"},"object_attributes":{"note":"hi","noteable_type":"Issue","noteable_iid":7,"url":"https://gitlab.test/note/1"},"issue":{"iid":7,"title":"Fix","state":"opened","url":"https://gitlab.test/acme/api/-/issues/7"}}`)
	ev, err := (&Adapter{}).Parse(body)
	if err != nil || ev.Kind != "comment" || ev.Text != "hi" {
		t.Fatalf("event = %+v, err = %v", ev, err)
	}
}

func TestRegisterWebhook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v4/projects/acme/api/hooks" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			URL   string `json:"url"`
			Token string `json:"token"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.URL != "https://orca.test/hook" || body.Token != "s3cret" {
			t.Errorf("body = %+v", body)
		}
		_, _ = w.Write([]byte(`{"id":42}`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client(), BaseURL: srv.URL}
	id, err := a.RegisterWebhook(t.Context(), integration.Credential{AccessToken: "tok"}, "acme/api", "https://orca.test/hook", "s3cret")
	if err != nil || id != "42" {
		t.Fatalf("id = %s, err = %v", id, err)
	}
}

func TestUnregisterWebhook(t *testing.T) {
	deleted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/acme/api/hooks":
			_, _ = w.Write([]byte(`[{"id":42,"url":"https://orca.test/hook"}]`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v4/projects/acme/api/hooks/42":
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client(), BaseURL: srv.URL}
	if err := a.UnregisterWebhook(t.Context(), integration.Credential{AccessToken: "tok"}, "acme/api", "https://orca.test/hook"); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("hook was not deleted")
	}
}

func TestParseMergeRequest(t *testing.T) {
	body := []byte(`{"object_kind":"merge_request","user":{"name":"Alice"},"project":{"path_with_namespace":"acme/api"},"object_attributes":{"iid":9,"title":"ENG-7 fix","state":"opened","url":"https://gitlab.test/mr/9","source_branch":"ENG-7"}}`)
	ev, err := (&Adapter{}).Parse(body)
	if err != nil || ev.Kind != "merge_request" {
		t.Fatalf("event = %+v, err = %v", ev, err)
	}
}
