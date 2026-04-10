package architect_test

import (
	"encoding/json"
	"testing"
	"time"

	"sdp_dev/internal/architect"
)

func TestArchitectureReportJSON(t *testing.T) {
	report := architect.ArchitectureReport{
		Version:           "1.0.0",
		AnalyzedAt:        time.Date(2026, 4, 10, 14, 30, 0, 0, time.UTC),
		RepoRoot:          "/tmp/test-repo",
		AnalysisDurationS: 45.2,
		LLMCostUSD:        0.03,
		Languages: architect.LanguageInfo{
			Primary:      "go",
			All:          []string{"go", "python"},
			Distribution: map[string]float64{"go": 0.8, "python": 0.2},
		},
		StyleHypothesis: architect.StyleHypothesis{
			Styles: []architect.StyleScore{
				{Style: architect.StyleMicroservices, Confidence: 0.82, Evidence: []string{"5 Dockerfiles"}},
				{Style: architect.StyleEventDriven, Confidence: 0.45, Evidence: []string{"kafka in deps"}},
			},
			HumanInputNeeded: []string{"hexagonal vs layered unclear"},
		},
		PatternsDetected: []architect.DetectedPattern{
			{Category: "ddd", Name: "aggregate_root", Confidence: 0.7, Location: "domain/"},
		},
		SpecsDiscovered: []architect.SpecArtifact{
			{Kind: "openapi", Path: "api/openapi.yaml", Version: "3.1"},
		},
		Risks: []architect.ArchRisk{
			{Severity: architect.SeverityHigh, Category: "missing_contract", Description: "No API spec for orders service", Affected: []string{"services/orders"}},
		},
		Metrics: architect.CodeMetrics{
			TotalFiles: 1247, TotalLOC: 48520, TestRatio: 0.32,
			LanguagesCount: 2, ContainersDetected: 5, ComponentsDetected: 23,
		},
		ConfidenceSummary: architect.ConfidenceSummary{
			Overall: 0.72, StructuralAnalysis: 0.85, StyleHypothesis: 0.65, ContractCoverage: 0.4,
		},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded architect.ArchitectureReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Version != report.Version {
		t.Errorf("version: got %q, want %q", decoded.Version, report.Version)
	}
	if len(decoded.StyleHypothesis.Styles) != 2 {
		t.Errorf("styles: got %d, want 2", len(decoded.StyleHypothesis.Styles))
	}
	if decoded.StyleHypothesis.Styles[0].Style != architect.StyleMicroservices {
		t.Errorf("style[0]: got %q, want %q", decoded.StyleHypothesis.Styles[0].Style, architect.StyleMicroservices)
	}
	if decoded.Metrics.TotalFiles != 1247 {
		t.Errorf("total_files: got %d, want 1247", decoded.Metrics.TotalFiles)
	}
}

func TestCodebaseProfileJSON(t *testing.T) {
	profile := architect.CodebaseProfile{
		FileTree: architect.FileTreeInfo{
			TotalFiles: 500, TotalDirs: 40, MaxDepth: 5,
			TopLevel:       []string{"cmd/", "internal/", "docs/"},
			NamingPatterns: map[string]int{"controller": 5, "service": 12},
		},
		Dependencies: architect.DependencyInfo{
			Manifests: []architect.ManifestInfo{
				{Path: "go.mod", Language: "go", DepsCount: 20},
			},
			NotableDeps: []architect.NotableDep{
				{Name: "kafka-go", FoundIn: 2, Signal: "event_driven"},
			},
		},
		ImportGraph: architect.ImportGraph{
			ExtractionMethod: "go/packages", AccuracyEstimate: 0.95,
			Nodes: 15, Edges: 42,
			Clusters: []architect.ImportCluster{
				{ID: "core", Packages: []string{"internal/kernel"}, InternalEdges: 5, ExternalEdges: 10},
			},
		},
		Infra: architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Source: "Dockerfile", Type: "service"},
			},
			Deployment: architect.DeploymentInfo{Type: "kubernetes", Evidence: []string{"k8s/"}},
		},
		Metrics: architect.CodeMetrics{TotalFiles: 500, TotalLOC: 20000},
	}

	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded architect.CodebaseProfile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.FileTree.TotalFiles != 500 {
		t.Errorf("total_files: got %d, want 500", decoded.FileTree.TotalFiles)
	}
	if decoded.ImportGraph.ExtractionMethod != "go/packages" {
		t.Errorf("extraction_method: got %q, want %q", decoded.ImportGraph.ExtractionMethod, "go/packages")
	}
}

func TestContractCatalogJSON(t *testing.T) {
	catalog := architect.ContractCatalog{
		Contracts: []architect.Contract{
			{
				ID: "auth-api", Type: "http_api", Format: "openapi",
				SourcePath: "services/auth/openapi.yaml",
				State:      architect.ContractObserved,
				Provider:   architect.ContractEndpoint{Container: "auth", Component: "handler"},
				Consumers:  []architect.ContractEndpoint{{Container: "gateway"}},
				Confidence: 1.0, ValidationStatus: "pass",
			},
		},
		Gaps: []architect.ContractGap{
			{
				Type:     "http_api",
				Between:  architect.ContractEndpoint{Container: "orders"},
				And:      architect.ContractEndpoint{Container: "payments"},
				Severity: architect.SeverityHigh,
				Note:     "HTTP call detected but no OpenAPI spec",
			},
		},
	}

	data, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded architect.ContractCatalog
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Contracts) != 1 {
		t.Fatalf("contracts: got %d, want 1", len(decoded.Contracts))
	}
	if decoded.Contracts[0].State != architect.ContractObserved {
		t.Errorf("state: got %q, want %q", decoded.Contracts[0].State, architect.ContractObserved)
	}
	if len(decoded.Gaps) != 1 {
		t.Fatalf("gaps: got %d, want 1", len(decoded.Gaps))
	}
}

func TestReferenceModelJSON(t *testing.T) {
	model := architect.ReferenceModel{
		Version: "1.0.0",
		State:   architect.ModelProposed,
		System:  architect.SystemInfo{Name: "E-Commerce", Description: "Microservices platform"},
		Actors:  []architect.Actor{{ID: "end_user"}},
		ExternalSystems: []architect.ExternalSystem{
			{ID: "stripe", Description: "Payment provider", Technology: "REST API"},
		},
		Containers: []architect.C4Container{
			{
				ID: "orders", Name: "Orders Service", Technology: "Go",
				Source: "services/orders/", Deploy: "services/orders/Dockerfile",
				Components: []architect.C4Component{
					{ID: "domain", Path: "services/orders/domain/", Confidence: 0.8},
				},
			},
		},
		Relationships: []architect.C4Relationship{
			{From: "orders", To: "stripe", Description: "Process payments", Type: "sync"},
		},
	}

	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded architect.ReferenceModel
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.State != architect.ModelProposed {
		t.Errorf("state: got %q, want %q", decoded.State, architect.ModelProposed)
	}
	if len(decoded.Containers) != 1 {
		t.Fatalf("containers: got %d, want 1", len(decoded.Containers))
	}
	if decoded.Containers[0].ID != "orders" {
		t.Errorf("container id: got %q, want %q", decoded.Containers[0].ID, "orders")
	}
}

func TestSQLAnalysisJSON(t *testing.T) {
	sql := architect.SQLAnalysis{
		DatabasesDetected: 1,
		Tables: []architect.TableDef{
			{Name: "users", Columns: []architect.ColumnDef{{Name: "id", Type: "bigint", PrimaryKey: true}}},
		},
		ForeignKeys:     []architect.ForeignKey{{FromTable: "orders", FromColumn: "user_id", ToTable: "users", ToColumn: "id"}},
		MigrationsCount: 23,
		PIIColumns: []architect.PIIColumn{
			{Table: "users", Column: "email", PIIType: "email_address", Confidence: 0.95},
		},
	}

	data, err := json.Marshal(sql)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded architect.SQLAnalysis
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.DatabasesDetected != 1 {
		t.Errorf("databases: got %d, want 1", decoded.DatabasesDetected)
	}
	if len(decoded.PIIColumns) != 1 {
		t.Fatalf("pii_columns: got %d, want 1", len(decoded.PIIColumns))
	}
}
