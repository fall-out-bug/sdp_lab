//go:build sdp_experimental

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/evals"
	"github.com/fall-out-bug/sdp_lab/internal/evals/f165"
	"github.com/fall-out-bug/sdp_lab/internal/evidence"
)

func main() {
	skill := flag.String("skill", "", "Skill name (e.g. oneshot). If empty, run all.")
	all := flag.Bool("all", false, "Run evals for all skills")
	projectRoot := flag.String("project-root", ".", "Project root")
	casesDir := flag.String("cases-dir", "", "Cases directory (default: internal/eval/cases)")
	piReport := flag.Bool("prompt-injection-report", false, "Run F164 prompt-injection static/advisory report and exit")
	piLive := flag.Bool("prompt-injection-live", false, "Include live-provider prompt-injection eval status as advisory")
	indirectPIReport := flag.Bool("indirect-pi-report", false, "Run F165 indirect prompt-injection demo report and exit")
	indirectPIJSON := flag.Bool("indirect-pi-json", false, "Emit F165 report as JSON (default: text)")
	flag.Parse()

	if *casesDir == "" {
		*casesDir = filepath.Join(*projectRoot, "internal", "eval", "cases")
	}

	absRoot, _ := filepath.Abs(*projectRoot)
	if err := evidence.ValidatePath(absRoot, ""); err != nil {
		fmt.Fprintf(os.Stderr, "project-root: %v\n", err)
		os.Exit(1)
	}

	if *piReport {
		if err := runPromptInjectionReport(absRoot, *piLive); err != nil {
			fmt.Fprintf(os.Stderr, "prompt-injection-report: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *indirectPIReport {
		if err := runIndirectPIReport(absRoot, *indirectPIJSON); err != nil {
			fmt.Fprintf(os.Stderr, "indirect-pi-report: %v\n", err)
			os.Exit(1)
		}
		return
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

	results, err := evals.Run(absRoot, absCases, skillFilter)
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

func runIndirectPIReport(projectRoot string, asJSON bool) error {
	testdataDir := filepath.Join(projectRoot, "internal", "evals", "testdata", "indirect_pi")
	report, err := f165.GenerateReport(testdataDir)
	if err != nil {
		return fmt.Errorf("generate F165 report: %w", err)
	}
	if asJSON {
		data, err := f165.RenderReportJSON(report)
		if err != nil {
			return fmt.Errorf("render JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}
	fmt.Println(f165.RenderReportText(report))
	return nil
}

func runPromptInjectionReport(projectRoot string, includeLive bool) error {
	testCasesPath := filepath.Join(projectRoot, "docs", "security", "f164-prompt-injection-test-cases.md")
	testCases, err := os.ReadFile(testCasesPath)
	if err != nil {
		return fmt.Errorf("read test cases: %w", err)
	}
	if !strings.Contains(string(testCases), "PI-013") || !strings.Contains(string(testCases), "Prompt Bundle Supply Chain") {
		return fmt.Errorf("PI-013 supply-chain case missing from prompt-injection corpus docs")
	}

	checkScript := filepath.Join(projectRoot, "scripts", "prompt-injection-check.sh")
	cmd := exec.Command(checkScript)
	cmd.Dir = projectRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("PI-013 prompt surface check failed: %w\n%s", err, out)
	}

	fmt.Println("prompt-injection static/mock report")
	fmt.Println("  static corpus docs: PASS (PI-013 supply-chain case present)")
	fmt.Println("  prompt surface PI-013 check: PASS")
	fmt.Println("  mock trace regressions: PASS when go test ./internal/evals passes")
	if includeLive {
		fmt.Println("  live-provider eval: ADVISORY (run cmd/sdp-pi-eval manually or scheduled; not a PR gate)")
	} else {
		fmt.Println("  live-provider eval: ADVISORY_DEGRADED (skipped; no live credentials required for CI)")
	}
	return nil
}
