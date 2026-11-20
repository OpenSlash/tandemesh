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
