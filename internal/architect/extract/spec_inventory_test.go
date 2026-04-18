package extract

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSpecInventoryScanner_OpenAPI(t *testing.T) {
	tmpDir := t.TempDir()

	openAPIFiles := []string{
		"openapi.yaml",
		"openapi.yml",
		"openapi.json",
		"swagger.yaml",
		"swagger.yml",
		"swagger.json",
	}

	for _, filename := range openAPIFiles {
		path := filepath.Join(tmpDir, filename)
		content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	scanner := SpecInventoryScanner{}
	frag, err := scanner.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Specs) != len(openAPIFiles) {
		t.Errorf("Expected %d specs, got %d", len(openAPIFiles), len(frag.Specs))
	}

	for _, spec := range frag.Specs {
		if spec.Kind != "openapi" {
			t.Errorf("Expected kind 'openapi', got '%s'", spec.Kind)
		}
	}
}

func TestSpecInventoryScanner_AsyncAPI(t *testing.T) {
	tmpDir := t.TempDir()

	asyncAPIFiles := []string{
		"asyncapi.yaml",
		"asyncapi.yml",
		"asyncapi.json",
	}

	for _, filename := range asyncAPIFiles {
		path := filepath.Join(tmpDir, filename)
		content := `asyncapi: 2.6.0
info:
  title: Test AsyncAPI
  version: 1.0.0
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	scanner := SpecInventoryScanner{}
	frag, err := scanner.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Specs) != len(asyncAPIFiles) {
		t.Errorf("Expected %d specs, got %d", len(asyncAPIFiles), len(frag.Specs))
	}

	for _, spec := range frag.Specs {
		if spec.Kind != "asyncapi" {
			t.Errorf("Expected kind 'asyncapi', got '%s'", spec.Kind)
		}
	}
}

func TestSpecInventoryScanner_Protobuf(t *testing.T) {
	tmpDir := t.TempDir()

	protoFiles := []string{
		"service.proto",
		"models.proto",
		"api/proto/v1/service.proto",
	}

	for _, filename := range protoFiles {
		path := filepath.Join(tmpDir, filename)
		content := `syntax = "proto3";
package test;
`
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	scanner := SpecInventoryScanner{}
	frag, err := scanner.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Specs) != len(protoFiles) {
		t.Errorf("Expected %d specs, got %d", len(protoFiles), len(frag.Specs))
	}

	for _, spec := range frag.Specs {
		if spec.Kind != "proto" {
			t.Errorf("Expected kind 'proto', got '%s'", spec.Kind)
		}
	}
}

func TestSpecInventoryScanner_GraphQL(t *testing.T) {
	tmpDir := t.TempDir()

	graphqlFiles := []string{
		"schema.graphql",
		"api.gql",
	}

	for _, filename := range graphqlFiles {
		path := filepath.Join(tmpDir, filename)
		content := `type Query {
  hello: String
}
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	scanner := SpecInventoryScanner{}
	frag, err := scanner.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Specs) != len(graphqlFiles) {
		t.Errorf("Expected %d specs, got %d", len(graphqlFiles), len(frag.Specs))
	}

	for _, spec := range frag.Specs {
		if spec.Kind != "graphql" {
			t.Errorf("Expected kind 'graphql', got '%s'", spec.Kind)
		}
	}
}

func TestSpecInventoryScanner_ADR(t *testing.T) {
	tmpDir := t.TempDir()

	adrFiles := []string{
		"adr/001-architecture.md",
		"adr/002-technology.md",
		"docs/adr/003-decision.md",
		"doc/adr/004-choice.md",
		"ADR-005-record.md",
		"ADR-006-another.md",
	}

	for _, filename := range adrFiles {
		path := filepath.Join(tmpDir, filename)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		content := `# Architecture Decision Record

## Status
Accepted

## Context
We need to make a decision.
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	scanner := SpecInventoryScanner{}
	frag, err := scanner.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Specs) != len(adrFiles) {
		t.Errorf("Expected %d specs, got %d", len(adrFiles), len(frag.Specs))
	}

	for _, spec := range frag.Specs {
		if spec.Kind != "adr" {
			t.Errorf("Expected kind 'adr', got '%s'", spec.Kind)
		}
	}
}

func TestSpecInventoryScanner_Docker(t *testing.T) {
	tmpDir := t.TempDir()

	dockerFiles := []string{
		"Dockerfile",
		"Dockerfile.prod",
		"docker-compose.yml",
		"docker-compose.yaml",
		"docker-compose.dev.yml",
	}

	for _, filename := range dockerFiles {
		path := filepath.Join(tmpDir, filename)
		content := `FROM alpine:latest
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	scanner := SpecInventoryScanner{}
	frag, err := scanner.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Specs) != len(dockerFiles) {
		t.Errorf("Expected %d specs, got %d", len(dockerFiles), len(frag.Specs))
	}

	for _, spec := range frag.Specs {
		if spec.Kind != "docker" {
			t.Errorf("Expected kind 'docker', got '%s'", spec.Kind)
		}
	}
}

func TestSpecInventoryScanner_Terraform(t *testing.T) {
	tmpDir := t.TempDir()

	tfFiles := []string{
		"main.tf",
		"variables.tf",
		"outputs.tf",
		"modules/network/main.tf",
	}

	for _, filename := range tfFiles {
		path := filepath.Join(tmpDir, filename)
		content := `resource "aws_instance" "example" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t2.micro"
}
`
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	scanner := SpecInventoryScanner{}
	frag, err := scanner.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Specs) != len(tfFiles) {
		t.Errorf("Expected %d specs, got %d", len(tfFiles), len(frag.Specs))
	}

	for _, spec := range frag.Specs {
		if spec.Kind != "terraform" {
			t.Errorf("Expected kind 'terraform', got '%s'", spec.Kind)
		}
	}
}

func TestSpecInventoryScanner_CI(t *testing.T) {
	tmpDir := t.TempDir()

	ciFiles := []string{
		".github/workflows/ci.yml",
		".github/workflows/test.yaml",
		".gitlab-ci.yml",
		"Jenkinsfile",
		".circleci/config.yml",
	}

	for _, filename := range ciFiles {
		path := filepath.Join(tmpDir, filename)
		content := `name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
`
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	scanner := SpecInventoryScanner{}
	frag, err := scanner.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Specs) != len(ciFiles) {
		t.Errorf("Expected %d specs, got %d", len(ciFiles), len(frag.Specs))
	}

	for _, spec := range frag.Specs {
		if spec.Kind != "ci" {
			t.Errorf("Expected kind 'ci', got '%s'", spec.Kind)
		}
	}
}

func TestSpecInventoryScanner_Kubernetes(t *testing.T) {
	tmpDir := t.TempDir()

	k8sFiles := []string{
		"k8s/deployment.yaml",
		"k8s/service.yml",
		"kubernetes/configmap.yaml",
		"kubernetes/ingress.yml",
		"deploy/namespace.yaml",
		"deploy/statefulset.yml",
	}

	for _, filename := range k8sFiles {
		path := filepath.Join(tmpDir, filename)
		content := `apiVersion: v1
kind: Service
metadata:
  name: test
`
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	scanner := SpecInventoryScanner{}
	frag, err := scanner.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Specs) != len(k8sFiles) {
		t.Errorf("Expected %d specs, got %d", len(k8sFiles), len(frag.Specs))
	}

	for _, spec := range frag.Specs {
		if spec.Kind != "k8s" {
			t.Errorf("Expected kind 'k8s', got '%s'", spec.Kind)
		}
	}
}

func TestSpecInventoryScanner_Migrations(t *testing.T) {
	tmpDir := t.TempDir()

	migrationFiles := []string{
		"migrations/001_initial.up.sql",
		"migrations/001_initial.down.sql",
		"db/migrations/002_users.up.sql",
		"migrate/003_posts.sql",
	}

	for _, filename := range migrationFiles {
		path := filepath.Join(tmpDir, filename)
		content := `CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  name TEXT
);
`
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	scanner := SpecInventoryScanner{}
	frag, err := scanner.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(frag.Specs) != len(migrationFiles) {
		t.Errorf("Expected %d specs, got %d", len(migrationFiles), len(frag.Specs))
	}

	for _, spec := range frag.Specs {
		if spec.Kind != "migration" {
			t.Errorf("Expected kind 'migration', got '%s'", spec.Kind)
		}
	}
}

func TestSpecInventoryScanner_AllSpecs(t *testing.T) {
	// Test detection of all spec types in acceptance criteria
	tmpDir := t.TempDir()

	specTests := []struct {
		path    string
		kind    string
		content string
	}{
		{"openapi.yaml", "openapi", "openapi: 3.0.0"},
		{"asyncapi.yaml", "asyncapi", "asyncapi: 2.0.0"},
		{"service.proto", "proto", "syntax = \"proto3\""},
		{"schema.graphql", "graphql", "type Query { hello: String }"},
		{"adr/001-test.md", "adr", "# ADR"},
		{"docs/adr/002-test.md", "adr", "# ADR"},
		{"Dockerfile", "docker", "FROM alpine"},
		{"main.tf", "terraform", "resource \"test\""},
		{".github/workflows/ci.yml", "ci", "name: CI"},
		{"k8s/deployment.yaml", "k8s", "apiVersion: v1"},
		{"migrations/001.sql", "migration", "CREATE TABLE"},
	}

	for _, tt := range specTests {
		path := filepath.Join(tmpDir, tt.path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", tt.path, err)
		}
	}

	scanner := SpecInventoryScanner{}
	frag, err := scanner.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	expectedCount := len(specTests)
	if len(frag.Specs) != expectedCount {
		t.Errorf("Expected %d specs, got %d", expectedCount, len(frag.Specs))
	}

	// Verify each spec type was detected
	for _, tt := range specTests {
		found := false
		for _, spec := range frag.Specs {
			if spec.Path == tt.path {
				found = true
				if spec.Kind != tt.kind {
					t.Errorf("Spec %s: expected kind '%s', got '%s'", tt.path, tt.kind, spec.Kind)
				}
				break
			}
		}
		if !found {
			t.Errorf("Spec %s was not detected", tt.path)
		}
	}
}

func TestSpecInventoryScanner_SkipDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create specs in skipped directories
	skipDirs := []string{
		"node_modules/openapi.yaml",
		"vendor/service.proto",
		".git/main.tf",
	}

	for _, path := range skipDirs {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Create a spec in a normal directory
	normalSpec := filepath.Join(tmpDir, "openapi.yaml")
	if err := os.WriteFile(normalSpec, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	scanner := SpecInventoryScanner{}
	frag, err := scanner.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should only detect the file in the normal directory
	if len(frag.Specs) != 1 {
		t.Errorf("Expected 1 spec, got %d", len(frag.Specs))
	}

	if len(frag.Specs) > 0 && frag.Specs[0].Path != "openapi.yaml" {
		t.Errorf("Expected openapi.yaml, got %s", frag.Specs[0].Path)
	}
}

func TestSpecInventoryScanner_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create many spec files
	for i := 0; i < 100; i++ {
		path := filepath.Join(tmpDir, filepath.Join("dir", "file.proto"))
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("syntax = \"proto3\""), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	scanner := SpecInventoryScanner{}
	_, err := scanner.Extract(ctx, tmpDir)
	if err == nil {
		t.Error("Expected error for cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}
