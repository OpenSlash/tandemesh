package writeback

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

type stubWork struct {
	called      int
	markdown    string
	transitions []string
}

func (s *stubWork) Comment(_ context.Context, _ domain.WorkRef, _ integration.Credential, md string) (string, error) {
	s.called++
	s.markdown = md
	return "https://example.test/comment/1", nil
}
func (s *stubWork) Transition(_ context.Context, _ domain.WorkRef, _ integration.Credential, to string) error {
	s.transitions = append(s.transitions, to)
	return nil
}
func (s *stubWork) Attach(context.Context, domain.WorkRef, integration.Credential, string, string) error {
	return nil
}

func testStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "wb.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestProcessOnce(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	work := &stubWork{}
	reg := integration.NewRegistry()
	if err := reg.Register(&integration.Integration{
		Manifest: integration.Manifest{
			ID: "github", Kind: integration.KindTask,
			Capabilities: []integration.Capability{integration.CapComment},
		},
		Work: work,
	}); err != nil {
		t.Fatal(err)
	}

	item := &domain.Writeback{
		ID:       "wb_1",
		RunID:    "run_1",
		Provider: "github",
		Action:   "comment",
		Work:     domain.WorkRef{Provider: "github", Instance: "acme/api", Type: "issue", ID: "7"},
		Markdown: "done",
		NextAt:   time.Now().UTC().Add(-time.Second),
	}
	if err := st.EnqueueWriteback(ctx, item); err != nil {
		t.Fatal(err)
	}

	p := New(st, reg, func(string, string, string) integration.Credential {
		return integration.Credential{AccessToken: "tok"}
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	n, err := p.ProcessOnce(ctx)
	if err != nil || n != 1 {
		t.Fatalf("processed = %d, err = %v", n, err)
	}
	if work.called != 1 || work.markdown != "done" {
		t.Fatalf("stub = %+v", work)
	}
	due, _ := st.DueWritebacks(ctx, time.Now().UTC(), 10)
	if len(due) != 0 {
		t.Fatalf("still due: %d", len(due))
	}
}

func TestProcessTransition(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	work := &stubWork{}
	reg := integration.NewRegistry()
	_ = reg.Register(&integration.Integration{
		Manifest: integration.Manifest{
			ID: "jira", Kind: integration.KindTask,
			Capabilities: []integration.Capability{integration.CapTransition},
		},
		Work: work,
	})
	if err := st.EnqueueWriteback(ctx, &domain.Writeback{
		ID: "wb_tr", RunID: "run_tr", Provider: "jira", Action: "transition",
		Work:   domain.WorkRef{Provider: "jira", Instance: "https://acme.atlassian.net", Type: "issue", ID: "ACME-7"},
		URL:    "Done",
		NextAt: time.Now().UTC().Add(-time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	p := New(st, reg, func(string, string, string) integration.Credential {
		return integration.Credential{AccessToken: "tok"}
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if n, err := p.ProcessOnce(ctx); err != nil || n != 1 {
		t.Fatalf("processed = %d err=%v", n, err)
	}
	if len(work.transitions) != 1 || work.transitions[0] != "Done" {
		t.Fatalf("transitions = %+v", work.transitions)
	}
}

func TestMissingCredentialDeadLetters(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	reg := integration.NewRegistry()
	_ = reg.Register(&integration.Integration{
		Manifest: integration.Manifest{ID: "github", Kind: integration.KindTask, Capabilities: []integration.Capability{integration.CapComment}},
		Work:     &stubWork{},
	})
	_ = st.EnqueueWriteback(ctx, &domain.Writeback{
		ID: "wb_2", RunID: "run_2", Provider: "github", Action: "comment",
		Work:   domain.WorkRef{Provider: "github", Instance: "acme/api", ID: "8"},
		NextAt: time.Now().UTC().Add(-time.Second),
	})

	p := New(st, reg, func(string, string, string) integration.Credential { return integration.Credential{} }, slog.New(slog.NewTextHandler(io.Discard, nil)))
	_, _ = p.ProcessOnce(ctx)
	due, _ := st.DueWritebacks(ctx, time.Now().UTC(), 10)
	if len(due) != 0 {
		t.Fatalf("dead row still due: %d", len(due))
	}
}

func TestPolicySkipsReadOnlyBinding(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	work := &stubWork{}
	reg := integration.NewRegistry()
	_ = reg.Register(&integration.Integration{
		Manifest: integration.Manifest{ID: "github", Kind: integration.KindTask, Capabilities: []integration.Capability{integration.CapComment}},
		Work:     work,
	})
	binding := &domain.ResourceBinding{
		ID: "bind_read", WorkspaceID: "ws_default", ConnectionID: "conn_1",
		Provider: "github", ExternalID: "acme/api", Direction: "read", Enabled: true,
	}
	if err := st.UpsertBinding(ctx, binding); err != nil {
		t.Fatal(err)
	}
	item := &domain.Writeback{
		ID: "wb_read", RunID: "run_read", Provider: "github", ConnectionID: "conn_1",
		Action: "comment", Work: domain.WorkRef{Provider: "github", Instance: "acme/api", Type: "issue", ID: "7"},
		Markdown: "done", NextAt: time.Now().UTC().Add(-time.Second),
	}
	if err := st.EnqueueWriteback(ctx, item); err != nil {
		t.Fatal(err)
	}
	p := New(st, reg, func(string, string, string) integration.Credential { return integration.Credential{AccessToken: "tok"} }, slog.New(slog.NewTextHandler(io.Discard, nil)))
	p.Policy = func(ctx context.Context, w *domain.Writeback) bool {
		return st.WritebackAllowed(ctx, w.Provider, w.ConnectionID, w.Work.Instance)
	}
	if n, err := p.ProcessOnce(ctx); err != nil || n != 1 {
		t.Fatalf("processed = %d, err = %v", n, err)
	}
	if work.called != 0 {
		t.Fatalf("read-only binding wrote back: %+v", work)
	}
	due, _ := st.DueWritebacks(ctx, time.Now().UTC(), 10)
	if len(due) != 0 {
		t.Fatalf("skipped row still due: %d", len(due))
	}
}
