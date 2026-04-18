package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/healthcheck"
)

func main() {
	jsonOut := flag.Bool("json", false, "output JSON")
	check := flag.String("check", "", "run only the named check")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	runner, err := healthcheck.NewRunner(healthcheck.Config{ProjectRoot: root, Only: *check})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	results := runner.Run(context.Background())

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
		os.Exit(exitCode(results))
	}

	fmt.Println("sdp-healthcheck")
	failed := 0
	for _, r := range results {
		icon := "✓"
		if r.Status == healthcheck.StatusFail {
			icon = "✗"
			failed++
		} else if r.Status == healthcheck.StatusWarn {
			icon = "!"
		}
		fmt.Printf("  %s %-14s %s\n", icon, r.Name, r.Detail)
	}
	fmt.Println()
	if failed > 0 {
		fmt.Printf("%d check(s) failed\n", failed)
	} else {
		fmt.Println("all checks passed")
	}
	os.Exit(exitCode(results))
}

func exitCode(results []healthcheck.CheckResult) int {
	for _, r := range results {
		if r.Status == healthcheck.StatusFail {
			return 1
		}
	}
	return 0
}
