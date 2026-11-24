package api

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestApprovalSignedLink(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "appr.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	run := &domain.Run{
		ID: "run_1", Work: domain.WorkRef{Provider: "manual", ID: "1"},
		Status: domain.RunRunning, CreatedAt: time.Now().UTC(),
	}
	if err := st.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	approval := &domain.Approval{
		ID: "apr_1", RunID: run.ID, Tool: "bash", Command: "git push", Risk: "high",
		Status: domain.ApprovalPending, RequestedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
	}
	if err := st.CreateApproval(ctx, approval); err != nil {
		t.Fatal(err)
	}

	srv := New(config.Config{ApprovalKey: "test-key"}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	exp := time.Now().Add(time.Minute).Unix()
	token := srv.approvalToken(approval.ID, "approved", exp)
	url := fmt.Sprintf("%s/api/v1/approvals/%s/decide?decision=approved&exp=%d&token=%s", ts.URL, approval.ID, exp, token)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	got, _ := st.GetApproval(ctx, approval.ID)
	if got == nil || got.Status != domain.ApprovalApproved {
		t.Fatalf("approval = %+v", got)
	}

	// A tampered token is rejected.
	bad := fmt.Sprintf("%s/api/v1/approvals/%s/decide?decision=denied&exp=%d&token=%s", ts.URL, approval.ID, exp, token)
	resp2, err := http.Get(bad)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("tampered status = %d", resp2.StatusCode)
	}
}

func TestApprovalLinkExpired(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "appr2.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	_ = st.Migrate(context.Background())

	srv := New(config.Config{ApprovalKey: "k"}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	exp := time.Now().Add(-time.Minute).Unix()
	token := srv.approvalToken("apr_x", "approved", exp)
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/approvals/apr_x/decide?decision=approved&exp=%d&token=%s", ts.URL, exp, token))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusGone {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}
