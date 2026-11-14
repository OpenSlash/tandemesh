package api

import (
	"net/http"
	"strconv"
	"time"
)

type usagePoint struct {
	Date   string  `json:"date"`
	Tokens int     `json:"tokens"`
	Cost   float64 `json:"cost"`
	Runs   int     `json:"runs"`
}

// usageTrend aggregates token and cost usage by day for the dashboard.
func (s *Server) usageTrend(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")
	if workspaceID != "" && !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	days := 14
	if value := r.URL.Query().Get("days"); value != "" {
		if n, err := strconv.Atoi(value); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}
	runs, err := s.store.ListRuns(r.Context(), workspaceID, "", 500)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -(days - 1)).Truncate(24 * time.Hour)
	byDay := map[string]*usagePoint{}
	for i := 0; i < days; i++ {
		key := start.AddDate(0, 0, i).Format("2006-01-02")
		byDay[key] = &usagePoint{Date: key}
	}
	totalTokens := 0
	totalCost := 0.0
	totalRuns := 0
	for _, run := range runs {
		if run.CreatedAt.Before(start) {
			continue
		}
		key := run.CreatedAt.UTC().Format("2006-01-02")
		point := byDay[key]
		if point == nil {
			continue
		}
		tokens := run.InputTokens + run.OutputTokens + run.CacheReadTokens + run.CacheWriteTokens
		point.Tokens += tokens
		point.Cost += run.CostUSD
		point.Runs++
		totalTokens += tokens
		totalCost += run.CostUSD
		totalRuns++
	}
	points := make([]*usagePoint, 0, days)
	for i := 0; i < days; i++ {
		key := start.AddDate(0, 0, i).Format("2006-01-02")
		points = append(points, byDay[key])
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"points": points, "totalTokens": totalTokens, "totalCost": totalCost, "totalRuns": totalRuns,
	})
}
