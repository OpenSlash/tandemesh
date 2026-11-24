package automation

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "auto.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestTickStartsDueAutomation(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)
	past := now.Add(-time.Minute)

	a := &domain.Automation{
		ID:       "auto_1",
		Name:     "nightly tests",
		Schedule: "0 * * * *",
		AgentID:  "claude",
		Repo:     "acme/api",
		Base:     "main",
		Prompt:   "run the nightly suite",
		Enabled:  true,
		NextRun:  &past,
	}
	if err := st.UpsertAutomation(ctx, a); err != nil {
		t.Fatal(err)
	}

	runner := New(st, slog.New(slog.NewTextHandler(io.Discard, nil)))
	runner.Now = func() time.Time { return now }

	n, err := runner.Tick(ctx)
	if err != nil || n != 1 {
		t.Fatalf("started = %d err=%v", n, err)
	}
	runs, _ := st.ListRuns(ctx, "", "", 10)
	if len(runs) != 1 || runs[0].Work.Provider != "automation" || runs[0].Prompt != "run the nightly suite" {
		t.Fatalf("runs = %+v", runs)
	}

	updated, _ := st.GetAutomation(ctx, a.ID)
	if updated.LastRun == nil || updated.NextRun == nil || !updated.NextRun.After(now) {
		t.Fatalf("automation schedule not advanced: %+v", updated)
	}

	// Not due any more.
	if n, _ := runner.Tick(ctx); n != 0 {
		t.Fatalf("started again: %d", n)
	}
}

func TestTickSkipsInvalidSchedule(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	past := now.Add(-time.Minute)
	_ = st.UpsertAutomation(ctx, &domain.Automation{
		ID: "auto_bad", Name: "bad", Schedule: "not a cron", AgentID: "claude", Enabled: true, NextRun: &past,
	})
	runner := New(st, slog.New(slog.NewTextHandler(io.Discard, nil)))
	runner.Now = func() time.Time { return now }
	if n, err := runner.Tick(ctx); err != nil || n != 0 {
		t.Fatalf("started = %d err=%v", n, err)
	}
}

func TestPrecheckFailureSkipsRun(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)
	past := now.Add(-time.Minute)
	a := &domain.Automation{
		ID: "auto_pre", Name: "precheck", Schedule: "0 * * * *", AgentID: "claude",
		Precheck: "exit 7", Enabled: true, NextRun: &past,
	}
	if err := st.UpsertAutomation(ctx, a); err != nil {
		t.Fatal(err)
	}
	runner := New(st, slog.New(slog.NewTextHandler(io.Discard, nil)))
	runner.Now = func() time.Time { return now }
	if n, err := runner.Tick(ctx); err != nil || n != 0 {
		t.Fatalf("started = %d err=%v", n, err)
	}
	runs, _ := st.ListRuns(ctx, "", "", 10)
	if len(runs) != 0 {
		t.Fatalf("precheck failure created run: %+v", runs)
	}
	updated, _ := st.GetAutomation(ctx, a.ID)
	if updated.LastStatus != "precheck_failed" || updated.NextRun == nil || !updated.NextRun.After(now) {
		t.Fatalf("automation = %+v", updated)
	}
}

func TestMissedGraceSkipsOldRun(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)
	old := now.Add(-2 * time.Hour)
	a := &domain.Automation{
		ID: "auto_grace", Name: "grace", Schedule: "0 * * * *", AgentID: "claude",
		MissedGraceSeconds: 300, Enabled: true, NextRun: &old,
	}
	if err := st.UpsertAutomation(ctx, a); err != nil {
		t.Fatal(err)
	}
	runner := New(st, slog.New(slog.NewTextHandler(io.Discard, nil)))
	runner.Now = func() time.Time { return now }
	if n, err := runner.Tick(ctx); err != nil || n != 0 {
		t.Fatalf("started = %d err=%v", n, err)
	}
	runs, _ := st.ListRuns(ctx, "", "", 10)
	if len(runs) != 0 {
		t.Fatalf("missed grace created run: %+v", runs)
	}
	updated, _ := st.GetAutomation(ctx, a.ID)
	if updated.LastStatus != "skipped_missed" {
		t.Fatalf("automation = %+v", updated)
	}
}

func TestAutomationReusesSession(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)
	a := &domain.Automation{
		ID: "auto_sess", WorkspaceID: "ws_default", Name: "session auto", Schedule: "0 * * * *",
		AgentID: "claude", Repo: "acme/api", Enabled: true, ReuseSession: true,
	}
	if err := st.UpsertAutomation(ctx, a); err != nil {
		t.Fatal(err)
	}
	first, err := StartRun(ctx, st, a, now)
	if err != nil {
		t.Fatal(err)
	}
	if first.SessionID == "" || first.Repo != "" {
		t.Fatalf("first run = %+v", first)
	}
	stored, _ := st.GetAutomation(ctx, a.ID)
	if stored.SessionID != first.SessionID {
		t.Fatalf("automation session = %q want %q", stored.SessionID, first.SessionID)
	}
	second, err := StartRun(ctx, st, stored, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if second.SessionID != first.SessionID {
		t.Fatalf("second session = %q want %q", second.SessionID, first.SessionID)
	}
}
