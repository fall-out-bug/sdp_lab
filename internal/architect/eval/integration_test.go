package eval

import (
	"testing"

	"sdp_dev/internal/architect"
)

// TestGoldenTestSuite_FullRun runs the full golden test suite with mock extractor.
func TestGoldenTestSuite_FullRun(t *testing.T) {
	fixturesDir := "fixtures"

	suite := NewGoldenTestSuite("full-suite", fixturesDir)
	if err := suite.LoadAll(); err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(suite.cases) == 0 {
		t.Skip("no test cases found in fixtures directory")
	}

	// Create mock extractor
	mock := NewMockExtractor()

	// Run suite with mock extractor
	result, err := suite.RunSuite(func(caseName string) (*architect.ProfileFragment, error) {
		return mock.Extract(caseName), nil
	})

	if err != nil {
		t.Fatalf("RunSuite failed: %v", err)
	}

	// Verify results
	if result.Total != len(suite.cases) {
		t.Errorf("expected Total %d, got %d", len(suite.cases), result.Total)
	}

	if result.Passed+result.Failed != result.Total {
		t.Errorf("Passed+Failed (%d) != Total (%d)", result.Passed+result.Failed, result.Total)
	}

	t.Logf("Suite Results: %d/%d passed (%.1f%%)", result.Passed, result.Total, result.PassRate())

	// All mock extractions should match perfectly
	if result.Passed != result.Total {
		t.Errorf("expected all %d cases to pass, but only %d passed", result.Total, result.Passed)
	}
}

// TestGoldenTestSuite_IntegrationByEcosystem runs tests grouped by ecosystem.
func TestGoldenTestSuite_IntegrationByEcosystem(t *testing.T) {
	fixturesDir := "fixtures"

	suite := NewGoldenTestSuite("ecosystem-test", fixturesDir)
	if err := suite.LoadAll(); err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(suite.cases) == 0 {
		t.Skip("no test cases found in fixtures directory")
	}

	// Test each ecosystem
	ecosystems := []string{"go", "python", "java", "typescript", "javascript", "sql"}
	for _, ecosystem := range ecosystems {
		t.Run(ecosystem, func(t *testing.T) {
			cases := suite.ByEcosystem(ecosystem)
			if len(cases) == 0 {
				t.Skipf("no test cases for ecosystem %s", ecosystem)
			}

			t.Logf("Found %d test cases for %s", len(cases), ecosystem)

			// Run each case with mock extractor
			mock := NewMockExtractor()
			for _, tc := range cases {
				actual := mock.Extract(tc.Name)
				if actual == nil {
					t.Errorf("mock.Extract(%q) returned nil", tc.Name)
					continue
				}

				// Basic sanity check
				if len(actual.Languages) == 0 && actual.SQLAnalysis == nil {
					t.Errorf("expected at least languages or SQL analysis for %q", tc.Name)
				}
			}
		})
	}
}

// TestMetricsAggregator_FullSuite runs metrics aggregation across all test cases.
func TestMetricsAggregator_FullSuite(t *testing.T) {
	fixturesDir := "fixtures"

	suite := NewGoldenTestSuite("metrics-test", fixturesDir)
	if err := suite.LoadAll(); err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(suite.cases) == 0 {
		t.Skip("no test cases found in fixtures directory")
	}

	mock := NewMockExtractor()
	aggregator := NewMetricsAggregator()

	// Add all test cases to aggregator
	for _, tc := range suite.cases {
		actual := mock.Extract(tc.Name)
		if err := aggregator.Add(tc.Name, tc.Ecosystem, &tc.Expected, actual); err != nil {
			t.Errorf("aggregator.Add failed for %s: %v", tc.Name, err)
		}
	}

	// Compute metrics
	metrics := aggregator.Compute()

	if len(metrics) == 0 {
		t.Error("expected at least one ecosystem metric")
	}

	t.Logf("Computed metrics for %d ecosystems", len(metrics))

	// Verify each ecosystem
	for _, m := range metrics {
		t.Logf("%s: ImportF1=%.3f StyleF1=%.3f C4F1=%.3f Overall=%.3f (samples=%d)",
			m.Ecosystem, m.ImportF1, m.StyleF1, m.C4F1, m.OverallScore, m.SampleCount)

		// All metrics should be perfect since we're using the same mock data
		if m.OverallScore < 0.99 {
			t.Errorf("%s: expected perfect scores from mock data, got %.3f", m.Ecosystem, m.OverallScore)
		}
	}
}

// TestGoldenTestSuite_CI_friendly verifies that evaluation is deterministic and fast.
func TestGoldenTestSuite_CI_friendly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CI-friendly test in short mode")
	}

	fixturesDir := "fixtures"

	suite := NewGoldenTestSuite("ci-test", fixturesDir)
	if err := suite.LoadAll(); err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(suite.cases) == 0 {
		t.Skip("no test cases found in fixtures directory")
	}

	mock := NewMockExtractor()

	// Run suite twice to verify determinism
	result1, err := suite.RunSuite(func(caseName string) (*architect.ProfileFragment, error) {
		return mock.Extract(caseName), nil
	})
	if err != nil {
		t.Fatalf("RunSuite failed: %v", err)
	}

	result2, err := suite.RunSuite(func(caseName string) (*architect.ProfileFragment, error) {
		return mock.Extract(caseName), nil
	})
	if err != nil {
		t.Fatalf("RunSuite failed: %v", err)
	}

	// Results should be identical
	if result1.Total != result2.Total {
		t.Errorf("Total mismatch: %d vs %d", result1.Total, result2.Total)
	}
	if result1.Passed != result2.Passed {
		t.Errorf("Passed mismatch: %d vs %d", result1.Passed, result2.Passed)
	}
	if result1.Failed != result2.Failed {
		t.Errorf("Failed mismatch: %d vs %d", result1.Failed, result2.Failed)
	}
}

// TestExitCriteria_AllEcosystems verifies exit criteria for all ecosystems.
func TestExitCriteria_AllEcosystems(t *testing.T) {
	ecosystems := []struct {
		name     string
		criteria ExitThresholds
	}{
		{"go", DefaultExitCriteria("go")},
		{"python", DefaultExitCriteria("python")},
		{"java", DefaultExitCriteria("java")},
		{"typescript", DefaultExitCriteria("typescript")},
		{"javascript", DefaultExitCriteria("javascript")},
		{"sql", DefaultExitCriteria("sql")},
	}

	for _, tc := range ecosystems {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("%s criteria: ImportF1>=%.2f StyleF1>=%.2f C4F1>=%.2f OverallF1>=%.2f",
				tc.name,
				tc.criteria.ImportMinF1,
				tc.criteria.StyleMinF1,
				tc.criteria.C4MinF1,
				tc.criteria.OverallMinF1)

			// Verify criteria are reasonable
			if tc.criteria.ImportMinF1 < 0 || tc.criteria.ImportMinF1 > 1 {
				t.Errorf("ImportMinF1 out of range: %.2f", tc.criteria.ImportMinF1)
			}
			if tc.criteria.StyleMinF1 < 0 || tc.criteria.StyleMinF1 > 1 {
				t.Errorf("StyleMinF1 out of range: %.2f", tc.criteria.StyleMinF1)
			}
			if tc.criteria.C4MinF1 < 0 || tc.criteria.C4MinF1 > 1 {
				t.Errorf("C4MinF1 out of range: %.2f", tc.criteria.C4MinF1)
			}
		})
	}
}
