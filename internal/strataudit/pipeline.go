package strataudit

import (
	"context"
	"fmt"
	"time"

	"sdp_dev/internal/strataudit/model"
	"sdp_dev/internal/strataudit/report"
)

// PipelineResult holds complete pipeline execution results.
type PipelineResult struct {
	Ingest   *IngestResult
	Extract  *ExtractResult
	Link     *LinkResult
	Analyze  *AnalyzeResult
	Duration time.Duration
}

// RunPipeline executes the full StratAudit pipeline: ingest → extract → link → analyze → report.
func RunPipeline(ctx context.Context, cfg *Config, store *SQLiteStore, llm *LLMClient) (*PipelineResult, error) {
	start := time.Now()
	result := &PipelineResult{}

	// Stage 1: Ingest
	ingestResult, err := Ingest(ctx, cfg, store)
	if err != nil {
		return nil, fmt.Errorf("ingest stage: %w", err)
	}
	result.Ingest = ingestResult
	saveCheckpoint(ctx, store, "ingest", stageStatus(ingestResult.Errors), ingestResult.New, ingestResult.Updated)

	// Stage 2: Extract
	extractResult, err := ExtractEntities(ctx, cfg, store, llm)
	if err != nil {
		return nil, fmt.Errorf("extract stage: %w", err)
	}
	result.Extract = extractResult
	saveCheckpoint(ctx, store, "extract", stageStatus(extractResult.Errors), extractResult.EntitiesExtracted, extractResult.Documents)

	// Stage 3: Link
	linkResult, err := LinkEntities(ctx, cfg, store, llm)
	if err != nil {
		return nil, fmt.Errorf("link stage: %w", err)
	}
	result.Link = linkResult
	saveCheckpoint(ctx, store, "link", stageStatus(linkResult.Errors), linkResult.TracesCreated, linkResult.CandidatesGenerated)

	// Stage 4: Analyze
	analyzeResult, err := Analyze(ctx, cfg, store)
	if err != nil {
		return nil, fmt.Errorf("analyze stage: %w", err)
	}
	result.Analyze = analyzeResult
	saveCheckpoint(ctx, store, "analyze", stageStatus(analyzeResult.Errors), analyzeResult.Findings, 0)

	// Stage 5: Report
	rpt, err := BuildReport(ctx, cfg, store)
	if err != nil {
		return nil, fmt.Errorf("build report: %w", err)
	}

	outputDir := cfg.Output.Dir
	for _, format := range cfg.Output.Formats {
		switch format {
		case "html":
			if err := report.WriteHTML(rpt, outputDir); err != nil {
				return nil, fmt.Errorf("html report: %w", err)
			}
		case "json":
			if err := report.WriteJSON(rpt, outputDir); err != nil {
				return nil, fmt.Errorf("json report: %w", err)
			}
		}
	}
	saveCheckpoint(ctx, store, "report", "completed", 1, 0)

	result.Duration = time.Since(start)
	return result, nil
}

func stageStatus(errs []error) string {
	if len(errs) == 0 {
		return "completed"
	}
	return "partial"
}

func saveCheckpoint(ctx context.Context, store *SQLiteStore, stage, status string, count, count2 int) {
	now := time.Now()
	_ = store.SavePipelineState(ctx, model.PipelineState{
		ID:          fmt.Sprintf("ps_%s_%d", stage, now.UnixMilli()),
		Stage:       stage,
		Status:      status,
		Checkpoint:  fmt.Sprintf(`{"count":%d,"count2":%d}`, count, count2),
		StartedAt:   now,
		CompletedAt: now,
	})
}
