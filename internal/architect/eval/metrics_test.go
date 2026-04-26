package eval

import (
	"testing"

	"sdp_dev/internal/architect"
)

func TestNewMetricsAggregator(t *testing.T) {
	ma := NewMetricsAggregator()
	if ma == nil {
		t.Fatal("NewMetricsAggregator returned nil")
	}
	if ma.byEcosystem == nil {
		t.Error("byEcosystem map not initialized")
	}
}

func TestMetricsAggregator_Add_SingleSample(t *testing.T) {
	ma := NewMetricsAggregator()

	expected := &architect.ProfileFragment{
		ImportGraph: &architect.ImportGraph{Nodes: 10, Edges: 20},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "web", Type: "service", Source: "Dockerfile"},
			},
		},
	}

	actual := &architect.ProfileFragment{
		ImportGraph: &architect.ImportGraph{Nodes: 10, Edges: 20},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "web", Type: "service", Source: "Dockerfile"},
			},
		},
	}

	err := ma.Add("test-repo", "go", expected, actual)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	metrics := ma.Compute()
	if len(metrics) != 1 {
		t.Fatalf("expected 1 ecosystem, got %d", len(metrics))
	}

	if metrics[0].Ecosystem != "go" {
		t.Errorf("expected ecosystem 'go', got %s", metrics[0].Ecosystem)
	}

	if metrics[0].SampleCount != 1 {
		t.Errorf("expected SampleCount 1, got %d", metrics[0].SampleCount)
	}

	// Perfect match should yield F1 = 1.0
	if metrics[0].ImportF1 != 1.0 {
		t.Errorf("expected ImportF1 1.0, got %.3f", metrics[0].ImportF1)
	}
	if metrics[0].C4F1 != 1.0 {
		t.Errorf("expected C4F1 1.0, got %.3f", metrics[0].C4F1)
	}
}

func TestMetricsAggregator_Add_MultipleSamples(t *testing.T) {
	ma := NewMetricsAggregator()

	// Sample 1: perfect match
	expected1 := &architect.ProfileFragment{
		ImportGraph: &architect.ImportGraph{Nodes: 10, Edges: 20},
	}
	actual1 := &architect.ProfileFragment{
		ImportGraph: &architect.ImportGraph{Nodes: 10, Edges: 20},
	}
	if err := ma.Add("repo1", "go", expected1, actual1); err != nil {
		t.Fatalf("ma.Add: %v", err)
	}

	// Sample 2: partial match
	expected2 := &architect.ProfileFragment{
		ImportGraph: &architect.ImportGraph{Nodes: 10, Edges: 20},
	}
	actual2 := &architect.ProfileFragment{
		ImportGraph: &architect.ImportGraph{Nodes: 8, Edges: 15},
	}
	_ = ma.Add("repo2", "go", expected2, actual2)

	metrics := ma.Compute()
	if len(metrics) != 1 {
		t.Fatalf("expected 1 ecosystem, got %d", len(metrics))
	}

	if metrics[0].SampleCount != 2 {
		t.Errorf("expected SampleCount 2, got %d", metrics[0].SampleCount)
	}

	// Should have some precision/recall between 0 and 1
	if metrics[0].ImportPrecision < 0 || metrics[0].ImportPrecision > 1 {
		t.Errorf("ImportPrecision out of range: %.3f", metrics[0].ImportPrecision)
	}
	if metrics[0].ImportRecall < 0 || metrics[0].ImportRecall > 1 {
		t.Errorf("ImportRecall out of range: %.3f", metrics[0].ImportRecall)
	}
}

func TestMetricsAggregator_Add_MultipleEcosystems(t *testing.T) {
	ma := NewMetricsAggregator()

	// Go sample
	goExpected := &architect.ProfileFragment{
		ImportGraph: &architect.ImportGraph{Nodes: 10, Edges: 20},
	}
	goActual := &architect.ProfileFragment{
		ImportGraph: &architect.ImportGraph{Nodes: 10, Edges: 20},
	}
	_ = ma.Add("go-repo", "go", goExpected, goActual)

	// Python sample
	pyExpected := &architect.ProfileFragment{
		ImportGraph: &architect.ImportGraph{Nodes: 5, Edges: 10},
	}
	pyActual := &architect.ProfileFragment{
		ImportGraph: &architect.ImportGraph{Nodes: 5, Edges: 10},
	}
	_ = ma.Add("py-repo", "python", pyExpected, pyActual)

	metrics := ma.Compute()
	if len(metrics) != 2 {
		t.Fatalf("expected 2 ecosystems, got %d", len(metrics))
	}

	// Find each ecosystem
	var goMetrics, pyMetrics *EcosystemMetrics
	for i := range metrics {
		if metrics[i].Ecosystem == "go" {
			goMetrics = &metrics[i]
		}
		if metrics[i].Ecosystem == "python" {
			pyMetrics = &metrics[i]
		}
	}

	if goMetrics == nil {
		t.Error("go ecosystem not found")
	}
	if pyMetrics == nil {
		t.Error("python ecosystem not found")
	}
}

func TestMetricsAggregator_Compute_Empty(t *testing.T) {
	ma := NewMetricsAggregator()
	metrics := ma.Compute()

	if len(metrics) != 0 {
		t.Errorf("expected 0 ecosystems, got %d", len(metrics))
	}
}

func TestComputeF1FromCounts_Perfect(t *testing.T) {
	f1 := computeF1FromCounts(10, 0, 0)
	if f1 != 1.0 {
		t.Errorf("expected F1 1.0, got %.3f", f1)
	}
}

func TestComputeF1FromCounts_Partial(t *testing.T) {
	// TP=8, FP=2, FN=3
	// Precision = 8/10 = 0.8
	// Recall = 8/11 ≈ 0.727
	// F1 = 2 * (0.8 * 0.727) / (0.8 + 0.727) ≈ 0.762
	f1 := computeF1FromCounts(8, 2, 3)
	if f1 < 0.76 || f1 > 0.77 {
		t.Errorf("expected F1 ≈ 0.762, got %.3f", f1)
	}
}

func TestComputeF1FromCounts_AllWrong(t *testing.T) {
	f1 := computeF1FromCounts(0, 5, 5)
	if f1 != 0.0 {
		t.Errorf("expected F1 0.0, got %.3f", f1)
	}
}

func TestComputeF1FromCounts_Empty(t *testing.T) {
	f1 := computeF1FromCounts(0, 0, 0)
	if f1 != 1.0 {
		t.Errorf("expected F1 1.0 (perfect empty), got %.3f", f1)
	}
}

func TestCheckThresholds_Go_Pass(t *testing.T) {
	metrics := EcosystemMetrics{
		Ecosystem:      "go",
		ImportF1:       0.95,
		StyleF1:        0.90,
		C4F1:           0.85,
		OverallScore:   0.90,
		SampleCount:    10,
	}

	passed, reason := CheckThresholds(metrics)
	if !passed {
		t.Errorf("expected pass, got fail with reason: %s", reason)
	}
}

func TestCheckThresholds_Go_FailImport(t *testing.T) {
	metrics := EcosystemMetrics{
		Ecosystem:      "go",
		ImportF1:       0.85, // below 0.90
		StyleF1:        0.90,
		C4F1:           0.85,
		OverallScore:   0.87,
		SampleCount:    10,
	}

	passed, reason := CheckThresholds(metrics)
	if passed {
		t.Error("expected fail for low import F1")
	}
	if reason == "" {
		t.Error("expected non-empty reason")
	}
}

func TestCheckThresholds_Go_FailStyle(t *testing.T) {
	metrics := EcosystemMetrics{
		Ecosystem:      "go",
		ImportF1:       0.95,
		StyleF1:        0.80, // below 0.85
		C4F1:           0.85,
		OverallScore:   0.87,
		SampleCount:    10,
	}

	passed, reason := CheckThresholds(metrics)
	if passed {
		t.Error("expected fail for low style F1")
	}
	if reason == "" {
		t.Error("expected non-empty reason")
	}
}

func TestCheckThresholds_Python_Pass(t *testing.T) {
	metrics := EcosystemMetrics{
		Ecosystem:      "python",
		ImportF1:       0.70,
		StyleF1:        0.80,
		C4F1:           0.75,
		OverallScore:   0.75,
		SampleCount:    10,
	}

	passed, _ := CheckThresholds(metrics)
	if !passed {
		t.Error("expected pass for python metrics")
	}
}

func TestCheckThresholds_Python_Fail(t *testing.T) {
	metrics := EcosystemMetrics{
		Ecosystem:      "python",
		ImportF1:       0.60, // below 0.65
		StyleF1:        0.80,
		C4F1:           0.75,
		OverallScore:   0.72,
		SampleCount:    10,
	}

	passed, _ := CheckThresholds(metrics)
	if passed {
		t.Error("expected fail for low import F1")
	}
}

func TestCheckThresholds_SQL_Pass(t *testing.T) {
	metrics := EcosystemMetrics{
		Ecosystem:      "sql",
		SchemaF1:       0.85,
		OverallScore:   0.85,
		SampleCount:    5,
	}

	passed, _ := CheckThresholds(metrics)
	if !passed {
		t.Error("expected pass for SQL metrics")
	}
}

func TestCheckThresholds_SQL_Fail(t *testing.T) {
	metrics := EcosystemMetrics{
		Ecosystem:      "sql",
		SchemaF1:       0.75, // below 0.80
		OverallScore:   0.75,
		SampleCount:    5,
	}

	passed, _ := CheckThresholds(metrics)
	if passed {
		t.Error("expected fail for low schema F1")
	}
}

func TestCompareC4_PerfectMatch(t *testing.T) {
	expected := &architect.ProfileFragment{
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "web", Type: "service", Source: "Dockerfile"},
				{Name: "db", Type: "database", Source: "docker-compose.yml"},
			},
		},
	}
	actual := &architect.ProfileFragment{
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "web", Type: "service", Source: "Dockerfile"},
				{Name: "db", Type: "database", Source: "docker-compose.yml"},
			},
		},
	}

	fa := compareC4(expected, actual)
	if fa.TruePositives != 2 {
		t.Errorf("expected TP=2, got %d", fa.TruePositives)
	}
	if fa.FalsePositives != 0 {
		t.Errorf("expected FP=0, got %d", fa.FalsePositives)
	}
	if fa.FalseNegatives != 0 {
		t.Errorf("expected FN=0, got %d", fa.FalseNegatives)
	}
}

func TestCompareC4_MissingContainer(t *testing.T) {
	expected := &architect.ProfileFragment{
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "web", Type: "service", Source: "Dockerfile"},
				{Name: "db", Type: "database", Source: "docker-compose.yml"},
			},
		},
	}
	actual := &architect.ProfileFragment{
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "web", Type: "service", Source: "Dockerfile"},
			},
		},
	}

	fa := compareC4(expected, actual)
	if fa.TruePositives != 1 {
		t.Errorf("expected TP=1, got %d", fa.TruePositives)
	}
	if fa.FalseNegatives != 1 {
		t.Errorf("expected FN=1, got %d", fa.FalseNegatives)
	}
}

func TestCompareC4_ExtraContainer(t *testing.T) {
	expected := &architect.ProfileFragment{
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "web", Type: "service", Source: "Dockerfile"},
			},
		},
	}
	actual := &architect.ProfileFragment{
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "web", Type: "service", Source: "Dockerfile"},
				{Name: "cache", Type: "cache", Source: "docker-compose.yml"},
			},
		},
	}

	fa := compareC4(expected, actual)
	if fa.TruePositives != 1 {
		t.Errorf("expected TP=1, got %d", fa.TruePositives)
	}
	if fa.FalsePositives != 1 {
		t.Errorf("expected FP=1, got %d", fa.FalsePositives)
	}
}

func TestFormatMetricsReport(t *testing.T) {
	metrics := []EcosystemMetrics{
		{
			Ecosystem:      "go",
			ImportF1:       0.95,
			StyleF1:        0.90,
			C4F1:           0.85,
			SchemaF1:       0.0, // not applicable
			OverallScore:   0.90,
			SampleCount:    10,
		},
		{
			Ecosystem:      "python",
			ImportF1:       0.70,
			StyleF1:        0.80,
			C4F1:           0.75,
			SchemaF1:       0.0,
			OverallScore:   0.75,
			SampleCount:    5,
		},
	}

	report := FormatMetricsReport(metrics)
	if report == "" {
		t.Error("FormatMetricsReport returned empty string")
	}

	// Check for key content
	if !contains(report, "go") {
		t.Error("report missing 'go' ecosystem")
	}
	if !contains(report, "python") {
		t.Error("report missing 'python' ecosystem")
	}
	if !contains(report, "PASS") {
		t.Error("report missing 'PASS' status")
	}
}
