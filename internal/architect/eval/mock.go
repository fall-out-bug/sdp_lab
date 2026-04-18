// Package eval provides mock extractor outputs for unit testing.
// It enables testing the evaluation pipeline without running real extractors.
package eval

import (
	"sdp_dev/internal/architect"
)

// MockExtractor generates mock ProfileFragment outputs for testing.
type MockExtractor struct {
	// Vary determines if the mock should introduce variations (imperfect matches).
	Vary bool
	// Drift determines the amount of drift from expected (0.0 = perfect, 1.0 = random).
	Drift float64
}

// NewMockExtractor creates a new mock extractor.
func NewMockExtractor() *MockExtractor {
	return &MockExtractor{
		Vary: false,
		Drift: 0.0,
	}
}

// NewVaryingMockExtractor creates a mock extractor that produces imperfect outputs.
func NewVaryingMockExtractor(drift float64) *MockExtractor {
	return &MockExtractor{
		Vary: true,
		Drift: drift,
	}
}

// Extract generates a mock ProfileFragment for the given repo.
func (m *MockExtractor) Extract(repoName string) *architect.ProfileFragment {
	fragment := m.getBaseFragment(repoName)

	if m.Vary {
		return m.applyVariation(fragment)
	}

	return fragment
}

// getBaseFragment returns the base (expected) fragment for a repo.
func (m *MockExtractor) getBaseFragment(repoName string) *architect.ProfileFragment {
	// Return predefined fragments for known test repos
	switch repoName {
	case "go-simple-cli":
		return m.goSimpleCLI()
	case "go-multi-module":
		return m.goMultiModule()
	case "go-grpc-service":
		return m.goGRPCService()
	case "go-gin-api":
		return m.ginAPI()
	case "python-flask":
		return m.flaskApp()
	case "python-fastapi":
		return m.fastAPIApp()
	case "python-django":
		return m.djangoMonolith()
	case "java-spring-boot":
		return m.springBootMonolith()
	case "java-gradle-multi":
		return m.gradleMultiProject()
	case "typescript-nextjs":
		return m.nextjsApp()
	case "typescript-nestjs":
		return m.nestJSService()
	case "javascript-express":
		return m.expressAPI()
	case "javascript-monorepo":
		return m.jsMonorepo()
	case "sql-migration-dir":
		return m.sqlMigrationDir()
	case "sql-orm-gorm":
		return m.sqlGORM()
	case "sql-orm-sqlalchemy":
		return m.sqlSQLAlchemy()
	case "sql-orm-prisma":
		return m.sqlPrisma()
	case "cross-repo-microservices":
		return m.microservicesRepo()
	case "cross-repo-monorepo":
		return m.mixedMonorepo()
	default:
		return &architect.ProfileFragment{}
	}
}

// applyVariation applies variations to simulate imperfect extraction.
func (m *MockExtractor) applyVariation(fragment *architect.ProfileFragment) *architect.ProfileFragment {
	// This is a simplified version - in real use, you'd apply more sophisticated variations
	// based on the drift parameter

	// For now, just return a slightly modified version
	result := *fragment

	// Add some false positives in languages (make a copy of the slice)
	if len(result.Languages) > 0 {
		newLanguages := make([]architect.LanguageInfo, len(result.Languages), len(result.Languages)+1)
		copy(newLanguages, result.Languages)
		newLanguages = append(newLanguages, architect.LanguageInfo{
			Primary: "rust",
			All:     []string{"rust"},
		})
		result.Languages = newLanguages
	}

	// Remove some containers (make a copy of the slice)
	if result.Infra != nil && len(result.Infra.Containers) > 1 {
		newContainers := make([]architect.ContainerInfo, len(result.Infra.Containers)-1)
		copy(newContainers, result.Infra.Containers[:len(result.Infra.Containers)-1])
		// Create a copy of InfraInfo with new containers
		newInfra := *result.Infra
		newInfra.Containers = newContainers
		result.Infra = &newInfra
	}

	return &result
}

// --- Go Golden Test Cases ---

func (m *MockExtractor) goSimpleCLI() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "go.mod", Language: "go", DepCount: 5},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 3,
			Edges: 4,
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{},
		},
	}
}

func (m *MockExtractor) goMultiModule() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "go.mod", Language: "go", DepCount: 10},
			{File: "cmd/service/go.mod", Language: "go", DepCount: 5},
			{File: "pkg/api/go.mod", Language: "go", DepCount: 3},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 15,
			Edges: 25,
			Clusters: []architect.ImportCluster{
				{ID: "cmd", Packages: []string{"cmd/server"}, InternalEdges: 3, ExternalEdges: 2},
				{ID: "pkg", Packages: []string{"pkg/api", "pkg/auth"}, InternalEdges: 5, ExternalEdges: 3},
			},
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Type: "service", Source: "Dockerfile"},
			},
		},
	}
}

func (m *MockExtractor) goGRPCService() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go", "protobuf"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "go.mod", Language: "go", DepCount: 8},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 12,
			Edges: 18,
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "grpc-server", Type: "service", Source: "Dockerfile"},
			},
		},
		Specs: []architect.SpecArtifact{
			{Kind: "protobuf", Path: "proto/service.proto"},
		},
	}
}

func (m *MockExtractor) ginAPI() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "go.mod", Language: "go", DepCount: 12, Signals: []string{"web_framework"}},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 20,
			Edges: 35,
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Type: "service", Source: "Dockerfile"},
				{Name: "postgres", Type: "database", Source: "docker-compose.yml"},
			},
		},
	}
}

// --- Python Golden Test Cases ---

func (m *MockExtractor) flaskApp() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "python", All: []string{"python"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "requirements.txt", Language: "python", DepCount: 8},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 10,
			Edges: 15,
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "web", Type: "service", Source: "Dockerfile"},
			},
		},
	}
}

func (m *MockExtractor) fastAPIApp() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "python", All: []string{"python"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "requirements.txt", Language: "python", DepCount: 10},
			{File: "pyproject.toml", Language: "python", DepCount: 5},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 15,
			Edges: 20,
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Type: "service", Source: "Dockerfile"},
			},
		},
		Specs: []architect.SpecArtifact{
			{Kind: "openapi", Path: "openapi.json"},
		},
	}
}

func (m *MockExtractor) djangoMonolith() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "python", All: []string{"python", "javascript"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "requirements.txt", Language: "python", DepCount: 25},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 40,
			Edges: 60,
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "web", Type: "service", Source: "Dockerfile"},
				{Name: "db", Type: "database", Source: "docker-compose.yml"},
				{Name: "cache", Type: "cache", Source: "docker-compose.yml"},
			},
		},
	}
}

// --- Java/Kotlin Golden Test Cases ---

func (m *MockExtractor) springBootMonolith() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "java", All: []string{"java", "xml"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "pom.xml", Language: "java", DepCount: 20},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 30,
			Edges: 50,
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "app", Type: "service", Source: "Dockerfile"},
				{Name: "mysql", Type: "database", Source: "docker-compose.yml"},
			},
		},
	}
}

func (m *MockExtractor) gradleMultiProject() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "java", All: []string{"java", "kotlin"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "build.gradle", Language: "java", DepCount: 15},
			{File: "api/build.gradle", Language: "java", DepCount: 8},
			{File: "service/build.gradle", Language: "kotlin", DepCount: 10},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 25,
			Edges: 40,
			Clusters: []architect.ImportCluster{
				{ID: "api", Packages: []string{"com.example.api"}, InternalEdges: 5, ExternalEdges: 3},
				{ID: "service", Packages: []string{"com.example.service"}, InternalEdges: 8, ExternalEdges: 5},
			},
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Type: "service", Source: "Dockerfile"},
				{Name: "service", Type: "service", Source: "Dockerfile"},
			},
		},
	}
}

// --- TypeScript/JavaScript Golden Test Cases ---

func (m *MockExtractor) nextjsApp() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "typescript", All: []string{"typescript", "javascript", "css"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "package.json", Language: "typescript", DepCount: 20},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 50,
			Edges: 80,
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "web", Type: "service", Source: "Dockerfile"},
			},
		},
	}
}

func (m *MockExtractor) nestJSService() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "typescript", All: []string{"typescript"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "package.json", Language: "typescript", DepCount: 15},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 20,
			Edges: 30,
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Type: "service", Source: "Dockerfile"},
				{Name: "postgres", Type: "database", Source: "docker-compose.yml"},
			},
		},
		Specs: []architect.SpecArtifact{
			{Kind: "openapi", Path: "api/openapi.yaml"},
		},
	}
}

func (m *MockExtractor) expressAPI() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "javascript", All: []string{"javascript"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "package.json", Language: "javascript", DepCount: 12},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 15,
			Edges: 20,
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Type: "service", Source: "Dockerfile"},
			},
		},
	}
}

func (m *MockExtractor) jsMonorepo() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "typescript", All: []string{"typescript", "javascript"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "package.json", Language: "typescript", DepCount: 10},
			{File: "packages/api/package.json", Language: "typescript", DepCount: 8},
			{File: "packages/web/package.json", Language: "typescript", DepCount: 15},
			{File: "packages/mobile/package.json", Language: "typescript", DepCount: 12},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 60,
			Edges: 100,
			Clusters: []architect.ImportCluster{
				{ID: "packages/api", Packages: []string{"@monorepo/api"}, InternalEdges: 10, ExternalEdges: 5},
				{ID: "packages/web", Packages: []string{"@monorepo/web"}, InternalEdges: 15, ExternalEdges: 8},
				{ID: "packages/mobile", Packages: []string{"@monorepo/mobile"}, InternalEdges: 12, ExternalEdges: 6},
			},
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Type: "service", Source: "packages/api/Dockerfile"},
				{Name: "web", Type: "service", Source: "packages/web/Dockerfile"},
			},
		},
	}
}

// --- SQL Golden Test Cases ---

func (m *MockExtractor) sqlMigrationDir() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "sql", All: []string{"sql"}},
		},
		SQLAnalysis: &architect.SQLAnalysis{
			Tables: []architect.Table{
				{Name: "users", Schema: "public"},
				{Name: "posts", Schema: "public"},
				{Name: "comments", Schema: "public"},
				{Name: "migrations", Schema: "public"},
			},
		},
	}
}

func (m *MockExtractor) sqlGORM() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
		SQLAnalysis: &architect.SQLAnalysis{
			Tables: []architect.Table{
				{Name: "User", Schema: "main"},
				{Name: "Profile", Schema: "main"},
				{Name: "Settings", Schema: "main"},
			},
		},
		Generated: []architect.GeneratedFile{
			{Path: "models/user.go", Reason: "gorm:model"},
			{Path: "models/profile.go", Reason: "gorm:model"},
		},
	}
}

func (m *MockExtractor) sqlSQLAlchemy() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "python", All: []string{"python"}},
		},
		SQLAnalysis: &architect.SQLAnalysis{
			Tables: []architect.Table{
				{Name: "users", Schema: "public"},
				{Name: "addresses", Schema: "public"},
				{Name: "orders", Schema: "public"},
			},
		},
		Generated: []architect.GeneratedFile{
			{Path: "models/user.py", Reason: "sqlalchemy:model"},
			{Path: "models/address.py", Reason: "sqlalchemy:model"},
		},
	}
}

func (m *MockExtractor) sqlPrisma() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "typescript", All: []string{"typescript", "prisma"}},
		},
		SQLAnalysis: &architect.SQLAnalysis{
			Tables: []architect.Table{
				{Name: "User", Schema: "public"},
				{Name: "Post", Schema: "public"},
				{Name: "Comment", Schema: "public"},
			},
		},
		Generated: []architect.GeneratedFile{
			{Path: "node_modules/.prisma/client/index.ts", Reason: "prisma:client"},
		},
	}
}

// --- Cross-Repo Golden Test Cases ---

func (m *MockExtractor) microservicesRepo() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go", "python", "typescript"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "api/go.mod", Language: "go", DepCount: 10},
			{File: "worker/requirements.txt", Language: "python", DepCount: 8},
			{File: "web/package.json", Language: "typescript", DepCount: 15},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 35,
			Edges: 50,
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Type: "service", Source: "api/Dockerfile"},
				{Name: "worker", Type: "service", Source: "worker/Dockerfile"},
				{Name: "web", Type: "service", Source: "web/Dockerfile"},
				{Name: "postgres", Type: "database", Source: "docker-compose.yml"},
				{Name: "redis", Type: "cache", Source: "docker-compose.yml"},
			},
			DeploymentType: "docker-compose",
		},
		Specs: []architect.SpecArtifact{
			{Kind: "openapi", Path: "api/openapi.yaml"},
			{Kind: "protobuf", Path: "proto/service.proto"},
		},
	}
}

func (m *MockExtractor) mixedMonorepo() *architect.ProfileFragment {
	return &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go", "python", "typescript", "rust"}},
		},
		Dependencies: []architect.DependencyInfo{
			{File: "go-services/api/go.mod", Language: "go", DepCount: 12},
			{File: "python-services/ml/Cargo.toml", Language: "rust", DepCount: 15},
			{File: "web-app/package.json", Language: "typescript", DepCount: 20},
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 80,
			Edges: 120,
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Type: "service", Source: "go-services/api/Dockerfile"},
				{Name: "ml", Type: "service", Source: "python-services/ml/Dockerfile"},
				{Name: "web", Type: "service", Source: "web-app/Dockerfile"},
				{Name: "db", Type: "database", Source: "docker-compose.yml"},
			},
			DeploymentType: "kubernetes",
		},
	}
}
