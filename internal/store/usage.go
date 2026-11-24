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
func (s *Store) UsageTrend(ctx context.Context, workspaceID string, from time.Time) (domain.UsageTrend, error) {
	query := `SELECT created_at_ms, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, cost_usd FROM runs WHERE created_at_ms >= ?`
	args := []any{ms(from)}
	if workspaceID != "" {
		query += " AND workspace_id = ?"
		args = append(args, workspaceID)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return domain.UsageTrend{}, err
	}
	defer rows.Close()

	byDay := map[string]*domain.UsagePoint{}
	for rows.Next() {
		var created int64
		var input, output, cacheRead, cacheWrite int
		var cost float64
		if err := rows.Scan(&created, &input, &output, &cacheRead, &cacheWrite, &cost); err != nil {
			return domain.UsageTrend{}, err
		}
		day := time.UnixMilli(created).UTC().Format("2006-01-02")
		point := byDay[day]
		if point == nil {
			point = &domain.UsagePoint{Date: day}
			byDay[day] = point
		}
		point.Tokens += input + output + cacheRead + cacheWrite
		point.Cost += cost
		point.Runs++
	}
	if err := rows.Err(); err != nil {
		return domain.UsageTrend{}, err
	}

	var trend domain.UsageTrend
	for day := from.UTC().Truncate(24 * time.Hour); !day.After(time.Now().UTC()); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		point := domain.UsagePoint{Date: key}
		if stored := byDay[key]; stored != nil {
			point = *stored
		}
		trend.Points = append(trend.Points, point)
		trend.TotalTokens += point.Tokens
		trend.TotalCost += point.Cost
		trend.TotalRuns += point.Runs
	}
	return trend, nil
}
