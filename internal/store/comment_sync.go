package store

import (
	"context"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Store) ListTasksForComments(ctx context.Context, provider, instance string, before time.Time, limit int) ([]*domain.Task, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, work_id, work_type, provider, instance
		FROM tasks WHERE provider = ? AND instance = ? AND comments_synced_at_ms < ?
		ORDER BY comments_synced_at_ms ASC LIMIT ?`, provider, instance, ms(before), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Task
	for rows.Next() {
		t := &domain.Task{}
		if err := rows.Scan(&t.ID, &t.WorkID, &t.WorkType, &t.Provider, &t.Instance); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) TouchTaskComments(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET comments_synced_at_ms = ? WHERE id = ?`, ms(time.Now()), id)
	return err
}
