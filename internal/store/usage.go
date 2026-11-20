package store

import (
	"context"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Store) UsageByAgent(ctx context.Context, workspaceID string, from time.Time) ([]domain.UsageRow, error) {
	query := `SELECT agent_id, model, COUNT(*), COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0),
		COALESCE(SUM(cache_read_tokens),0), COALESCE(SUM(cache_write_tokens),0)
		FROM runs WHERE created_at_ms >= ?`
	args := []any{ms(from)}
	if workspaceID != "" {
		query += " AND workspace_id = ?"
		args = append(args, workspaceID)
	}
	query += " GROUP BY agent_id, model ORDER BY SUM(input_tokens + output_tokens) DESC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.UsageRow
	for rows.Next() {
		var row domain.UsageRow
		if err := rows.Scan(&row.AgentID, &row.Model, &row.Runs, &row.InputTokens, &row.OutputTokens, &row.CacheReadTokens, &row.CacheWriteTokens); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
