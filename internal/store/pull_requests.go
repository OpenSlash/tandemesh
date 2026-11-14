package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// UpsertTaskPullRequest links a PR to a task.
func (s *Store) UpsertTaskPullRequest(ctx context.Context, pr *domain.TaskPullRequest) error {
	now := time.Now().UTC()
	if pr.CreatedAt.IsZero() {
		pr.CreatedAt = now
	}
	pr.UpdatedAt = now
	var backlinked any
	if pr.BacklinkedAt != nil {
		backlinked = ms(*pr.BacklinkedAt)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO task_pull_requests
		(id, task_id, workspace_id, provider, repo, number, url, title, state, branch, author, backlinked_at_ms, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(task_id, provider, repo, number) DO UPDATE SET
			url = excluded.url, title = excluded.title, state = excluded.state,
			branch = excluded.branch, author = excluded.author,
			backlinked_at_ms = COALESCE(excluded.backlinked_at_ms, task_pull_requests.backlinked_at_ms), updated_at_ms = excluded.updated_at_ms`,
		pr.ID, pr.TaskID, pr.WorkspaceID, pr.Provider, pr.Repo, pr.Number,
		pr.URL, pr.Title, pr.State, pr.Branch, pr.Author, backlinked, ms(pr.CreatedAt), ms(pr.UpdatedAt))
	return err
}

// ListTaskPullRequests returns linked pull requests for a task.
func (s *Store) ListTaskPullRequests(ctx context.Context, taskID string) ([]*domain.TaskPullRequest, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, task_id, workspace_id, provider, repo, number,
		url, title, state, branch, author, backlinked_at_ms, created_at_ms, updated_at_ms
		FROM task_pull_requests WHERE task_id = ? ORDER BY updated_at_ms DESC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.TaskPullRequest
	for rows.Next() {
		var (
			pr         domain.TaskPullRequest
			backlinked sql.NullInt64
			created    int64
			updated    int64
		)
		if err := rows.Scan(&pr.ID, &pr.TaskID, &pr.WorkspaceID, &pr.Provider, &pr.Repo, &pr.Number,
			&pr.URL, &pr.Title, &pr.State, &pr.Branch, &pr.Author, &backlinked, &created, &updated); err != nil {
			return nil, err
		}
		pr.BacklinkedAt = fromNull(backlinked)
		pr.CreatedAt = fromMS(created)
		pr.UpdatedAt = fromMS(updated)
		out = append(out, &pr)
	}
	return out, rows.Err()
}

// GetTaskPullRequest returns one PR link or nil.
func (s *Store) GetTaskPullRequest(ctx context.Context, id string) (*domain.TaskPullRequest, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, task_id, workspace_id, provider, repo, number,
		url, title, state, branch, author, backlinked_at_ms, created_at_ms, updated_at_ms
		FROM task_pull_requests WHERE id = ?`, id)
	var (
		pr         domain.TaskPullRequest
		backlinked sql.NullInt64
		created    int64
		updated    int64
	)
	err := row.Scan(&pr.ID, &pr.TaskID, &pr.WorkspaceID, &pr.Provider, &pr.Repo, &pr.Number,
		&pr.URL, &pr.Title, &pr.State, &pr.Branch, &pr.Author, &backlinked, &created, &updated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	pr.BacklinkedAt = fromNull(backlinked)
	pr.CreatedAt = fromMS(created)
	pr.UpdatedAt = fromMS(updated)
	return &pr, nil
}

// DeleteTaskPullRequest unlinks one PR.
func (s *Store) DeleteTaskPullRequest(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM task_pull_requests WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
