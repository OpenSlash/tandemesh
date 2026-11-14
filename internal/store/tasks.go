package store

import (
	"context"
	"time"
)

// SetTaskExternalAssignee updates the platform assignee projection.
func (s *Store) SetTaskExternalAssignee(ctx context.Context, id, assignee string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET assignee = ?, updated_at_ms = ? WHERE id = ?`,
		assignee, ms(time.Now()), id)
	return err
}
