package coveragegate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCoverprofile(t *testing.T) {
	// Create a temporary coverprofile file
	tmp := t.TempDir()
	covFile := filepath.Join(tmp, "cov.out")

	coverprofile := `mode: set
internal/foo/bar.go:10.1,12.16 3 1
internal/foo/bar.go:15.1,18.20 5 0
internal/baz/qux.go:20.1,22.10 2 2
`
	if err := os.WriteFile(covFile, []byte(coverprofile), 0o644); err != nil {
		t.Fatalf("write coverprofile: %v", err)
	}

	funcs, totalStmts, coveredStmts, err := ParseCoverprofile(covFile)
	if err != nil {
		t.Fatalf("ParseCoverprofile: %v", err)
	}

	if totalStmts != 10 {
		t.Errorf("totalStmts = %d, want 10", totalStmts)
	}
	if coveredStmts != 5 {
		t.Errorf("coveredStmts = %d, want 5", coveredStmts)
	}
	if len(funcs) != 2 {
		t.Errorf("len(funcs) = %d, want 2", len(funcs))
	}

	// Check coverage calculation
	expectedPct := 50.0
	actualPct := float64(coveredStmts) / float64(totalStmts) * 100.0
	if actualPct != expectedPct {
		t.Errorf("coverage = %.1f%%, want %.1f%%", actualPct, expectedPct)
	}
}

func TestParseCoverprofile_EmptyFile(t *testing.T) {
	tmp := t.TempDir()
	covFile := filepath.Join(tmp, "cov.out")

	if err := os.WriteFile(covFile, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, totalStmts, coveredStmts, err := ParseCoverprofile(covFile)
	if err != nil {
		t.Fatalf("ParseCoverprofile: %v", err)
	}
	if totalStmts != 0 {
		t.Errorf("totalStmts = %d, want 0", totalStmts)
	}
	if coveredStmts != 0 {
		t.Errorf("coveredStmts = %d, want 0", coveredStmts)
	}
}

func TestParseCoverprofile_NonexistentFile(t *testing.T) {
	_, _, _, err := ParseCoverprofile("/nonexistent/path/cov.out")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestParseFuncOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    float64
		wantErr bool
	}{
		{
			name: "standard total line",
			output: `internal/foo/bar.go:Bar 100.0%
total:                                                  (statements) 85.7%
`,
			want:    85.7,
			wantErr: false,
		},
		{
			name: "total with no parentheses",
			output: `internal/foo/bar.go:Bar 100.0%
total:                                                  92.3%
`,
			want:    92.3,
			wantErr: false,
		},
		{
			name: "no total line",
			output: `internal/foo/bar.go:Bar 100.0%
internal/baz/qux.go:Qux 50.0%
`,
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty output",
			output:  "",
			want:    0,
			wantErr: true,
		},
		{
			name: "100 percent coverage",
			output: `total:                                                  100.0%
`,
			want:    100.0,
			wantErr: false,
		},
		{
			name: "zero percent coverage",
			output: `total:                                                  0.0%
`,
			want:    0.0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFuncOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFuncOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFuncOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadBaseline(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		want       float64
		wantErr    bool
		noFile     bool
	}{
		{
			name:    "valid percentage",
			content: "85.7\n",
			want:    85.7,
			wantErr: false,
		},
		{
			name:    "integer percentage",
			content: "70\n",
			want:    70.0,
			wantErr: false,
		},
		{
			name:    "trailing whitespace",
			content: "92.3  \n",
			want:    92.3,
			wantErr: false,
		},
		{
			name:    "empty file",
			content: "",
			want:    0.0,
			wantErr: false,
		},
		{
			name:    "no file returns zero",
			noFile:  true,
			want:    0.0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			if !tt.noFile {
				dir := filepath.Join(tmp, ".sdp", "metrics")
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				path := filepath.Join(dir, "coverage.txt")
				if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
					t.Fatalf("write: %v", err)
				}
			}

			got, err := ReadBaseline(tmp)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadBaseline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadBaseline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteBaseline(t *testing.T) {
	tmp := t.TempDir()

	if err := WriteBaseline(tmp, 87.3); err != nil {
		t.Fatalf("WriteBaseline: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, DefaultMetricsPath))
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}

	expected := "87.3\n"
	if string(data) != expected {
		t.Errorf("written content = %q, want %q", string(data), expected)
	}

	// Verify we can read it back
	pct, err := ReadBaseline(tmp)
	if err != nil {
		t.Fatalf("ReadBaseline after WriteBaseline: %v", err)
	}
	if pct != 87.3 {
		t.Errorf("ReadBaseline = %v, want 87.3", pct)
	}
}

func TestCheckCoverage_Pass(t *testing.T) {
	tmp := t.TempDir()

	// Write coverprofile
	covFile := filepath.Join(tmp, "cov.out")
	coverprofile := `mode: set
internal/foo/bar.go:10.1,12.16 3 1
internal/foo/bar.go:15.1,18.20 5 0
internal/baz/qux.go:20.1,22.10 2 2
`
	if err := os.WriteFile(covFile, []byte(coverprofile), 0o644); err != nil {
		t.Fatalf("write coverprofile: %v", err)
	}

	// Write baseline (40% — current is 50%, so delta is +10pp, should pass)
	writeMetrics(t, tmp, "40.0")

	report, err := CheckCoverage(tmp, covFile)
	if err != nil {
		t.Fatalf("CheckCoverage: %v", err)
	}

	if !report.Passed {
		t.Errorf("Passed = false, want true; message: %s", report.Message)
	}
	if report.TotalCoverage != 50.0 {
		t.Errorf("TotalCoverage = %.1f, want 50.0", report.TotalCoverage)
	}
	if report.Baseline != 40.0 {
		t.Errorf("Baseline = %.1f, want 40.0", report.Baseline)
	}
}

func TestCheckCoverage_Fail_DropExceedsThreshold(t *testing.T) {
	tmp := t.TempDir()

	// Write coverprofile with low coverage (1/10 = 10%)
	covFile := filepath.Join(tmp, "cov.out")
	coverprofile := `mode: set
internal/foo/bar.go:10.1,12.16 3 0
internal/foo/bar.go:15.1,18.20 5 0
internal/baz/qux.go:20.1,22.10 2 1
`
	if err := os.WriteFile(covFile, []byte(coverprofile), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Baseline is 80%, current is 10%, drop is 70pp (way over 2pp threshold)
	writeMetrics(t, tmp, "80.0")

	report, err := CheckCoverage(tmp, covFile)
	if err != nil {
		t.Fatalf("CheckCoverage: %v", err)
	}

	if report.Passed {
		t.Errorf("Passed = true, want false; message: %s", report.Message)
	}
}

func TestCheckCoverage_ExactThreshold(t *testing.T) {
	tmp := t.TempDir()

	// Coverprofile: 5/10 = 50%
	covFile := filepath.Join(tmp, "cov.out")
	coverprofile := `mode: set
internal/foo/bar.go:10.1,12.16 3 1
internal/foo/bar.go:15.1,18.20 5 0
internal/baz/qux.go:20.1,22.10 2 2
`
	if err := os.WriteFile(covFile, []byte(coverprofile), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Baseline is 52%, current is 50%, drop is exactly 2pp — should pass
	writeMetrics(t, tmp, "52.0")

	report, err := CheckCoverage(tmp, covFile)
	if err != nil {
		t.Fatalf("CheckCoverage: %v", err)
	}

	if !report.Passed {
		t.Errorf("Passed = false at exact threshold; message: %s", report.Message)
	}
}

func TestCheckCoverage_OnePastThreshold(t *testing.T) {
	tmp := t.TempDir()

	// Coverprofile: 5/10 = 50%
	covFile := filepath.Join(tmp, "cov.out")
	coverprofile := `mode: set
internal/foo/bar.go:10.1,12.16 3 1
internal/foo/bar.go:15.1,18.20 5 0
internal/baz/qux.go:20.1,22.10 2 2
`
	if err := os.WriteFile(covFile, []byte(coverprofile), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Baseline is 52.2%, current is 50%, drop is 2.2pp — should fail
	writeMetrics(t, tmp, "52.2")

	report, err := CheckCoverage(tmp, covFile)
	if err != nil {
		t.Fatalf("CheckCoverage: %v", err)
	}

	if report.Passed {
		t.Errorf("Passed = true when drop exceeds threshold; message: %s", report.Message)
	}
}

func TestCheckCoverage_NoBaseline(t *testing.T) {
	tmp := t.TempDir()

	// Coverprofile exists, no baseline file
	covFile := filepath.Join(tmp, "cov.out")
	coverprofile := `mode: set
internal/foo/bar.go:10.1,12.16 3 1
`
	if err := os.WriteFile(covFile, []byte(coverprofile), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// No baseline file → baseline is 0.0, current is 100%, should pass
	report, err := CheckCoverage(tmp, covFile)
	if err != nil {
		t.Fatalf("CheckCoverage: %v", err)
	}

	if !report.Passed {
		t.Errorf("Passed = false with no baseline; message: %s", report.Message)
	}
	if report.Baseline != 0.0 {
		t.Errorf("Baseline = %.1f, want 0.0", report.Baseline)
	}
}

func TestCheckCoverage_ZeroStatements(t *testing.T) {
	tmp := t.TempDir()

	covFile := filepath.Join(tmp, "cov.out")
	if err := os.WriteFile(covFile, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := CheckCoverage(tmp, covFile)
	if err == nil {
		t.Fatal("expected error for zero statements")
	}
}

func TestCheckCoverageWithThreshold_NegativeThreshold(t *testing.T) {
	tmp := t.TempDir()
	covFile := filepath.Join(tmp, "cov.out")
	if err := os.WriteFile(covFile, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := CheckCoverageWithThreshold(tmp, covFile, -1.0)
	if err == nil {
		t.Fatal("expected error for negative threshold")
	}
}

func TestCheckCoverageWithThreshold_CustomPass(t *testing.T) {
	tmp := t.TempDir()

	covFile := filepath.Join(tmp, "cov.out")
	coverprofile := `mode: set
internal/foo/bar.go:10.1,12.16 3 1
internal/foo/bar.go:15.1,18.20 5 0
internal/baz/qux.go:20.1,22.10 2 2
`
	if err := os.WriteFile(covFile, []byte(coverprofile), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// 50% current vs 52% baseline = -2pp drop, threshold 1.0 → should fail
	writeMetrics(t, tmp, "52.0")

	report, err := CheckCoverageWithThreshold(tmp, covFile, 1.0)
	if err != nil {
		t.Fatalf("CheckCoverageWithThreshold: %v", err)
	}

	if report.Passed {
		t.Errorf("should fail with tight threshold; message: %s", report.Message)
	}
	if report.Threshold != 1.0 {
		t.Errorf("Threshold = %.1f, want 1.0", report.Threshold)
	}
}

// Helper to write the metrics baseline file.
func writeMetrics(t *testing.T, projectRoot, content string) {
	t.Helper()
	dir := filepath.Join(projectRoot, ".sdp", "metrics")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir metrics: %v", err)
	}
	path := filepath.Join(dir, "coverage.txt")
	if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
		t.Fatalf("write metrics: %v", err)
	}
}
