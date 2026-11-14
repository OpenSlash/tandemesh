package api

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/store"
)

type stubChat struct{ calls int }

func (s *stubChat) Send(context.Context, integration.Credential, integration.ChatMessage) (string, error) {
	s.calls++
	return "ref", nil
}
func (s *stubChat) Update(context.Context, integration.Credential, string, integration.ChatMessage) error {
	return nil
}

func TestRemindOnce(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "remind.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	approval := &domain.Approval{
		ID: "apr_1", RunID: "run_1", Tool: "bash", Command: "git push", Risk: "high",
		Status: domain.ApprovalPending, RequestedAt: now.Add(-6 * time.Minute), ExpiresAt: now.Add(20 * time.Minute),
	}
	if err := st.CreateApproval(ctx, approval); err != nil {
		t.Fatal(err)
	}

	chat := &stubChat{}
	reg := integration.NewRegistry()
	if err := reg.Register(&integration.Integration{
		Manifest: integration.Manifest{ID: "slack", Kind: integration.KindChat, Auth: integration.AuthBot},
		Chat:     chat,
	}); err != nil {
		t.Fatal(err)
	}

	srv := New(config.Config{SlackToken: "tok", SlackChannel: "C1", ApprovalKey: "k"}, st, nil, hub.New(), reg, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv.RemindOnce(ctx, now)
	if chat.calls != 1 {
		t.Fatalf("reminders = %d", chat.calls)
	}
	srv.RemindOnce(ctx, now.Add(time.Minute))
	if chat.calls != 1 {
		t.Fatalf("reminder repeated too early: %d", chat.calls)
	}
	srv.RemindOnce(ctx, now.Add(11*time.Minute))
	if chat.calls != 2 {
		t.Fatalf("reminder not repeated: %d", chat.calls)
	}
}
