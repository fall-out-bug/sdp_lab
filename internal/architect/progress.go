package architect

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// ProgressReporter writes pipeline progress to stderr with optional verbose mode.
type ProgressReporter struct {
	w       io.Writer
	verbose bool
	mu      sync.Mutex
	start   time.Time
	stages  []stageInfo
}

type stageInfo struct {
	Stage     PipelineStage
	Msg       string
	Timestamp time.Time
	Timing    *ExtractorTiming
}

// NewProgressReporter creates a reporter that writes to stderr.
func NewProgressReporter(verbose bool) *ProgressReporter {
	return &ProgressReporter{
		w:       os.Stderr,
		verbose: verbose,
		start:   time.Now(),
		stages:  make([]stageInfo, 0),
	}
}

// Callback returns a ProgressCallback compatible with Pipeline.
func (r *ProgressReporter) Callback() ProgressCallback {
	return func(stage PipelineStage, msg string, timing *ExtractorTiming) {
		r.mu.Lock()
		defer r.mu.Unlock()

		info := stageInfo{
			Stage:     stage,
			Msg:       msg,
			Timestamp: time.Now(),
			Timing:    timing,
		}
		r.stages = append(r.stages, info)

		elapsed := time.Since(r.start).Round(time.Millisecond)
		stageLabel := r.stageLabel(stage)

		if timing != nil && r.verbose {
			status := "OK"
			if !timing.Success {
				status = "FAIL"
			}
			fmt.Fprintf(r.w, "[architect] %s [%s] %s (%s) -- %s\n",
				elapsed, stageLabel, timing.Name, timing.Duration.Round(time.Millisecond), status)
		} else {
			fmt.Fprintf(r.w, "[architect] %s [%s] %s\n", elapsed, stageLabel, msg)
		}
	}
}

// Report writes a single message to stderr (for CLI layer use).
func (r *ProgressReporter) Report(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	elapsed := time.Since(r.start).Round(time.Millisecond)
	fmt.Fprintf(r.w, "[architect] %s %s\n", elapsed, msg)
}

// Summary prints a final summary of all stages.
func (r *ProgressReporter) Summary() {
	r.mu.Lock()
	defer r.mu.Unlock()

	fmt.Fprintf(r.w, "\n--- Pipeline Summary ---\n")
	fmt.Fprintf(r.w, "Total time: %s\n", time.Since(r.start).Round(time.Millisecond))

	stageCounts := make(map[PipelineStage]int)
	for _, s := range r.stages {
		stageCounts[s.Stage]++
	}

	allStages := []PipelineStage{StageExtract, StageAssemble, StageFilter, StageEnrich, StageModel, StageOutput}
	for _, stage := range allStages {
		if count, ok := stageCounts[stage]; ok {
			fmt.Fprintf(r.w, "  %s: %d events\n", r.stageLabel(stage), count)
		}
	}

	if r.verbose {
		fmt.Fprintf(r.w, "\nExtractor Timing:\n")
		for _, s := range r.stages {
			if s.Timing != nil {
				fmt.Fprintf(r.w, "  %-25s %s\n", s.Timing.Name, s.Timing.Duration.Round(time.Millisecond))
			}
		}
	}
}

func (r *ProgressReporter) stageLabel(stage PipelineStage) string {
	switch stage {
	case StageExtract:
		return "EXTRACT"
	case StageAssemble:
		return "ASSEMBLE"
	case StageFilter:
		return "FILTER"
	case StageEnrich:
		return "ENRICH"
	case StageModel:
		return "MODEL"
	case StageOutput:
		return "OUTPUT"
	default:
		return strings.ToUpper(string(stage))
	}
}
