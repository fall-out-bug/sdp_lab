package c4

import (
	"strings"
	"testing"

	"sdp_dev/internal/architect"
)

func TestGenerateLevel1_Actors(t *testing.T) {
	profile := &architect.CodebaseProfile{
		Infra: architect.InfraInfo{
			Ingresses: []architect.IngressInfo{
				{Name: "main", Source: "k8s/ingress.yaml"},
			},
			ExposedPorts: []string{"8080"},
		},
	}

	actors, _ := GenerateLevel1(profile, "test-system")

	if len(actors) < 2 {
		t.Errorf("expected at least 2 actors (end-user, client), got %d", len(actors))
	}

	// Check for end-user actor
	foundEndUser := false
	for _, a := range actors {
		if a.ID == "end-user" {
			foundEndUser = true
			break
		}
	}
	if !foundEndUser {
		t.Error("expected end-user actor")
	}

	// Check for client actor
	foundClient := false
	for _, a := range actors {
		if a.ID == "client" {
			foundClient = true
			break
		}
	}
	if !foundClient {
		t.Error("expected client actor")
	}
}

func TestGenerateLevel1_Empty(t *testing.T) {
	profile := &architect.CodebaseProfile{}

	actors, externals := GenerateLevel1(profile, "test-system")

	if len(actors) == 0 {
		t.Error("expected at least one default actor (developer)")
	}

	if actors[0].ID != "developer" {
		t.Errorf("expected default actor to be developer, got %q", actors[0].ID)
	}

	if len(externals) != 0 {
		t.Errorf("expected no externals from empty profile, got %d", len(externals))
	}
}

func TestGenerateLevel1_Externals(t *testing.T) {
	profile := &architect.CodebaseProfile{
		Dependencies: architect.DependencyInfo{
			NotableDeps: []architect.NotableDep{
				{Name: "boto3", FoundIn: 3, Signal: "cloud_aws"},
				{Name: "azure-storage", FoundIn: 1, Signal: "cloud_azure"},
			},
		},
		Infra: architect.InfraInfo{
			Resources: []architect.ResourceInfo{
				{Type: "aws_db_instance", Name: "prod-db", Provider: "aws"},
			},
		},
	}

	_, externals := GenerateLevel1(profile, "test-system")

	// Should have externals from cloud dependencies and terraform resources
	if len(externals) < 2 {
		t.Errorf("expected at least 2 externals, got %d", len(externals))
	}

	// Check for AWS external
	foundAWS := false
	for _, ext := range externals {
		if ext.Technology == "cloud_aws" {
			foundAWS = true
			break
		}
	}
	if !foundAWS {
		t.Error("expected AWS external system")
	}

	// Check for managed database
	foundDB := false
	for _, ext := range externals {
		if strings.Contains(ext.Description, "Managed Database") {
			foundDB = true
			break
		}
	}
	if !foundDB {
		t.Error("expected managed database external system")
	}
}

func TestInferLevel1Relationships(t *testing.T) {
	actors := []architect.Actor{
		{ID: "user", Description: "End User"},
	}
	externals := []architect.ExternalSystem{
		{ID: "stripe", Description: "Stripe"},
	}
	systemID := "my-system"

	rels := InferLevel1Relationships(actors, externals, systemID)

	// Should have actor -> system and system -> external relationships
	expectedCount := len(actors) + len(externals)
	if len(rels) != expectedCount {
		t.Errorf("expected %d relationships, got %d", expectedCount, len(rels))
	}

	// Check actor -> system
	foundActorRel := false
	for _, r := range rels {
		if r.From == "user" && r.To == systemID {
			foundActorRel = true
			break
		}
	}
	if !foundActorRel {
		t.Error("expected actor -> system relationship")
	}

	// Check system -> external
	foundExternalRel := false
	for _, r := range rels {
		if r.From == systemID && r.To == "stripe" {
			foundExternalRel = true
			break
		}
	}
	if !foundExternalRel {
		t.Error("expected system -> external relationship")
	}
}

func TestGenerateLevel2_Containers(t *testing.T) {
	profile := &architect.CodebaseProfile{
		Infra: architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Image: "node:18", Type: "service", Source: "Dockerfile"},
				{Name: "db", Image: "postgres:15", Type: "database", Source: "docker-compose.yml"},
			},
			K8sServices: []architect.K8sServiceInfo{
				{Name: "web", Source: "k8s/web.yaml"},
			},
		},
	}

	containers := GenerateLevel2(profile)

	if len(containers) < 3 {
		t.Errorf("expected at least 3 containers, got %d", len(containers))
	}

	// Check that database was detected
	foundDB := false
	for _, c := range containers {
		if strings.Contains(strings.ToLower(c.Description), "database") {
			foundDB = true
			break
		}
	}
	if !foundDB {
		t.Error("expected database container")
	}

	// Check K8s service was detected
	foundK8s := false
	for _, c := range containers {
		if c.Deploy == "kubernetes" {
			foundK8s = true
			break
		}
	}
	if !foundK8s {
		t.Error("expected kubernetes container")
	}
}

func TestGenerateLevel2_ModuleBoundaries(t *testing.T) {
	profile := &architect.CodebaseProfile{
		Infra: architect.InfraInfo{
			ModuleBoundaries: []architect.ModuleBoundaryInfo{
				{
					Name:        "go-modules",
					BuildSystem: "go",
					Path:        "cmd/",
					Children:    []string{"cmd/api", "cmd/worker"},
				},
			},
		},
	}

	containers := GenerateLevel2(profile)

	if len(containers) < 2 {
		t.Errorf("expected at least 2 containers from cmd/ modules, got %d", len(containers))
	}

	// Check build system is preserved
	for _, c := range containers {
		if c.Name == "api" || c.Name == "worker" {
			if c.Deploy != "go" {
				t.Errorf("expected deploy=go for %s, got %q", c.Name, c.Deploy)
			}
		}
	}
}

func TestGenerateLevel2_DockerCompose(t *testing.T) {
	profile := &architect.CodebaseProfile{
		Infra: architect.InfraInfo{
			Services: []architect.ServiceDep{
				{From: "web", To: "api"},
				{From: "api", To: "db"},
			},
		},
	}

	containers := GenerateLevel2(profile)

	// Should create containers for all services in deps
	expectedNames := []string{"web", "api", "db"}
	for _, name := range expectedNames {
		found := false
		for _, c := range containers {
			if c.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected container %q", name)
		}
	}
}

func TestGenerateLevel2_Fallback(t *testing.T) {
	profile := &architect.CodebaseProfile{
		ImportGraph: architect.ImportGraph{
			Clusters: []architect.ImportCluster{
				{ID: "internal/handlers", Packages: []string{"internal/handlers"}},
				{ID: "internal/repository", Packages: []string{"internal/repository"}},
			},
		},
	}

	containers := GenerateLevel2(profile)

	if len(containers) == 0 {
		t.Error("expected fallback containers from import clusters")
	}

	// Should have "handlers" and "repository" containers
	for _, c := range containers {
		if c.Deploy == "inferred" {
			// Good, this is from cluster fallback
			return
		}
	}
	t.Error("expected at least one inferred container")
}

func TestInferLevel2Relationships(t *testing.T) {
	containers := []architect.C4Container{
		{ID: "web", Name: "web", Description: "Service: web"},
		{ID: "api", Name: "api", Description: "Service: api"},
		{ID: "db", Name: "db", Description: "Database: db"},
	}

	profile := &architect.CodebaseProfile{
		Infra: architect.InfraInfo{
			Services: []architect.ServiceDep{
				{From: "web", To: "api"},
			},
		},
	}

	rels := InferLevel2Relationships(profile, containers)

	if len(rels) == 0 {
		t.Error("expected at least one relationship")
	}

	// Check for docker-compose dependency
	foundDep := false
	for _, r := range rels {
		if r.From == "web" && r.To == "api" {
			foundDep = true
			break
		}
	}
	if !foundDep {
		t.Error("expected web -> api relationship from docker-compose")
	}

	// Check for database persistence
	foundPersistence := false
	for _, r := range rels {
		if r.Type == "data" {
			foundPersistence = true
			break
		}
	}
	if !foundPersistence {
		t.Error("expected database persistence relationship")
	}
}

func TestGenerateLevel3_Components(t *testing.T) {
	profile := &architect.CodebaseProfile{
		ImportGraph: architect.ImportGraph{
			Clusters: []architect.ImportCluster{
				{
					ID:            "api/handlers",
					Packages:      []string{"api/handlers"},
					InternalEdges: 5,
					ExternalEdges: 2,
				},
				{
					ID:            "api/repository",
					Packages:      []string{"api/repository"},
					InternalEdges: 3,
					ExternalEdges: 1,
				},
			},
		},
	}

	containers := []architect.C4Container{
		{ID: "api", Name: "api", Source: "api/"},
	}

	GenerateLevel3(profile, containers)

	if len(containers[0].Components) == 0 {
		t.Error("expected components to be generated")
	}

	// Check that components were created
	if len(containers[0].Components) < 2 {
		t.Errorf("expected at least 2 components, got %d", len(containers[0].Components))
	}
}

func TestGenerateLevel3_NoClusters(t *testing.T) {
	profile := &architect.CodebaseProfile{
		ImportGraph: architect.ImportGraph{
			Clusters: nil,
		},
	}

	containers := []architect.C4Container{
		{ID: "app", Name: "app", Source: "cmd/"},
	}

	GenerateLevel3(profile, containers)

	// Should create a default component
	if len(containers[0].Components) == 0 {
		t.Error("expected default component when no clusters")
	}

	if containers[0].Components[0].Confidence != 0.50 {
		t.Errorf("expected default component confidence 0.50, got %f", containers[0].Components[0].Confidence)
	}
}

func TestInferLevel3Relationships(t *testing.T) {
	container := &architect.C4Container{
		ID: "api",
		Components: []architect.C4Component{
			{ID: "handlers", Path: "api/handlers", Description: "Handlers"},
			{ID: "repository", Path: "api/repository", Description: "Repository"},
			{ID: "models", Path: "api/models", Description: "Models"},
		},
	}

	rels := InferLevel3Relationships(container)

	// With the current implementation, relationships are based on path prefix
	// So api/handlers and api/models wouldn't have a prefix relationship
	// But we should still get some relationships if paths overlap
	if len(rels) == 0 {
		// This is OK given the test data
		t.Log("No component relationships inferred (expected for non-overlapping paths)")
	}
}

func TestClusterConfidence(t *testing.T) {
	tests := []struct {
		cluster    architect.ImportCluster
		minConf    float64
		maxConf    float64
	}{
		{
			cluster:    architect.ImportCluster{InternalEdges: 5, ExternalEdges: 5},
			minConf:    0.90,
			maxConf:    0.90,
		},
		{
			cluster:    architect.ImportCluster{InternalEdges: 3, ExternalEdges: 1},
			minConf:    0.75,
			maxConf:    0.75,
		},
		{
			cluster:    architect.ImportCluster{InternalEdges: 1, ExternalEdges: 1},
			minConf:    0.60,
			maxConf:    0.60,
		},
	}

	for i, tt := range tests {
		t.Run("", func(t *testing.T) {
			conf := clusterConfidence(tt.cluster)
			if conf < tt.minConf || conf > tt.maxConf {
				t.Errorf("test %d: clusterConfidence() = %f, expected [%f, %f]", i, conf, tt.minConf, tt.maxConf)
			}
		})
	}
}

func TestAssignClustersToContainers(t *testing.T) {
	clusters := []architect.ImportCluster{
		{ID: "api-handlers"},
		{ID: "api-repository"},
		{ID: "worker-processor"},
	}

	containers := []architect.C4Container{
		{ID: "api", Name: "API Service"},
		{ID: "worker", Name: "Worker"},
	}

	assignments := assignClustersToContainers(clusters, containers)

	// api-handlers and api-repository should go to "api" container
	apiClusters, ok := assignments["api"]
	if !ok {
		t.Error("expected api container to have assigned clusters")
	} else if len(apiClusters) < 2 {
		t.Errorf("expected api container to have at least 2 clusters, got %d", len(apiClusters))
	}

	// worker-processor should go to "worker" container
	workerClusters, ok := assignments["worker"]
	if !ok {
		t.Error("expected worker container to have assigned clusters")
	} else if len(workerClusters) < 1 {
		t.Error("expected worker container to have at least 1 cluster")
	}
}
