package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestMetrics(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "metrics.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{
		ID: "run_metrics", Work: domain.WorkRef{Provider: "manual", ID: "1"},
		Status: domain.RunQueued, CreatedAt: time.Now().UTC(),
	}
	if err := st.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}

	srv := New(config.Config{}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	body := string(raw)
	for _, want := range []string{
		`orca_runs_total{status="queued"} 1`,
		"orca_approvals_pending",
		"orca_providers",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics missing %q:\n%s", want, body)
		}
	}
}
