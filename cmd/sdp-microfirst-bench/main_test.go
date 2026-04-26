package main

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/inference/decompose"
	"sdp_dev/internal/inference/microfirst/bdtype"
	"sdp_dev/internal/inference/microfirst/knn"
	"sdp_dev/internal/inference/microfirst/wsverdict"
)

const testDataDir = "testdata"

// TestBenchWsVerdict_Smoke runs wsverdict bench with a tiny corpus (5 items).
func TestBenchWsVerdict_Smoke(t *testing.T) {
	result, err := benchWsVerdict(context.Background(), testDataDir, 5)
	if err != nil {
		t.Fatalf("benchWsVerdict: %v", err)
	}
	if result.TotalRequests < 5 {
		t.Errorf("expected TotalRequests >= 5, got %d", result.TotalRequests)
	}
	if result.MicroHandled < 0 {
		t.Errorf("MicroHandled must not be negative, got %d", result.MicroHandled)
	}
	if result.Classifier != "wsverdict" {
		t.Errorf("expected classifier=wsverdict, got %q", result.Classifier)
	}
}

// TestBenchBdSeverity_Smoke runs bdseverity bench with a tiny corpus.
func TestBenchBdSeverity_Smoke(t *testing.T) {
	result, err := benchBdSeverity(context.Background(), testDataDir, 5)
	if err != nil {
		t.Fatalf("benchBdSeverity: %v", err)
	}
	if result.TotalRequests < 1 {
		t.Errorf("expected TotalRequests >= 1, got %d", result.TotalRequests)
	}
	if result.MicroHandled < 0 {
		t.Errorf("MicroHandled must not be negative, got %d", result.MicroHandled)
	}
	if result.Classifier != "bdseverity" {
		t.Errorf("expected classifier=bdseverity, got %q", result.Classifier)
	}
}

// TestBenchBdType_Smoke runs bdtype bench with a tiny corpus.
func TestBenchBdType_Smoke(t *testing.T) {
	result, err := benchBdType(context.Background(), testDataDir, 5)
	if err != nil {
		t.Fatalf("benchBdType: %v", err)
	}
	if result.TotalRequests < 1 {
		t.Errorf("expected TotalRequests >= 1, got %d", result.TotalRequests)
	}
	if result.MicroHandled < 0 {
		t.Errorf("MicroHandled must not be negative, got %d", result.MicroHandled)
	}
	if result.Classifier != "bdtype" {
		t.Errorf("expected classifier=bdtype, got %q", result.Classifier)
	}
}

// TestBenchRouting_Smoke runs routing bench with a tiny corpus.
func TestBenchRouting_Smoke(t *testing.T) {
	result, err := benchRouting(context.Background(), testDataDir, 5)
	if err != nil {
		t.Fatalf("benchRouting: %v", err)
	}
	if result.TotalRequests < 1 {
		t.Errorf("expected TotalRequests >= 1, got %d", result.TotalRequests)
	}
	if result.MicroHandled < 0 {
		t.Errorf("MicroHandled must not be negative, got %d", result.MicroHandled)
	}
	if result.Classifier != "routing" {
		t.Errorf("expected classifier=routing, got %q", result.Classifier)
	}
}

// TestBenchResult_JSONRoundTrip ensures the output JSON parses correctly.
func TestBenchResult_JSONRoundTrip(t *testing.T) {
	r := BenchResult{
		Classifier:      "wsverdict",
		TotalRequests:   30,
		MicroHandled:    25,
		Escalated:       5,
		FallbackRate:    0.1667,
		P50Ms:           0.05,
		P95Ms:           0.12,
		Accuracy:        92.0,
		LLMCallsSaved:   83.3,
		EstTokenSavings: 83.3,
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var r2 BenchResult
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r2.MicroHandled != r.MicroHandled {
		t.Errorf("MicroHandled mismatch: %d != %d", r2.MicroHandled, r.MicroHandled)
	}
	if r2.Classifier != r.Classifier {
		t.Errorf("Classifier mismatch: %q != %q", r2.Classifier, r.Classifier)
	}
}

// TestWriteJSON ensures JSON output file is created and parseable.
func TestWriteJSON(t *testing.T) {
	dir := t.TempDir()
	r := BenchResult{
		Classifier:    "wsverdict",
		TotalRequests: 10,
		MicroHandled:  8,
	}
	path := filepath.Join(dir, "wsverdict.json")
	if err := writeJSON(path, r); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var r2 BenchResult
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if r2.MicroHandled < 0 {
		t.Errorf("MicroHandled must not be negative, got %d", r2.MicroHandled)
	}
}

// TestWriteMarkdown ensures the markdown report is created without error.
func TestWriteMarkdown(t *testing.T) {
	dir := t.TempDir()
	results := []BenchResult{
		{Classifier: "wsverdict", TotalRequests: 30, MicroHandled: 27, Escalated: 3, LLMCallsSaved: 90.0},
		{Classifier: "bdseverity", TotalRequests: 30, MicroHandled: 20, Escalated: 10, LLMCallsSaved: 66.7},
	}
	path := filepath.Join(dir, "report.md")
	if err := writeMarkdown(path, results); err != nil {
		t.Fatalf("writeMarkdown: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if len(data) == 0 {
		t.Error("report.md is empty")
	}
}

// TestMockEmbedder_Deterministic verifies mockEmbedder returns the same vector for same input.
func TestMockEmbedder_Deterministic(t *testing.T) {
	emb := &mockEmbedder{}
	ctx := context.Background()
	v1, err := emb.Embed(ctx, "hello world")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	v2, err := emb.Embed(ctx, "hello world")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Errorf("vectors differ at index %d: %f != %f", i, v1[i], v2[i])
		}
	}
}

// TestMockEmbedder_UnitVector verifies the returned vector has unit L2 norm.
func TestMockEmbedder_UnitVector(t *testing.T) {
	emb := &mockEmbedder{}
	ctx := context.Background()
	vec, err := emb.Embed(ctx, "test input")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	const eps = 1e-9
	if abs(norm-1.0) > eps {
		t.Errorf("expected unit vector, got norm=%.6f", norm)
	}
}

// TestPercentiles verifies percentile computation.
func TestPercentiles(t *testing.T) {
	vals := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	p50, p95 := percentiles(vals)
	if p50 <= 0 {
		t.Errorf("p50 should be positive, got %f", p50)
	}
	if p95 <= p50 {
		t.Errorf("p95 (%f) should be >= p50 (%f)", p95, p50)
	}
}

// TestPercentiles_Empty handles empty input gracefully.
func TestPercentiles_Empty(t *testing.T) {
	p50, p95 := percentiles(nil)
	if p50 != 0 || p95 != 0 {
		t.Errorf("expected 0,0 for empty slice, got %f,%f", p50, p95)
	}
}

// TestLoadWsVerdictCases verifies JSON corpus loading.
func TestLoadWsVerdictCases(t *testing.T) {
	cases, err := loadWsVerdictCases(filepath.Join(testDataDir, "wsverdict_cases.json"))
	if err != nil {
		t.Fatalf("loadWsVerdictCases: %v", err)
	}
	if len(cases) < 35 {
		t.Errorf("expected >= 35 cases, got %d", len(cases))
	}
}

// TestLoadTextCasesJSONL verifies JSONL corpus loading for each classifier type.
func TestLoadTextCasesJSONL(t *testing.T) {
	tests := []struct {
		file string
		key  string
		min  int
	}{
		{"bdseverity_cases.jsonl", "expected_priority", 35},
		{"bdtype_cases.jsonl", "expected_type", 35},
		{"routing_cases.jsonl", "expected_capability", 35},
	}
	for _, tt := range tests {
		cases, err := loadTextCasesJSONL(filepath.Join(testDataDir, tt.file), tt.key)
		if err != nil {
			t.Fatalf("loadTextCasesJSONL(%s): %v", tt.file, err)
		}
		if len(cases) < tt.min {
			t.Errorf("%s: expected >= %d cases, got %d", tt.file, tt.min, len(cases))
		}
	}
}

// TestExtendToMin verifies case extension logic.
func TestExtendToMin(t *testing.T) {
	cases := []wsVerdictCase{
		{Input: wsverdict.WsVerdictInput{}, ExpectedVerdict: "pass"},
		{Input: wsverdict.WsVerdictInput{}, ExpectedVerdict: "fail"},
	}
	extended := extendToMin(cases, 7)
	if len(extended) < 7 {
		t.Errorf("expected >= 7, got %d", len(extended))
	}
	// Already enough cases, no extension needed.
	big := []wsVerdictCase{
		{}, {}, {}, {}, {}, {}, {}, {}, {}, {},
	}
	same := extendToMin(big, 5)
	if len(same) != 10 {
		t.Errorf("expected 10 (no truncation), got %d", len(same))
	}
}

// TestExtendTextCasesToMin verifies text case extension logic.
func TestExtendTextCasesToMin(t *testing.T) {
	cases := []textCase{
		{Title: "a", Description: "b", Expected: "bug"},
		{Title: "c", Description: "d", Expected: "feature"},
	}
	extended := extendTextCasesToMin(cases, 8)
	if len(extended) < 8 {
		t.Errorf("expected >= 8, got %d", len(extended))
	}
	// Already enough, no extension needed.
	big := make([]textCase, 10)
	same := extendTextCasesToMin(big, 5)
	if len(same) != 10 {
		t.Errorf("expected 10 (no truncation), got %d", len(same))
	}
	// Empty input.
	empty := extendTextCasesToMin(nil, 5)
	if len(empty) != 0 {
		t.Errorf("expected 0 for empty input, got %d", len(empty))
	}
}

// TestComputeResult verifies the metric computation.
func TestComputeResult(t *testing.T) {
	latencies := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	r := computeResult("wsverdict", 10, 8, latencies, 7)
	if r.TotalRequests != 10 {
		t.Errorf("TotalRequests: got %d, want 10", r.TotalRequests)
	}
	if r.MicroHandled != 8 {
		t.Errorf("MicroHandled: got %d, want 8", r.MicroHandled)
	}
	if r.Escalated != 2 {
		t.Errorf("Escalated: got %d, want 2", r.Escalated)
	}
	if r.LLMCallsSaved != 80.0 {
		t.Errorf("LLMCallsSaved: got %f, want 80.0", r.LLMCallsSaved)
	}
}

// TestBdTypeMicro_WithMockEmbedder verifies bdtype classifier builds and runs with mock embedder.
func TestBdTypeMicro_WithMockEmbedder(t *testing.T) {
	emb := &mockEmbedder{}
	ctx := context.Background()

	corpus := []bdtype.CorpusEntry{
		{ID: "1", Text: "fix nil pointer crash bug", Label: "bug"},
		{ID: "2", Text: "add new feature export csv", Label: "feature"},
		{ID: "3", Text: "refactor database layer task", Label: "task"},
		{ID: "4", Text: "authentication crash bug nil", Label: "bug"},
		{ID: "5", Text: "implement new export feature", Label: "feature"},
	}
	clf, err := bdtype.NewBdTypeMicro(ctx, emb, corpus)
	if err != nil {
		t.Fatalf("NewBdTypeMicro: %v", err)
	}

	result, _, err := clf.Run(ctx, bdtype.BdInput{Title: "fix crash", Description: "nil pointer bug"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// MicroHandled check - status should be ok or unsure (no error)
	_ = result
}

// TestKnnMajorityVote_StatusOK checks majority vote produces StatusOK with identical labels.
func TestKnnMajorityVote_StatusOK(t *testing.T) {
	matches := []knn.Match[string]{
		{Label: "bug", Score: 0.95},
		{Label: "bug", Score: 0.92},
		{Label: "bug", Score: 0.88},
	}
	result := knn.MajorityVote(matches, 0.85)
	if result.Status != decompose.StatusOK {
		t.Errorf("expected StatusOK, got %s", result.Status)
	}
	if result.Label != "bug" {
		t.Errorf("expected label=bug, got %s", result.Label)
	}
}

// TestWsVerdictMicro_ClearPass verifies clear-pass cases produce StatusOK.
func TestWsVerdictMicro_ClearPass(t *testing.T) {
	clf := wsverdict.New(wsverdict.Default())
	in := wsverdict.WsVerdictInput{
		Report: wsverdict.TestReport{Failed: 0, Errored: 0, Total: 10},
		Guard:  wsverdict.GuardDiff{OutOfScope: nil},
	}
	result, _, err := clf.Run(context.Background(), in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.ConfStatus() != decompose.StatusOK {
		t.Errorf("expected StatusOK for clear pass, got %s", result.ConfStatus())
	}
}

// abs is a helper for float64 absolute value.
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
