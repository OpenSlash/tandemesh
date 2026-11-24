package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// AddTaskEvent appends a human or system action to a task timeline.
func (s *Store) AddTaskEvent(ctx context.Context, e *domain.TaskEvent) error {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	payload, _ := json.Marshal(e.Payload)
	_, err := s.db.ExecContext(ctx, `INSERT INTO task_events (id, task_id, actor, type, payload, created_at_ms)
		VALUES (?,?,?,?,?,?)`, e.ID, e.TaskID, e.Actor, e.Type, string(payload), ms(e.CreatedAt))
	return err
}

// ListTaskEvents returns task events in creation order.
func (s *Store) ListTaskEvents(ctx context.Context, taskID string) ([]*domain.TaskEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, task_id, actor, type, payload, created_at_ms
		FROM task_events WHERE task_id = ? ORDER BY created_at_ms ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.TaskEvent
	for rows.Next() {
		var (
			e       domain.TaskEvent
			payload string
			created int64
		)
		if err := rows.Scan(&e.ID, &e.TaskID, &e.Actor, &e.Type, &payload, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(payload), &e.Payload)
		e.CreatedAt = fromMS(created)
		out = append(out, &e)
	}
	return out, rows.Err()
}
