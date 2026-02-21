package retrospective

import (
	"os"
	"path/filepath"
)

// Aggregator walks epic child tree and collects run/evidence paths.
type Aggregator struct {
	workDir string
}

// NewAggregator creates an Aggregator.
func NewAggregator(workDir string) *Aggregator {
	return &Aggregator{workDir: workDir}
}

// CollectPaths returns paths to .sdp/runs/, .sdp/evidence/, and intake.jsonl.
func (a *Aggregator) CollectPaths(epicID string) (runs []string, evidence []string, intakePath string) {
	runsDir := filepath.Join(a.workDir, ".sdp", "runs")
	evDir := filepath.Join(a.workDir, ".sdp", "evidence")
	intakePath = filepath.Join(a.workDir, ".sdp", "intake.jsonl")

	if entries, err := os.ReadDir(runsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				runs = append(runs, filepath.Join(runsDir, e.Name()))
			}
		}
	}
	if entries, err := os.ReadDir(evDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				evidence = append(evidence, filepath.Join(evDir, e.Name()))
			}
		}
	}
	_ = epicID
	return runs, evidence, intakePath
}
