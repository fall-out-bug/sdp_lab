package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"sdp_dev/internal/eval"
	"sdp_dev/internal/evidence"
)

func main() {
	skill := flag.String("skill", "", "Skill name (e.g. oneshot). If empty, run all.")
	all := flag.Bool("all", false, "Run evals for all skills")
	projectRoot := flag.String("project-root", ".", "Project root")
	casesDir := flag.String("cases-dir", "", "Cases directory (default: internal/eval/cases)")
	flag.Parse()

	if *casesDir == "" {
		*casesDir = filepath.Join(*projectRoot, "internal", "eval", "cases")
	}

	absRoot, _ := filepath.Abs(*projectRoot)
	if err := evidence.ValidatePath(absRoot, ""); err != nil {
		fmt.Fprintf(os.Stderr, "project-root: %v\n", err)
		os.Exit(1)
	}
	absCases, _ := filepath.Abs(*casesDir)
	if err := evidence.ValidatePath(absCases, absRoot); err != nil {
		fmt.Fprintf(os.Stderr, "cases-dir: %v\n", err)
		os.Exit(1)
	}

	skillFilter := *skill
	if *all {
		skillFilter = ""
	}
	if !*all && skillFilter == "" {
		fmt.Fprintln(os.Stderr, "error: --skill <name> or --all required")
		flag.Usage()
		os.Exit(1)
	}

	results, err := eval.Run(absRoot, absCases, skillFilter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	passed := 0
	for _, r := range results {
		status := "FAIL"
		if r.Pass {
			status = "PASS"
			passed++
		}
		fmt.Printf("  %s: %s", r.Case, status)
		if !r.Pass && r.Reason != "" {
			fmt.Printf(" (%s)", r.Reason)
		}
		fmt.Println()
	}

	skillLabel := "all"
	if skillFilter != "" {
		skillLabel = skillFilter
	}
	fmt.Printf("\n%s: %d/%d passed\n", skillLabel, passed, len(results))
	if passed < len(results) {
		os.Exit(1)
	}
}
