package api

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestLinkGitLabMergeRequest(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "mr.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	task, err := st.EnsureTaskIn(ctx, "ws_default", domain.WorkRef{
		Provider: "gitlab", Instance: "acme/api", Type: "issue", ID: "ENG-7", URL: "https://gitlab.test/acme/api/-/issues/7",
	}, domain.Snapshot{Title: "Fix"})
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{store: st, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	raw := []byte(`{"user":{"name":"Alice"},"project":{"path_with_namespace":"acme/api"},"object_attributes":{"iid":9,"title":"ENG-7 fix","state":"merged","url":"https://gitlab.test/mr/9","source_branch":"ENG-7"}}`)
	if linked := s.linkGitLabMergeRequest(ctx, raw, "ws_default"); linked != 1 {
		t.Fatalf("linked = %d", linked)
	}
	items, err := st.ListTaskPullRequests(ctx, task.ID)
	if err != nil || len(items) != 1 || items[0].State != "merged" {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
}

func TestLinkGitLabPipeline(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "pipeline.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	task, err := st.EnsureTaskIn(ctx, "ws_default", domain.WorkRef{
		Provider: "gitlab", Instance: "acme/api", Type: "issue", ID: "ENG-7",
	}, domain.Snapshot{Title: "Fix"})
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{store: st, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	raw := []byte(`{"user":{"name":"Alice"},"project":{"path_with_namespace":"acme/api"},"object_attributes":{"id":1,"ref":"ENG-7","status":"success","url":"https://gitlab.test/pipeline/1","sha":"abc"},"commit":{"message":"ENG-7 fix"}}`)
	if linked := s.linkGitLabPipeline(ctx, raw, "ws_default"); linked != 1 {
		t.Fatalf("linked = %d", linked)
	}
	events, err := st.ListTaskEvents(ctx, task.ID)
	if err != nil || len(events) == 0 {
		t.Fatalf("events = %+v, err = %v", events, err)
	}
}

func TestEnqueuePipelineFollowUp(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "pipeline-follow.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	task, err := st.EnsureTaskIn(ctx, "ws_default", domain.WorkRef{
		Provider: "gitlab", Instance: "acme/api", Type: "issue", ID: "ENG-7",
	}, domain.Snapshot{Title: "Fix"})
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{store: st, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	automation := &domain.Automation{
		ID: "auto_1", WorkspaceID: "ws_default", Provider: "gitlab", Instance: "acme/api",
		TriggerKind: "pipeline", AgentID: "codex", Enabled: true,
	}
	raw := []byte(`{"project":{"path_with_namespace":"acme/api"},"object_attributes":{"id":1,"ref":"ENG-7","status":"failed","url":"https://gitlab.test/pipeline/1","sha":"abc"},"commit":{"message":"ENG-7 fix"}}`)
	if queued := s.enqueuePipelineFollowUp(ctx, raw, automation); queued != 1 {
		t.Fatalf("queued = %d", queued)
	}
	if queued := s.enqueuePipelineFollowUp(ctx, raw, automation); queued != 0 {
		t.Fatalf("duplicate queued = %d", queued)
	}
	runs, err := st.ListRunsByTask(ctx, task.ID)
	if err != nil || len(runs) != 1 || runs[0].AgentID != "codex" {
		t.Fatalf("runs = %+v, err = %v", runs, err)
	}
}

func TestPipelineFollowUpPolicy(t *testing.T) {
	automation := &domain.Automation{ID: "auto_1", Writeback: map[string]any{
		"pipelineBranches": "main,release/*", "pipelineLabels": "bug",
	}}
	task := &domain.Task{Labels: []string{"bug"}}
	if !pipelineFollowUpAllowed(nil, automation, task, "main") {
		t.Fatal("main should be allowed")
	}
	if !pipelineFollowUpAllowed(nil, automation, task, "release/1.0") {
		t.Fatal("release branch should be allowed")
	}
	if pipelineFollowUpAllowed(nil, automation, task, "feature/x") {
		t.Fatal("feature branch should be rejected")
	}
	task.Labels = []string{"docs"}
	if pipelineFollowUpAllowed(nil, automation, task, "main") {
		t.Fatal("label mismatch should be rejected")
	}
}
