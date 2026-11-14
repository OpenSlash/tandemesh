package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// TestPostgresStore runs only when a disposable database is provided:
//
//	ORCA_TEST_POSTGRES=postgres://orca:orca@localhost:5432/orca_test?sslmode=disable go test ./internal/store
func TestPostgresStore(t *testing.T) {
	dsn := os.Getenv("ORCA_TEST_POSTGRES")
	if dsn == "" {
		t.Skip("ORCA_TEST_POSTGRES is not set")
	}
	st, err := Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	id := "run_pg_" + time.Now().Format("20060102150405.000000")
	run := &domain.Run{
		ID:        id,
		Work:      domain.WorkRef{Provider: "manual", Instance: "test", Type: "task", ID: "1"},
		Status:    domain.RunQueued,
		CreatedAt: time.Now().UTC(),
	}
	if err := st.CreateRun(ctx, run); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := st.GetRun(ctx, id)
	if err != nil || got == nil || got.ID != id {
		t.Fatalf("get = %+v err=%v", got, err)
	}
	if _, err := st.AppendEvent(ctx, id, "run.created", map[string]any{"provider": "manual"}); err != nil {
		t.Fatalf("append event: %v", err)
	}
	events, err := st.ListEvents(ctx, id, 0)
	if err != nil || len(events) == 0 {
		t.Fatalf("events = %+v err=%v", events, err)
	}
}
