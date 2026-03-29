package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"sdp_dev/internal/dispatch"
)

func runBench() error {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	task := fs.String("task", "", "Task type (feature, bugfix, refactor)")
	lang := fs.String("lang", "go", "Language")
	harnessFilter := fs.String("harness", "", "Specific harness to bench (default: all)")
	projectRoot := fs.String("project", "", "Project root (default: cwd)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	if *task == "" {
		return fmt.Errorf("--task is required (feature, bugfix, refactor)")
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
		fmt.Println("No profiles found. Run 'sdp-dispatch profile' to create profiles first.")
		return nil
	}

	fmt.Fprintln(os.Stderr, "Bench scaffolding: real benchmarks require gastown. Showing profile-based estimates.")

	now := time.Now().UTC()
	var results []dispatch.BenchResult
	for _, p := range profiles {
		if *harnessFilter != "" && p.Harness != *harnessFilter {
			continue
		}

		key := fmt.Sprintf("%s:%s", *task, *lang)
		cap, hasCap := p.Capabilities[key]

		var dur time.Duration
		var testsPassed, testsTotal int
		if hasCap {
			dur = time.Duration(cap.AvgDuration * float64(time.Minute))
			// Estimate tests from pass rate: assume 10 total tests as baseline.
			testsTotal = 10
			testsPassed = int(cap.TestPassRate * float64(testsTotal))
		}

		results = append(results, dispatch.BenchResult{
			Harness:   p.Harness,
			Provider:  p.Provider,
			Model:     p.Model,
			Task:      *task,
			TaskType:  *task,
			Language:  *lang,
			Duration:  dur,
			Timestamp: now,

			TestsTotal:  testsTotal,
			TestsPassed: testsPassed,
		})
	}

	if len(results) == 0 {
		fmt.Println("No matching harnesses found.")
		return nil
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	fmt.Print(dispatch.FormatCompareTable(results))
	return nil
}
