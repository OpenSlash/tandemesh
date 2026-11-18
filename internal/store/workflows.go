package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Store) CreateWorkflow(ctx context.Context, wf *domain.Workflow) error {
	if wf.ID == "" {
		wf.ID = "wf_" + ulid.Make().String()
	}
	now := time.Now().UTC()
	if wf.CreatedAt.IsZero() {
		wf.CreatedAt = now
	}
	wf.UpdatedAt = now
	nodes, _ := json.Marshal(wf.Nodes)
	_, err := s.db.ExecContext(ctx, `INSERT INTO workflows (id, workspace_id, name, description, nodes_json, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?)`, wf.ID, wf.WorkspaceID, wf.Name, wf.Description, string(nodes), ms(wf.CreatedAt), ms(wf.UpdatedAt))
	return err
}

func (s *Store) GetWorkflow(ctx context.Context, id string) (*domain.Workflow, error) {
	var wf domain.Workflow
	var nodes string
	var created, updated int64
	err := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, name, description, nodes_json, created_at_ms, updated_at_ms FROM workflows WHERE id = ?`, id).
		Scan(&wf.ID, &wf.WorkspaceID, &wf.Name, &wf.Description, &nodes, &created, &updated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(nodes), &wf.Nodes)
	wf.CreatedAt = fromMS(created)
	wf.UpdatedAt = fromMS(updated)
	return &wf, nil
}

func (s *Store) ListWorkflows(ctx context.Context, workspaceID string) ([]*domain.Workflow, error) {
	q := `SELECT id, workspace_id, name, description, nodes_json, created_at_ms, updated_at_ms FROM workflows`
	args := []any{}
	if workspaceID != "" {
		q += " WHERE workspace_id = ?"
		args = append(args, workspaceID)
	}
	q += " ORDER BY updated_at_ms DESC"
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Workflow
	for rows.Next() {
		var wf domain.Workflow
		var nodes string
		var created, updated int64
		if err := rows.Scan(&wf.ID, &wf.WorkspaceID, &wf.Name, &wf.Description, &nodes, &created, &updated); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(nodes), &wf.Nodes)
		wf.CreatedAt = fromMS(created)
		wf.UpdatedAt = fromMS(updated)
		out = append(out, &wf)
	}
	return out, rows.Err()
}

func (s *Store) CreateWorkflowRun(ctx context.Context, run *domain.WorkflowRun) error {
	if run.ID == "" {
		run.ID = "wfr_" + ulid.Make().String()
	}
	now := time.Now().UTC()
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	run.UpdatedAt = now
	nodes, _ := json.Marshal(run.Nodes)
	_, err := s.db.ExecContext(ctx, `INSERT INTO workflow_runs (id, workflow_id, workspace_id, task_id, status, current_node, nodes_json, error, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?)`, run.ID, run.WorkflowID, run.WorkspaceID, run.TaskID, run.Status, run.CurrentNode, string(nodes), run.Error, ms(run.CreatedAt), ms(run.UpdatedAt))
	return err
}

func (s *Store) UpdateWorkflowRun(ctx context.Context, run *domain.WorkflowRun) error {
	run.UpdatedAt = time.Now().UTC()
	nodes, _ := json.Marshal(run.Nodes)
	_, err := s.db.ExecContext(ctx, `UPDATE workflow_runs SET status = ?, current_node = ?, nodes_json = ?, error = ?, updated_at_ms = ? WHERE id = ?`,
		run.Status, run.CurrentNode, string(nodes), run.Error, ms(run.UpdatedAt), run.ID)
	return err
}

func (s *Store) GetWorkflowRun(ctx context.Context, id string) (*domain.WorkflowRun, error) {
	var run domain.WorkflowRun
	var nodes string
	var created, updated int64
	err := s.db.QueryRowContext(ctx, `SELECT id, workflow_id, workspace_id, task_id, status, current_node, nodes_json, error, created_at_ms, updated_at_ms FROM workflow_runs WHERE id = ?`, id).
		Scan(&run.ID, &run.WorkflowID, &run.WorkspaceID, &run.TaskID, &run.Status, &run.CurrentNode, &nodes, &run.Error, &created, &updated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(nodes), &run.Nodes)
	run.CreatedAt = fromMS(created)
	run.UpdatedAt = fromMS(updated)
	return &run, nil
}

func (s *Store) ListWorkflowRuns(ctx context.Context, workspaceID, taskID string, limit int) ([]*domain.WorkflowRun, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := `SELECT id, workflow_id, workspace_id, task_id, status, current_node, nodes_json, error, created_at_ms, updated_at_ms FROM workflow_runs`
	args := []any{}
	if workspaceID != "" {
		q += " WHERE workspace_id = ?"
		args = append(args, workspaceID)
	}
	if taskID != "" {
		q += " AND task_id = ?"
		args = append(args, taskID)
	}
	q += " ORDER BY created_at_ms DESC LIMIT ?"
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.WorkflowRun
	for rows.Next() {
		var run domain.WorkflowRun
		var nodes string
		var created, updated int64
		if err := rows.Scan(&run.ID, &run.WorkflowID, &run.WorkspaceID, &run.TaskID, &run.Status, &run.CurrentNode, &nodes, &run.Error, &created, &updated); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(nodes), &run.Nodes)
		run.CreatedAt = fromMS(created)
		run.UpdatedAt = fromMS(updated)
		out = append(out, &run)
	}
	return out, rows.Err()
}
