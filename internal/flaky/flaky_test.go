package flaky

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// helper to build go test -json lines.
func jsonLine(action, pkg, test string) string {
	return fmt.Sprintf(`{"Time":"2026-04-17T12:00:00Z","Action":"%s","Package":"%s","Test":"%s"}`, action, pkg, test)
}

func TestDetectFlakes_NoFlaky(t *testing.T) {
	// Two runs, all tests pass.
	output := strings.Join([]string{
		jsonLine("pass", "pkg/a", "TestFoo"),
		jsonLine("pass", "pkg/a", "TestBar"),
		jsonLine("pass", "pkg/a", "TestFoo"),
		jsonLine("pass", "pkg/a", "TestBar"),
	}, "\n")

	reports := DetectFlakes(output, 2)
	if len(reports) != 0 {
		t.Errorf("expected 0 flaky reports, got %d: %+v", len(reports), reports)
	}
}

func TestDetectFlakes_SingleFlaky(t *testing.T) {
	// TestFoo passes once, fails once. TestBar always passes.
	output := strings.Join([]string{
		jsonLine("pass", "pkg/a", "TestFoo"),
		jsonLine("pass", "pkg/a", "TestBar"),
		jsonLine("fail", "pkg/a", "TestFoo"),
		jsonLine("pass", "pkg/a", "TestBar"),
	}, "\n")

	reports := DetectFlakes(output, 2)
	if len(reports) != 1 {
		t.Fatalf("expected 1 flaky report, got %d", len(reports))
	}

	r := reports[0]
	if r.Package != "pkg/a" {
		t.Errorf("Package = %q, want %q", r.Package, "pkg/a")
	}
	if r.Test != "TestFoo" {
		t.Errorf("Test = %q, want %q", r.Test, "TestFoo")
	}
	if r.PassCount != 1 {
		t.Errorf("PassCount = %d, want 1", r.PassCount)
	}
	if r.FailCount != 1 {
		t.Errorf("FailCount = %d, want 1", r.FailCount)
	}
	if !r.IsFlaky {
		t.Error("IsFlaky = false, want true")
	}
}

func TestDetectFlakes_MultipleFlaky(t *testing.T) {
	output := strings.Join([]string{
		jsonLine("pass", "pkg/a", "TestA"),
		jsonLine("fail", "pkg/a", "TestB"),
		jsonLine("fail", "pkg/a", "TestA"),
		jsonLine("pass", "pkg/a", "TestB"),
	}, "\n")

	reports := DetectFlakes(output, 2)
	if len(reports) != 2 {
		t.Fatalf("expected 2 flaky reports, got %d", len(reports))
	}
}

func TestDetectFlakes_AlwaysFails(t *testing.T) {
	// Test that always fails is NOT flaky — it's consistently failing.
	output := strings.Join([]string{
		jsonLine("fail", "pkg/a", "TestX"),
		jsonLine("fail", "pkg/a", "TestX"),
		jsonLine("fail", "pkg/a", "TestX"),
	}, "\n")

	reports := DetectFlakes(output, 3)
	if len(reports) != 0 {
		t.Errorf("expected 0 flaky reports for always-failing test, got %d", len(reports))
	}
}

func TestDetectFlakes_SkippedIgnored(t *testing.T) {
	output := strings.Join([]string{
		jsonLine("pass", "pkg/a", "TestA"),
		jsonLine("skip", "pkg/a", "TestB"),
		jsonLine("fail", "pkg/a", "TestA"),
	}, "\n")

	reports := DetectFlakes(output, 2)
	if len(reports) != 1 {
		t.Fatalf("expected 1 flaky report (skip should be ignored), got %d", len(reports))
	}
	if reports[0].Test != "TestA" {
		t.Errorf("flaky test = %q, want TestA", reports[0].Test)
	}
}

func TestDetectFlakes_PackageLevelIgnored(t *testing.T) {
	// Package-level entries (Test="") should be ignored.
	output := strings.Join([]string{
		`{"Time":"2026-04-17T12:00:00Z","Action":"pass","Package":"pkg/a","Test":""}`,
		jsonLine("pass", "pkg/a", "TestA"),
		jsonLine("fail", "pkg/a", "TestA"),
	}, "\n")

	reports := DetectFlakes(output, 2)
	if len(reports) != 1 {
		t.Fatalf("expected 1 flaky report, got %d", len(reports))
	}
}

func TestDetectFlakes_InvalidJSON(t *testing.T) {
	output := strings.Join([]string{
		"not json at all",
		jsonLine("pass", "pkg/a", "TestA"),
		jsonLine("fail", "pkg/a", "TestA"),
		"",
		"also not json",
	}, "\n")

	reports := DetectFlakes(output, 2)
	if len(reports) != 1 {
		t.Fatalf("expected 1 flaky report (bad lines ignored), got %d", len(reports))
	}
}

func TestDetectFlakes_EmptyInput(t *testing.T) {
	reports := DetectFlakes("", 3)
	if len(reports) != 0 {
		t.Errorf("expected 0 reports for empty input, got %d", len(reports))
	}
}

func TestFormatReport_NoFlaky(t *testing.T) {
	msg := FormatReport(nil, 3)
	if !strings.Contains(msg, "No flaky tests detected") {
		t.Errorf("expected 'No flaky tests detected' in output, got: %s", msg)
	}
	if !strings.Contains(msg, "3 run(s)") {
		t.Errorf("expected '3 run(s)' in output, got: %s", msg)
	}
}

func TestFormatReport_WithFlaky(t *testing.T) {
	reports := []FlakeReport{
		{
			Package:   "github.com/example/pkg/a",
			Test:      "TestSomething",
			PassCount: 2,
			FailCount: 1,
			TotalRuns: 3,
			IsFlaky:   true,
		},
	}
	msg := FormatReport(reports, 3)
	if !strings.Contains(msg, "1 flaky test(s)") {
		t.Errorf("expected flaky count in output, got: %s", msg)
	}
	if !strings.Contains(msg, "Advisory") {
		t.Errorf("expected advisory note in output, got: %s", msg)
	}
}

func TestRetryFlaky_SucceedsFirstTry(t *testing.T) {
	calls := 0
	err := RetryFlaky(func() error {
		calls++
		return nil
	}, 3)
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetryFlaky_SucceedsOnRetry(t *testing.T) {
	calls := 0
	err := RetryFlaky(func() error {
		calls++
		if calls < 3 {
			return errors.New("transient failure")
		}
		return nil
	}, 5)
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetryFlaky_AllAttemptsFail(t *testing.T) {
	err := RetryFlaky(func() error {
		return errors.New("always fails")
	}, 2)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "3 attempt(s) failed") {
		t.Errorf("expected '3 attempt(s) failed' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "always fails") {
		t.Errorf("expected original error message in error, got: %v", err)
	}
}

func TestRetryFlaky_ZeroRetries(t *testing.T) {
	calls := 0
	err := RetryFlaky(func() error {
		calls++
		return errors.New("nope")
	}, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retries), got %d", calls)
	}
}

func TestSplitKey(t *testing.T) {
	tests := []struct {
		input     string
		wantPkg   string
		wantTest  string
	}{
		{"pkg/a.TestFoo", "pkg/a", "TestFoo"},
		{"pkg/sub.TestBar", "pkg/sub", "TestBar"},
		{"NoDot", "", "NoDot"},
		{"pkg.Test.Nested", "pkg.Test", "Nested"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg, test := splitKey(tt.input)
			if pkg != tt.wantPkg {
				t.Errorf("pkg = %q, want %q", pkg, tt.wantPkg)
			}
			if test != tt.wantTest {
				t.Errorf("test = %q, want %q", test, tt.wantTest)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"too long string here", 10, "too lon..."},
		{"ab", 3, "ab"},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.max)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
		}
	}
}

func TestRetryFlaky_BackoffTiming(t *testing.T) {
	// Verify that retries actually happen with some delay.
	start := time.Now()
	_ = RetryFlaky(func() error {
		return errors.New("fail")
	}, 2)
	elapsed := time.Since(start)

	// Expected: 100ms (1st retry) + 200ms (2nd retry) = ~300ms minimum
	// Allow generous margin for CI scheduling jitter.
	if elapsed < 250*time.Millisecond {
		t.Errorf("retry total time %v seems too fast, expected ~300ms of backoff", elapsed)
	}
}
