package strataudit

import (
	"context"
	"fmt"
	"log/slog"
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

// PipelineOpts controls which stages to run.
type PipelineOpts struct {
	SkipIngest  bool
	SkipExtract bool
}

// RunPipeline executes the full StratAudit pipeline: ingest → extract → link → analyze → report.
func RunPipeline(ctx context.Context, cfg *Config, store *SQLiteStore, llm *LLMClient, opts PipelineOpts) (*PipelineResult, error) {
	start := time.Now()
	result := &PipelineResult{}

	// Stage 1: Ingest
	if opts.SkipIngest {
		slog.Info("pipeline: skipping ingest")
		result.Ingest = &IngestResult{}
	} else {
		slog.Info("pipeline: starting ingest")
		ingestResult, err := Ingest(ctx, cfg, store)
		if err != nil {
			return nil, fmt.Errorf("ingest stage: %w", err)
		}
		result.Ingest = ingestResult
		saveCheckpoint(ctx, store, "ingest", stageStatus(ingestResult.Errors), ingestResult.New, ingestResult.Updated)
		slog.Info("pipeline: ingest done", "new", ingestResult.New, "updated", ingestResult.Updated, "unchanged", ingestResult.Unchanged, "errors", len(ingestResult.Errors))
	}

	// Stage 2: Extract
	if opts.SkipExtract {
		slog.Info("pipeline: skipping extract")
		result.Extract = &ExtractResult{}
	} else {
		slog.Info("pipeline: starting extract")
		extractResult, err := ExtractEntities(ctx, cfg, store, llm)
		if err != nil {
			return nil, fmt.Errorf("extract stage: %w", err)
		}
		result.Extract = extractResult
		saveCheckpoint(ctx, store, "extract", stageStatus(extractResult.Errors), extractResult.EntitiesExtracted, extractResult.Documents)
		slog.Info("pipeline: extract done", "entities", extractResult.EntitiesExtracted, "docs", extractResult.Documents, "errors", len(extractResult.Errors))
	}

	// Stage 3: Link
	slog.Info("pipeline: starting link")
	linkResult, err := LinkEntities(ctx, cfg, store, llm)
	if err != nil {
		return nil, fmt.Errorf("link stage: %w", err)
	}
	result.Link = linkResult
	saveCheckpoint(ctx, store, "link", stageStatus(linkResult.Errors), linkResult.TracesCreated, linkResult.CandidatesGenerated)
	slog.Info("pipeline: link done", "traces", linkResult.TracesCreated, "candidates", linkResult.CandidatesGenerated, "pairs", linkResult.Pairs, "errors", len(linkResult.Errors))

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
