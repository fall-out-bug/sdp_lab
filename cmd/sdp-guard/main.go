package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"sdp_dev/internal/guard"
)

func main() {
	ws := flag.String("ws", "", "Workstream ID (e.g. 00-023-01)")
	cached := flag.Bool("cached", false, "Use git diff --cached (staged) instead of HEAD~1")
	flag.Parse()

	if *ws == "" {
		fmt.Fprintln(os.Stderr, "error: --ws is required")
		flag.Usage()
		os.Exit(1)
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	projectRoot, err := findProjectRoot(wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	verdict, err := guard.CheckScope(projectRoot, *ws, *cached)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(verdict.Warnings) > 0 {
		for _, w := range verdict.Warnings {
			fmt.Fprintf(os.Stderr, "WARN: %s (allowlisted)\n", w)
		}
	}

	if verdict.Pass {
		os.Exit(0)
	}

	for _, v := range verdict.Violations {
		fmt.Fprintf(os.Stderr, "SCOPE VIOLATION: %s\n", v)
	}
	fmt.Fprintf(os.Stderr, "out-of-scope changes detected (%d files)\n", len(verdict.Violations))
	os.Exit(1)
}

func findProjectRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for d := abs; d != "" && d != "/"; d = filepath.Dir(d) {
		check := filepath.Join(d, "docs", "workstreams", "backlog")
		if ents, _ := filepath.Glob(filepath.Join(check, "*.md")); len(ents) > 0 {
			return d, nil
		}
	}
	return "", fmt.Errorf("project root not found (no docs/workstreams/backlog)")
}
