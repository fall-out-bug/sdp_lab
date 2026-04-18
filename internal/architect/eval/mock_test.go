package eval

import (
	"testing"

	"sdp_dev/internal/architect"
)

func TestNewMockExtractor(t *testing.T) {
	me := NewMockExtractor()
	if me == nil {
		t.Fatal("NewMockExtractor returned nil")
	}
	if me.Vary {
		t.Error("expected Vary=false")
	}
	if me.Drift != 0.0 {
		t.Errorf("expected Drift=0.0, got %.2f", me.Drift)
	}
}

func TestNewVaryingMockExtractor(t *testing.T) {
	me := NewVaryingMockExtractor(0.3)
	if me == nil {
		t.Fatal("NewVaryingMockExtractor returned nil")
	}
	if !me.Vary {
		t.Error("expected Vary=true")
	}
	if me.Drift != 0.3 {
		t.Errorf("expected Drift=0.3, got %.2f", me.Drift)
	}
}

func TestMockExtractor_Extract_GoSimpleCLI(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("go-simple-cli")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if len(fragment.Languages) != 1 {
		t.Errorf("expected 1 language, got %d", len(fragment.Languages))
	}
	if fragment.Languages[0].Primary != "go" {
		t.Errorf("expected primary language 'go', got %s", fragment.Languages[0].Primary)
	}
}

func TestMockExtractor_Extract_GoMultiModule(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("go-multi-module")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if len(fragment.Dependencies) != 3 {
		t.Errorf("expected 3 dependencies, got %d", len(fragment.Dependencies))
	}
	if fragment.ImportGraph == nil {
		t.Error("expected non-nil ImportGraph")
	}
	if len(fragment.ImportGraph.Clusters) != 2 {
		t.Errorf("expected 2 clusters, got %d", len(fragment.ImportGraph.Clusters))
	}
}

func TestMockExtractor_Extract_GoGRPCService(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("go-grpc-service")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if len(fragment.Specs) != 1 {
		t.Errorf("expected 1 spec, got %d", len(fragment.Specs))
	}
	if fragment.Specs[0].Kind != "protobuf" {
		t.Errorf("expected spec kind 'protobuf', got %s", fragment.Specs[0].Kind)
	}
}

func TestMockExtractor_Extract_GinAPI(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("go-gin-api")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if len(fragment.Infra.Containers) != 2 {
		t.Errorf("expected 2 containers, got %d", len(fragment.Infra.Containers))
	}
}

func TestMockExtractor_Extract_FlaskApp(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("python-flask")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if fragment.Languages[0].Primary != "python" {
		t.Errorf("expected primary language 'python', got %s", fragment.Languages[0].Primary)
	}
}

func TestMockExtractor_Extract_FastAPIApp(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("python-fastapi")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if len(fragment.Specs) != 1 {
		t.Errorf("expected 1 spec, got %d", len(fragment.Specs))
	}
}

func TestMockExtractor_Extract_DjangoMonolith(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("python-django")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if len(fragment.Infra.Containers) != 3 {
		t.Errorf("expected 3 containers, got %d", len(fragment.Infra.Containers))
	}
}

func TestMockExtractor_Extract_SpringBootMonolith(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("java-spring-boot")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if fragment.Languages[0].Primary != "java" {
		t.Errorf("expected primary language 'java', got %s", fragment.Languages[0].Primary)
	}
}

func TestMockExtractor_Extract_GradleMultiProject(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("java-gradle-multi")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if len(fragment.Dependencies) != 3 {
		t.Errorf("expected 3 dependencies, got %d", len(fragment.Dependencies))
	}
	if len(fragment.ImportGraph.Clusters) != 2 {
		t.Errorf("expected 2 clusters, got %d", len(fragment.ImportGraph.Clusters))
	}
}

func TestMockExtractor_Extract_NextjsApp(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("typescript-nextjs")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if fragment.Languages[0].Primary != "typescript" {
		t.Errorf("expected primary language 'typescript', got %s", fragment.Languages[0].Primary)
	}
}

func TestMockExtractor_Extract_NestJSService(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("typescript-nestjs")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if len(fragment.Specs) != 1 {
		t.Errorf("expected 1 spec, got %d", len(fragment.Specs))
	}
}

func TestMockExtractor_Extract_ExpressAPI(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("javascript-express")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if fragment.Languages[0].Primary != "javascript" {
		t.Errorf("expected primary language 'javascript', got %s", fragment.Languages[0].Primary)
	}
}

func TestMockExtractor_Extract_JSMonorepo(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("javascript-monorepo")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if len(fragment.Dependencies) != 4 {
		t.Errorf("expected 4 dependencies, got %d", len(fragment.Dependencies))
	}
	if len(fragment.ImportGraph.Clusters) != 3 {
		t.Errorf("expected 3 clusters, got %d", len(fragment.ImportGraph.Clusters))
	}
}

func TestMockExtractor_Extract_SQLMigrationDir(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("sql-migration-dir")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if fragment.SQLAnalysis == nil {
		t.Error("expected non-nil SQLAnalysis")
	}
	if len(fragment.SQLAnalysis.Tables) != 4 {
		t.Errorf("expected 4 tables, got %d", len(fragment.SQLAnalysis.Tables))
	}
}

func TestMockExtractor_Extract_SQLGORM(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("sql-orm-gorm")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if len(fragment.Generated) != 2 {
		t.Errorf("expected 2 generated files, got %d", len(fragment.Generated))
	}
}

func TestMockExtractor_Extract_SQLSQLAlchemy(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("sql-orm-sqlalchemy")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if len(fragment.Generated) != 2 {
		t.Errorf("expected 2 generated files, got %d", len(fragment.Generated))
	}
}

func TestMockExtractor_Extract_SQLPrisma(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("sql-orm-prisma")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if len(fragment.Generated) != 1 {
		t.Errorf("expected 1 generated file, got %d", len(fragment.Generated))
	}
}

func TestMockExtractor_Extract_MicroservicesRepo(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("cross-repo-microservices")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if len(fragment.Languages[0].All) != 3 {
		t.Errorf("expected 3 languages, got %d", len(fragment.Languages[0].All))
	}
	if len(fragment.Infra.Containers) != 5 {
		t.Errorf("expected 5 containers, got %d", len(fragment.Infra.Containers))
	}
}

func TestMockExtractor_Extract_MixedMonorepo(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("cross-repo-monorepo")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	if len(fragment.Languages[0].All) != 4 {
		t.Errorf("expected 4 languages, got %d", len(fragment.Languages[0].All))
	}
	if fragment.Infra.DeploymentType != "kubernetes" {
		t.Errorf("expected deployment type 'kubernetes', got %s", fragment.Infra.DeploymentType)
	}
}

func TestMockExtractor_Extract_UnknownRepo(t *testing.T) {
	me := NewMockExtractor()
	fragment := me.Extract("unknown-repo")

	if fragment == nil {
		t.Fatal("Extract returned nil")
	}

	// Should return empty fragment for unknown repos
	if len(fragment.Languages) != 0 {
		t.Errorf("expected 0 languages for unknown repo, got %d", len(fragment.Languages))
	}
}

func TestMockExtractor_Varying(t *testing.T) {
	me := NewVaryingMockExtractor(0.3)
	base := me.getBaseFragment("go-simple-cli")
	varying := me.Extract("go-simple-cli")

	// Varying should be different from base
	if len(base.Languages) == len(varying.Languages) {
		t.Error("expected varying fragment to have different language count")
	}
}

func TestMockExtractor_ApplyVariation(t *testing.T) {
	me := &MockExtractor{
		Vary: true,
		Drift: 0.5,
	}

	fragment := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "web", Type: "service", Source: "Dockerfile"},
				{Name: "db", Type: "database", Source: "docker-compose.yml"},
				{Name: "cache", Type: "cache", Source: "docker-compose.yml"},
			},
		},
	}

	varied := me.applyVariation(fragment)

	if len(varied.Languages) <= len(fragment.Languages) {
		t.Error("expected variation to add languages")
	}

	if varied.Infra == nil {
		t.Error("expected non-nil Infra")
	}

	if len(varied.Infra.Containers) >= len(fragment.Infra.Containers) {
		t.Errorf("expected variation to remove containers, got %d (original %d)", len(varied.Infra.Containers), len(fragment.Infra.Containers))
	}
}

func TestGetBaseFragment_AllKnownRepos(t *testing.T) {
	me := NewMockExtractor()

	knownRepos := []string{
		"go-simple-cli",
		"go-multi-module",
		"go-grpc-service",
		"go-gin-api",
		"python-flask",
		"python-fastapi",
		"python-django",
		"java-spring-boot",
		"java-gradle-multi",
		"typescript-nextjs",
		"typescript-nestjs",
		"javascript-express",
		"javascript-monorepo",
		"sql-migration-dir",
		"sql-orm-gorm",
		"sql-orm-sqlalchemy",
		"sql-orm-prisma",
		"cross-repo-microservices",
		"cross-repo-monorepo",
	}

	for _, repo := range knownRepos {
		fragment := me.getBaseFragment(repo)
		if fragment == nil {
			t.Errorf("getBaseFragment(%q) returned nil", repo)
		}
	}
}
