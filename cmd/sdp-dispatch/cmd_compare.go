package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"sdp_dev/internal/dispatch"
)

func runCompare() error {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	task := fs.String("task", "feature", "Task type")
	lang := fs.String("lang", "go", "Language")
	projectRoot := fs.String("project", "", "Project root (default: cwd)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	root, err := resolveProjectRoot(*projectRoot)
	if err != nil {
		return err
	}

	store := dispatch.NewProfileStore(root)
	profiles, err := store.LoadAll()
	if err != nil {
		return fmt.Errorf("loading profiles: %w", err)
	}

	if len(profiles) == 0 {
		fmt.Println("No profiles found. Run 'sdp-dispatch bench' to generate profiles.")
		return nil
	}

	var results []dispatch.BenchResult
	for _, p := range profiles {
		key := fmt.Sprintf("%s:%s", *task, *lang)
		cap, hasCap := p.Capabilities[key]

		var dur time.Duration
		var testsPassed, testsTotal int
		if hasCap {
			dur = time.Duration(cap.AvgDuration * float64(time.Minute))
			testsTotal = 10
			testsPassed = int(cap.TestPassRate * float64(testsTotal))
		}

		results = append(results, dispatch.BenchResult{
			Harness:     p.Harness,
			Provider:    p.Provider,
			Model:       p.Model,
			Task:        *task,
			TaskType:    *task,
			Language:    *lang,
			Duration:    dur,
			TestsTotal:  testsTotal,
			TestsPassed: testsPassed,
			Timestamp:   time.Now().UTC(),
		})
	}

	ranked := dispatch.RankBenchResults(results)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(ranked)
	}

	fmt.Print(dispatch.FormatCompareTable(ranked))
	return nil
}
