// Package flaky provides utilities for detecting flaky tests and retrying
// flaky-prone operations. This is advisory-only (P2) — it never blocks CI.
//
// Detection works by parsing `go test -json` output from multiple runs and
// identifying tests that both pass and fail across those runs.
package flaky

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FlakeReport summarizes flake detection results for a single test.
type FlakeReport struct {
	Package   string `json:"package"`
	Test      string `json:"test"`
	PassCount int    `json:"pass_count"`
	FailCount int    `json:"fail_count"`
	TotalRuns int    `json:"total_runs"`
	IsFlaky   bool   `json:"is_flaky"`
}

// testResult captures the outcome of a single test execution from go test -json.
type testResult struct {
	Package string
	Test    string
	Action  string // "pass", "fail", "skip"
}

// DetectFlakes analyzes combined test output from multiple `go test` runs
// and returns reports for tests that both passed and failed.
//
// The testOutput parameter should contain the concatenated stdout from running
// `go test -json ./...` N times. Each line is a JSON object with at least
// {Time, Action, Package, Test} fields (standard go test -json format).
//
// The runs parameter indicates how many separate runs are represented in the
// output. It is used for informational purposes in the report (TotalRuns).
func DetectFlakes(testOutput string, runs int) []FlakeReport {
	// Track per-test outcomes: package.Test -> set of distinct actions seen
	type track struct {
		passes int
		fails  int
	}
	outcomes := make(map[string]*track)

	scanner := bufio.NewScanner(strings.NewReader(testOutput))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry struct {
			Action  string `json:"Action"`
			Package string `json:"Package"`
			Test    string `json:"Test"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // skip non-JSON lines
		}

		// We only care about leaf test results, not package-level summaries.
		// go test -json emits package-level entries with Test="".
		if entry.Test == "" {
			continue
		}

		action := strings.ToLower(entry.Action)
		if action != "pass" && action != "fail" {
			continue
		}

		key := entry.Package + "." + entry.Test
		t, ok := outcomes[key]
		if !ok {
			t = &track{}
			outcomes[key] = t
		}
		if action == "pass" {
			t.passes++
		} else {
			t.fails++
		}
	}

	// Build reports for tests that have both passes and failures.
	var reports []FlakeReport
	for key, t := range outcomes {
		pkg, test := splitKey(key)
		isFlaky := t.passes > 0 && t.fails > 0
		reports = append(reports, FlakeReport{
			Package:   pkg,
			Test:      test,
			PassCount: t.passes,
			FailCount: t.fails,
			TotalRuns: t.passes + t.fails,
			IsFlaky:   isFlaky,
		})
	}

	// Filter to only flaky tests for the return value.
	var flaky []FlakeReport
	for _, r := range reports {
		if r.IsFlaky {
			flaky = append(flaky, r)
		}
	}

	return flaky
}

// FormatReport produces a human-readable summary of flake detection results.
func FormatReport(reports []FlakeReport, runs int) string {
	if len(reports) == 0 {
		return fmt.Sprintf("No flaky tests detected across %d run(s).", runs)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Flaky tests detected (%d run(s), %d flaky test(s)):\n\n", runs, len(reports)))
	b.WriteString("  PACKAGE                                   TEST                           PASS  FAIL  TOTAL\n")
	b.WriteString("  -------                                   ----                           ----  ----  -----\n")
	for _, r := range reports {
		b.WriteString(fmt.Sprintf("  %-42s %-30s %4d  %4d  %5d\n",
			truncate(r.Package, 42),
			truncate(r.Test, 30),
			r.PassCount,
			r.FailCount,
			r.TotalRuns,
		))
	}
	b.WriteString("\nAdvisory: investigate flaky tests above. Not a CI gate.\n")
	return b.String()
}

// RetryFlaky executes testFunc up to maxRetries+1 times, returning nil on
// the first success. If all attempts fail, it returns the last error.
//
// This is useful for wrapping operations known to be unreliable (network
// calls, race-prone code, etc.) without silently swallowing errors.
func RetryFlaky(testFunc func() error, maxRetries int) error {
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		if err := testFunc(); err != nil {
			lastErr = err
			if i < maxRetries {
				// Brief backoff: 100ms, 200ms, 400ms...
				time.Sleep(time.Duration(100) * time.Millisecond << uint(i))
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("all %d attempt(s) failed, last error: %w", maxRetries+1, lastErr)
}

// splitKey splits "package.Test" back into (package, test).
// If there is no dot, the full string is returned as test with empty package.
func splitKey(key string) (string, string) {
	idx := strings.LastIndex(key, ".")
	if idx < 0 {
		return "", key
	}
	return key[:idx], key[idx+1:]
}

// truncate shortens s to maxLen with "..." if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
