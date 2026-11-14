package desktop

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
	"github.com/OpenSlash/tandemesh/internal/integration/github"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestServiceSurface(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "desktop.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	reg := integration.NewRegistry()
	if err := reg.Register(github.New("")); err != nil {
		t.Fatal(err)
	}
	svc := New(st, reg, "test")

	if svc.Version() != "test" {
		t.Fatalf("version = %q", svc.Version())
	}
	if got := svc.Providers(); len(got) != 1 || got[0].ID != "github" {
		t.Fatalf("providers = %+v", got)
	}

	run, err := svc.CreateRun(ctx, CreateRunInput{Repo: "acme/api", Prompt: "fix"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	runs, err := svc.ListRuns(ctx, "")
	if err != nil || len(runs) != 1 || runs[0].ID != run.ID {
		t.Fatalf("runs = %+v err=%v", runs, err)
	}
	if _, err := svc.GetRun(ctx, run.ID); err != nil {
		t.Fatalf("get: %v", err)
	}
	if _, err := svc.GetRun(ctx, "run_missing"); err == nil {
		t.Fatal("missing run did not error")
	}
	if ok, err := svc.CancelRun(ctx, run.ID); err != nil || !ok {
		t.Fatalf("cancel: %v %v", ok, err)
	}
}
