package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sdp_dev/internal/metrics"
	"time"
)

func runMetrics(args []string) {
	fs := flag.NewFlagSet("metrics", flag.ExitOnError)
	format := fs.String("format", "json", "output format: json, text")
	output := fs.String("output", "", "write report to this directory")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp metrics [--format json|text] [--output DIR] <repo-path>")
		os.Exit(2)
	}
	repoPath := fs.Arg(0)

	switch *format {
	case "json", "text":
	default:
		fmt.Fprintf(os.Stderr, "error: unknown format %q (use json or text)\n", *format)
		os.Exit(2)
	}

	start := time.Now()
	data, err := metrics.Collect(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: metrics collection failed: %v\n", err)
		os.Exit(1)
	}
	data = metrics.Filter(data)

	report := metrics.MetricsReport{
		Version:         "1.0.0",
		GeneratedAt:     start.UTC(),
		RepoPath:        repoPath,
		DurationMs:      time.Since(start).Milliseconds(),
		CommitsAnalyzed: len(data.Commits),
	}

	if len(data.Commits) > 0 {
		report.Period = metrics.TimePeriod{
			From: data.Commits[len(data.Commits)-1].Date,
			To:   data.Commits[0].Date,
		}
	}

	switch *format {
	case "json":
		out, jerr := json.MarshalIndent(report, "", "  ")
		if jerr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", jerr)
			os.Exit(1)
		}
		fmt.Print(string(out) + "\n")
	case "text":
		fmt.Fprintf(os.Stdout, " %s — %d commits analyzed\n", repoPath, report.CommitsAnalyzed)
		fmt.Fprintf(os.Stdout, " Tags: %d | Branches: %d | Merges: %d\n",
			len(data.Tags), len(data.Branches), data.MergeCount)
		fmt.Fprintf(os.Stdout, " Duration: %dms\n", report.DurationMs)
	}

	if *output != "" {
		if err := os.MkdirAll(*output, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not create output dir: %v\n", err)
		} else {
			b, _ := json.MarshalIndent(report, "", "  ")
			path := *output + "/report.json"
			if err := os.WriteFile(path, b, 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not write report: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "artifact: %s\n", path)
			}
		}
	}
}
