package extract

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestDockerfileParsing tests Dockerfile extraction.
func TestDockerfileParsing(t *testing.T) {
	dockerfile := `FROM alpine:3.18
EXPOSE 8080
ENV PORT=8080
ENTRYPOINT ["/app/server"]
CMD ["--port=8080"]
`

	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		t.Fatalf("failed to write Dockerfile: %v", err)
	}

	extractor := &InfraExtractor{}
	frag, err := extractor.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	info := frag.Infra
	if info == nil {
		t.Fatal("InfraInfo should not be nil")
	}

	// Check base image
	found := false
	for _, img := range info.BaseImages {
		if img == "alpine:3.18" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("BaseImages should contain alpine:3.18, got %v", info.BaseImages)
	}

	// Check exposed ports
	found = false
	for _, port := range info.ExposedPorts {
		if port == "8080" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ExposedPorts should contain 8080, got %v", info.ExposedPorts)
	}

	// Check containers
	if len(info.Containers) == 0 {
		t.Error("Containers should not be empty")
	} else {
		container := info.Containers[0]
		if container.Entrypoint != "/app/server" {
			t.Errorf("Entrypoint = %q, want %q", container.Entrypoint, "/app/server")
		}
		if container.Cmd != "--port=8080" {
			t.Errorf("Cmd = %q, want %q", container.Cmd, "--port=8080")
		}
	}
}

// TestDockerComposeParsing tests docker-compose.yml extraction.
func TestDockerComposeParsing(t *testing.T) {
	composeContent := `version: '3.8'
services:
  web:
    image: nginx:alpine
    ports:
      - "80:80"
    depends_on:
      - api
    networks:
      - frontend
  api:
    build: ./api
    ports:
      - "8080:8080"
    depends_on:
      - db
    networks:
      - frontend
      - backend
  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: secret
    volumes:
      - db_data:/var/lib/postgresql/data
    networks:
      - backend
networks:
  frontend:
  backend:
volumes:
  db_data:
`

	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write docker-compose.yml: %v", err)
	}

	extractor := &InfraExtractor{}
	frag, err := extractor.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	info := frag.Infra
	if info == nil {
		t.Fatal("InfraInfo should not be nil")
	}

	// Check services (containers)
	if len(info.Containers) < 3 {
		t.Errorf("Expected at least 3 containers, got %d", len(info.Containers))
	}

	// Check service dependencies
	if len(info.Services) == 0 {
		t.Error("Expected service dependencies")
	}

	// Check networks
	if len(info.Networks) < 2 {
		t.Errorf("Expected at least 2 networks, got %d", len(info.Networks))
	}

	// Check volumes
	if len(info.Volumes) < 1 {
		t.Errorf("Expected at least 1 volume, got %d", len(info.Volumes))
	}

	// Check deployment type
	if info.DeploymentType != "docker-compose" {
		t.Errorf("DeploymentType = %q, want %q", info.DeploymentType, "docker-compose")
	}
}

// TestKubernetesYAMLParsing tests Kubernetes manifest extraction.
func TestKubernetesYAMLParsing(t *testing.T) {
	k8sContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-deployment
  namespace: production
spec:
  template:
    spec:
      containers:
      - name: web
        image: nginx:1.21
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: web-service
  namespace: production
spec:
  type: LoadBalancer
  selector:
    app: web
  ports:
  - name: http
    port: 80
    targetPort: 80
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: web-ingress
  namespace: production
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: web-config
  namespace: production
data:
  config.yaml: |
    key: value
`

	tmpDir := t.TempDir()
	k8sDir := filepath.Join(tmpDir, "k8s")
	if err := os.MkdirAll(k8sDir, 0755); err != nil {
		t.Fatalf("failed to create k8s dir: %v", err)
	}
	k8sPath := filepath.Join(k8sDir, "deployment.yaml")
	if err := os.WriteFile(k8sPath, []byte(k8sContent), 0644); err != nil {
		t.Fatalf("failed to write k8s yaml: %v", err)
	}

	extractor := &InfraExtractor{}
	frag, err := extractor.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	info := frag.Infra
	if info == nil {
		t.Fatal("InfraInfo should not be nil")
	}

	// Check containers from deployment
	if len(info.Containers) == 0 {
		t.Error("Expected containers from deployment")
	}

	// Check services
	if len(info.K8sServices) == 0 {
		t.Error("Expected Kubernetes services")
	}

	// Check ingresses
	if len(info.Ingresses) == 0 {
		t.Error("Expected ingresses")
	}

	// Check configmaps
	if len(info.ConfigMaps) == 0 {
		t.Error("Expected configmaps")
	}

	// Check deployment type
	if info.DeploymentType != "kubernetes" {
		t.Errorf("DeploymentType = %q, want %q", info.DeploymentType, "kubernetes")
	}
}

// TestTerraformParsing tests Terraform .tf file extraction.
func TestTerraformParsing(t *testing.T) {
	terraformContent := `# Provider configuration
provider "aws" {
  region = "us-east-1"
}

# S3 bucket
resource "aws_s3_bucket" "example" {
  bucket = "my-tf-test-bucket"
}

# EC2 instance
resource "aws_instance" "web" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"
}

# Data source
data "aws_ami" "ubuntu" {
  most_recent = true
}

# Module
module "vpc" {
  source = "./modules/vpc"
  cidr   = "10.0.0.0/16"
}

# Variables
variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
}
`

	tmpDir := t.TempDir()
	terraformPath := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(terraformPath, []byte(terraformContent), 0644); err != nil {
		t.Fatalf("failed to write terraform file: %v", err)
	}

	extractor := &InfraExtractor{}
	frag, err := extractor.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	info := frag.Infra
	if info == nil {
		t.Fatal("InfraInfo should not be nil")
	}

	// Check resources (including data sources and modules)
	if len(info.Resources) < 4 {
		t.Errorf("Expected at least 4 resources, got %d", len(info.Resources))
	}

	// Check Terraform variables
	if len(info.TerraformVars) < 2 {
		t.Errorf("Expected at least 2 variables, got %d", len(info.TerraformVars))
	}
}

// TestGitHubActionsParsing tests GitHub Actions workflow extraction.
func TestGitHubActionsParsing(t *testing.T) {
	workflowContent := `name: CI/CD Pipeline

on:
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        ports:
          - 5432:5432
    steps:
      - uses: actions/checkout@v3

  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to production
        uses: aws-actions/aws-cloudformation-github-deploy@v1
`

	tmpDir := t.TempDir()
	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("failed to create workflows dir: %v", err)
	}
	workflowPath := filepath.Join(workflowDir, "ci.yml")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	extractor := &InfraExtractor{}
	frag, err := extractor.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	info := frag.Infra
	if info == nil {
		t.Fatal("InfraInfo should not be nil")
	}

	// Check CI jobs
	if len(info.CIJobs) < 2 {
		t.Errorf("Expected at least 2 CI jobs, got %d", len(info.CIJobs))
	}

	// Check service containers
	if len(info.Containers) < 1 {
		t.Errorf("Expected at least 1 service container, got %d", len(info.Containers))
	}
}

// TestGitLabCIParsing tests GitLab CI extraction.
func TestGitLabCIParsing(t *testing.T) {
	gitlabCIContent := `stages:
  - build
  - test
  - deploy

build:
  stage: build
  image: docker:24
  services:
    - docker:dind
  script:
    - docker build -t myapp .

test:
  stage: test
  image: golang:1.21
  services:
    - postgres:15
  script:
    - go test ./...

deploy:
  stage: deploy
  image: alpine:3.18
  only:
    - main
  script:
    - kubectl apply -f k8s/
`

	tmpDir := t.TempDir()
	gitlabCIPath := filepath.Join(tmpDir, ".gitlab-ci.yml")
	if err := os.WriteFile(gitlabCIPath, []byte(gitlabCIContent), 0644); err != nil {
		t.Fatalf("failed to write .gitlab-ci.yml: %v", err)
	}

	extractor := &InfraExtractor{}
	frag, err := extractor.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	info := frag.Infra
	if info == nil {
		t.Fatal("InfraInfo should not be nil")
	}

	// Check CI jobs
	if len(info.CIJobs) < 3 {
		t.Errorf("Expected at least 3 CI jobs, got %d", len(info.CIJobs))
	}

	// Check service containers
	if len(info.Containers) < 2 {
		t.Errorf("Expected at least 2 service containers, got %d", len(info.Containers))
	}
}

// TestJenkinsfileParsing tests Jenkinsfile extraction.
func TestJenkinsfileParsing(t *testing.T) {
	jenkinsfileContent := `pipeline {
    agent any

    stages {
        stage('Build') {
            agent {
                docker {
                    image 'golang:1.21'
                }
            }
            steps {
                sh 'go build'
            }
        }

        stage('Test') {
            agent {
                docker {
                    image 'golang:1.21'
                }
            }
            steps {
                sh 'go test'
            }
        }

        stage('Deploy to Production') {
            when {
                branch 'main'
            }
            steps {
                sh 'kubectl apply -f k8s/'
            }
        }
    }
}
`

	tmpDir := t.TempDir()
	jenkinsfilePath := filepath.Join(tmpDir, "Jenkinsfile")
	if err := os.WriteFile(jenkinsfilePath, []byte(jenkinsfileContent), 0644); err != nil {
		t.Fatalf("failed to write Jenkinsfile: %v", err)
	}

	extractor := &InfraExtractor{}
	frag, err := extractor.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	info := frag.Infra
	if info == nil {
		t.Fatal("InfraInfo should not be nil")
	}

	// Check CI jobs (stages)
	if len(info.CIJobs) < 3 {
		t.Errorf("Expected at least 3 CI jobs (stages), got %d", len(info.CIJobs))
	}

	// Check base images from agent directives
	if len(info.BaseImages) == 0 {
		t.Error("Expected to find base images from agent directives")
	}
}

// TestModuleBoundaryDetection tests module boundary detection.
func TestModuleBoundaryDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Maven pom.xml with modules
	pomXML := `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <modules>
        <module>core</module>
        <module>api</module>
        <module>web</module>
    </modules>
</project>`
	if err := os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte(pomXML), 0644); err != nil {
		t.Fatalf("failed to write pom.xml: %v", err)
	}

	// Create Gradle settings.gradle
	gradleSettings := `rootProject.name = 'myapp'
include 'core'
include 'api'
include 'web'`
	if err := os.WriteFile(filepath.Join(tmpDir, "settings.gradle"), []byte(gradleSettings), 0644); err != nil {
		t.Fatalf("failed to write settings.gradle: %v", err)
	}

	// Create npm package.json with workspaces
	packageJSON := `{
  "name": "myapp",
  "workspaces": [
    "packages/*",
    "services/*"
  ]
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Create Go cmd directories
	cmdDir := filepath.Join(tmpDir, "cmd")
	if err := os.MkdirAll(filepath.Join(cmdDir, "app1"), 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cmdDir, "app2"), 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "app1", "main.go"), []byte("package main\nfunc main() {}"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "app2", "main.go"), []byte("package main\nfunc main() {}"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	extractor := &InfraExtractor{}
	frag, err := extractor.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	info := frag.Infra
	if info == nil {
		t.Fatal("InfraInfo should not be nil")
	}

	// Check that we found module boundaries for all build systems
	if len(info.ModuleBoundaries) < 4 {
		t.Errorf("Expected at least 4 module boundaries (maven, gradle, npm, go), got %d", len(info.ModuleBoundaries))
	}

	// Verify each build system
	foundMaven := false
	foundGradle := false
	foundNpm := false
	foundGo := false

	for _, boundary := range info.ModuleBoundaries {
		switch boundary.BuildSystem {
		case "maven":
			foundMaven = true
			if len(boundary.Children) < 3 {
				t.Errorf("Maven: expected at least 3 children, got %d", len(boundary.Children))
			}
		case "gradle":
			foundGradle = true
			if len(boundary.Children) < 3 {
				t.Errorf("Gradle: expected at least 3 children, got %d", len(boundary.Children))
			}
		case "npm":
			foundNpm = true
			if len(boundary.Children) < 2 {
				t.Errorf("npm: expected at least 2 children, got %d", len(boundary.Children))
			}
		case "go":
			foundGo = true
			if len(boundary.Children) < 2 {
				t.Errorf("Go: expected at least 2 children, got %d", len(boundary.Children))
			}
		}
	}

	if !foundMaven {
		t.Error("Expected to find Maven module boundary")
	}
	if !foundGradle {
		t.Error("Expected to find Gradle module boundary")
	}
	if !foundNpm {
		t.Error("Expected to find npm workspace boundary")
	}
	if !foundGo {
		t.Error("Expected to find Go cmd module boundary")
	}
}

// TestExtractIntegration tests the full extraction pipeline.
func TestExtractIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Dockerfile
	dockerfile := `FROM golang:1.21
WORKDIR /app
COPY . .
EXPOSE 8080
ENV PORT=8080
CMD ["go", "run", "main.go"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte(dockerfile), 0644); err != nil {
		t.Fatalf("failed to write Dockerfile: %v", err)
	}

	// Create docker-compose.yml
	compose := `version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      - db
  db:
    image: postgres:15
`
	if err := os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(compose), 0644); err != nil {
		t.Fatalf("failed to write docker-compose.yml: %v", err)
	}

	// Create Kubernetes manifests
	k8sDir := filepath.Join(tmpDir, "k8s")
	if err := os.MkdirAll(k8sDir, 0755); err != nil {
		t.Fatalf("failed to create k8s dir: %v", err)
	}
	k8sYAML := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
      - name: app
        image: myapp:latest
`
	if err := os.WriteFile(filepath.Join(k8sDir, "deployment.yaml"), []byte(k8sYAML), 0644); err != nil {
		t.Fatalf("failed to write k8s yaml: %v", err)
	}

	// Create Terraform file
	tfFile := `resource "aws_instance" "app" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(tfFile), 0644); err != nil {
		t.Fatalf("failed to write terraform file: %v", err)
	}

	// Run extraction
	extractor := &InfraExtractor{}
	frag, err := extractor.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	info := frag.Infra
	if info == nil {
		t.Fatal("InfraInfo should not be nil")
	}

	// Verify comprehensive extraction
	if len(info.Containers) == 0 {
		t.Error("Expected containers")
	}
	if len(info.BaseImages) == 0 {
		t.Error("Expected base images")
	}
	if len(info.ExposedPorts) == 0 {
		t.Error("Expected exposed ports")
	}
	if len(frag.InfraArtifacts) == 0 {
		t.Error("Expected infra artifacts")
	}

	// Verify deployment type detection (kubernetes should win over docker-compose)
	if info.DeploymentType != "kubernetes" {
		t.Errorf("DeploymentType = %q, want %q (kubernetes should take priority)", info.DeploymentType, "kubernetes")
	}
}

// TestUniqueHelper tests the appendUnique helper.
func TestUniqueHelper(t *testing.T) {
	slice := []string{"a", "b", "c"}

	// Add existing element
	result := appendUnique(slice, "b")
	if len(result) != 3 {
		t.Errorf("appendUnique with existing element should not change length")
	}

	// Add new element
	result = appendUnique(slice, "d")
	if len(result) != 4 {
		t.Errorf("appendUnique with new element should increase length")
	}
	if result[3] != "d" {
		t.Errorf("appendUnique should add new element at end")
	}
}

// TestFileMatching tests file pattern matching.
func TestFileMatching(t *testing.T) {
	// Test Dockerfile matching
	if !isDockerfile("Dockerfile") {
		t.Error("isDockerfile(Dockerfile) should return true")
	}
	if !isDockerfile("Dockerfile.prod") {
		t.Error("isDockerfile(Dockerfile.prod) should return true")
	}
	// Note: "dockerfile.txt" matches because HasPrefix("dockerfile.txt", "dockerfile") is true
	// This is acceptable behavior for the heuristic
	if !isDockerfile("dockerfile.txt") {
		t.Error("isDockerfile(dockerfile.txt) should return true (HasPrefix matches)")
	}
	if isDockerfile("my-dockerfile") {
		t.Error("isDockerfile(my-dockerfile) should return false")
	}

	// Test docker-compose matching
	if !isComposeFile("docker-compose.yml") {
		t.Error("isComposeFile(docker-compose.yml) should return true")
	}
	if !isComposeFile("compose.yaml") {
		t.Error("isComposeFile(compose.yaml) should return true")
	}

	// Test Kubernetes YAML matching
	if !isKubernetesYAML("k8s/deployment.yaml", "deployment.yaml") {
		t.Error("isKubernetesYAML(k8s/deployment.yaml, deployment.yaml) should return true")
	}
	if isKubernetesYAML("app/config.yaml", "config.yaml") {
		t.Error("isKubernetesYAML(app/config.yaml, config.yaml) should return false")
	}

	// Test Terraform matching
	if !isTerraformFile("main.tf") {
		t.Error("isTerraformFile(main.tf) should return true")
	}
	if !isTerraformFile("modules/vpc.tf") {
		t.Error("isTerraformFile(modules/vpc.tf) should return true")
	}

	// Test GitHub workflow matching
	if !isGitHubWorkflow(".github/workflows/ci.yml") {
		t.Error("isGitHubWorkflow(.github/workflows/ci.yml) should return true")
	}
	if isGitHubWorkflow("workflows/ci.yml") {
		t.Error("isGitHubWorkflow(workflows/ci.yml) should return false")
	}

	// Test GitLab CI matching
	if !isGitLabCI("", ".gitlab-ci.yml") {
		t.Error("isGitLabCI(, .gitlab-ci.yml) should return true")
	}

	// Test Jenkinsfile matching
	if !isJenkinsfile("Jenkinsfile") {
		t.Error("isJenkinsfile(Jenkinsfile) should return true")
	}
	if !isJenkinsfile("Jenkinsfile.prod") {
		t.Error("isJenkinsfile(Jenkinsfile.prod) should return true")
	}
}
