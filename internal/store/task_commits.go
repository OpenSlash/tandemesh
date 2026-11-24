package store

import (
	"context"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// UpsertTaskCommit links a commit to a task.
func (s *Store) UpsertTaskCommit(ctx context.Context, link *domain.TaskCommitLink) error {
	if link.CreatedAt.IsZero() {
		link.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO task_commits
		(id, task_id, workspace_id, provider, repo, sha, url, message, author, created_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(task_id, provider, repo, sha) DO UPDATE SET
			url = excluded.url, message = excluded.message, author = excluded.author`,
		link.ID, link.TaskID, link.WorkspaceID, link.Provider, link.Repo, link.SHA,
		link.URL, link.Message, link.Author, ms(link.CreatedAt))
	return err
}

// ListTaskCommits returns commits linked to a task.
func (s *Store) ListTaskCommits(ctx context.Context, taskID string) ([]*domain.TaskCommitLink, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, task_id, workspace_id, provider, repo, sha,
		url, message, author, created_at_ms FROM task_commits WHERE task_id = ? ORDER BY created_at_ms DESC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.TaskCommitLink
	for rows.Next() {
		var (
			link    domain.TaskCommitLink
			created int64
		)
		if err := rows.Scan(&link.ID, &link.TaskID, &link.WorkspaceID, &link.Provider, &link.Repo,
			&link.SHA, &link.URL, &link.Message, &link.Author, &created); err != nil {
			return nil, err
		}
		link.CreatedAt = fromMS(created)
		out = append(out, &link)
	}
	return out, rows.Err()
}

// DeleteTaskCommit removes one commit link.
func (s *Store) DeleteTaskCommit(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM task_commits WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
