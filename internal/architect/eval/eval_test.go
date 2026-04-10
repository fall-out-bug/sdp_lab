package eval

import (
	"math"
	"testing"

	"sdp_dev/internal/architect"
)

// --- Precision / Recall / F1 unit tests ---

func TestFieldAccuracyPerfectMatch(t *testing.T) {
	fa := FieldAccuracy{
		FieldName:      "test",
		TruePositives:  10,
		FalsePositives: 0,
		FalseNegatives: 0,
	}
	assertFloat(t, "precision", 1.0, fa.Precision())
	assertFloat(t, "recall", 1.0, fa.Recall())
	assertFloat(t, "f1", 1.0, fa.F1())
}

func TestFieldAccuracyAllWrong(t *testing.T) {
	fa := FieldAccuracy{
		FieldName:      "test",
		TruePositives:  0,
		FalsePositives: 5,
		FalseNegatives: 10,
	}
	assertFloat(t, "precision", 0.0, fa.Precision())
	assertFloat(t, "recall", 0.0, fa.Recall())
	assertFloat(t, "f1", 0.0, fa.F1())
}

func TestFieldAccuracyTypicalCase(t *testing.T) {
	// 8 correct, 2 false positives, 3 false negatives
	fa := FieldAccuracy{
		FieldName:      "test",
		TruePositives:  8,
		FalsePositives: 2,
		FalseNegatives: 3,
	}
	// Precision = 8 / (8+2) = 0.8
	assertFloat(t, "precision", 0.8, fa.Precision())
	// Recall = 8 / (8+3) = 0.7272...
	assertFloat(t, "recall", 8.0/11.0, fa.Recall())
	// F1 = 2 * (0.8 * 0.7272) / (0.8 + 0.7272)
	expectedF1 := 2.0 * (0.8 * (8.0 / 11.0)) / (0.8 + (8.0 / 11.0))
	assertFloat(t, "f1", expectedF1, fa.F1())
}

func TestFieldAccuracyEmptySets(t *testing.T) {
	// Nothing expected, nothing produced: perfect
	fa := FieldAccuracy{FieldName: "test"}
	assertFloat(t, "precision", 1.0, fa.Precision())
	assertFloat(t, "recall", 1.0, fa.Recall())
	assertFloat(t, "f1", 1.0, fa.F1())
}

func TestFieldAccuracyNoExpectedButProduced(t *testing.T) {
	// Nothing expected, but extractor produced items: all false positives
	fa := FieldAccuracy{
		FieldName:      "test",
		FalsePositives: 5,
	}
	assertFloat(t, "precision", 0.0, fa.Precision())
	assertFloat(t, "recall", 1.0, fa.Recall()) // 0/(0+0) = 1.0 by convention
	assertFloat(t, "f1", 0.0, fa.F1())
}

func TestFieldAccuracyExpectedButNoneProduced(t *testing.T) {
	// Items expected but none produced: all false negatives.
	// Precision: TP=0, FP=0 -> denom=0. Since FN>0, the extractor missed
	// everything so 0/0 is treated as 0.0 (cannot claim any precision).
	fa := FieldAccuracy{
		FieldName:      "test",
		FalseNegatives: 5,
	}
	assertFloat(t, "precision", 0.0, fa.Precision())
	assertFloat(t, "recall", 0.0, fa.Recall())
	assertFloat(t, "f1", 0.0, fa.F1())
}

func TestComputeMetrics(t *testing.T) {
	p, r, f1 := ComputeMetrics(8, 2, 3)
	assertFloat(t, "precision", 0.8, p)
	assertFloat(t, "recall", 8.0/11.0, r)
	expectedF1 := 2.0 * (0.8 * (8.0 / 11.0)) / (0.8 + (8.0 / 11.0))
	assertFloat(t, "f1", expectedF1, f1)
}

func TestComputeMetricsPerfect(t *testing.T) {
	p, r, f1 := ComputeMetrics(10, 0, 0)
	assertFloat(t, "precision", 1.0, p)
	assertFloat(t, "recall", 1.0, r)
	assertFloat(t, "f1", 1.0, f1)
}

func TestComputeMetricsZero(t *testing.T) {
	p, r, f1 := ComputeMetrics(0, 5, 5)
	assertFloat(t, "precision", 0.0, p)
	assertFloat(t, "recall", 0.0, r)
	assertFloat(t, "f1", 0.0, f1)
}

// --- Set comparison tests ---

func TestComputeSetAccuracyExactMatch(t *testing.T) {
	expected := map[string]bool{"go": true, "python": true}
	actual := map[string]bool{"go": true, "python": true}
	fa := computeSetAccuracy("test_field", expected, actual)
	if fa.TruePositives != 2 {
		t.Errorf("TP = %d, want 2", fa.TruePositives)
	}
	if fa.FalsePositives != 0 {
		t.Errorf("FP = %d, want 0", fa.FalsePositives)
	}
	if fa.FalseNegatives != 0 {
		t.Errorf("FN = %d, want 0", fa.FalseNegatives)
	}
}

func TestComputeSetAccuracyPartialOverlap(t *testing.T) {
	expected := map[string]bool{"go": true, "python": true, "java": true}
	actual := map[string]bool{"go": true, "rust": true} // missing python, java; added rust
	fa := computeSetAccuracy("test_field", expected, actual)
	if fa.TruePositives != 1 {
		t.Errorf("TP = %d, want 1", fa.TruePositives)
	}
	if fa.FalsePositives != 1 {
		t.Errorf("FP = %d, want 1", fa.FalsePositives) // rust
	}
	if fa.FalseNegatives != 2 {
		t.Errorf("FN = %d, want 2", fa.FalseNegatives) // python, java
	}
}

func TestComputeSetAccuracyEmptySets(t *testing.T) {
	fa := computeSetAccuracy("test_field", map[string]bool{}, map[string]bool{})
	if fa.TruePositives != 0 || fa.FalsePositives != 0 || fa.FalseNegatives != 0 {
		t.Errorf("expected all zeros, got TP=%d FP=%d FN=%d",
			fa.TruePositives, fa.FalsePositives, fa.FalseNegatives)
	}
}

// --- EvalResult aggregate tests ---

func TestEvalResultOverallMetrics(t *testing.T) {
	result := &EvalResult{
		FieldResults: []FieldAccuracy{
			{FieldName: "a", TruePositives: 5, FalsePositives: 1, FalseNegatives: 2},
			{FieldName: "b", TruePositives: 3, FalsePositives: 2, FalseNegatives: 1},
		},
	}
	// Overall: TP=8, FP=3, FN=3
	// Precision = 8/11, Recall = 8/11, F1 = 8/11
	assertFloat(t, "overall_precision", 8.0/11.0, result.OverallPrecision())
	assertFloat(t, "overall_recall", 8.0/11.0, result.OverallRecall())
	assertFloat(t, "overall_f1", 8.0/11.0, result.OverallF1())
}

// --- Harness Evaluate tests ---

func TestHarnessEvaluatePerfectMatch(t *testing.T) {
	fragment := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "go.mod", Language: "go", DepCount: 3},
		},
	}

	gt := GroundTruth{
		RepoName:  "go-simple-cli",
		Ecosystem: "go",
		Expected:  *fragment,
	}

	h := NewHarness([]GroundTruth{gt})
	result, err := h.Evaluate("go-simple-cli", "test-extractor", fragment)
	if err != nil {
		t.Fatal(err)
	}

	if result.OverallF1() != 1.0 {
		t.Errorf("expected perfect F1, got %.3f", result.OverallF1())
	}
}

func TestHarnessEvaluatePartialMatch(t *testing.T) {
	expected := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
			{Primary: "python", All: []string{"python"}},
		},
	}

	actual := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
			{Primary: "rust", All: []string{"rust"}},
		},
	}

	gt := GroundTruth{
		RepoName:  "multi-lang",
		Ecosystem: "mixed",
		Expected:  *expected,
	}

	h := NewHarness([]GroundTruth{gt})
	result, err := h.Evaluate("multi-lang", "test-extractor", actual)
	if err != nil {
		t.Fatal(err)
	}

	// Languages: TP=1 (go), FP=1 (rust), FN=1 (python)
	// Precision = 1/2 = 0.5, Recall = 1/2 = 0.5, F1 = 0.5
	langFA := findFieldResult(t, result, "languages")
	assertFloat(t, "lang_precision", 0.5, langFA.Precision())
	assertFloat(t, "lang_recall", 0.5, langFA.Recall())
	assertFloat(t, "lang_f1", 0.5, langFA.F1())
}

func TestHarnessEvaluateMissingRepo(t *testing.T) {
	h := NewHarness([]GroundTruth{})
	_, err := h.Evaluate("nonexistent", "test", &architect.ProfileFragment{})
	if err == nil {
		t.Error("expected error for missing ground truth")
	}
}

// --- Import graph comparison tests ---

func TestCompareImportGraphsNil(t *testing.T) {
	fa := compareImportGraphs(nil, nil)
	assertFloat(t, "precision", 1.0, fa.Precision())
	assertFloat(t, "recall", 1.0, fa.Recall())
	assertFloat(t, "f1", 1.0, fa.F1())
}

func TestCompareImportGraphsExpectedNilActualPresent(t *testing.T) {
	actual := &architect.ImportGraph{
		Nodes: 10,
		Edges: 20,
	}
	fa := compareImportGraphs(nil, actual)
	if fa.FalsePositives != 30 {
		t.Errorf("FP = %d, want 30", fa.FalsePositives)
	}
}

func TestCompareImportGraphsActualNilExpectedPresent(t *testing.T) {
	expected := &architect.ImportGraph{
		Nodes: 10,
		Edges: 20,
	}
	fa := compareImportGraphs(expected, nil)
	if fa.FalseNegatives != 30 {
		t.Errorf("FN = %d, want 30", fa.FalseNegatives)
	}
}

func TestCompareImportGraphsExactMatch(t *testing.T) {
	graph := &architect.ImportGraph{
		Nodes: 10,
		Edges: 20,
	}
	fa := compareImportGraphs(graph, graph)
	assertFloat(t, "precision", 1.0, fa.Precision())
	assertFloat(t, "recall", 1.0, fa.Recall())
}

func TestCompareImportGraphsUndercount(t *testing.T) {
	expected := &architect.ImportGraph{Nodes: 10, Edges: 20}
	actual := &architect.ImportGraph{Nodes: 8, Edges: 15}
	fa := compareImportGraphs(expected, actual)
	// TP = min(10,8)+min(20,15) = 8+15 = 23
	// FN = (10-8)+(20-15) = 2+5 = 7
	// FP = 0
	if fa.TruePositives != 23 {
		t.Errorf("TP = %d, want 23", fa.TruePositives)
	}
	if fa.FalseNegatives != 7 {
		t.Errorf("FN = %d, want 7", fa.FalseNegatives)
	}
	if fa.FalsePositives != 0 {
		t.Errorf("FP = %d, want 0", fa.FalsePositives)
	}
}

func TestCompareImportGraphsOvercount(t *testing.T) {
	expected := &architect.ImportGraph{Nodes: 5, Edges: 10}
	actual := &architect.ImportGraph{Nodes: 8, Edges: 15}
	fa := compareImportGraphs(expected, actual)
	// TP = min(5,8)+min(10,15) = 5+10 = 15
	// FP = (8-5)+(15-10) = 3+5 = 8
	// FN = 0
	if fa.TruePositives != 15 {
		t.Errorf("TP = %d, want 15", fa.TruePositives)
	}
	if fa.FalsePositives != 8 {
		t.Errorf("FP = %d, want 8", fa.FalsePositives)
	}
	if fa.FalseNegatives != 0 {
		t.Errorf("FN = %d, want 0", fa.FalseNegatives)
	}
}

// --- Infra comparison tests ---

func TestCompareInfraBothNil(t *testing.T) {
	fa := compareInfra(nil, nil)
	assertFloat(t, "precision", 1.0, fa.Precision())
	assertFloat(t, "recall", 1.0, fa.Recall())
}

func TestCompareInfraContainerMatch(t *testing.T) {
	expected := &architect.InfraInfo{
		Containers: []architect.ContainerInfo{
			{Name: "web", Type: "service", Source: "Dockerfile"},
			{Name: "db", Type: "database", Source: "docker-compose.yml"},
		},
	}
	actual := &architect.InfraInfo{
		Containers: []architect.ContainerInfo{
			{Name: "web", Type: "service", Source: "Dockerfile"},
		},
	}
	fa := compareInfra(expected, actual)
	// TP=1 (web), FP=0, FN=1 (db)
	if fa.TruePositives != 1 {
		t.Errorf("TP = %d, want 1", fa.TruePositives)
	}
	if fa.FalseNegatives != 1 {
		t.Errorf("FN = %d, want 1", fa.FalseNegatives)
	}
}

// --- Format report test ---

func TestFormatReport(t *testing.T) {
	result := &EvalResult{
		RepoName:      "test-repo",
		Ecosystem:     "go",
		ExtractorName: "test-extractor",
		FieldResults: []FieldAccuracy{
			{FieldName: "languages", TruePositives: 2, FalsePositives: 0, FalseNegatives: 1, MatchType: "exact"},
		},
	}
	report := FormatReport(result)
	if report == "" {
		t.Error("FormatReport returned empty string")
	}
	// Verify key content is present
	if !contains(report, "test-repo") {
		t.Error("report missing repo name")
	}
	if !contains(report, "languages") {
		t.Error("report missing field name")
	}
	if !contains(report, "OVERALL") {
		t.Error("report missing overall metrics")
	}
}

// --- Diff tests (basic coverage) ---

func TestDiffFragmentsIdentical(t *testing.T) {
	frag := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
	}
	diff := DiffFragments(frag, frag)
	if diff.HasDiffs() {
		t.Errorf("identical fragments should have no diffs, got %d", len(diff.Entries))
	}
}

func TestDiffFragmentsAddedLanguage(t *testing.T) {
	expected := &architect.ProfileFragment{}
	actual := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
	}
	diff := DiffFragments(expected, actual)
	if !diff.HasDiffs() {
		t.Error("expected diffs for added language")
	}
	found := false
	for _, e := range diff.Entries {
		if e.Field == "languages" && e.Action == DiffAdded && e.Key == "go" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'added' diff entry for language 'go'")
	}
}

func TestDiffFragmentsRemovedContainer(t *testing.T) {
	expected := &architect.ProfileFragment{
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Type: "service", Source: "Dockerfile"},
			},
		},
	}
	actual := &architect.ProfileFragment{
		Infra: &architect.InfraInfo{},
	}
	diff := DiffFragments(expected, actual)
	found := false
	for _, e := range diff.Entries {
		if e.Field == "infra.containers" && e.Action == DiffRemoved && e.Key == "api" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'removed' diff entry for container 'api'")
	}
}

func TestDiffSummary(t *testing.T) {
	diff := &FragmentDiff{
		Entries: []DiffEntry{
			{Field: "f1", Action: DiffAdded, Key: "a"},
			{Field: "f2", Action: DiffRemoved, Key: "b"},
			{Field: "f3", Action: DiffModified, Key: "c"},
			{Field: "f4", Action: DiffAdded, Key: "d"},
		},
	}
	summary := diff.Summary()
	if !contains(summary, "2 added") {
		t.Errorf("expected '2 added' in summary, got: %s", summary)
	}
	if !contains(summary, "1 removed") {
		t.Errorf("expected '1 removed' in summary, got: %s", summary)
	}
	if !contains(summary, "1 modified") {
		t.Errorf("expected '1 modified' in summary, got: %s", summary)
	}
}

func TestFormatDiffEmpty(t *testing.T) {
	diff := &FragmentDiff{Entries: nil}
	output := FormatDiff(diff)
	if output != "No differences found." {
		t.Errorf("expected 'No differences found.', got: %s", output)
	}
}

// --- Spec comparison test ---

func TestCompareSpecsMatch(t *testing.T) {
	expected := []architect.SpecArtifact{
		{Kind: "openapi", Path: "api/openapi.yaml"},
		{Kind: "protobuf", Path: "proto/service.proto"},
	}
	actual := []architect.SpecArtifact{
		{Kind: "openapi", Path: "api/openapi.yaml"},
		{Kind: "protobuf", Path: "proto/service.proto"},
	}
	fa := compareSpecs(expected, actual)
	if fa.TruePositives != 2 {
		t.Errorf("TP = %d, want 2", fa.TruePositives)
	}
	assertFloat(t, "f1", 1.0, fa.F1())
}

func TestCompareSpecsPartial(t *testing.T) {
	expected := []architect.SpecArtifact{
		{Kind: "openapi", Path: "api/openapi.yaml"},
		{Kind: "protobuf", Path: "proto/service.proto"},
	}
	actual := []architect.SpecArtifact{
		{Kind: "openapi", Path: "api/openapi.yaml"},
		{Kind: "graphql", Path: "schema.graphql"},
	}
	fa := compareSpecs(expected, actual)
	// TP=1 (openapi), FP=1 (graphql), FN=1 (protobuf)
	if fa.TruePositives != 1 {
		t.Errorf("TP = %d, want 1", fa.TruePositives)
	}
	if fa.FalsePositives != 1 {
		t.Errorf("FP = %d, want 1", fa.FalsePositives)
	}
	if fa.FalseNegatives != 1 {
		t.Errorf("FN = %d, want 1", fa.FalseNegatives)
	}
}

// --- SQL comparison test ---

func TestCompareSQLMatch(t *testing.T) {
	expected := &architect.SQLAnalysis{
		Tables: []architect.Table{
			{Name: "users"},
			{Name: "orders"},
		},
	}
	actual := &architect.SQLAnalysis{
		Tables: []architect.Table{
			{Name: "users"},
			{Name: "orders"},
		},
	}
	fa := compareSQL(expected, actual)
	assertFloat(t, "f1", 1.0, fa.F1())
}

// --- Helpers ---

func assertFloat(t *testing.T, name string, expected, actual float64) {
	t.Helper()
	if math.Abs(expected-actual) > 1e-9 {
		t.Errorf("%s: expected %.6f, got %.6f", name, expected, actual)
	}
}

func findFieldResult(t *testing.T, result *EvalResult, fieldName string) FieldAccuracy {
	t.Helper()
	for _, fr := range result.FieldResults {
		if fr.FieldName == fieldName {
			return fr
		}
	}
	t.Fatalf("field %q not found in results", fieldName)
	return FieldAccuracy{}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
