package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch"
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
		results = append(results, profileToBenchResult(p, *task, *lang))
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
