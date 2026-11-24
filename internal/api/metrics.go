package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// metrics exposes a small Prometheus text surface. It is intentionally
// hand-rolled: the control plane has a handful of counters, not a metrics
// pipeline.
func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	runs, err := s.store.CountRunsByStatus(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	approvals, _ := s.store.CountApprovals(ctx, domain.ApprovalPending)
	outbox, _ := s.store.CountOutboxPending(ctx)
	machines, _ := s.store.CountOnlineMachines(ctx, time.Now().UTC().Add(-45*time.Second))

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintln(w, "# HELP orca_runs_total Runs by status.")
	fmt.Fprintln(w, "# TYPE orca_runs_total gauge")
	statuses := make([]string, 0, len(runs))
	for status := range runs {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	for _, status := range statuses {
		fmt.Fprintf(w, "orca_runs_total{status=%q} %d\n", status, runs[status])
	}
	fmt.Fprintln(w, "# HELP orca_approvals_pending Pending approvals.")
	fmt.Fprintln(w, "# TYPE orca_approvals_pending gauge")
	fmt.Fprintf(w, "orca_approvals_pending %d\n", approvals)
	fmt.Fprintln(w, "# HELP orca_outbox_pending Queued write-backs.")
	fmt.Fprintln(w, "# TYPE orca_outbox_pending gauge")
	fmt.Fprintf(w, "orca_outbox_pending %d\n", outbox)
	fmt.Fprintln(w, "# HELP orca_machines_online Machines seen in the last 45 seconds.")
	fmt.Fprintln(w, "# TYPE orca_machines_online gauge")
	fmt.Fprintf(w, "orca_machines_online %d\n", machines)
	fmt.Fprintln(w, "# HELP orca_providers Registered integrations.")
	fmt.Fprintln(w, "# TYPE orca_providers gauge")
	fmt.Fprintf(w, "orca_providers %d\n", len(s.integrations.List()))
	s.metricsMu.Lock()
	keys := make([]string, 0, len(s.httpCounts))
	for key := range s.httpCounts {
		keys = append(keys, key)
	}
	s.metricsMu.Unlock()
	sort.Strings(keys)
	fmt.Fprintln(w, "# HELP orca_http_requests_total HTTP requests by method and status.")
	fmt.Fprintln(w, "# TYPE orca_http_requests_total counter")
	for _, key := range keys {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}
		s.metricsMu.Lock()
		count := s.httpCounts[key]
		s.metricsMu.Unlock()
		fmt.Fprintf(w, "orca_http_requests_total{method=%q,status=%q} %d\n", parts[0], parts[1], count)
	}
}
