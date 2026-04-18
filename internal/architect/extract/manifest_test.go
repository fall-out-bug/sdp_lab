package extract

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDependencyManifestParser_GoMod(t *testing.T) {
	tmpDir := t.TempDir()

	goModContent := `module example.com/myproject

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/grpc-ecosystem/grpc-gateway v2.15.0+incompatible
	google.golang.org/grpc v1.58.0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/stretchr/testify v1.8.4 // indirect
)
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	parser := DependencyManifestParser{}
	frag, err := parser.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Dependencies) == 0 {
		t.Fatal("No dependencies found")
	}

	dep := frag.Dependencies[0]
	if dep.File != "go.mod" {
		t.Errorf("Expected file 'go.mod', got '%s'", dep.File)
	}
	if dep.Language != "go" {
		t.Errorf("Expected language 'go', got '%s'", dep.Language)
	}
	if dep.DepCount != 4 {
		t.Errorf("Expected 4 direct dependencies, got %d", dep.DepCount)
	}

	// Check for grpc signal
	foundGRPC := false
	for _, signal := range dep.Signals {
		if signal == "grpc" {
			foundGRPC = true
			break
		}
	}
	if !foundGRPC {
		t.Error("Expected grpc signal to be detected")
	}
}

func TestDependencyManifestParser_PackageJSON(t *testing.T) {
	tmpDir := t.TempDir()

	packageJSONContent := `{
  "name": "my-project",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.18.0",
    "graphql": "^16.8.0",
    "@prisma/client": "^5.7.0",
    "redis": "^4.6.0"
  },
  "devDependencies": {
    "typescript": "^5.3.0",
    "@types/node": "^20.10.0"
  }
}
`
	packageJSONPath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(packageJSONPath, []byte(packageJSONContent), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	parser := DependencyManifestParser{}
	frag, err := parser.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Dependencies) == 0 {
		t.Fatal("No dependencies found")
	}

	dep := frag.Dependencies[0]
	if dep.File != "package.json" {
		t.Errorf("Expected file 'package.json', got '%s'", dep.File)
	}
	if dep.Language != "javascript" {
		t.Errorf("Expected language 'javascript', got '%s'", dep.Language)
	}
	// Should count both dependencies and devDependencies
	if dep.DepCount < 4 {
		t.Errorf("Expected at least 4 dependencies, got %d", dep.DepCount)
	}

	// Check for graphql and prisma signals
	signalsMap := make(map[string]bool)
	for _, signal := range dep.Signals {
		signalsMap[signal] = true
	}
	if !signalsMap["graphql"] {
		t.Error("Expected graphql signal to be detected")
	}
	if !signalsMap["orm"] {
		t.Error("Expected orm signal (prisma) to be detected")
	}
}

func TestDependencyManifestParser_RequirementsTxt(t *testing.T) {
	tmpDir := t.TempDir()

	requirementsContent := `fastapi==0.104.0
uvicorn[standard]==0.24.0
sqlalchemy==2.0.23
grpcio==1.60.0
pytest==7.4.3
# Comment line
redis==5.0.1
`
	requirementsPath := filepath.Join(tmpDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte(requirementsContent), 0644); err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}

	parser := DependencyManifestParser{}
	frag, err := parser.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Dependencies) == 0 {
		t.Fatal("No dependencies found")
	}

	dep := frag.Dependencies[0]
	if dep.File != "requirements.txt" {
		t.Errorf("Expected file 'requirements.txt', got '%s'", dep.File)
	}
	if dep.Language != "python" {
		t.Errorf("Expected language 'python', got '%s'", dep.Language)
	}
	// Should count 6 dependencies (excluding comments)
	if dep.DepCount != 6 {
		t.Errorf("Expected 6 dependencies, got %d", dep.DepCount)
	}

	// Check for signals
	signalsMap := make(map[string]bool)
	for _, signal := range dep.Signals {
		signalsMap[signal] = true
	}
	if !signalsMap["grpc"] {
		t.Error("Expected grpc signal to be detected")
	}
	if !signalsMap["cache"] {
		t.Error("Expected cache signal (redis) to be detected")
	}
}

func TestDependencyManifestParser_CargoToml(t *testing.T) {
	tmpDir := t.TempDir()

	cargoContent := `[package]
name = "my-project"
version = "0.1.0"

[dependencies]
tokio = { version = "1.35", features = ["full"] }
serde = "1.0"
prometheus = "0.13"
`
	cargoPath := filepath.Join(tmpDir, "Cargo.toml")
	if err := os.WriteFile(cargoPath, []byte(cargoContent), 0644); err != nil {
		t.Fatalf("Failed to create Cargo.toml: %v", err)
	}

	parser := DependencyManifestParser{}
	frag, err := parser.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Dependencies) == 0 {
		t.Fatal("No dependencies found")
	}

	dep := frag.Dependencies[0]
	if dep.File != "Cargo.toml" {
		t.Errorf("Expected file 'Cargo.toml', got '%s'", dep.File)
	}
	if dep.Language != "rust" {
		t.Errorf("Expected language 'rust', got '%s'", dep.Language)
	}
	if dep.DepCount < 1 {
		t.Errorf("Expected at least 1 dependency, got %d", dep.DepCount)
	}
}

func TestDependencyManifestParser_PomXML(t *testing.T) {
	tmpDir := t.TempDir()

	pomContent := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.example</groupId>
  <artifactId>my-project</artifactId>
  <version>1.0.0</version>

  <dependencies>
    <dependency>
      <groupId>org.apache.kafka</groupId>
      <artifactId>kafka-clients</artifactId>
      <version>3.6.0</version>
    </dependency>
    <dependency>
      <groupId>org.hibernate</groupId>
      <artifactId>hibernate-core</artifactId>
      <version>6.4.0</version>
    </dependency>
  </dependencies>
</project>
`
	pomPath := filepath.Join(tmpDir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte(pomContent), 0644); err != nil {
		t.Fatalf("Failed to create pom.xml: %v", err)
	}

	parser := DependencyManifestParser{}
	frag, err := parser.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Dependencies) == 0 {
		t.Fatal("No dependencies found")
	}

	dep := frag.Dependencies[0]
	if dep.File != "pom.xml" {
		t.Errorf("Expected file 'pom.xml', got '%s'", dep.File)
	}
	if dep.Language != "java" {
		t.Errorf("Expected language 'java', got '%s'", dep.Language)
	}
	if dep.DepCount != 2 {
		t.Errorf("Expected 2 dependencies, got %d", dep.DepCount)
	}

	// Check for event_driven and orm signals
	signalsMap := make(map[string]bool)
	for _, signal := range dep.Signals {
		signalsMap[signal] = true
	}
	if !signalsMap["event_driven"] {
		t.Error("Expected event_driven signal (kafka) to be detected")
	}
	if !signalsMap["orm"] {
		t.Error("Expected orm signal (hibernate) to be detected")
	}
}

func TestDependencyManifestParser_Gemfile(t *testing.T) {
	tmpDir := t.TempDir()

	gemfileContent := `source "https://rubygems.org"

ruby "3.2.0"

gem "rails", "~> 7.1"
gem "pg", "~> 1.5"
gem "redis", "~> 5.0"
gem "sidekiq", "~> 7.0"

group :development, :test do
  gem "rspec-rails"
end
`
	gemfilePath := filepath.Join(tmpDir, "Gemfile")
	if err := os.WriteFile(gemfilePath, []byte(gemfileContent), 0644); err != nil {
		t.Fatalf("Failed to create Gemfile: %v", err)
	}

	parser := DependencyManifestParser{}
	frag, err := parser.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Dependencies) == 0 {
		t.Fatal("No dependencies found")
	}

	dep := frag.Dependencies[0]
	if dep.File != "Gemfile" {
		t.Errorf("Expected file 'Gemfile', got '%s'", dep.File)
	}
	if dep.Language != "ruby" {
		t.Errorf("Expected language 'ruby', got '%s'", dep.Language)
	}
	if dep.DepCount != 4 {
		t.Errorf("Expected 4 dependencies (excluding groups), got %d", dep.DepCount)
	}
}

func TestDependencyManifestParser_Csproj(t *testing.T) {
	tmpDir := t.TempDir()

	csprojContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>

  <ItemGroup>
    <PackageReference Include="Grpc.Core" Version="2.0.0" />
    <PackageReference Include="StackExchange.Redis" Version="2.7.0" />
    <PackageReference Include="Serilog" Version="3.1.0" />
  </ItemGroup>
</Project>
`
	csprojPath := filepath.Join(tmpDir, "MyProject.csproj")
	if err := os.WriteFile(csprojPath, []byte(csprojContent), 0644); err != nil {
		t.Fatalf("Failed to create .csproj file: %v", err)
	}

	parser := DependencyManifestParser{}
	frag, err := parser.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Dependencies) == 0 {
		t.Fatal("No dependencies found")
	}

	dep := frag.Dependencies[0]
	if dep.Language != "csharp" {
		t.Errorf("Expected language 'csharp', got '%s'", dep.Language)
	}
	if dep.DepCount != 3 {
		t.Errorf("Expected 3 dependencies, got %d", dep.DepCount)
	}

	// Check for grpc and cache signals
	signalsMap := make(map[string]bool)
	for _, signal := range dep.Signals {
		signalsMap[signal] = true
	}
	if !signalsMap["grpc"] {
		t.Error("Expected grpc signal to be detected")
	}
	if !signalsMap["cache"] {
		t.Error("Expected cache signal (redis) to be detected")
	}
}

func TestDependencyManifestParser_MultipleManifests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple manifest files
	files := map[string]string{
		"go.mod":         "module test\n\nrequire github.com/test v1.0.0",
		"package.json":   `{"dependencies": {"express": "^4.0.0"}}`,
		"requirements.txt": "fastapi==0.100.0",
		"Cargo.toml":      `[dependencies]\ntokio = "1.0"`,
		"pom.xml":         `<project><dependencies><dependency><groupId>test</groupId><artifactId>test</artifactId></dependency></dependencies></project>`,
		"Gemfile":         "gem 'rails'",
	}

	for filename, content := range files {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	parser := DependencyManifestParser{}
	frag, err := parser.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	expectedCount := 6
	if len(frag.Dependencies) != expectedCount {
		t.Errorf("Expected %d dependency entries, got %d", expectedCount, len(frag.Dependencies))
	}

	// Check that each manifest was detected
	foundFiles := make(map[string]bool)
	for _, dep := range frag.Dependencies {
		foundFiles[dep.File] = true
	}

	for filename := range files {
		if !foundFiles[filename] {
			t.Errorf("Manifest file %s was not detected", filename)
		}
	}
}

func TestDependencyManifestParser_SignalsDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// Test all notable signals
	tests := []struct {
		depName   string
		signal    string
		manifest  string
		content   string
	}{
		{"confluent-kafka", "event_driven", "requirements.txt", "confluent-kafka==2.3.0"},
		{"nats", "event_driven", "go.mod", "require github.com/nats-io/nats.go v1.0.0"},
		{"prisma", "orm", "package.json", `{"prisma": "^5.0.0"}`},
		{"sequelize", "orm", "package.json", `{"sequelize": "^6.0.0"}`},
		{"grpcio", "grpc", "requirements.txt", "grpcio==1.60.0"},
		{"graphql", "graphql", "package.json", `{"graphql": "^16.0.0"}`},
		{"redis", "cache", "requirements.txt", "redis==5.0.0"},
		{"memcached", "cache", "requirements.txt", "python-memcached==1.0.0"},
		{"prometheus", "observability", "go.mod", "require github.com/prometheus/client_golang v1.0.0"},
		{"opentelemetry", "observability", "package.json", `{"opentelemetry": "^1.0.0"}`},
		{"jaeger", "observability", "requirements.txt", "jaeger-client==4.0.0"},
		{"docker", "container", "go.mod", "require github.com/docker/docker v1.0.0"},
		{"kubernetes", "container", "requirements.txt", "kubernetes==28.0.0"},
		{"terraform", "iac", "go.mod", "require github.com/hashicorp/terraform v1.0.0"},
		{"pulumi", "iac", "package.json", `{"pulumi": "^3.0.0"}`},
	}

	for i, tt := range tests {
		t.Run(tt.depName, func(t *testing.T) {
			// Create a subdirectory for each test to avoid conflicts
			subDir := filepath.Join(tmpDir, "test", filepath.Join("dir", string(rune('a'+i))))
			if err := os.MkdirAll(subDir, 0755); err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}

			manifestPath := filepath.Join(subDir, tt.manifest)
			if err := os.WriteFile(manifestPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create manifest: %v", err)
			}

			parser := DependencyManifestParser{}
			frag, err := parser.Extract(context.Background(), subDir)
			if err != nil {
				t.Fatalf("Extract failed: %v", err)
			}

			if len(frag.Dependencies) == 0 {
				t.Fatal("No dependencies found")
			}

			dep := frag.Dependencies[0]
			found := false
			for _, signal := range dep.Signals {
				if signal == tt.signal {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected signal '%s' for dependency '%s', got %v", tt.signal, tt.depName, dep.Signals)
			}
		})
	}
}

func TestDependencyManifestParser_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create many manifest files
	for i := 0; i < 100; i++ {
		path := filepath.Join(tmpDir, filepath.Join("dir", "go.mod"))
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("module test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	parser := DependencyManifestParser{}
	_, err := parser.Extract(ctx, tmpDir)
	if err == nil {
		t.Error("Expected error for cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}
