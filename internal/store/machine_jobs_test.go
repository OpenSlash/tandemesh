package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestMachineJobLifecycle(t *testing.T) {
	ctx := context.Background()
	st, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	job, err := st.CreateMachineJob(ctx, "mch_a", domain.MachineJobACPInstall, map[string]string{"registryId": "cursor"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateMachineJob(ctx, "mch_b", domain.MachineJobRuntimeCheck, struct{}{}); err != nil {
		t.Fatal(err)
	}

	claimed, err := st.ClaimMachineJobs(ctx, "mch_a", 5)
	if err != nil || len(claimed) != 1 || claimed[0].ID != job.ID || claimed[0].Status != "running" {
		t.Fatalf("claimed = %+v err=%v", claimed, err)
	}
	if again, _ := st.ClaimMachineJobs(ctx, "mch_a", 5); len(again) != 0 {
		t.Fatalf("job handed out twice: %+v", again)
	}

	if err := st.FinishMachineJob(ctx, job.ID, "mch_b", "done", nil, ""); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("another machine finished the job: err=%v", err)
	}
	if err := st.FinishMachineJob(ctx, job.ID, "mch_a", "done", json.RawMessage(`{"version":"1.0"}`), ""); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetMachineJob(ctx, job.ID)
	if err != nil || got == nil || got.Status != "done" || got.CompletedAt == nil || string(got.Result) != `{"version":"1.0"}` {
		t.Fatalf("job = %+v err=%v", got, err)
	}
}

func TestSetMachineRuntimes(t *testing.T) {
	ctx := context.Background()
	st, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertMachine(ctx, &domain.Machine{ID: "mch_a", Name: "a", Agents: []string{"claude"}, Status: "online"}); err != nil {
		t.Fatal(err)
	}
	details := []domain.RuntimeStatus{{ID: "codex", Status: domain.RuntimeReady}, {ID: "claude", Status: domain.RuntimeAdapterMissing}}
	if err := st.SetMachineRuntimes(ctx, "mch_a", map[string]string{"codex": "1.0"}, details, nil); err != nil {
		t.Fatal(err)
	}
	m, err := st.GetMachine(ctx, "mch_a")
	if err != nil || m == nil {
		t.Fatalf("machine = %+v err=%v", m, err)
	}
	if len(m.RuntimeDetails) != 2 || m.Runtimes["codex"] != "1.0" || len(m.Agents) != 1 || m.Agents[0] != "claude" {
		t.Fatalf("nil agents should keep backends: %+v", m)
	}
	if err := st.SetMachineRuntimes(ctx, "mch_a", nil, nil, []string{"codex"}); err != nil {
		t.Fatal(err)
	}
	if m, _ = st.GetMachine(ctx, "mch_a"); len(m.Agents) != 1 || m.Agents[0] != "codex" || len(m.RuntimeDetails) != 0 {
		t.Fatalf("agents not replaced: %+v", m)
	}
}

func TestDeleteMachine(t *testing.T) {
	ctx := context.Background()
	st, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"mch_free", "mch_busy"} {
		if err := st.UpsertMachine(ctx, &domain.Machine{ID: id, Name: id, Status: "offline"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.CreateMachineJob(ctx, "mch_free", domain.MachineJobRuntimeCheck, struct{}{}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().ExecContext(ctx, `INSERT INTO agents (id, name, machine_id, created_at_ms, updated_at_ms) VALUES ('agt_1', 'a', 'mch_busy', 0, 0)`); err != nil {
		t.Fatal(err)
	}

	if err := st.DeleteMachine(ctx, "mch_busy"); !errors.Is(err, ErrMachineInUse) {
		t.Fatalf("delete with an agent bound: err=%v", err)
	}
	if err := st.DeleteMachine(ctx, "mch_free"); err != nil {
		t.Fatal(err)
	}
	if m, _ := st.GetMachine(ctx, "mch_free"); m != nil {
		t.Fatalf("machine still present: %+v", m)
	}
	if jobs, _ := st.ClaimMachineJobs(ctx, "mch_free", 5); len(jobs) != 0 {
		t.Fatalf("queued jobs not removed: %+v", jobs)
	}
	if err := st.DeleteMachine(ctx, "mch_free"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("second delete: err=%v", err)
	}
}
