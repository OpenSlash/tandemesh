package store

import (
	"context"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// ListAssignmentRules returns workspace rules ordered by priority.
func (s *Store) ListAssignmentRules(ctx context.Context, workspaceID string) ([]*domain.AssignmentRule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, name, provider, instance, label, author, assignee,
		agent_id, group_id, repo, auto_dispatch, policy, enabled, priority, created_at_ms
		FROM assignment_rules WHERE workspace_id = ? ORDER BY priority DESC, created_at_ms ASC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.AssignmentRule
	for rows.Next() {
		var (
			r            domain.AssignmentRule
			autoDispatch int
			enabled      int
			created      int64
		)
		if err := rows.Scan(&r.ID, &r.WorkspaceID, &r.Name, &r.Provider, &r.Instance, &r.Label, &r.Author,
			&r.Assignee, &r.AgentID, &r.GroupID, &r.Repo, &autoDispatch, &r.Policy, &enabled, &r.Priority, &created); err != nil {
			return nil, err
		}
		r.AutoDispatch = autoDispatch == 1
		r.Enabled = enabled == 1
		r.CreatedAt = fromMS(created)
		out = append(out, &r)
	}
	return out, rows.Err()
}

// UpsertAssignmentRule stores a rule.
func (s *Store) UpsertAssignmentRule(ctx context.Context, r *domain.AssignmentRule) error {
	if r.WorkspaceID == "" {
		r.WorkspaceID = "ws_default"
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	auto, enabled := 0, 0
	if r.AutoDispatch {
		auto = 1
	}
	if r.Enabled {
		enabled = 1
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO assignment_rules
		(id, workspace_id, name, provider, instance, label, author, assignee, agent_id, group_id, repo,
		 auto_dispatch, policy, enabled, priority, created_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name, provider = excluded.provider, instance = excluded.instance,
			label = excluded.label, author = excluded.author, assignee = excluded.assignee,
			agent_id = excluded.agent_id, group_id = excluded.group_id, repo = excluded.repo, auto_dispatch = excluded.auto_dispatch,
			enabled = excluded.enabled, priority = excluded.priority, policy = excluded.policy`,
		r.ID, r.WorkspaceID, r.Name, r.Provider, r.Instance, r.Label, r.Author, r.Assignee,
		r.AgentID, r.GroupID, r.Repo, auto, r.Policy, enabled, r.Priority, ms(r.CreatedAt))
	return err
}

// DeleteAssignmentRule removes a rule.
func (s *Store) DeleteAssignmentRule(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM assignment_rules WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// SetTaskAutoDispatched records that a rule already dispatched a task.
func (s *Store) SetTaskAutoDispatched(ctx context.Context, id string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET auto_dispatched_at_ms = ? WHERE id = ?`, ms(at), id)
	return err
}
