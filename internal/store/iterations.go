package store

import (
	"context"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// CreateIteration inserts a sprint-like period.
func (s *Store) CreateIteration(ctx context.Context, it *domain.Iteration) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO iterations
		(id, workspace_id, name, start_date, end_date, status, created_at_ms)
		VALUES (?,?,?,?,?,?,?)`,
		it.ID, it.WorkspaceID, it.Name, it.StartDate, it.EndDate, it.Status, ms(time.Now()))
	return err
}

// ListIterations returns iterations in a workspace.
func (s *Store) ListIterations(ctx context.Context, workspaceID string) ([]*domain.Iteration, error) {
	q := `SELECT id, workspace_id, name, start_date, end_date, status, created_at_ms,
		(SELECT COUNT(*) FROM tasks WHERE tasks.iteration_id = iterations.id),
		(SELECT COUNT(*) FROM tasks WHERE tasks.iteration_id = iterations.id AND tasks.status IN ('done', 'cancelled'))
		FROM iterations`
	var args []any
	if workspaceID != "" {
		q += " WHERE workspace_id = ?"
		args = append(args, workspaceID)
	}
	q += " ORDER BY start_date DESC, created_at_ms DESC"
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Iteration
	for rows.Next() {
		var (
			it      domain.Iteration
			created int64
		)
		if err := rows.Scan(&it.ID, &it.WorkspaceID, &it.Name, &it.StartDate, &it.EndDate, &it.Status, &created,
			&it.TaskCount, &it.DoneCount); err != nil {
			return nil, err
		}
		it.CreatedAt = fromMS(created)
		out = append(out, &it)
	}
	return out, rows.Err()
}

// SetTaskDoneAt records when a task entered done.
func (s *Store) SetTaskDoneAt(ctx context.Context, id string, at *time.Time) error {
	var value any
	if at != nil {
		value = ms(*at)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET done_at_ms = ? WHERE id = ?`, value, id)
	return err
}

// GetIteration returns one iteration with task counts.
func (s *Store) GetIteration(ctx context.Context, id string) (*domain.Iteration, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, name, start_date, end_date, status, created_at_ms,
		(SELECT COUNT(*) FROM tasks WHERE tasks.iteration_id = iterations.id),
		(SELECT COUNT(*) FROM tasks WHERE tasks.iteration_id = iterations.id AND tasks.status IN ('done', 'cancelled'))
		FROM iterations WHERE id = ?`, id)
	var (
		it      domain.Iteration
		created int64
	)
	err := row.Scan(&it.ID, &it.WorkspaceID, &it.Name, &it.StartDate, &it.EndDate, &it.Status, &created,
		&it.TaskCount, &it.DoneCount)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	it.CreatedAt = fromMS(created)
	return &it, nil
}

// IterationBurndown builds a daily remaining-task series.
func (s *Store) IterationBurndown(ctx context.Context, iterationID string) (*domain.Iteration, []domain.BurndownPoint, domain.IterationStats, error) {
	it, err := s.GetIteration(ctx, iterationID)
	if err != nil || it == nil {
		return it, nil, domain.IterationStats{}, err
	}
	tasks, err := s.ListTasks(ctx, TaskFilter{Workspace: it.WorkspaceID, Iteration: iterationID, Limit: 200})
	if err != nil {
		return nil, nil, domain.IterationStats{}, err
	}
	stats := domain.IterationStats{Total: len(tasks)}
	for _, task := range tasks {
		switch task.Status {
		case domain.TaskDone, domain.TaskCancelled:
			stats.Done++
		case domain.TaskOpen:
			stats.Open++
		default:
			stats.InProgress++
		}
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	start, err := time.Parse("2006-01-02", it.StartDate)
	if err != nil {
		start = today.AddDate(0, 0, -13)
	}
	end, err := time.Parse("2006-01-02", it.EndDate)
	if err != nil || end.After(today) {
		end = today
	}
	if start.After(end) {
		start = end
	}

	var points []domain.BurndownPoint
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		dayEnd := day.AddDate(0, 0, 1)
		remaining := 0
		for _, task := range tasks {
			if domain.TaskClosed(task.Status) {
				if task.DoneAt == nil || task.DoneAt.Before(dayEnd) {
					continue
				}
			}
			remaining++
		}
		points = append(points, domain.BurndownPoint{
			Date: day.Format("2006-01-02"), Remaining: remaining, Total: len(tasks),
		})
	}
	return it, points, stats, nil
}

// SetIterationStatus updates the status of an iteration.
func (s *Store) SetIterationStatus(ctx context.Context, id, status string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE iterations SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// DeleteIteration removes an iteration and clears task assignments.
func (s *Store) DeleteIteration(ctx context.Context, id string) (bool, error) {
	if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET iteration_id = '' WHERE iteration_id = ?`, id); err != nil {
		return false, err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM iterations WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
