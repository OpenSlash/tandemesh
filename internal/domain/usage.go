package domain

type UsageRow struct {
	AgentID          string `json:"agentId"`
	Model            string `json:"model"`
	Runs             int    `json:"runs"`
	InputTokens      int    `json:"inputTokens"`
	OutputTokens     int    `json:"outputTokens"`
	CacheReadTokens  int    `json:"cacheReadTokens"`
	CacheWriteTokens int    `json:"cacheWriteTokens"`
}

type UsageSummary struct {
	From  int64      `json:"from"`
	To    int64      `json:"to"`
	Items []UsageRow `json:"items"`
	UsageRow
}
type UsagePoint struct {
	Date   string  `json:"date"`
	Tokens int     `json:"tokens"`
	Cost   float64 `json:"cost"`
	Runs   int     `json:"runs"`
}

type UsageTrend struct {
	Points      []UsagePoint `json:"points"`
	TotalTokens int          `json:"totalTokens"`
	TotalCost   float64      `json:"totalCost"`
	TotalRuns   int          `json:"totalRuns"`
}
