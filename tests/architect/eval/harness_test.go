package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
)

func TestComputeStyleMetrics_PerfectMatch(t *testing.T) {
	h := &Harness{}
	predicted := []architect.StyleScore{
		{Style: architect.StyleMicroservices, Confidence: 0.8},
		{Style: architect.StyleEventDriven, Confidence: 0.7},
	}
	expected := []string{"microservices", "event_driven"}

	precision, recall := h.computeStyleMetrics(predicted, expected)
	if precision != 1.0 {
		t.Errorf("expected precision 1.0, got %f", precision)
	}
	if recall != 1.0 {
		t.Errorf("expected recall 1.0, got %f", recall)
	}
}

func TestComputeStyleMetrics_PartialMatch(t *testing.T) {
	h := &Harness{}
	predicted := []architect.StyleScore{
		{Style: architect.StyleMicroservices, Confidence: 0.8},
		{Style: architect.StyleLayered, Confidence: 0.6}, // not in expected
	}
	expected := []string{"microservices", "event_driven"}

	precision, recall := h.computeStyleMetrics(predicted, expected)
	if precision != 0.5 {
		t.Errorf("expected precision 0.5, got %f", precision)
	}
	if recall != 0.5 {
		t.Errorf("expected recall 0.5, got %f", recall)
	}
}

func TestComputeStyleMetrics_NoMatch(t *testing.T) {
	h := &Harness{}
	predicted := []architect.StyleScore{
		{Style: architect.StyleLayered, Confidence: 0.8},
	}
	expected := []string{"microservices"}

	precision, recall := h.computeStyleMetrics(predicted, expected)
	if precision != 0.0 {
		t.Errorf("expected precision 0.0, got %f", precision)
	}
	if recall != 0.0 {
		t.Errorf("expected recall 0.0, got %f", recall)
	}
}

func TestComputeStyleMetrics_LowConfidenceIgnored(t *testing.T) {
	h := &Harness{}
	predicted := []architect.StyleScore{
		{Style: architect.StyleMicroservices, Confidence: 0.8},
		{Style: architect.StyleEventDriven, Confidence: 0.2}, // below threshold
	}
	expected := []string{"microservices", "event_driven"}

	precision, recall := h.computeStyleMetrics(predicted, expected)
	if precision != 1.0 {
		t.Errorf("expected precision 1.0, got %f", precision)
	}
	if recall != 0.5 {
		t.Errorf("expected recall 0.5, got %f", recall)
	}
}

func TestComputeStyleMetrics_EmptyBoth(t *testing.T) {
	h := &Harness{}
	predicted := []architect.StyleScore{}
	expected := []string{}

	precision, recall := h.computeStyleMetrics(predicted, expected)
	if precision != 1.0 {
		t.Errorf("expected precision 1.0, got %f", precision)
	}
	if recall != 1.0 {
		t.Errorf("expected recall 1.0, got %f", recall)
	}
}

func TestComputeStyleMetrics_EmptyPredicted(t *testing.T) {
	h := &Harness{}
	predicted := []architect.StyleScore{}
	expected := []string{"microservices"}

	precision, recall := h.computeStyleMetrics(predicted, expected)
	if precision != 0.0 {
		t.Errorf("expected precision 0.0, got %f", precision)
	}
	if recall != 0.0 {
		t.Errorf("expected recall 0.0, got %f", recall)
	}
}

func TestComputeStyleMetrics_EmptyExpected(t *testing.T) {
	h := &Harness{}
	predicted := []architect.StyleScore{
		{Style: architect.StyleMicroservices, Confidence: 0.8},
	}
	expected := []string{}

	precision, recall := h.computeStyleMetrics(predicted, expected)
	if precision != 0.0 {
		t.Errorf("expected precision 0.0, got %f", precision)
	}
	if recall != 1.0 {
		t.Errorf("expected recall 1.0, got %f", recall)
	}
}

func TestComputeF1(t *testing.T) {
	tests := []struct {
		name      string
		precision float64
		recall    float64
		expected  float64
	}{
		{"perfect", 1.0, 1.0, 1.0},
		{"balanced", 0.5, 0.5, 0.5},
		{"precision biased", 0.75, 0.5, 0.6},
		{"recall biased", 0.5, 0.75, 0.6},
		{"both zero", 0.0, 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeF1(tt.precision, tt.recall)
			if result != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestComputeC4Completeness(t *testing.T) {
	h := &Harness{}

	report := &architect.ArchitectureReport{
		Metrics: architect.CodeMetrics{
			ContainersDetected: 3,
		},
	}

	result := h.computeC4Completeness(report, 5)
	if result != 0.6 {
		t.Errorf("expected 0.6, got %f", result)
	}
}

func TestComputeC4Completeness_OverDetected(t *testing.T) {
	h := &Harness{}

	report := &architect.ArchitectureReport{
		Metrics: architect.CodeMetrics{
			ContainersDetected: 10,
		},
	}

	result := h.computeC4Completeness(report, 5)
	if result != 1.0 {
		t.Errorf("expected 1.0, got %f", result)
	}
}

func TestComputeC4Completeness_NoExpected(t *testing.T) {
	h := &Harness{}

	report := &architect.ArchitectureReport{
		Metrics: architect.CodeMetrics{
			ContainersDetected: 0,
		},
	}

	result := h.computeC4Completeness(report, 0)
	if result != 1.0 {
		t.Errorf("expected 1.0, got %f", result)
	}
}

func TestComputeC4Completeness_NoExpected_ButDetected(t *testing.T) {
	h := &Harness{}

	report := &architect.ArchitectureReport{
		Metrics: architect.CodeMetrics{
			ContainersDetected: 3,
		},
	}

	result := h.computeC4Completeness(report, 0)
	if result != 0.0 {
		t.Errorf("expected 0.0, got %f", result)
	}
}

func TestComputeLangAccuracy(t *testing.T) {
	h := &Harness{}

	detected := architect.LanguageInfo{
		All: []string{"go", "python", "javascript"},
	}
	expected := []string{"go", "python"}

	result := h.computeLangAccuracy(detected, expected)
	if result != 1.0 {
		t.Errorf("expected 1.0, got %f", result)
	}
}

func TestComputeLangAccuracy_Partial(t *testing.T) {
	h := &Harness{}

	detected := architect.LanguageInfo{
		All: []string{"go", "javascript"},
	}
	expected := []string{"go", "python"}

	result := h.computeLangAccuracy(detected, expected)
	if result != 0.5 {
		t.Errorf("expected 0.5, got %f", result)
	}
}

func TestComputeLangAccuracy_NoMatch(t *testing.T) {
	h := &Harness{}

	detected := architect.LanguageInfo{
		All: []string{"javascript"},
	}
	expected := []string{"go", "python"}

	result := h.computeLangAccuracy(detected, expected)
	if result != 0.0 {
		t.Errorf("expected 0.0, got %f", result)
	}
}

func TestComputeLangAccuracy_EmptyExpected(t *testing.T) {
	h := &Harness{}

	detected := architect.LanguageInfo{
		All: []string{"go", "python"},
	}
	expected := []string{}

	result := h.computeLangAccuracy(detected, expected)
	if result != 1.0 {
		t.Errorf("expected 1.0, got %f", result)
	}
}

func TestPasses_AllAboveThreshold(t *testing.T) {
	h := NewHarness([]GoldenRepo{}, DefaultThresholds())

	metrics := EvalMetrics{
		RepoName:       "test",
		StylePrecision: 0.7,
		StyleRecall:    0.6,
		StyleF1:        0.65,
		ImportAccuracy: 0.8,
		C4Completeness: 0.6,
		LangAccuracy:   0.9,
		OverallScore:   0.7,
	}

	if !h.Passes(metrics) {
		t.Error("expected metrics to pass")
	}
}

func TestPasses_BelowThreshold(t *testing.T) {
	h := NewHarness([]GoldenRepo{}, DefaultThresholds())

	metrics := EvalMetrics{
		RepoName:       "test",
		StylePrecision: 0.5, // below 0.6
		StyleRecall:    0.6,
		StyleF1:        0.55,
		ImportAccuracy: 0.8,
		C4Completeness: 0.6,
		LangAccuracy:   0.9,
		OverallScore:   0.6,
	}

	if h.Passes(metrics) {
		t.Error("expected metrics to fail")
	}
}

func TestPasses_WithError(t *testing.T) {
	h := NewHarness([]GoldenRepo{}, DefaultThresholds())

	metrics := EvalMetrics{
		RepoName: "test",
		Error:    "failed to load report",
	}

	if h.Passes(metrics) {
		t.Error("expected metrics with error to fail")
	}
}

func TestDefaultThresholds(t *testing.T) {
	thresholds := DefaultThresholds()

	if thresholds.StylePrecision != 0.6 {
		t.Errorf("expected StylePrecision 0.6, got %f", thresholds.StylePrecision)
	}
	if thresholds.StyleRecall != 0.5 {
		t.Errorf("expected StyleRecall 0.5, got %f", thresholds.StyleRecall)
	}
	if thresholds.C4Completeness != 0.5 {
		t.Errorf("expected C4Completeness 0.5, got %f", thresholds.C4Completeness)
	}
	if thresholds.LangAccuracy != 0.8 {
		t.Errorf("expected LangAccuracy 0.8, got %f", thresholds.LangAccuracy)
	}
}

func TestStandardGoldenRepos(t *testing.T) {
	repos := StandardGoldenRepos()

	if len(repos) != 7 {
		t.Errorf("expected 7 repos, got %d", len(repos))
	}

	// Check first repo
	if repos[0].Name != "go-microservices-demo" {
		t.Errorf("expected first repo name 'go-microservices-demo', got %s", repos[0].Name)
	}
	if len(repos[0].ExpectedStyles) != 1 {
		t.Errorf("expected 1 style for first repo, got %d", len(repos[0].ExpectedStyles))
	}
	if repos[0].ExpectedStyles[0] != "microservices" {
		t.Errorf("expected style 'microservices', got %s", repos[0].ExpectedStyles[0])
	}
}

func TestWriteReport(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.json")

	result := &EvalResult{
		Timestamp:  "2026-04-10T12:00:00Z",
		TotalRepos: 2,
		Passed:     1,
		Failed:     1,
		Thresholds: DefaultThresholds(),
		Metrics: []EvalMetrics{
			{
				RepoName:       "test-repo",
				StylePrecision: 0.8,
				StyleRecall:    0.7,
				StyleF1:        0.75,
				ImportAccuracy: 0.9,
				C4Completeness: 0.8,
				LangAccuracy:   1.0,
				OverallScore:   0.825,
			},
		},
	}

	err := WriteReport(result, reportPath)
	if err != nil {
		t.Fatalf("failed to write report: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("report file was not created")
	}

	// Verify content
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("failed to read report: %v", err)
	}

	var loaded EvalResult
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal report: %v", err)
	}

	if loaded.TotalRepos != result.TotalRepos {
		t.Errorf("expected TotalRepos %d, got %d", result.TotalRepos, loaded.TotalRepos)
	}
	if loaded.Passed != result.Passed {
		t.Errorf("expected Passed %d, got %d", result.Passed, loaded.Passed)
	}
}

func TestEvalResultJSON(t *testing.T) {
	result := EvalResult{
		Timestamp:  "2026-04-10T12:00:00Z",
		TotalRepos: 1,
		Passed:     1,
		Failed:     0,
		Thresholds: DefaultThresholds(),
		Metrics: []EvalMetrics{
			{
				RepoName:       "test",
				StylePrecision: 0.8,
				StyleRecall:    0.7,
				StyleF1:        0.75,
				ImportAccuracy: 0.9,
				C4Completeness: 0.8,
				LangAccuracy:   1.0,
				OverallScore:   0.825,
			},
		},
	}

	// Marshal
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal
	var unmarshaled EvalResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify fields match
	if unmarshaled.Timestamp != result.Timestamp {
		t.Errorf("timestamp mismatch: %s vs %s", unmarshaled.Timestamp, result.Timestamp)
	}
	if unmarshaled.TotalRepos != result.TotalRepos {
		t.Errorf("TotalRepos mismatch: %d vs %d", unmarshaled.TotalRepos, result.TotalRepos)
	}
	if len(unmarshaled.Metrics) != len(result.Metrics) {
		t.Errorf("Metrics length mismatch: %d vs %d", len(unmarshaled.Metrics), len(result.Metrics))
	}
}

func TestRunLocal_SkipsReposWithoutLocalPath(t *testing.T) {
	repos := []GoldenRepo{
		{Name: "repo-without-path", ExpectedStyles: []string{"microservices"}},
		{Name: "repo-with-path", LocalPath: "/some/path", ExpectedStyles: []string{"microservices"}},
	}

	h := NewHarness(repos, DefaultThresholds())
	result, err := h.RunLocal("/repo/root")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the repo with a local path should be evaluated (and fail because path doesn't exist)
	if result.TotalRepos != 2 {
		t.Errorf("expected TotalRepos 2, got %d", result.TotalRepos)
	}
	if len(result.Metrics) != 1 {
		t.Errorf("expected 1 metric (only repo with path), got %d", len(result.Metrics))
	}
	if result.Metrics[0].RepoName != "repo-with-path" {
		t.Errorf("expected repo 'repo-with-path', got %s", result.Metrics[0].RepoName)
	}
}

func TestNewHarness(t *testing.T) {
	repos := []GoldenRepo{{Name: "test"}}
	thresholds := DefaultThresholds()

	h := NewHarness(repos, thresholds)

	if h == nil {
		t.Fatal("NewHarness returned nil")
	}
	if len(h.repos) != 1 {
		t.Errorf("expected 1 repo, got %d", len(h.repos))
	}
	if h.thresholds.StylePrecision != thresholds.StylePrecision {
		t.Error("thresholds not set correctly")
	}
}

func TestGoldenRepoJSON(t *testing.T) {
	repo := GoldenRepo{
		Name:           "test-repo",
		URL:            "https://github.com/test/repo",
		LocalPath:      "/path/to/repo",
		ExpectedStyles: []string{"microservices", "event_driven"},
		ExpectedLangs:  []string{"go"},
		Containers:     5,
		HasContracts:   true,
		Complexity:     "medium",
	}

	data, err := json.Marshal(repo)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled GoldenRepo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.Name != repo.Name {
		t.Errorf("name mismatch: %s vs %s", unmarshaled.Name, repo.Name)
	}
	if len(unmarshaled.ExpectedStyles) != len(repo.ExpectedStyles) {
		t.Errorf("ExpectedStyles length mismatch: %d vs %d", len(unmarshaled.ExpectedStyles), len(repo.ExpectedStyles))
	}
}

func TestLoadReport_InvalidPath(t *testing.T) {
	h := &Harness{}
	_, err := h.loadReport("/nonexistent/path/report.json")

	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestLoadReport_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.json")

	// Write invalid JSON
	if err := os.WriteFile(reportPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	h := &Harness{}
	_, err := h.loadReport(reportPath)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadReport_Success(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.json")

	report := architect.ArchitectureReport{
		Version:   "1.0",
		AnalyzedAt: time.Now(),
		RepoRoot:  "/test",
		Languages: architect.LanguageInfo{
			Primary: "go",
			All:     []string{"go"},
		},
		Metrics: architect.CodeMetrics{
			TotalFiles:          100,
			TotalLOC:            10000,
			ContainersDetected:  3,
			ComponentsDetected:  10,
			ContractsDiscovered: 2,
		},
		ConfidenceSummary: architect.ConfidenceSummary{
			Overall:            0.8,
			StructuralAnalysis: 0.9,
			StyleHypothesis:    0.7,
		},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	h := &Harness{}
	loaded, err := h.loadReport(reportPath)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loaded.RepoRoot != report.RepoRoot {
		t.Errorf("RepoRoot mismatch: %s vs %s", loaded.RepoRoot, report.RepoRoot)
	}
	if loaded.Metrics.ContainersDetected != report.Metrics.ContainersDetected {
		t.Errorf("ContainersDetected mismatch: %d vs %d", loaded.Metrics.ContainersDetected, report.Metrics.ContainersDetected)
	}
}
