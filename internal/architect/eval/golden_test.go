package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/architect"
)

func TestNewGoldenTestSuite(t *testing.T) {
	suite := NewGoldenTestSuite("test-suite", "/tmp/test-fixtures")
	if suite == nil {
		t.Fatal("NewGoldenTestSuite returned nil")
	}
	if suite.Name != "test-suite" {
		t.Errorf("expected name 'test-suite', got %s", suite.Name)
	}
	if suite.root != "/tmp/test-fixtures" {
		t.Errorf("expected root '/tmp/test-fixtures', got %s", suite.root)
	}
	if len(suite.cases) != 0 {
		t.Errorf("expected 0 cases, got %d", len(suite.cases))
	}
}

func TestDefaultExitCriteria(t *testing.T) {
	tests := []struct {
		name     string
		expected ExitThresholds
	}{
		{
			name: "go",
			expected: ExitThresholds{
				ImportMinF1:  0.90,
				StyleMinF1:   0.85,
				C4MinF1:      0.80,
				OverallMinF1: 0.85,
			},
		},
		{
			name: "python",
			expected: ExitThresholds{
				ImportMinF1:  0.65,
				StyleMinF1:   0.75,
				C4MinF1:      0.70,
				OverallMinF1: 0.70,
			},
		},
		{
			name: "java",
			expected: ExitThresholds{
				ImportMinF1:  0.65,
				StyleMinF1:   0.75,
				C4MinF1:      0.70,
				OverallMinF1: 0.70,
			},
		},
		{
			name: "typescript",
			expected: ExitThresholds{
				ImportMinF1:  0.65,
				StyleMinF1:   0.70,
				C4MinF1:      0.65,
				OverallMinF1: 0.67,
			},
		},
		{
			name: "javascript",
			expected: ExitThresholds{
				ImportMinF1:  0.65,
				StyleMinF1:   0.70,
				C4MinF1:      0.65,
				OverallMinF1: 0.67,
			},
		},
		{
			name: "sql",
			expected: ExitThresholds{
				ImportMinF1:  0.0,
				StyleMinF1:   0.0,
				C4MinF1:      0.0,
				SchemaMinF1:  0.80,
				OverallMinF1: 0.80,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			criteria := DefaultExitCriteria(tt.name)
			if criteria != tt.expected {
				t.Errorf("criteria mismatch:\ngot:  %+v\nwant: %+v", criteria, tt.expected)
			}
		})
	}
}

func TestExtractEcosystem(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"go-simple-cli", "go"},
		{"python-flask", "python"},
		{"java-spring-boot", "java"},
		{"typescript-nextjs", "typescript"},
		{"javascript-express", "javascript"},
		{"sql-migration-dir", "sql"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractEcosystem(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGoldenTestSuite_LoadCase(t *testing.T) {
	tmpDir := t.TempDir()
	suite := NewGoldenTestSuite("test", tmpDir)

	// Create test case directory
	caseDir := filepath.Join(tmpDir, "go-test-case")
	if err := os.Mkdir(caseDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write expected.json
	expected := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
	}
	if err := WriteExpected(caseDir, expected); err != nil {
		t.Fatalf("WriteExpected: %v", err)
	}

	// Load the case
	if err := suite.LoadCase("go-test-case"); err != nil {
		t.Fatalf("LoadCase failed: %v", err)
	}

	// Verify
	if len(suite.cases) != 1 {
		t.Fatalf("expected 1 case, got %d", len(suite.cases))
	}

	tc := suite.cases[0]
	if tc.Name != "go-test-case" {
		t.Errorf("expected name 'go-test-case', got %s", tc.Name)
	}
	if tc.Ecosystem != "go" {
		t.Errorf("expected ecosystem 'go', got %s", tc.Ecosystem)
	}
}

func TestGoldenTestSuite_LoadCase_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	suite := NewGoldenTestSuite("test", tmpDir)

	err := suite.LoadCase("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent case")
	}
}

func TestGoldenTestSuite_LoadCase_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	suite := NewGoldenTestSuite("test", tmpDir)

	// Create test case directory
	caseDir := filepath.Join(tmpDir, "go-test-case")
	if err := os.Mkdir(caseDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write invalid JSON
	expectedPath := filepath.Join(caseDir, "expected.json")
	if err := os.WriteFile(expectedPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := suite.LoadCase("go-test-case")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestGoldenTestSuite_LoadAll(t *testing.T) {
	tmpDir := t.TempDir()
	suite := NewGoldenTestSuite("test", tmpDir)

	// Create test case directories
	cases := []string{"go-test-1", "python-test-1"}
	for _, caseName := range cases {
		caseDir := filepath.Join(tmpDir, caseName)
		if err := os.Mkdir(caseDir, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", caseName, err)
		}

		expected := &architect.ProfileFragment{
			Languages: []architect.LanguageInfo{
				{Primary: "test", All: []string{"test"}},
			},
		}
		if err := WriteExpected(caseDir, expected); err != nil {
			t.Fatalf("WriteExpected: %v", err)
		}
	}

	// Load all
	if err := suite.LoadAll(); err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	// Verify
	if len(suite.cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(suite.cases))
	}
}

func TestGoldenTestSuite_GetCase(t *testing.T) {
	tmpDir := t.TempDir()
	suite := NewGoldenTestSuite("test", tmpDir)

	// Create test case
	caseDir := filepath.Join(tmpDir, "go-test")
	if err := os.Mkdir(caseDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	expected := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
	}
	if err := WriteExpected(caseDir, expected); err != nil {
		t.Fatalf("WriteExpected: %v", err)
	}

	if err := suite.LoadCase("go-test"); err != nil {
		t.Fatalf("LoadCase: %v", err)
	}

	// Get case
	tc, err := suite.GetCase("go-test")
	if err != nil {
		t.Fatalf("GetCase failed: %v", err)
	}

	if tc.Name != "go-test" {
		t.Errorf("expected name 'go-test', got %s", tc.Name)
	}
}

func TestGoldenTestSuite_GetCase_NotFound(t *testing.T) {
	suite := NewGoldenTestSuite("test", "/tmp")
	_, err := suite.GetCase("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent case")
	}
}

func TestGoldenTestSuite_ByEcosystem(t *testing.T) {
	tmpDir := t.TempDir()
	suite := NewGoldenTestSuite("test", tmpDir)

	// Create test cases for different ecosystems
	cases := []struct {
		name      string
		ecosystem string
	}{
		{"go-test-1", "go"},
		{"go-test-2", "go"},
		{"python-test-1", "python"},
	}

	for _, tc := range cases {
		caseDir := filepath.Join(tmpDir, tc.name)
		if err := os.Mkdir(caseDir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		expected := &architect.ProfileFragment{
			Languages: []architect.LanguageInfo{
				{Primary: tc.ecosystem, All: []string{tc.ecosystem}},
			},
		}
		if err := WriteExpected(caseDir, expected); err != nil {
			t.Fatalf("WriteExpected: %v", err)
		}

		if err := suite.LoadCase(tc.name); err != nil {
			t.Fatalf("LoadCase: %v", err)
		}
	}

	// Filter by ecosystem
	goCases := suite.ByEcosystem("go")
	if len(goCases) != 2 {
		t.Errorf("expected 2 go cases, got %d", len(goCases))
	}

	pyCases := suite.ByEcosystem("python")
	if len(pyCases) != 1 {
		t.Errorf("expected 1 python case, got %d", len(pyCases))
	}

	jsCases := suite.ByEcosystem("javascript")
	if len(jsCases) != 0 {
		t.Errorf("expected 0 javascript cases, got %d", len(jsCases))
	}
}

func TestGoldenTestSuite_RunCase_PerfectMatch(t *testing.T) {
	tmpDir := t.TempDir()
	suite := NewGoldenTestSuite("test", tmpDir)

	// Create test case
	caseDir := filepath.Join(tmpDir, "go-test")
	if err := os.Mkdir(caseDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	expected := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
	}
	if err := WriteExpected(caseDir, expected); err != nil {
		t.Fatalf("WriteExpected: %v", err)
	}

	if err := suite.LoadCase("go-test"); err != nil {
		t.Fatalf("LoadCase: %v", err)
	}

	// Run with perfect match extractor
	result, err := suite.RunCase("go-test", func() (*architect.ProfileFragment, error) {
		return &architect.ProfileFragment{
			Languages: []architect.LanguageInfo{
				{Primary: "go", All: []string{"go"}},
			},
		}, nil
	})

	if err != nil {
		t.Fatalf("RunCase failed: %v", err)
	}

	if result.OverallF1() != 1.0 {
		t.Errorf("expected perfect F1, got %.3f", result.OverallF1())
	}
}

func TestGoldenTestSuite_RunCase_ExtractorError(t *testing.T) {
	tmpDir := t.TempDir()
	suite := NewGoldenTestSuite("test", tmpDir)

	// Create test case
	caseDir := filepath.Join(tmpDir, "go-test")
	if err := os.Mkdir(caseDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	expected := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
	}
	if err := WriteExpected(caseDir, expected); err != nil {
		t.Fatalf("WriteExpected: %v", err)
	}

	if err := suite.LoadCase("go-test"); err != nil {
		t.Fatalf("LoadCase: %v", err)
	}

	// Run with failing extractor
	_, err := suite.RunCase("go-test", func() (*architect.ProfileFragment, error) {
		return nil, fmt.Errorf("extractor failed")
	})

	if err == nil {
		t.Error("expected error for failing extractor")
	}
}

func TestSuiteResult_PassRate(t *testing.T) {
	result := &SuiteResult{
		Total:  10,
		Passed: 8,
		Failed: 2,
	}

	passRate := result.PassRate()
	if passRate != 80.0 {
		t.Errorf("expected pass rate 80.0, got %.1f", passRate)
	}
}

func TestSuiteResult_PassRate_Empty(t *testing.T) {
	result := &SuiteResult{
		Total:  0,
		Passed: 0,
		Failed: 0,
	}

	passRate := result.PassRate()
	if passRate != 100.0 {
		t.Errorf("expected pass rate 100.0 for empty suite, got %.1f", passRate)
	}
}

func TestWriteExpected(t *testing.T) {
	tmpDir := t.TempDir()
	caseDir := filepath.Join(tmpDir, "test-case")

	expected := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
	}

	if err := WriteExpected(caseDir, expected); err != nil {
		t.Fatalf("WriteExpected failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(caseDir, "expected.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected.json was not created")
	}

	// Verify content
	loaded, err := LoadExpected(caseDir)
	if err != nil {
		t.Fatalf("LoadExpected failed: %v", err)
	}

	if len(loaded.Languages) != 1 {
		t.Errorf("expected 1 language, got %d", len(loaded.Languages))
	}
	if loaded.Languages[0].Primary != "go" {
		t.Errorf("expected primary 'go', got %s", loaded.Languages[0].Primary)
	}
}

func TestLoadExpected_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadExpected(tmpDir)
	if err == nil {
		t.Error("expected error for missing expected.json")
	}
}

func TestMeetsCriteria_AllPass(t *testing.T) {
	suite := &GoldenTestSuite{}
	criteria := ExitThresholds{
		ImportMinF1:  0.90,
		StyleMinF1:   0.85,
		C4MinF1:      0.80,
		OverallMinF1: 0.85,
	}

	// Create mock result with perfect scores
	result := &EvalResult{
		FieldResults: []FieldAccuracy{
			{FieldName: "import_graph", TruePositives: 10, FalsePositives: 0, FalseNegatives: 0},
			{FieldName: "languages", TruePositives: 10, FalsePositives: 0, FalseNegatives: 0},
			{FieldName: "infra_containers", TruePositives: 10, FalsePositives: 0, FalseNegatives: 0},
		},
	}

	passes := suite.meetsCriteria(result, criteria)
	if !passes {
		t.Error("expected criteria to pass")
	}
}

func TestMeetsCriteria_ImportFails(t *testing.T) {
	suite := &GoldenTestSuite{}
	criteria := ExitThresholds{
		ImportMinF1:  0.90,
		StyleMinF1:   0.85,
		C4MinF1:      0.80,
		OverallMinF1: 0.85,
	}

	// Create mock result with low import F1
	result := &EvalResult{
		FieldResults: []FieldAccuracy{
			{FieldName: "import_graph", TruePositives: 5, FalsePositives: 5, FalseNegatives: 0}, // F1 = 0.5
			{FieldName: "languages", TruePositives: 10, FalsePositives: 0, FalseNegatives: 0},
			{FieldName: "infra_containers", TruePositives: 10, FalsePositives: 0, FalseNegatives: 0},
		},
	}

	passes := suite.meetsCriteria(result, criteria)
	if passes {
		t.Error("expected criteria to fail due to low import F1")
	}
}
