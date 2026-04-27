// Command sdp-ft-dataset assembles the train/eval JSONL pair for the F133
// complexity-classifier fine-tune.
//
// Usage:
//
//	sdp-ft-dataset --out internal/dispatch/training [--seed 42]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/fall-out-bug/sdp_lab/internal/finetune"
)

func main() {
	var (
		out       = flag.String("out", "internal/dispatch/training", "output directory")
		wsDir     = flag.String("ws", "docs/workstreams/backlog", "workstream backlog directory")
		beadsPath = flag.String("beads", ".beads/issues.jsonl", "beads issues JSONL")
		seed      = flag.Int64("seed", 42, "shuffle seed (deterministic split)")
		evalRatio = flag.Float64("eval-ratio", 0.2, "fraction of samples held out for eval")
		minSize   = flag.Int("min", 50, "minimum total samples after dedup")
	)
	flag.Parse()

	split, report, err := finetune.Build(finetune.BuildOptions{
		WSDir:      *wsDir,
		BeadsPath:  *beadsPath,
		EvalRatio:  *evalRatio,
		Seed:       *seed,
		MinSamples: *minSize,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "build dataset: %v\n", err)
		os.Exit(1)
	}

	trainPath := filepath.Join(*out, "train.jsonl")
	evalPath := filepath.Join(*out, "eval.jsonl")
	reportPath := filepath.Join(*out, "build_report.json")

	if err := finetune.WriteJSONL(trainPath, split.Train); err != nil {
		fmt.Fprintf(os.Stderr, "write train: %v\n", err)
		os.Exit(1)
	}
	if err := finetune.WriteJSONL(evalPath, split.Eval); err != nil {
		fmt.Fprintf(os.Stderr, "write eval: %v\n", err)
		os.Exit(1)
	}

	if err := writeReport(reportPath, split, report); err != nil {
		fmt.Fprintf(os.Stderr, "write report: %v\n", err)
		os.Exit(1)
	}

	printSummary(split, report, trainPath, evalPath, reportPath)
}

func writeReport(path string, split finetune.Split, report finetune.BuildReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload := map[string]any{
		"ws_loaded":          report.WSLoaded,
		"beads_loaded":       report.BeadsLoaded,
		"ws_skipped":         report.WSSkipped,
		"beads_skipped":      report.BeadsSkipped,
		"duplicates_dropped": report.Duplicates,
		"total_after_dedup":  report.TotalAfterDedup,
		"label_distribution": report.LabelDistribution,
		"train_count":        len(split.Train),
		"eval_count":         len(split.Eval),
		"real_share": map[string]int{
			"train_real": countReal(split.Train),
			"eval_real":  countReal(split.Eval),
		},
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func countReal(samples []finetune.Sample) int {
	n := 0
	for _, s := range samples {
		if s.Meta.Real {
			n++
		}
	}
	return n
}

func printSummary(split finetune.Split, report finetune.BuildReport, trainPath, evalPath, reportPath string) {
	fmt.Printf("ws loaded:      %d\n", report.WSLoaded)
	fmt.Printf("beads loaded:   %d\n", report.BeadsLoaded)
	fmt.Printf("duplicates:     %d\n", report.Duplicates)
	fmt.Printf("after dedup:    %d\n", report.TotalAfterDedup)
	fmt.Printf("train:          %d -> %s\n", len(split.Train), trainPath)
	fmt.Printf("eval:           %d -> %s\n", len(split.Eval), evalPath)
	fmt.Printf("report:         %s\n", reportPath)

	fmt.Println("\nlabel distribution (complexity/task_type/risk):")
	keys := make([]string, 0, len(report.LabelDistribution))
	for k := range report.LabelDistribution {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-30s %d\n", k, report.LabelDistribution[k])
	}
	fmt.Println("\nws skip reasons:")
	printMap(report.WSSkipped)
	fmt.Println("beads skip reasons:")
	printMap(report.BeadsSkipped)
}

func printMap(m map[string]int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-30s %d\n", k, m[k])
	}
}
