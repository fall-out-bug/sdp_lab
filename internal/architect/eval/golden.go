// Package eval provides golden test suite management.
// It loads golden test cases from fixture directories and manages expected results.
package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"sdp_dev/internal/architect"
)

// GoldenTestSuite manages a collection of golden test cases.
type GoldenTestSuite struct {
	Name   string
	cases  []GoldenTestCase
	root   string
}

// GoldenTestCase represents a single golden test case.
type GoldenTestCase struct {
	// Name is the test case name (e.g., "go-simple-cli").
	Name string `json:"name"`

	// Ecosystem is the primary language/ecosystem.
	Ecosystem string `json:"ecosystem"`

	// Description describes what this test case covers.
	Description string `json:"description,omitempty"`

	// ExpectedPath is the path to the expected.json file.
	ExpectedPath string `json:"expected_path"`

	// RepoPath is the path to the fixture repository (if running extraction).
	RepoPath string `json:"repo_path,omitempty"`

	// Expected is the expected ProfileFragment (loaded from ExpectedPath).
	Expected architect.ProfileFragment `json:"expected"`

	// ExitCriteria specifies the minimum acceptable scores for this test case.
	ExitCriteria ExitThresholds `json:"exit_criteria"`
}

// ExitThresholds defines the minimum acceptable scores for a test case.
type ExitThresholds struct {
	ImportMinF1     float64 `json:"import_min_f1"`
	StyleMinF1      float64 `json:"style_min_f1"`
	C4MinF1         float64 `json:"c4_min_f1"`
	SchemaMinF1     float64 `json:"schema_min_f1,omitempty"` // for SQL tests
	OverallMinF1    float64 `json:"overall_min_f1"`
}

// DefaultExitCriteria returns the default exit criteria for an ecosystem.
func DefaultExitCriteria(ecosystem string) ExitThresholds {
	switch ecosystem {
	case "go":
		return ExitThresholds{
			ImportMinF1:  0.90,
			StyleMinF1:   0.85,
			C4MinF1:      0.80,
			OverallMinF1: 0.85,
		}
	case "python", "java":
		return ExitThresholds{
			ImportMinF1:  0.65,
			StyleMinF1:   0.75,
			C4MinF1:      0.70,
			OverallMinF1: 0.70,
		}
	case "typescript", "javascript":
		return ExitThresholds{
			ImportMinF1:  0.65,
			StyleMinF1:   0.70,
			C4MinF1:      0.65,
			OverallMinF1: 0.67,
		}
	case "sql":
		return ExitThresholds{
			ImportMinF1:  0.0, // not applicable
			StyleMinF1:   0.0, // not applicable
			C4MinF1:      0.0, // not applicable
			SchemaMinF1:  0.80,
			OverallMinF1: 0.80,
		}
	default:
		return ExitThresholds{
			ImportMinF1:  0.70,
			StyleMinF1:   0.70,
			C4MinF1:      0.70,
			OverallMinF1: 0.70,
		}
	}
}

// NewGoldenTestSuite creates a new golden test suite.
func NewGoldenTestSuite(name, root string) *GoldenTestSuite {
	return &GoldenTestSuite{
		Name:  name,
		root:  root,
		cases: []GoldenTestCase{},
	}
}

// LoadCase loads a golden test case from a directory.
func (suite *GoldenTestSuite) LoadCase(caseName string) error {
	caseDir := filepath.Join(suite.root, caseName)
	expectedPath := filepath.Join(caseDir, "expected.json")

	// Check if directory exists
	if _, err := os.Stat(caseDir); os.IsNotExist(err) {
		return fmt.Errorf("case directory not found: %s", caseDir)
	}

	// Load expected.json
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		return fmt.Errorf("read expected.json: %w", err)
	}

	var expected architect.ProfileFragment
	if err := json.Unmarshal(data, &expected); err != nil {
		return fmt.Errorf("unmarshal expected.json: %w", err)
	}

	// Determine ecosystem from case name prefix
	ecosystem := extractEcosystem(caseName)

	// Create test case
	tc := GoldenTestCase{
		Name:         caseName,
		Ecosystem:    ecosystem,
		ExpectedPath: expectedPath,
		RepoPath:     caseDir,
		Expected:     expected,
		ExitCriteria: DefaultExitCriteria(ecosystem),
	}

	suite.cases = append(suite.cases, tc)
	return nil
}

// LoadAll loads all test cases from the suite root.
func (suite *GoldenTestSuite) LoadAll() error {
	entries, err := os.ReadDir(suite.root)
	if err != nil {
		return fmt.Errorf("read suite root: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		caseName := entry.Name()
		// Skip hidden directories
		if caseName[0] == '.' {
			continue
		}

		if err := suite.LoadCase(caseName); err != nil {
			return fmt.Errorf("load case %s: %w", caseName, err)
		}
	}

	return nil
}

// Cases returns all loaded test cases.
func (suite *GoldenTestSuite) Cases() []GoldenTestCase {
	return suite.cases
}

// GetCase returns a test case by name.
func (suite *GoldenTestSuite) GetCase(name string) (*GoldenTestCase, error) {
	for i := range suite.cases {
		if suite.cases[i].Name == name {
			return &suite.cases[i], nil
		}
	}
	return nil, fmt.Errorf("test case not found: %s", name)
}

// ByEcosystem returns test cases filtered by ecosystem.
func (suite *GoldenTestSuite) ByEcosystem(ecosystem string) []GoldenTestCase {
	var filtered []GoldenTestCase
	for _, tc := range suite.cases {
		if tc.Ecosystem == ecosystem {
			filtered = append(filtered, tc)
		}
	}
	return filtered
}

// RunCase runs a single test case against an extractor function.
func (suite *GoldenTestSuite) RunCase(caseName string, extractor func() (*architect.ProfileFragment, error)) (*EvalResult, error) {
	tc, err := suite.GetCase(caseName)
	if err != nil {
		return nil, err
	}

	actual, err := extractor()
	if err != nil {
		return nil, fmt.Errorf("extractor failed: %w", err)
	}

	h := NewHarness([]GroundTruth{{
		RepoName:  tc.Name,
		Ecosystem: tc.Ecosystem,
		Expected:  tc.Expected,
	}})

	result, err := h.Evaluate(tc.Name, "test-extractor", actual)
	if err != nil {
		return nil, fmt.Errorf("evaluation failed: %w", err)
	}

	return result, nil
}

// RunSuite runs all test cases against an extractor function.
func (suite *GoldenTestSuite) RunSuite(extractor func(caseName string) (*architect.ProfileFragment, error)) (*SuiteResult, error) {
	result := &SuiteResult{
		SuiteName: suite.Name,
		Total:     len(suite.cases),
	}

	for _, tc := range suite.cases {
		actual, err := extractor(tc.Name)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", tc.Name, err))
			result.Failed++
			continue
		}

		h := NewHarness([]GroundTruth{{
			RepoName:  tc.Name,
			Ecosystem: tc.Ecosystem,
			Expected:  tc.Expected,
		}})

		evalResult, err := h.Evaluate(tc.Name, "test-extractor", actual)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", tc.Name, err))
			result.Failed++
			continue
		}

		// Check if case meets exit criteria
		if suite.meetsCriteria(evalResult, tc.ExitCriteria) {
			result.Passed++
		} else {
			result.Failed++
		}

		result.Results = append(result.Results, evalResult)
	}

	return result, nil
}

// meetsCriteria checks if an evaluation result meets the exit criteria.
func (suite *GoldenTestSuite) meetsCriteria(result *EvalResult, criteria ExitThresholds) bool {
	// Find the import accuracy field
	var importF1, styleF1, c4F1 float64
	for _, fa := range result.FieldResults {
		switch fa.FieldName {
		case "import_graph":
			importF1 = fa.F1()
		case "languages":
			styleF1 = fa.F1()
		case "infra_containers":
			c4F1 = fa.F1()
		}
	}

	overallF1 := result.OverallF1()

	// Check each criterion
	if importF1 < criteria.ImportMinF1 {
		return false
	}
	if styleF1 < criteria.StyleMinF1 {
		return false
	}
	if c4F1 < criteria.C4MinF1 {
		return false
	}
	if criteria.SchemaMinF1 > 0 {
		// Find schema F1
		for _, fa := range result.FieldResults {
			if fa.FieldName == "sql_tables" {
				if fa.F1() < criteria.SchemaMinF1 {
					return false
				}
			}
		}
	}
	if overallF1 < criteria.OverallMinF1 {
		return false
	}

	return true
}

// SuiteResult holds the results of running a test suite.
type SuiteResult struct {
	SuiteName string         `json:"suite_name"`
	Total     int            `json:"total"`
	Passed    int            `json:"passed"`
	Failed    int            `json:"failed"`
	Results   []*EvalResult  `json:"results"`
	Errors    []string       `json:"errors,omitempty"`
}

// PassRate returns the pass rate as a percentage.
func (sr *SuiteResult) PassRate() float64 {
	if sr.Total == 0 {
		return 100.0
	}
	return 100.0 * float64(sr.Passed) / float64(sr.Total)
}

// extractEcosystem extracts the ecosystem from a case name.
// Example: "go-simple-cli" -> "go"
func extractEcosystem(caseName string) string {
	// Simple heuristic: extract prefix before first dash
	for i, ch := range caseName {
		if ch == '-' {
			return caseName[:i]
		}
	}
	return caseName
}

// WriteExpected writes an expected.json file for a test case.
func WriteExpected(caseDir string, expected *architect.ProfileFragment) error {
	if err := os.MkdirAll(caseDir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	data, err := json.MarshalIndent(expected, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	path := filepath.Join(caseDir, "expected.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

// LoadExpected loads an expected.json file for a test case.
func LoadExpected(caseDir string) (*architect.ProfileFragment, error) {
	path := filepath.Join(caseDir, "expected.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var expected architect.ProfileFragment
	if err := json.Unmarshal(data, &expected); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return &expected, nil
}
