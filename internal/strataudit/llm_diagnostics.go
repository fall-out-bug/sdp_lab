package strataudit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/strataudit/model"
)

type llmDiagnosticsSummary struct {
	Total             int            `json:"total"`
	ByStage           map[string]int `json:"by_stage,omitempty"`
	Cached            int            `json:"cached"`
	ReasoningFallback int            `json:"reasoning_fallback"`
	Errors            int            `json:"errors"`
}

type llmDiagnosticsReport struct {
	GeneratedAt string                `json:"generated_at"`
	OutputDir   string                `json:"output_dir"`
	Summary     llmDiagnosticsSummary `json:"summary"`
	Invocations []model.LLMInvocation `json:"invocations"`
}

func recordLLMInvocation(ctx context.Context, store *SQLiteStore, req LLMRequest, resp *LLMResponse, err error) {
	if store == nil {
		return
	}

	now := time.Now().UTC()
	inv := model.LLMInvocation{
		ID:        fmt.Sprintf("llm_%d_%s", now.UnixNano(), sha256Hash([]byte(req.Stage + "|" + req.Model + "|" + req.User))[:10]),
		Stage:     req.Stage,
		Model:     req.Model,
		Metadata:  cloneStringMap(req.Metadata),
		CreatedAt: now,
	}
	if resp != nil {
		inv.PromptHash = resp.PromptHash
		inv.TokensIn = resp.TokensIn
		inv.TokensOut = resp.TokensOut
		inv.DurationMs = int(resp.DurationMs)
		inv.Cached = resp.Cached
		inv.ContentSource = resp.ContentSource
		inv.ResponseContent = resp.Content
		inv.ResponseReasoning = resp.Reasoning
	}
	if err != nil {
		inv.Error = err.Error()
	}

	_ = store.SaveLLMInvocation(ctx, inv)

	if resp != nil && resp.PromptHash != "" && resp.Content != "" {
		_ = store.SaveLLMCacheEntry(ctx, model.LLMCacheEntry{
			PromptHash: resp.PromptHash,
			Model:      req.Model,
			Response:   resp.Content,
			TokensIn:   resp.TokensIn,
			TokensOut:  resp.TokensOut,
			CreatedAt:  now,
		})
	}
}

func WriteLLMDiagnostics(ctx context.Context, cfg *Config, store *SQLiteStore) error {
	if store == nil {
		return nil
	}
	invocations, err := store.AllLLMInvocations(ctx)
	if err != nil {
		return fmt.Errorf("load llm invocations: %w", err)
	}

	outputDir := ".strataudit"
	if cfg != nil && cfg.Output.Dir != "" {
		outputDir = cfg.Output.Dir
	}

	summary := llmDiagnosticsSummary{
		Total:   len(invocations),
		ByStage: make(map[string]int),
	}
	for _, inv := range invocations {
		summary.ByStage[inv.Stage]++
		if inv.Cached {
			summary.Cached++
		}
		if inv.ContentSource == "reasoning" {
			summary.ReasoningFallback++
		}
		if inv.Error != "" {
			summary.Errors++
		}
	}

	report := llmDiagnosticsReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		OutputDir:   outputDir,
		Summary:     summary,
		Invocations: invocations,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal llm diagnostics: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create llm diagnostics dir: %w", err)
	}
	path := filepath.Join(outputDir, "llm_diagnostics.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write llm diagnostics: %w", err)
	}
	return nil
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
