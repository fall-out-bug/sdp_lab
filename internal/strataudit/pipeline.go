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

// PipelineOpts controls pipeline execution.
type PipelineOpts struct {
	Resume bool // auto-skip stages with "completed" checkpoint
}

// isStageCompleted checks if a pipeline stage has a completed checkpoint.
func isStageCompleted(ctx context.Context, store *SQLiteStore, stage string) bool {
	state, err := store.LoadPipelineState(ctx, stage)
	if err != nil || state == nil {
		return false
	}
	return state.Status == "completed"
}

// RunPipeline executes the full StratAudit pipeline: ingest → extract → link → analyze → report.
func RunPipeline(ctx context.Context, cfg *Config, store *SQLiteStore, runtime ModelRuntime, opts PipelineOpts) (*PipelineResult, error) {
	start := time.Now()
	result := &PipelineResult{}

	// Stage 1: Ingest
	if opts.Resume && isStageCompleted(ctx, store, "ingest") {
		slog.Info("pipeline: resume — skipping ingest")
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
	if opts.Resume && isStageCompleted(ctx, store, "extract") {
		slog.Info("pipeline: resume — skipping extract")
		result.Extract = &ExtractResult{}
	} else {
		slog.Info("pipeline: starting extract")
		extractResult, err := ExtractEntities(ctx, cfg, store, runtime)
		if err != nil {
			return nil, fmt.Errorf("extract stage: %w", err)
		}
		result.Extract = extractResult
		saveExtractCheckpoint(ctx, store, extractResult)
		slog.Info("pipeline: extract done",
			"entities", extractResult.EntitiesExtracted,
			"verified", extractResult.VerifiedEntities,
			"suspect", extractResult.SuspectEntities,
			"rejected", extractResult.RejectedEntities,
			"docs", extractResult.Documents,
			"errors", len(extractResult.Errors))
	}

	// Stage 3: Link
	if opts.Resume && isStageCompleted(ctx, store, "link") {
		slog.Info("pipeline: resume — skipping link")
		result.Link = &LinkResult{}
	} else {
		slog.Info("pipeline: starting link")
		linkResult, err := LinkEntities(ctx, cfg, store, runtime)
		if err != nil {
			return nil, fmt.Errorf("link stage: %w", err)
		}
		result.Link = linkResult
		saveCheckpoint(ctx, store, "link", stageStatus(linkResult.Errors), linkResult.TracesCreated, linkResult.CandidatesGenerated)
		slog.Info("pipeline: link done", "traces", linkResult.TracesCreated, "candidates", linkResult.CandidatesGenerated, "pairs", linkResult.Pairs, "errors", len(linkResult.Errors))
	}

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

func saveExtractCheckpoint(ctx context.Context, store *SQLiteStore, result *ExtractResult) {
	now := time.Now()
	checkpoint := fmt.Sprintf(
		`{"verified":%d,"suspect":%d,"rejected":%d,"documents":%d,"saved":%d}`,
		result.VerifiedEntities,
		result.SuspectEntities,
		result.RejectedEntities,
		result.Documents,
		result.EntitiesExtracted,
	)
	_ = store.SavePipelineState(ctx, model.PipelineState{
		ID:          fmt.Sprintf("ps_extract_%d", now.UnixMilli()),
		Stage:       "extract",
		Status:      stageStatus(result.Errors),
		Checkpoint:  checkpoint,
		StartedAt:   now,
		CompletedAt: now,
	})
}
