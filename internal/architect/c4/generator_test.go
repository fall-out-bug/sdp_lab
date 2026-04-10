package c4

import (
	"strings"
	"testing"

	"sdp_dev/internal/architect"
)

func TestGenerate_NilProfile(t *testing.T) {
	_, err := Generate(nil, GenerateOptions{})
	if err == nil {
		t.Fatal("expected error for nil profile")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected nil-related error, got: %v", err)
	}
}

func TestGenerate_EmptyProfile(t *testing.T) {
	profile := &architect.CodebaseProfile{
		Name: "test-project",
	}
	model, err := Generate(profile, GenerateOptions{RepoName: "test-project"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if model.System.Name != "test-project" {
		t.Errorf("expected system name 'test-project', got %q", model.System.Name)
	}
	if model.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", model.Version)
	}
	if model.State != architect.ModelObserved {
		t.Errorf("expected state 'observed', got %q", model.State)
	}
	// Should have at least one fallback container.
	if len(model.Containers) == 0 {
		t.Error("expected at least one container")
	}
}

func TestGenerate_WithDockerContainers(t *testing.T) {
	profile := &architect.CodebaseProfile{
		Name: "microservices-demo",
		Infra: architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api-gateway", Image: "node:18", Type: "service", Source: "Dockerfile"},
				{Name: "orders", Image: "golang:1.21", Type: "service", Source: "services/orders/Dockerfile"},
				{Name: "postgres", Image: "postgres:15", Type: "database", Source: "docker-compose.yml"},
			},
			Services: []architect.ServiceDep{
				{From: "api-gateway", To: "orders"},
			},
			DeploymentType: "docker-compose",
		},
	}

	model, err := Generate(profile, GenerateOptions{RepoName: "microservices-demo"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should have containers from Dockerfile services + compose deps.
	if len(model.Containers) < 3 {
		t.Errorf("expected at least 3 containers, got %d", len(model.Containers))
	}

	// Check that the database container was detected.
	foundDB := false
	for _, c := range model.Containers {
		if strings.Contains(strings.ToLower(c.Description), "database") {
			foundDB = true
			break
		}
	}
	if !foundDB {
		t.Error("expected to find a database container")
	}

	// Check relationships include docker-compose depends_on.
	foundDep := false
	for _, r := range model.Relationships {
		if strings.Contains(r.From, "api") && strings.Contains(r.To, "orders") {
			foundDep = true
			break
		}
	}
	if !foundDep {
		t.Error("expected relationship from api-gateway to orders")
	}
}

func TestGenerate_WithK8sServices(t *testing.T) {
	profile := &architect.CodebaseProfile{
		Name: "k8s-app",
		Infra: architect.InfraInfo{
			K8sServices: []architect.K8sServiceInfo{
				{Name: "web-frontend", Source: "k8s/frontend.yaml"},
				{Name: "backend-api", Source: "k8s/backend.yaml"},
			},
		},
	}

	model, err := Generate(profile, GenerateOptions{RepoName: "k8s-app"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(model.Containers) < 2 {
		t.Errorf("expected at least 2 containers from K8s services, got %d", len(model.Containers))
	}
}

func TestGenerate_WithModuleBoundaries(t *testing.T) {
	profile := &architect.CodebaseProfile{
		Name: "monorepo",
		Infra: architect.InfraInfo{
			ModuleBoundaries: []architect.ModuleBoundaryInfo{
				{
					Name:        "go-cmd",
					BuildSystem: "go",
					Path:        "cmd/",
					Children:    []string{"cmd/api", "cmd/worker", "cmd/migrate"},
				},
			},
		},
	}

	model, err := Generate(profile, GenerateOptions{RepoName: "monorepo"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(model.Containers) < 3 {
		t.Errorf("expected at least 3 containers from cmd/ modules, got %d", len(model.Containers))
	}
}

func TestGenerate_WithImportClusters(t *testing.T) {
	profile := &architect.CodebaseProfile{
		Name: "clustered-app",
		ImportGraph: architect.ImportGraph{
			ExtractionMethod: "go/packages",
			AccuracyEstimate: 0.93,
			Nodes:            15,
			Edges:            20,
			Clusters: []architect.ImportCluster{
				{
					ID:            "internal/handlers",
					Packages:      []string{"internal/handlers/users", "internal/handlers/orders"},
					InternalEdges: 8,
					ExternalEdges: 3,
				},
				{
					ID:            "internal/repository",
					Packages:      []string{"internal/repository/postgres", "internal/repository/redis"},
					InternalEdges: 5,
					ExternalEdges: 2,
				},
			},
		},
	}

	model, err := Generate(profile, GenerateOptions{RepoName: "clustered-app"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// When no infra containers, clusters become fallback containers.
	if len(model.Containers) == 0 {
		t.Error("expected containers from import clusters")
	}
}

func TestGenerate_ActorsAndExternals(t *testing.T) {
	profile := &architect.CodebaseProfile{
		Name: "public-api",
		Infra: architect.InfraInfo{
			Ingresses: []architect.IngressInfo{
				{Name: "main-ingress", Hosts: []string{"api.example.com"}, Source: "k8s/ingress.yaml"},
			},
			ExposedPorts: []string{"8080", "443"},
		},
		Dependencies: architect.DependencyInfo{
			NotableDeps: []architect.NotableDep{
				{Name: "boto3", FoundIn: 1, Signal: "cloud_aws"},
			},
		},
	}

	model, err := Generate(profile, GenerateOptions{RepoName: "public-api"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should have at least an end-user and client actor.
	if len(model.Actors) < 1 {
		t.Error("expected at least one actor")
	}

	// Should have external system from dependency signals.
	if len(model.ExternalSystems) < 1 {
		t.Error("expected at least one external system")
	}
}

func TestGenerate_Relationships(t *testing.T) {
	profile := &architect.CodebaseProfile{
		Name: "rel-test",
		Infra: architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "frontend", Image: "react", Type: "service", Source: "Dockerfile"},
				{Name: "backend", Image: "golang", Type: "service", Source: "Dockerfile"},
				{Name: "db", Image: "postgres:15", Type: "database", Source: "docker-compose.yml"},
			},
			Services: []architect.ServiceDep{
				{From: "frontend", To: "backend"},
			},
		},
	}

	model, err := Generate(profile, GenerateOptions{RepoName: "rel-test"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(model.Relationships) == 0 {
		t.Error("expected at least one relationship")
	}

	// Should have frontend -> backend (from depends_on).
	foundServiceDep := false
	for _, r := range model.Relationships {
		if strings.Contains(r.From, "frontend") && strings.Contains(r.To, "backend") {
			foundServiceDep = true
		}
	}
	if !foundServiceDep {
		t.Error("expected frontend -> backend relationship from service deps")
	}

	// Should have service -> database persistence.
	foundPersistence := false
	for _, r := range model.Relationships {
		if r.Type == "data" {
			foundPersistence = true
		}
	}
	if !foundPersistence {
		t.Error("expected at least one persistence (data) relationship")
	}
}

func TestSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"My Service", "my-service"},
		{"api_gateway", "api-gateway"},
		{"API.Gateway", "apigateway"},
		{"service@prod", "serviceprod"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := slug(tt.input)
			if result != tt.expected {
				t.Errorf("slug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContainerConfidence(t *testing.T) {
	tests := []struct {
		deploy    string
		expected  float64
	}{
		{"docker", 0.95},
		{"docker-compose", 0.95},
		{"kubernetes", 0.90},
		{"npm", 0.85},
		{"maven", 0.80},
		{"gradle", 0.80},
		{"inferred", 0.50},
		{"other", 0.60},
	}

	for _, tt := range tests {
		t.Run(tt.deploy, func(t *testing.T) {
			c := &architect.C4Container{Deploy: tt.deploy}
			result := containerConfidence(c)
			if result != tt.expected {
				t.Errorf("containerConfidence(deploy=%q) = %f, want %f", tt.deploy, result, tt.expected)
			}
		})
	}
}

func TestConfidenceMarker(t *testing.T) {
	tests := []struct {
		confidence float64
		expected   string
	}{
		{0.95, ""},
		{0.80, ""},
		{0.75, "[AUTO?] "},
		{0.60, "[AUTO?] "},
		{0.50, "[AUTO] "},
		{0.30, "[AUTO] "},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := confidenceMarker(tt.confidence)
			if result != tt.expected {
				t.Errorf("confidenceMarker(%f) = %q, want %q", tt.confidence, result, tt.expected)
			}
		})
	}
}

func TestGenerateReviewReport(t *testing.T) {
	model := &architect.ReferenceModel{
		System: architect.SystemInfo{Name: "test"},
		Containers: []architect.C4Container{
			{ID: "svc", Name: "Service", Deploy: "docker"},        // high confidence
			{ID: "inferred", Name: "Inferred", Deploy: "inferred"}, // low confidence
		},
	}

	report := GenerateReviewReport(model)

	if len(report.ReviewRequired) == 0 {
		t.Error("expected at least one review item for inferred container")
	}

	if report.Stats.TotalNodes < 2 {
		t.Errorf("expected at least 2 total nodes, got %d", report.Stats.TotalNodes)
	}

	if report.Stats.OverallConfidence <= 0 {
		t.Error("expected positive overall confidence")
	}
}

func TestExportLevel1(t *testing.T) {
	model := &architect.ReferenceModel{
		System: architect.SystemInfo{Name: "TestSystem"},
		Actors: []architect.Actor{
			{ID: "user", Description: "End User"},
		},
		ExternalSystems: []architect.ExternalSystem{
			{ID: "stripe", Description: "Payment", Technology: "REST"},
		},
		Containers: []architect.C4Container{
			{ID: "api", Name: "API Service"},
		},
		Relationships: []architect.C4Relationship{
			{From: "user", To: "system", Description: "uses", Type: "sync"},
		},
	}

	data := ExportLevel1(model)

	if data.Level != "L1" {
		t.Errorf("expected level L1, got %q", data.Level)
	}
	if len(data.Nodes) == 0 {
		t.Error("expected at least one node")
	}
	if len(data.Edges) == 0 {
		t.Error("expected at least one edge")
	}
}

func TestExportLevel2(t *testing.T) {
	model := &architect.ReferenceModel{
		System: architect.SystemInfo{Name: "TestSystem"},
		Containers: []architect.C4Container{
			{ID: "api", Name: "API", Technology: "Go"},
			{ID: "db", Name: "DB", Description: "Database: postgres"},
		},
	}

	data := ExportLevel2(model)

	if data.Level != "L2" {
		t.Errorf("expected level L2, got %q", data.Level)
	}
	if len(data.Nodes) < 2 {
		t.Errorf("expected at least 2 nodes, got %d", len(data.Nodes))
	}
}

func TestExportLevel3(t *testing.T) {
	model := &architect.ReferenceModel{
		System: architect.SystemInfo{Name: "TestSystem"},
		Containers: []architect.C4Container{
			{
				ID:   "api",
				Name: "API",
				Components: []architect.C4Component{
					{ID: "api-handlers", Description: "Handlers"},
					{ID: "api-repo", Description: "Repository"},
				},
			},
		},
		Relationships: []architect.C4Relationship{
			{From: "api-handlers", To: "api-repo", Description: "calls", Type: "sync"},
		},
	}

	data, err := ExportLevel3(model, "api")
	if err != nil {
		t.Fatalf("ExportLevel3 failed: %v", err)
	}

	if data.Level != "L3" {
		t.Errorf("expected level L3, got %q", data.Level)
	}
	if len(data.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(data.Nodes))
	}
	if len(data.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(data.Edges))
	}
}

func TestExportLevel3_NotFound(t *testing.T) {
	model := &architect.ReferenceModel{
		System: architect.SystemInfo{Name: "Test"},
	}

	_, err := ExportLevel3(model, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent container")
	}
}

func TestShouldExport(t *testing.T) {
	if ShouldExport(10) {
		t.Error("ShouldExport(10) should be false")
	}
	if ShouldExport(15) {
		t.Error("ShouldExport(15) should be false (at threshold)")
	}
	if !ShouldExport(16) {
		t.Error("ShouldExport(16) should be true")
	}
}

func TestMarshalExportData(t *testing.T) {
	data := &ExportData{
		Level:  "L1",
		System: "test",
		Nodes: []ExportNode{
			{ID: "sys", Type: "System", Label: "Test"},
		},
	}

	jsonStr, err := MarshalExportData(data)
	if err != nil {
		t.Fatalf("MarshalExportData failed: %v", err)
	}
	if !strings.Contains(jsonStr, `"level": "L1"`) {
		t.Error("expected JSON to contain level field")
	}
}

func TestGenerate_RepoRootFallback(t *testing.T) {
	profile := &architect.CodebaseProfile{}

	model, err := Generate(profile, GenerateOptions{RepoRoot: "/path/to/my-project"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if model.System.Name != "my-project" {
		t.Errorf("expected system name from RepoRoot basename, got %q", model.System.Name)
	}
}

func TestGenerate_GeneratedAt(t *testing.T) {
	profile := &architect.CodebaseProfile{}
	model, err := Generate(profile, GenerateOptions{RepoName: "test"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if model.GeneratedAt == "" {
		t.Error("expected GeneratedAt to be set")
	}
}

func TestGenerate_AnalyzedCommit(t *testing.T) {
	profile := &architect.CodebaseProfile{}
	model, err := Generate(profile, GenerateOptions{
		RepoName:    "test",
		CommitHash:  "abc123",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if model.AnalyzedCommit != "abc123" {
		t.Errorf("expected AnalyzedCommit 'abc123', got %q", model.AnalyzedCommit)
	}
}

func TestImageToTech(t *testing.T) {
	tests := []struct {
		image    string
		expected string
	}{
		{"node:18", "node"},
		{"golang:1.21-alpine", "golang"},
		{"postgres:15", "postgres"},
		{"registry.example.com/my-service:v1", "my-service"},
		{"nginx@sha256:abc123", "nginx"},
	}

	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			result := imageToTech(tt.image)
			if result != tt.expected {
				t.Errorf("imageToTech(%q) = %q, want %q", tt.image, result, tt.expected)
			}
		})
	}
}

func TestMatchScore(t *testing.T) {
	tests := []struct {
		a, b     string
		minScore int
	}{
		{"api-gateway", "api-gateway", 100},
		{"api", "api-gateway", 50},
		{"internal/handlers", "handlers", 40},
		{"foo", "bar", 0},
	}

	for _, tt := range tests {
		t.Run(tt.a+"-"+tt.b, func(t *testing.T) {
			result := matchScore(tt.a, tt.b)
			if result < tt.minScore {
				t.Errorf("matchScore(%q, %q) = %d, want >= %d", tt.a, tt.b, result, tt.minScore)
			}
		})
	}
}

func TestRendererSecurityLevel(t *testing.T) {
	model := sampleModel()
	opts := RenderOptions{Theme: "default"}

	l1, err := RenderL1(model, opts)
	if err != nil {
		t.Fatalf("RenderL1 failed: %v", err)
	}
	if !strings.Contains(l1.MermaidCode, "securityLevel") {
		t.Error("expected securityLevel in L1 Mermaid output")
	}
	if !strings.Contains(l1.MermaidCode, "'strict'") {
		t.Error("expected 'strict' securityLevel in L1 Mermaid output")
	}

	l2, err := RenderL2(model, opts)
	if err != nil {
		t.Fatalf("RenderL2 failed: %v", err)
	}
	if !strings.Contains(l2.MermaidCode, "securityLevel") {
		t.Error("expected securityLevel in L2 Mermaid output")
	}

	l3, err := RenderL3(model, "orders", opts)
	if err != nil {
		t.Fatalf("RenderL3 failed: %v", err)
	}
	if !strings.Contains(l3.MermaidCode, "securityLevel") {
		t.Error("expected securityLevel in L3 Mermaid output")
	}
}

func TestModelConfidence(t *testing.T) {
	model := &architect.ReferenceModel{
		System: architect.SystemInfo{Name: "test"},
		Containers: []architect.C4Container{
			{ID: "svc", Name: "Service", Deploy: "docker"},
		},
		Relationships: []architect.C4Relationship{
			{From: "a", To: "b"},
		},
	}

	conf := modelConfidence(model)
	if conf <= 0 || conf > 1.0 {
		t.Errorf("modelConfidence = %f, expected (0, 1]", conf)
	}
}

func TestGenerate_CompleteFlow(t *testing.T) {
	// End-to-end test: Generate -> Render -> Export.
	profile := &architect.CodebaseProfile{
		Name: "e2e-test",
		Infra: architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "web", Image: "node:18", Type: "service", Source: "Dockerfile"},
				{Name: "api", Image: "golang:1.21", Type: "service", Source: "Dockerfile"},
				{Name: "db", Image: "postgres:15", Type: "database", Source: "docker-compose.yml"},
			},
			Services: []architect.ServiceDep{
				{From: "web", To: "api"},
			},
			Ingresses: []architect.IngressInfo{
				{Name: "main", Hosts: []string{"example.com"}, Source: "k8s/ingress.yaml"},
			},
		},
		ImportGraph: architect.ImportGraph{
			Clusters: []architect.ImportCluster{
				{ID: "handlers", Packages: []string{"api/handlers"}, InternalEdges: 5},
				{ID: "repository", Packages: []string{"api/repository"}, InternalEdges: 3},
			},
		},
	}

	model, err := Generate(profile, GenerateOptions{RepoName: "e2e-test"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify model structure.
	if len(model.Containers) < 3 {
		t.Errorf("expected at least 3 containers, got %d", len(model.Containers))
	}
	if len(model.Actors) == 0 {
		t.Error("expected actors")
	}
	if len(model.Relationships) == 0 {
		t.Error("expected relationships")
	}

	// Render L1.
	l1, err := RenderL1(model, RenderOptions{})
	if err != nil {
		t.Fatalf("RenderL1 failed: %v", err)
	}
	if l1.NodeCount == 0 {
		t.Error("L1: expected nodes")
	}

	// Render L2.
	l2, err := RenderL2(model, RenderOptions{})
	if err != nil {
		t.Fatalf("RenderL2 failed: %v", err)
	}
	if l2.NodeCount == 0 {
		t.Error("L2: expected nodes")
	}

	// Render L3 for first container with components.
	renderedL3 := false
	for _, c := range model.Containers {
		if len(c.Components) > 0 {
			l3, err := RenderL3(model, c.ID, RenderOptions{})
			if err != nil {
				t.Fatalf("RenderL3 for %s failed: %v", c.ID, err)
			}
			if l3.NodeCount == 0 {
				t.Errorf("L3 for %s: expected nodes", c.ID)
			}
			renderedL3 = true
		}
	}
	if !renderedL3 {
		t.Log("No containers had components for L3 rendering (this may be OK)")
	}

	// Generate review report.
	report := GenerateReviewReport(model)
	if report.Stats.OverallConfidence <= 0 {
		t.Error("expected positive overall confidence")
	}
}
