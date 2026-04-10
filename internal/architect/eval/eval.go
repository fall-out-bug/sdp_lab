// Package eval provides the evaluation harness for measuring extractor accuracy
// against golden test repositories. It computes precision, recall, and F1 metrics
// per field of a ProfileFragment, enabling systematic validation of each extractor.
package eval

import (
	"fmt"
	"strings"

	"sdp_dev/internal/architect"
)

// --- Ground Truth ---

// GroundTruth holds the expected ProfileFragment for a known repository.
// It is loaded from a golden test fixture (typically expected.json)
// and compared against the actual extractor output.
type GroundTruth struct {
	// RepoName identifies the fixture (e.g. "go-simple-cli", "python-flask").
	RepoName string `json:"repo_name"`

	// Ecosystem is the primary language/ecosystem of the fixture.
	Ecosystem string `json:"ecosystem"`

	// Expected is the ProfileFragment that extractors should produce
	// when run against this fixture repository.
	Expected architect.ProfileFragment `json:"expected"`
}

// --- Field Accuracy ---

// FieldAccuracy measures how well an extractor performed on a single field
// of the ProfileFragment.
type FieldAccuracy struct {
	// FieldName is the ProfileFragment field being measured (e.g. "languages",
	// "dependencies", "import_graph").
	FieldName string `json:"field_name"`

	// TruePositives is the count of correctly identified items.
	TruePositives int `json:"true_positives"`

	// FalsePositives is the count of items the extractor produced that are
	// not in the ground truth.
	FalsePositives int `json:"false_positives"`

	// FalseNegatives is the count of items in the ground truth that the
	// extractor missed.
	FalseNegatives int `json:"false_negatives"`

	// MatchType indicates how matching was performed.
	MatchType string `json:"match_type"` // "exact" or "partial"
}

// Precision returns TP / (TP + FP). Returns 1.0 when TP+FP == 0 and TP == 0
// (no items in either set).
func (fa FieldAccuracy) Precision() float64 {
	denom := float64(fa.TruePositives + fa.FalsePositives)
	if denom == 0 {
		if fa.FalseNegatives == 0 {
			return 1.0 // perfect: nothing expected, nothing produced
		}
		return 0.0
	}
	return float64(fa.TruePositives) / denom
}

// Recall returns TP / (TP + FN). Returns 1.0 when TP+FN == 0 (nothing expected).
func (fa FieldAccuracy) Recall() float64 {
	denom := float64(fa.TruePositives + fa.FalseNegatives)
	if denom == 0 {
		return 1.0
	}
	return float64(fa.TruePositives) / denom
}

// F1 returns the harmonic mean of Precision and Recall.
// Returns 0.0 when both precision and recall are 0.
func (fa FieldAccuracy) F1() float64 {
	p := fa.Precision()
	r := fa.Recall()
	if p+r == 0 {
		return 0.0
	}
	return 2 * (p * r) / (p + r)
}

// --- Evaluation Result ---

// EvalResult holds the full evaluation outcome for a single extractor run
// against a golden repository.
type EvalResult struct {
	// RepoName is the fixture that was tested.
	RepoName string `json:"repo_name"`

	// Ecosystem is the primary language/ecosystem.
	Ecosystem string `json:"ecosystem"`

	// ExtractorName is the Name() of the extractor that was evaluated.
	ExtractorName string `json:"extractor_name"`

	// FieldResults maps field name to its accuracy measurement.
	FieldResults []FieldAccuracy `json:"field_results"`

	// Diff is the structural diff between actual and expected fragments.
	// May be nil if diff computation was skipped.
	Diff *FragmentDiff `json:"diff,omitempty"`
}

// OverallPrecision computes the micro-averaged precision across all fields.
func (er *EvalResult) OverallPrecision() float64 {
	var tp, fp int
	for _, fr := range er.FieldResults {
		tp += fr.TruePositives
		fp += fr.FalsePositives
	}
	denom := float64(tp + fp)
	if denom == 0 {
		return 1.0
	}
	return float64(tp) / denom
}

// OverallRecall computes the micro-averaged recall across all fields.
func (er *EvalResult) OverallRecall() float64 {
	var tp, fn int
	for _, fr := range er.FieldResults {
		tp += fr.TruePositives
		fn += fr.FalseNegatives
	}
	denom := float64(tp + fn)
	if denom == 0 {
		return 1.0
	}
	return float64(tp) / denom
}

// OverallF1 computes the micro-averaged F1 across all fields.
func (er *EvalResult) OverallF1() float64 {
	p := er.OverallPrecision()
	r := er.OverallRecall()
	if p+r == 0 {
		return 0.0
	}
	return 2 * (p * r) / (p + r)
}

// --- Evaluation Harness ---

// Harness runs extractors against golden repos and computes accuracy metrics.
type Harness struct {
	groundTruths []GroundTruth
}

// NewHarness creates a Harness with the given ground truth fixtures.
func NewHarness(gt []GroundTruth) *Harness {
	return &Harness{groundTruths: gt}
}

// Evaluate compares an actual ProfileFragment against the ground truth for
// the named fixture, computing per-field precision/recall/F1.
func (h *Harness) Evaluate(repoName, extractorName string, actual *architect.ProfileFragment) (*EvalResult, error) {
	var gt *GroundTruth
	for i := range h.groundTruths {
		if h.groundTruths[i].RepoName == repoName {
			gt = &h.groundTruths[i]
			break
		}
	}
	if gt == nil {
		return nil, fmt.Errorf("ground truth not found for repo %q", repoName)
	}

	diff := DiffFragments(&gt.Expected, actual)

	result := &EvalResult{
		RepoName:      repoName,
		Ecosystem:     gt.Ecosystem,
		ExtractorName: extractorName,
		Diff:          diff,
	}

	// Compute per-field accuracy
	result.FieldResults = computeFieldAccuracies(&gt.Expected, actual, diff)

	return result, nil
}

// --- Metric Helpers ---

// ComputeMetrics computes precision, recall, and F1 from raw TP/FP/FN counts.
func ComputeMetrics(tp, fp, fn int) (precision, recall, f1 float64) {
	fa := FieldAccuracy{
		TruePositives:  tp,
		FalsePositives: fp,
		FalseNegatives: fn,
	}
	return fa.Precision(), fa.Recall(), fa.F1()
}

// computeFieldAccuracies measures per-field accuracy between expected and actual.
func computeFieldAccuracies(expected, actual *architect.ProfileFragment, diff *FragmentDiff) []FieldAccuracy {
	var results []FieldAccuracy

	// Languages: compare by primary language string
	results = append(results, compareLanguages(expected.Languages, actual.Languages))

	// Dependencies: compare by manifest file path
	results = append(results, compareDependencies(expected.Dependencies, actual.Dependencies))

	// Import graph: compare edges and clusters
	results = append(results, compareImportGraphs(expected.ImportGraph, actual.ImportGraph))

	// Infra: compare containers by name
	results = append(results, compareInfra(expected.Infra, actual.Infra))

	// FileTree: compare top-level entries and extension counts
	results = append(results, compareFileTree(expected.FileTree, actual.FileTree))

	// Specs: compare by path
	results = append(results, compareSpecs(expected.Specs, actual.Specs))

	// SQL: compare table names
	results = append(results, compareSQL(expected.SQLAnalysis, actual.SQLAnalysis))

	return results
}

// --- Field Comparison Functions ---

func compareLanguages(expected, actual []architect.LanguageInfo) FieldAccuracy {
	expectedSet := make(map[string]bool)
	for _, l := range expected {
		expectedSet[l.Primary] = true
	}
	actualSet := make(map[string]bool)
	for _, l := range actual {
		actualSet[l.Primary] = true
	}
	return computeSetAccuracy("languages", expectedSet, actualSet)
}

func compareDependencies(expected, actual []architect.DependencyInfo) FieldAccuracy {
	expectedSet := make(map[string]bool)
	for _, d := range expected {
		key := d.File
		if key == "" {
			key = d.Language
		}
		expectedSet[key] = true
	}
	actualSet := make(map[string]bool)
	for _, d := range actual {
		key := d.File
		if key == "" {
			key = d.Language
		}
		actualSet[key] = true
	}
	return computeSetAccuracy("dependencies", expectedSet, actualSet)
}

func compareImportGraphs(expected, actual *architect.ImportGraph) FieldAccuracy {
	fa := FieldAccuracy{FieldName: "import_graph", MatchType: "exact"}

	if expected == nil && actual == nil {
		fa.MatchType = "exact"
		return fa // 0/0/0 -> precision=1, recall=1, F1=1
	}
	if expected == nil {
		// Expected nil, actual non-nil: all actual items are false positives
		fa.FalsePositives = actual.Nodes + actual.Edges
		return fa
	}
	if actual == nil {
		// Expected non-nil, actual nil: all expected items are false negatives
		fa.FalseNegatives = expected.Nodes + expected.Edges
		return fa
	}

	// Compare nodes and edges as counts (directional comparison)
	fa.TruePositives = min(expected.Nodes, actual.Nodes) + min(expected.Edges, actual.Edges)
	if actual.Nodes > expected.Nodes {
		fa.FalsePositives = actual.Nodes - expected.Nodes
	}
	if expected.Nodes > actual.Nodes {
		fa.FalseNegatives = expected.Nodes - actual.Nodes
	}
	if actual.Edges > expected.Edges {
		fa.FalsePositives += actual.Edges - expected.Edges
	}
	if expected.Edges > actual.Edges {
		fa.FalseNegatives += expected.Edges - actual.Edges
	}

	return fa
}

func compareInfra(expected, actual *architect.InfraInfo) FieldAccuracy {
	expectedNames := make(map[string]bool)
	if expected != nil {
		for _, c := range expected.Containers {
			expectedNames[c.Name] = true
		}
	}
	actualNames := make(map[string]bool)
	if actual != nil {
		for _, c := range actual.Containers {
			actualNames[c.Name] = true
		}
	}
	return computeSetAccuracy("infra_containers", expectedNames, actualNames)
}

func compareFileTree(expected, actual *architect.FileTreeInfo) FieldAccuracy {
	fa := FieldAccuracy{FieldName: "file_tree", MatchType: "exact"}

	if expected == nil && actual == nil {
		return fa
	}
	if expected == nil || actual == nil {
		if expected != nil {
			fa.FalseNegatives = expected.TotalFiles
		}
		if actual != nil {
			fa.FalsePositives = actual.TotalFiles
		}
		return fa
	}

	// Compare top-level entries as a set
	expectedTopLevel := make(map[string]bool)
	for _, dir := range expected.TopLevel {
		expectedTopLevel[dir] = true
	}
	actualTopLevel := make(map[string]bool)
	for _, dir := range actual.TopLevel {
		actualTopLevel[dir] = true
	}

	sub := computeSetAccuracy("file_tree_top_level", expectedTopLevel, actualTopLevel)
	fa.TruePositives = sub.TruePositives
	fa.FalsePositives = sub.FalsePositives
	fa.FalseNegatives = sub.FalseNegatives
	return fa
}

func compareSpecs(expected, actual []architect.SpecArtifact) FieldAccuracy {
	expectedPaths := make(map[string]bool)
	for _, s := range expected {
		expectedPaths[s.Path] = true
	}
	actualPaths := make(map[string]bool)
	for _, s := range actual {
		actualPaths[s.Path] = true
	}
	return computeSetAccuracy("specs", expectedPaths, actualPaths)
}

func compareSQL(expected, actual *architect.SQLAnalysis) FieldAccuracy {
	expectedTables := make(map[string]bool)
	if expected != nil {
		for _, t := range expected.Tables {
			expectedTables[t.Name] = true
		}
	}
	actualTables := make(map[string]bool)
	if actual != nil {
		for _, t := range actual.Tables {
			actualTables[t.Name] = true
		}
	}
	return computeSetAccuracy("sql_tables", expectedTables, actualTables)
}

// computeSetAccuracy compares two sets and returns TP/FP/FN counts.
func computeSetAccuracy(fieldName string, expected, actual map[string]bool) FieldAccuracy {
	fa := FieldAccuracy{
		FieldName: fieldName,
		MatchType: "exact",
	}

	// True positives: items in both expected and actual
	for item := range expected {
		if actual[item] {
			fa.TruePositives++
		} else {
			fa.FalseNegatives++
		}
	}

	// False positives: items in actual but not in expected
	for item := range actual {
		if !expected[item] {
			fa.FalsePositives++
		}
	}

	return fa
}

// --- Formatting ---

// FormatReport produces a human-readable evaluation report.
func FormatReport(result *EvalResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "=== Evaluation Report ===\n")
	fmt.Fprintf(&b, "Repo:       %s\n", result.RepoName)
	fmt.Fprintf(&b, "Ecosystem:  %s\n", result.Ecosystem)
	fmt.Fprintf(&b, "Extractor:  %s\n\n", result.ExtractorName)

	fmt.Fprintf(&b, "%-25s  %8s  %8s  %8s  %s\n", "Field", "Prec", "Recall", "F1", "Type")
	fmt.Fprintf(&b, "%s\n", strings.Repeat("-", 70))

	for _, fr := range result.FieldResults {
		fmt.Fprintf(&b, "%-25s  %8.3f  %8.3f  %8.3f  %s (TP=%d FP=%d FN=%d)\n",
			fr.FieldName, fr.Precision(), fr.Recall(), fr.F1(),
			fr.MatchType, fr.TruePositives, fr.FalsePositives, fr.FalseNegatives)
	}

	fmt.Fprintf(&b, "%s\n", strings.Repeat("-", 70))
	fmt.Fprintf(&b, "%-25s  %8.3f  %8.3f  %8.3f  (micro-avg)\n",
		"OVERALL", result.OverallPrecision(), result.OverallRecall(), result.OverallF1())

	if result.Diff != nil && len(result.Diff.Entries) > 0 {
		fmt.Fprintf(&b, "\n%s\n", FormatDiff(result.Diff))
	}

	return b.String()
}

// --- LanguageInfo access helper ---
// LanguageInfo is referenced from profile.go. We need it for the comparison.
