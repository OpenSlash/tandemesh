package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestApprovalExpiry(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "expiry.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	a := &domain.Approval{
		ID: "apr_old", RunID: "run_1", Tool: "bash", Command: "git push", Risk: "high",
		Status: domain.ApprovalPending, RequestedAt: now.Add(-time.Hour), ExpiresAt: now.Add(-time.Minute),
	}
	if err := st.CreateApproval(ctx, a); err != nil {
		t.Fatal(err)
	}
	fresh := &domain.Approval{
		ID: "apr_fresh", RunID: "run_1", Tool: "bash", Risk: "low",
		Status: domain.ApprovalPending, RequestedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	if err := st.CreateApproval(ctx, fresh); err != nil {
		t.Fatal(err)
	}

	expired, err := st.ListExpiredApprovals(ctx, now)
	if err != nil || len(expired) != 1 || expired[0].ID != "apr_old" {
		t.Fatalf("expired = %+v err=%v", expired, err)
	}
	ok, err := st.ExpireApproval(ctx, "apr_old")
	if err != nil || !ok {
		t.Fatalf("expire = %v %v", ok, err)
	}
	got, _ := st.GetApproval(ctx, "apr_old")
	if got.Status != domain.ApprovalExpired {
		t.Fatalf("status = %q", got.Status)
	}
	// A decision cannot override an expired approval.
	if ok, _ := st.DecideApproval(ctx, "apr_old", domain.ApprovalApproved, "web"); ok {
		t.Fatal("expired approval was decided")
	}
}
