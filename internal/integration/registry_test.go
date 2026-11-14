package integration

import (
	"net/http"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestRegistry(t *testing.T) {
	reg := NewRegistry()
	gh := &Integration{
		Manifest: Manifest{
			ID: "github", Name: "GitHub", Kind: KindTask, Auth: AuthToken,
			Capabilities: []Capability{CapWebhook},
		},
		Webhook: stubWebhook{},
	}
	if err := reg.Register(gh); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := reg.Register(gh); err == nil {
		t.Fatal("duplicate registration accepted")
	}
	got, ok := reg.Get("github")
	if !ok || got.Manifest.Name != "GitHub" {
		t.Fatalf("get = %+v, %v", got, ok)
	}
	if !gh.Supported(CapWebhook) {
		t.Fatal("webhook should be supported")
	}
	if gh.Supported(CapSend) {
		t.Fatal("send should not be supported")
	}
	if list := reg.ListByKind(KindTask); len(list) != 1 {
		t.Fatalf("task integrations = %d", len(list))
	}
	if list := reg.ListByKind(KindChat); len(list) != 0 {
		t.Fatalf("chat integrations = %d", len(list))
	}
}

func TestRegisterValidatesKind(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(&Integration{Manifest: Manifest{ID: "x", Kind: "unknown"}}); err == nil {
		t.Fatal("unknown kind accepted")
	}
	if err := reg.Register(&Integration{Manifest: Manifest{Kind: KindTask}}); err == nil {
		t.Fatal("missing id accepted")
	}
}

type stubWebhook struct{}

func (stubWebhook) Verify(*http.Request, []byte) error { return nil }

func (stubWebhook) Parse([]byte) (Event, error) { return Event{}, nil }

func (stubWebhook) Normalize(Event) (domain.WorkRef, domain.Snapshot, string, string, error) {
	return domain.WorkRef{}, domain.Snapshot{}, "", "", nil
}
