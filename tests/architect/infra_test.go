package architect_test

import (
	"context"
	"path/filepath"
	"sort"
	"testing"

	"sdp_dev/internal/architect"
	"sdp_dev/internal/architect/extract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TestInfraExtractor_DockerCompose — 3-service compose
// ---------------------------------------------------------------------------

func TestInfraExtractor_DockerCompose(t *testing.T) {
	root := t.TempDir()

	composeYAML := `
services:
  web:
    image: nginx:1.25
    ports:
      - "8080:80"
    depends_on:
      - api
      - db
  api:
    build: ./api
    image: myapp/api:latest
    ports:
      - "3000:3000"
    depends_on:
      - db
  db:
    image: postgres:16
    ports:
      - "5432:5432"
`
	writeFile(t, root, "docker-compose.yml", composeYAML)

	ext := &extract.InfraExtractor{}
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.Infra)

	// 3 containers
	require.Len(t, frag.Infra.Containers, 3)
	names := containerNames(frag.Infra.Containers)
	sort.Strings(names)
	assert.Equal(t, []string{"api", "db", "web"}, names)

	// All containers have type "service"
	for _, c := range frag.Infra.Containers {
		assert.Equal(t, "service", c.Type, "container %s should be service", c.Name)
	}

	// depends_on edges
	require.True(t, len(frag.Infra.Services) >= 3, "expected at least 3 dependency edges, got %d", len(frag.Infra.Services))
	depSet := depEdges(frag.Infra.Services)
	assert.Contains(t, depSet, "web->api")
	assert.Contains(t, depSet, "web->db")
	assert.Contains(t, depSet, "api->db")

	// Ports present
	assert.NotEmpty(t, frag.Infra.ExposedPorts)

	// Deployment type should be docker-compose
	assert.Equal(t, "docker-compose", frag.Infra.DeploymentType)

	// Artifacts list
	assert.Contains(t, frag.InfraArtifacts, "docker-compose.yml")
}

// ---------------------------------------------------------------------------
// TestInfraExtractor_Dockerfile — multi-stage build
// ---------------------------------------------------------------------------

func TestInfraExtractor_Dockerfile(t *testing.T) {
	root := t.TempDir()

	dockerfile := `
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server .

# Runtime stage
FROM alpine:3.19
COPY --from=builder /app/server /usr/local/bin/server
EXPOSE 8080
EXPOSE 9090/udp
CMD ["server"]
`
	writeFile(t, root, "Dockerfile", dockerfile)

	ext := &extract.InfraExtractor{}
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.Infra)

	// Two base images (multi-stage)
	require.Len(t, frag.Infra.BaseImages, 2)
	assert.Contains(t, frag.Infra.BaseImages, "golang:1.22-alpine")
	assert.Contains(t, frag.Infra.BaseImages, "alpine:3.19")

	// Two exposed ports
	require.Len(t, frag.Infra.ExposedPorts, 2)
	assert.Contains(t, frag.Infra.ExposedPorts, "8080")
	assert.Contains(t, frag.Infra.ExposedPorts, "9090/udp")

	// Artifacts
	assert.Contains(t, frag.InfraArtifacts, "Dockerfile")

	// Deployment type bare (no compose/k8s)
	assert.Equal(t, "bare", frag.Infra.DeploymentType)
}

// ---------------------------------------------------------------------------
// TestInfraExtractor_Kubernetes — Deployment YAML
// ---------------------------------------------------------------------------

func TestInfraExtractor_Kubernetes(t *testing.T) {
	root := t.TempDir()

	k8sYAML := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: web
          image: myapp/web:v1.2.3
          ports:
            - containerPort: 80
            - containerPort: 443
        - name: sidecar
          image: envoyproxy/envoy:v1.28
          ports:
            - containerPort: 9901
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: cache
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: redis
          image: redis:7.2
          ports:
            - containerPort: 6379
`
	writeFile(t, root, "k8s/app.yaml", k8sYAML)

	ext := &extract.InfraExtractor{}
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.Infra)

	// 3 containers total (web + sidecar from Deployment, redis from StatefulSet)
	require.Len(t, frag.Infra.Containers, 3)
	names := containerNames(frag.Infra.Containers)
	sort.Strings(names)
	assert.Equal(t, []string{"redis", "sidecar", "web"}, names)

	// Container types
	typeMap := make(map[string]string)
	for _, c := range frag.Infra.Containers {
		typeMap[c.Name] = c.Type
	}
	assert.Equal(t, "deployment", typeMap["web"])
	assert.Equal(t, "deployment", typeMap["sidecar"])
	assert.Equal(t, "statefulset", typeMap["redis"])

	// Images recorded
	assert.Contains(t, frag.Infra.BaseImages, "myapp/web:v1.2.3")
	assert.Contains(t, frag.Infra.BaseImages, "envoyproxy/envoy:v1.28")
	assert.Contains(t, frag.Infra.BaseImages, "redis:7.2")

	// Ports
	assert.Contains(t, frag.Infra.ExposedPorts, "80")
	assert.Contains(t, frag.Infra.ExposedPorts, "443")
	assert.Contains(t, frag.Infra.ExposedPorts, "6379")

	// Deployment type should be kubernetes (highest priority)
	assert.Equal(t, "kubernetes", frag.Infra.DeploymentType)

	// Artifacts
	require.Len(t, frag.InfraArtifacts, 1)
	assert.Equal(t, filepath.Join("k8s", "app.yaml"), frag.InfraArtifacts[0])
}

// ---------------------------------------------------------------------------
// TestInfraExtractor_DeploymentTypeDetection — kubernetes vs compose vs bare
// ---------------------------------------------------------------------------

func TestInfraExtractor_DeploymentTypeDetection(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected string
	}{
		{
			name: "kubernetes wins over compose",
			files: map[string]string{
				"docker-compose.yml": `services:
  web:
    image: nginx
`,
				"k8s/deploy.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
        - name: app
          image: app:latest
`,
			},
			expected: "kubernetes",
		},
		{
			name: "compose when no k8s",
			files: map[string]string{
				"docker-compose.yml": `services:
  web:
    image: nginx
`,
				"Dockerfile": `FROM node:20
EXPOSE 3000
`,
			},
			expected: "docker-compose",
		},
		{
			name: "bare with only Dockerfile",
			files: map[string]string{
				"Dockerfile": `FROM python:3.12
EXPOSE 8000
`,
			},
			expected: "bare",
		},
		{
			name: "bare with only terraform",
			files: map[string]string{
				"main.tf": `resource "aws_lambda_function" "handler" {
  function_name = "my-lambda"
}
`,
			},
			expected: "bare",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for relPath, content := range tt.files {
				writeFile(t, root, relPath, content)
			}

			ext := &extract.InfraExtractor{}
			frag, err := ext.Extract(context.Background(), root)
			require.NoError(t, err)
			require.NotNil(t, frag.Infra)
			assert.Equal(t, tt.expected, frag.Infra.DeploymentType)
		})
	}
}

// ---------------------------------------------------------------------------
// TestInfraExtractor_Terraform — resource extraction
// ---------------------------------------------------------------------------

func TestInfraExtractor_Terraform(t *testing.T) {
	root := t.TempDir()

	tfContent := `
resource "aws_s3_bucket" "data" {
  bucket = "my-data-bucket"
}

resource "aws_lambda_function" "handler" {
  function_name = "my-handler"
  runtime       = "go1.x"
}

resource "google_compute_instance" "web" {
  name         = "web-server"
  machine_type = "e2-medium"
}
`
	writeFile(t, root, "main.tf", tfContent)

	ext := &extract.InfraExtractor{}
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.Infra)

	require.Len(t, frag.Infra.Resources, 3)

	resMap := make(map[string]string)
	provMap := make(map[string]string)
	for _, r := range frag.Infra.Resources {
		resMap[r.Name] = r.Type
		provMap[r.Name] = r.Provider
	}

	assert.Equal(t, "aws_s3_bucket", resMap["data"])
	assert.Equal(t, "aws_lambda_function", resMap["handler"])
	assert.Equal(t, "google_compute_instance", resMap["web"])

	assert.Equal(t, "aws", provMap["data"])
	assert.Equal(t, "aws", provMap["handler"])
	assert.Equal(t, "google", provMap["web"])

	assert.Contains(t, frag.InfraArtifacts, "main.tf")
}

// ---------------------------------------------------------------------------
// TestInfraExtractor_GitHubActions — services in workflow
// ---------------------------------------------------------------------------

func TestInfraExtractor_GitHubActions(t *testing.T) {
	root := t.TempDir()

	workflowYAML := `
name: CI
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        ports:
          - "5432:5432"
      redis:
        image: redis:7
        ports:
          - "6379:6379"
    steps:
      - uses: actions/checkout@v4
`
	writeFile(t, root, ".github/workflows/ci.yml", workflowYAML)

	ext := &extract.InfraExtractor{}
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.Infra)

	require.Len(t, frag.Infra.Containers, 2)
	names := containerNames(frag.Infra.Containers)
	sort.Strings(names)
	assert.Equal(t, []string{"postgres", "redis"}, names)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func containerNames(cs []architect.ContainerInfo) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Name
	}
	return out
}

type depEdge = string

func depEdges(deps []architect.ServiceDep) []depEdge {
	out := make([]depEdge, len(deps))
	for i, d := range deps {
		out[i] = d.From + "->" + d.To
	}
	return out
}
