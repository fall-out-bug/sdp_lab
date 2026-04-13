package model

import "time"

type LLMInvocation struct {
	ID                string
	Stage             string
	Model             string
	PromptHash        string
	Metadata          map[string]string
	TokensIn          int
	TokensOut         int
	CostUSD           float64
	DurationMs        int
	Cached            bool
	ContentSource     string
	ResponseContent   string
	ResponseReasoning string
	Error             string
	CreatedAt         time.Time
}

type LLMCacheEntry struct {
	PromptHash string
	Model      string
	Response   string
	TokensIn   int
	TokensOut  int
	CreatedAt  time.Time
}
