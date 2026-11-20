package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) usageSummary(w http.ResponseWriter, r *http.Request) {
	s.usageTrend(w, r)
}
func (s *Server) usageTrend(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 365 {
		days = 30
	}
	from := time.Now().UTC().AddDate(0, 0, -days)
	items, err := s.store.UsageByAgent(r.Context(), workspaceID, from)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	summary := domain.UsageSummary{From: from.UnixMilli(), To: time.Now().UnixMilli(), Items: items}
	for _, item := range items {
		summary.Runs += item.Runs
		summary.InputTokens += item.InputTokens
		summary.OutputTokens += item.OutputTokens
		summary.CacheReadTokens += item.CacheReadTokens
		summary.CacheWriteTokens += item.CacheWriteTokens
	}
	writeJSON(w, http.StatusOK, summary)
}
