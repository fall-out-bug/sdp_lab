// Package c4 provides C4 model generation from extractor outputs.
//
// It converts CodebaseProfile / ProfileFragment data into ReferenceModel
// instances containing C4 containers, components, and relationships with
// confidence scoring.  Phase 1 is deterministic (no LLM); Phase 2 LLM
// enrichment is out of scope for this package.
package c4

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"sdp_dev/internal/architect"
)

// GenerateOptions controls C4 model generation.
type GenerateOptions struct {
	// RepoName overrides the system name.  When empty, the basename of
	// RepoRoot is used.
	RepoName string

	// RepoRoot is the absolute path to the analysed repository.
	RepoRoot string

	// CommitHash is the git commit being analysed (optional).
	CommitHash string
}

// Generate builds a ReferenceModel from a CodebaseProfile.
//
// Phase 1 is fully deterministic. It runs container detection heuristics,
// creates components from import graph clusters, infers relationships, and
// computes confidence scores.
func Generate(profile *architect.CodebaseProfile, opts GenerateOptions) (*architect.ReferenceModel, error) {
	if profile == nil {
		return nil, fmt.Errorf("c4: profile is nil")
	}

	// Determine system name.
	name := opts.RepoName
	if name == "" && opts.RepoRoot != "" {
		name = filepath.Base(opts.RepoRoot)
	}
	if name == "" {
		name = "unknown-system"
	}

	model := &architect.ReferenceModel{
		Version:        "1.0.0",
		State:          architect.ModelObserved,
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		AnalyzedCommit: opts.CommitHash,
		System: architect.SystemInfo{
			Name:        name,
			Description: "",
		},
	}

	// --- Phase 1: Deterministic node/edge creation ---

	// Step 1: Detect containers (L2).
	containers := detectContainers(profile)
	model.Containers = containers

	// Step 2: Create components within containers (L3).
	assignComponents(profile, model)

	// Step 3: Infer relationships.
	model.Relationships = inferRelationships(profile, model)

	// Step 4: Derive actors and external systems from infra/dependency signals.
	model.Actors, model.ExternalSystems = detectActorsAndExternals(profile, model)

	// Step 5: Compute confidence scores.
	scoreModel(model, profile)

	return model, nil
}

// systemIDHash returns a short deterministic ID for the system node.
func systemIDHash(repoRoot string) string {
	h := sha256.Sum256([]byte(repoRoot))
	return fmt.Sprintf("%x", h[:8])
}

// containerID creates a canonical container ID from a name.
func containerID(name string) string {
	return slug(name)
}

// slug converts a name to a lowercase, hyphen-separated identifier.
func slug(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	// Remove characters that are not alphanumeric or hyphen.
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	result := b.String()
	if result == "" {
		return "unnamed"
	}
	return result
}

// detectContainers applies the priority-ordered heuristics from the C4 spec
// Section 2.2 to discover deployable units.
func detectContainers(profile *architect.CodebaseProfile) []architect.C4Container {
	var containers []architect.C4Container
	seen := make(map[string]bool)

	add := func(c architect.C4Container) {
		id := containerID(c.Name)
		if seen[id] {
			return
		}
		seen[id] = true
		c.ID = id
		containers = append(containers, c)
	}

	// Priority 1: Dockerfile services (from infra extractor).
	if profile.Infra.Containers != nil {
		for _, ci := range profile.Infra.Containers {
			// Skip CI-only containers.
			if isCIContainer(ci) {
				continue
			}

			cType := "service"
			tech := ""
			if ci.Image != "" {
				tech = imageToTech(ci.Image)
			}

			// Detect databases and message queues from image names.
			imgLower := strings.ToLower(ci.Image)
			switch {
			case containsAny(imgLower, "postgres", "mysql", "mongo", "mariadb", "cockroach", "cassandra", "dynamodb-local"):
				cType = "database"
			case containsAny(imgLower, "redis", "memcached"):
				cType = "cache"
			case containsAny(imgLower, "rabbitmq", "kafka", "nats", "sqs", "eventstore"):
				cType = "message_broker"
			}

			add(architect.C4Container{
				Name:        ci.Name,
				Technology:  tech,
				Source:      ci.Source,
				Deploy:      "docker",
				Description: containerDescription(cType, ci.Name),
			})
		}
	}

	// Priority 2: Kubernetes workloads.
	for _, k8sSvc := range profile.Infra.K8sServices {
		add(architect.C4Container{
			Name:        k8sSvc.Name,
			Technology:  "kubernetes",
			Source:      k8sSvc.Source,
			Deploy:      "kubernetes",
			Description: "Kubernetes service: " + k8sSvc.Name,
		})
	}

	// Priority 3: docker-compose service dependencies (captures cross-service edges).
	for _, sd := range profile.Infra.Services {
		// Ensure both the source and target service have containers.
		add(architect.C4Container{
			Name:        sd.From,
			Technology:  "",
			Source:      "docker-compose",
			Deploy:      "docker-compose",
			Description: "Service from docker-compose",
		})
		add(architect.C4Container{
			Name:        sd.To,
			Technology:  "",
			Source:      "docker-compose",
			Deploy:      "docker-compose",
			Description: "Service from docker-compose",
		})
	}

	// Priority 4: Module boundaries (cmd/, maven, gradle, npm workspaces).
	for _, mb := range profile.Infra.ModuleBoundaries {
		for _, child := range mb.Children {
			childName := filepath.Base(child)
			if childName == "." || childName == "" {
				childName = child
			}
			add(architect.C4Container{
				Name:        childName,
				Technology:  mb.BuildSystem,
				Source:      mb.Path,
				Deploy:      mb.BuildSystem,
				Description: mb.BuildSystem + " module: " + child,
			})
		}
	}

	// Priority 5: Import graph clusters as fallback containers.
	if profile.ImportGraph.Clusters != nil && len(containers) == 0 {
		for _, cluster := range profile.ImportGraph.Clusters {
			name := cluster.ID
			if name == "" && len(cluster.Packages) > 0 {
				name = cluster.Packages[0]
			}
			if name == "" {
				continue
			}
			// Use last path segment as name.
			parts := strings.Split(name, "/")
			shortName := parts[len(parts)-1]
			add(architect.C4Container{
				Name:        shortName,
				Technology:  "inferred",
				Source:      "import_cluster",
				Deploy:      "inferred",
				Description: "Inferred from import graph cluster",
			})
		}
	}

	// Fallback: If still no containers, create a single application container.
	if len(containers) == 0 {
		containers = append(containers, architect.C4Container{
			ID:          "app",
			Name:        "Application",
			Technology:  primaryTech(profile),
			Deploy:      "inferred",
			Description: "Single application container (no deploy boundaries detected)",
		})
	}

	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Name < containers[j].Name
	})

	return containers
}

// assignComponents populates C4Component entries within each container based
// on import graph clusters and directory structure.
func assignComponents(profile *architect.CodebaseProfile, model *architect.ReferenceModel) {
	// If there are no import graph clusters, we create one component per
	// container from its source path.
	if profile.ImportGraph.Clusters == nil || len(profile.ImportGraph.Clusters) == 0 {
		for i := range model.Containers {
			model.Containers[i].Components = []architect.C4Component{
				{
					ID:          model.Containers[i].ID + "-main",
					Path:        model.Containers[i].Source,
					Description: model.Containers[i].Name + " main module",
					Confidence:  0.50,
				},
			}
		}
		return
	}

	// Build a mapping from cluster packages to container IDs.
	// Strategy: assign each cluster to the container whose name best matches
	// the cluster's path.
	clusterAssignments := assignClustersToContainers(profile.ImportGraph.Clusters, model.Containers)

	for containerIdx := range model.Containers {
		cid := model.Containers[containerIdx].ID
		assignedClusters, ok := clusterAssignments[cid]
		if !ok || len(assignedClusters) == 0 {
			// No cluster assigned; create a single default component.
			model.Containers[containerIdx].Components = []architect.C4Component{
				{
					ID:          cid + "-default",
					Path:        model.Containers[containerIdx].Source,
					Description: model.Containers[containerIdx].Name + " default component",
					Confidence:  0.50,
				},
			}
			continue
		}

		for _, clusterIdx := range assignedClusters {
			cluster := profile.ImportGraph.Clusters[clusterIdx]
			compName := cluster.ID
			if compName == "" {
				compName = "component"
			}
			// Use the last path segment as the human-friendly name.
			parts := strings.Split(compName, "/")
			shortName := parts[len(parts)-1]

			comp := architect.C4Component{
				ID:          cid + "-" + slug(shortName),
				Path:        compName,
				Description: shortName + " component",
				Confidence:  clusterConfidence(cluster),
			}
			model.Containers[containerIdx].Components = append(
				model.Containers[containerIdx].Components, comp,
			)
		}
	}
}

// assignClustersToContainers maps each import cluster to the best-matching container.
// Returns map[containerID][]clusterIndex.
func assignClustersToContainers(clusters []architect.ImportCluster, containers []architect.C4Container) map[string][]int {
	result := make(map[string][]int)

	for ci, cluster := range clusters {
		bestContainer := ""
		bestScore := 0

		clusterLower := strings.ToLower(cluster.ID)
		for _, c := range containers {
			containerLower := strings.ToLower(c.Name)
			score := matchScore(clusterLower, containerLower)
			if score > bestScore {
				bestScore = score
				bestContainer = c.ID
			}
		}

		// If no match at all, assign to the first container.
		if bestContainer == "" && len(containers) > 0 {
			bestContainer = containers[0].ID
		}

		if bestContainer != "" {
			result[bestContainer] = append(result[bestContainer], ci)
		}
	}

	return result
}

// matchScore returns a simple string-overlap score between two identifiers.
func matchScore(a, b string) int {
	if a == b {
		return 100
	}
	if strings.Contains(a, b) || strings.Contains(b, a) {
		return 50
	}
	// Check if last path segments match.
	aParts := strings.Split(a, "/")
	bParts := strings.Split(b, "/")
	if aParts[len(aParts)-1] == bParts[len(bParts)-1] {
		return 40
	}
	return 0
}

// clusterConfidence returns a confidence score for a component derived from
// the given import cluster.
func clusterConfidence(cluster architect.ImportCluster) float64 {
	total := cluster.InternalEdges + cluster.ExternalEdges
	if total >= 5 {
		return 0.90
	}
	if total >= 3 {
		return 0.75
	}
	return 0.60
}

// imageToTech extracts a human-readable technology tag from a Docker image reference.
func imageToTech(image string) string {
	// Remove registry and tag.
	img := image
	if at := strings.Index(img, "@"); at >= 0 {
		img = img[:at]
	}
	if colon := strings.LastIndex(img, ":"); colon >= 0 {
		img = img[:colon]
	}
	// Use last path segment.
	parts := strings.Split(img, "/")
	return parts[len(parts)-1]
}

// containerDescription returns a default description based on container type.
func containerDescription(cType, name string) string {
	switch cType {
	case "database":
		return "Database: " + name
	case "cache":
		return "Cache: " + name
	case "message_broker":
		return "Message broker: " + name
	default:
		return "Service: " + name
	}
}

// primaryTech returns the primary language/technology from the profile.
func primaryTech(profile *architect.CodebaseProfile) string {
	if profile.Dependencies.Language != "" {
		return profile.Dependencies.Language
	}
	return "unknown"
}

// detectActorsAndExternals derives L1 actors and external systems from the
// profile data.
func detectActorsAndExternals(profile *architect.CodebaseProfile, model *architect.ReferenceModel) ([]architect.Actor, []architect.ExternalSystem) {
	var actors []architect.Actor
	var externals []architect.ExternalSystem

	// Detect actors from ingress / exposed ports.
	if len(profile.Infra.Ingresses) > 0 {
		actors = append(actors, architect.Actor{
			ID:          "end-user",
			Description: "End User",
		})
	}
	if len(profile.Infra.ExposedPorts) > 0 {
		actors = append(actors, architect.Actor{
			ID:          "client",
			Description: "API Client",
		})
	}

	// Detect external systems from notable dependencies.
	seenExternal := make(map[string]bool)
	for _, dep := range profile.Dependencies.NotableDeps {
		name := dep.Name
		signal := dep.Signal
		switch signal {
		case "cloud_aws", "cloud_azure", "cloud_gcp":
			if !seenExternal[name] {
				seenExternal[name] = true
				externals = append(externals, architect.ExternalSystem{
					ID:          slug(name),
					Description: name,
					Technology:  signal,
					Evidence:    fmt.Sprintf("Found in %d manifest(s)", dep.FoundIn),
				})
			}
		}
	}

	// Detect external databases/services from Terraform resources.
	for _, res := range profile.Infra.Resources {
		switch {
		case strings.Contains(res.Type, "db_instance"),
			strings.Contains(res.Type, "rds_cluster"),
			strings.Contains(res.Type, "sql_database"):
			if !seenExternal[res.Name] {
				seenExternal[res.Name] = true
				externals = append(externals, architect.ExternalSystem{
					ID:          slug(res.Name),
					Description: "Managed Database: " + res.Name,
					Technology:  res.Provider,
					Evidence:    "Terraform resource: " + res.Type,
				})
			}
		}
	}

	// Always have at least one actor for L1.
	if len(actors) == 0 {
		actors = append(actors, architect.Actor{
			ID:          "developer",
			Description: "Developer",
		})
	}

	return actors, externals
}

// isCIContainer returns true if the container appears to be a CI/build-time
// image rather than a runtime deploy unit.
func isCIContainer(ci architect.ContainerInfo) bool {
	src := strings.ToLower(ci.Source)
	name := strings.ToLower(ci.Name)

	// Source path indicates CI/test infrastructure.
	ciPathPatterns := []string{
		".github/", ".ci/", "ci/", ".circleci/", ".gitlab/",
		"dev/docker/", "dev/spark-test-image/",
		"-test-image/", "test/docker/", "testing/",
	}
	for _, p := range ciPathPatterns {
		if strings.Contains(src, p) {
			return true
		}
	}

	// Known CI/test purpose names (exact or prefix match).
	ciNames := []string{
		"lint", "test", "docs", "binder", "check", "build",
		"coverage", "python", "pypy",
	}
	for _, ciName := range ciNames {
		if name == ciName || strings.HasPrefix(name, ciName+"-") {
			return true
		}
	}

	return false
}

// containsAny checks if s contains any of the substrings.
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// sortStrings sorts a string slice in place.
func sortStrings(s []string) {
	sort.Strings(s)
}
