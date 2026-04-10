package architect

import (
	"path/filepath"
	"strings"
)

// CrossLangDep represents a dependency between services in different languages.
type CrossLangDep struct {
	FromContainer string  `json:"from_container"`
	FromLanguage  string  `json:"from_language"`
	ToContainer   string  `json:"to_container"`
	ToLanguage    string  `json:"to_language"`
	BridgeType    string  `json:"bridge_type"` // "openapi", "protobuf", "json_schema", "orm_sql", "shared_config"
	BridgePath    string  `json:"bridge_path"`
	Confidence    float64 `json:"confidence"`
}

// CrossLangResult holds the results of cross-language dependency analysis.
type CrossLangResult struct {
	Dependencies []CrossLangDep `json:"dependencies,omitempty"`
	SharedSpecs  []SharedSpec   `json:"shared_specs,omitempty"`
}

// SharedSpec represents a specification shared between multiple services.
type SharedSpec struct {
	Path         string   `json:"path"`
	Kind         string   `json:"kind"` // "openapi", "protobuf", "json_schema", "graphql"
	ReferencedBy []string `json:"referenced_by"` // container names
}

// DetectCrossLangDeps analyzes the profile to find cross-language dependencies.
func DetectCrossLangDeps(profile *CodebaseProfile) *CrossLangResult {
	result := &CrossLangResult{
		Dependencies: []CrossLangDep{},
		SharedSpecs:  []SharedSpec{},
	}

	// Detect shared specs
	specDeps := detectSharedSpecs(profile)
	result.Dependencies = append(result.Dependencies, specDeps...)

	// Detect ORM-SQL mappings
	ormDeps := detectORMSQLMapping(profile)
	result.Dependencies = append(result.Dependencies, ormDeps...)

	// Detect shared configs
	result.SharedSpecs = detectSharedConfig(profile)

	return result
}

// detectSharedSpecs finds OpenAPI/Proto/GraphQL specs referenced by multiple containers.
func detectSharedSpecs(profile *CodebaseProfile) []CrossLangDep {
	var deps []CrossLangDep

	// Build a map of spec path to referencing containers
	specRefs := make(map[string][]string)
	for _, spec := range profile.Specs {
		// Only consider cross-language contract types
		if spec.Kind != "openapi" && spec.Kind != "protobuf" && spec.Kind != "graphql" && spec.Kind != "asyncapi" {
			continue
		}

		// Find which containers reference this spec
		var refContainers []string
		for _, container := range profile.Infra.Containers {
			if container.Source == "" {
				continue
			}
			containerDir := filepath.Dir(container.Source)
			// Check if spec is within or near the container's source directory
			if strings.HasPrefix(spec.Path, containerDir) ||
				strings.Contains(filepath.Dir(spec.Path), containerDir) ||
				isAdjacentPath(spec.Path, containerDir) {
				refContainers = append(refContainers, container.Name)
			}
		}
		specRefs[spec.Path] = refContainers
	}

	// Create cross-lang deps for specs referenced by containers in different languages
	for specPath, containers := range specRefs {
		if len(containers) < 2 {
			continue
		}

		// Find the spec kind
		var specKind string
		for _, spec := range profile.Specs {
			if spec.Path == specPath {
				specKind = spec.Kind
				break
			}
		}

		// Check all pairs of containers for cross-language dependencies
		for i := 0; i < len(containers); i++ {
			for j := i + 1; j < len(containers); j++ {
				fromContainer := containers[i]
				toContainer := containers[j]

				fromLang := ContainerLanguage(profile, fromContainer)
				toLang := ContainerLanguage(profile, toContainer)

				// Only create dep if languages differ
				if fromLang != "" && toLang != "" && fromLang != toLang {
					deps = append(deps, CrossLangDep{
						FromContainer: fromContainer,
						FromLanguage:  fromLang,
						ToContainer:   toContainer,
						ToLanguage:    toLang,
						BridgeType:    specKind,
						BridgePath:    specPath,
						Confidence:    1.0,
					})
				}
			}
		}
	}

	return deps
}

// detectORMSQLMapping finds ORM models that map to SQL migrations.
func detectORMSQLMapping(profile *CodebaseProfile) []CrossLangDep {
	var deps []CrossLangDep

	if profile.SQLAnalysis == nil || len(profile.SQLAnalysis.ORMModels) == 0 {
		return deps
	}

	// Map ORM models to their containers
	ormContainers := make(map[string]string) // file -> container name
	for _, container := range profile.Infra.Containers {
		if container.Source == "" {
			continue
		}
		containerDir := filepath.Dir(container.Source)
		for _, orm := range profile.SQLAnalysis.ORMModels {
			if strings.HasPrefix(orm.File, containerDir) {
				ormContainers[orm.File] = container.Name
			}
		}
	}

	// Find SQL migration files and their containers
	sqlContainers := make(map[string]string) // file -> container name
	if profile.SQLAnalysis.Migrations != nil {
		migrationDir := profile.SQLAnalysis.Migrations.Dir
		for _, container := range profile.Infra.Containers {
			if container.Source == "" {
				continue
			}
			containerDir := filepath.Dir(container.Source)
			// Check if migrations are in this container's directory
			if strings.HasPrefix(migrationDir, containerDir) {
				sqlContainers[migrationDir] = container.Name
			}
		}
	}

	// Create cross-lang deps when ORM and SQL are in different containers
	for ormFile, ormContainer := range ormContainers {
		for sqlDir, sqlContainer := range sqlContainers {
			// Skip if same container
			if ormContainer == sqlContainer {
				continue
			}

			// Check if ORM file references SQL dir (they're related)
			if strings.Contains(filepath.Dir(ormFile), sqlDir) ||
				strings.Contains(sqlDir, filepath.Dir(ormFile)) ||
				isAdjacentPath(ormFile, sqlDir) {

				fromLang := ContainerLanguage(profile, ormContainer)
				toLang := ContainerLanguage(profile, sqlContainer)

				if fromLang != "" && toLang != "" && fromLang != toLang {
					deps = append(deps, CrossLangDep{
						FromContainer: ormContainer,
						FromLanguage:  fromLang,
						ToContainer:   sqlContainer,
						ToLanguage:    toLang,
						BridgeType:    "orm_sql",
						BridgePath:    ormFile + " -> " + sqlDir,
						Confidence:    0.7,
					})
				}
			}
		}
	}

	return deps
}

// detectSharedConfig finds specs that appear in multiple container source trees.
func detectSharedConfig(profile *CodebaseProfile) []SharedSpec {
	var sharedSpecs []SharedSpec

	// Group specs by path
	specContainers := make(map[string][]string)
	for _, spec := range profile.Specs {
		// Find which containers reference this spec
		var refContainers []string
		for _, container := range profile.Infra.Containers {
			if container.Source == "" {
				continue
			}
			containerDir := filepath.Dir(container.Source)
			if strings.HasPrefix(spec.Path, containerDir) ||
				strings.Contains(filepath.Dir(spec.Path), containerDir) ||
				isAdjacentPath(spec.Path, containerDir) {
				refContainers = append(refContainers, container.Name)
			}
		}
		if len(refContainers) > 0 {
			specContainers[spec.Path] = refContainers
		}
	}

	// Only include specs referenced by multiple containers
	for path, containers := range specContainers {
		if len(containers) >= 2 {
			// Find spec kind
			var kind string
			for _, spec := range profile.Specs {
				if spec.Path == path {
					kind = spec.Kind
					break
				}
			}
			sharedSpecs = append(sharedSpecs, SharedSpec{
				Path:         path,
				Kind:         kind,
				ReferencedBy: containers,
			})
		}
	}

	return sharedSpecs
}

// ContainerLanguage detects the language from manifest files in the container's directory.
func ContainerLanguage(profile *CodebaseProfile, containerName string) string {
	// Find the container
	var container *ContainerInfo
	for i := range profile.Infra.Containers {
		if profile.Infra.Containers[i].Name == containerName {
			container = &profile.Infra.Containers[i]
			break
		}
	}
	if container == nil || container.Source == "" {
		return ""
	}

	containerDir := filepath.Dir(container.Source)

	// Check manifest files to determine language
	for _, manifest := range profile.Dependencies.Manifests {
		manifestDir := filepath.Dir(manifest.Path)

		// Check if manifest is in this container's directory
		if strings.HasPrefix(manifestDir, containerDir) || strings.HasPrefix(containerDir, manifestDir) {
			// Use the language from the manifest
			if manifest.Language != "" {
				return normalizeLanguage(manifest.Language)
			}

			// Fallback: detect from filename
			base := filepath.Base(manifest.Path)
			switch base {
			case "go.mod":
				return "go"
			case "package.json":
				return "typescript"
			case "requirements.txt", "pyproject.toml", "setup.py":
				return "python"
			case "pom.xml", "build.gradle", "build.gradle.kts":
				return "java"
			case "Cargo.toml":
				return "rust"
			case "composer.json":
				return "php"
			case "Gemfile":
				return "ruby"
			case "csproj", "fsproj":
				return "c#"
			}
		}
	}

	// Additional fallback: check file extensions in the directory
	for ext, count := range profile.FileTree.ExtCounts {
		if count == 0 {
			continue
		}
		switch ext {
		case ".go":
			return "go"
		case ".ts", ".tsx":
			return "typescript"
		case ".js", ".jsx":
			return "javascript"
		case ".py":
			return "python"
		case ".java":
			return "java"
		case ".rs":
			return "rust"
		}
	}

	return ""
}

// normalizeLanguage normalizes language names to a standard format.
func normalizeLanguage(lang string) string {
	lang = strings.ToLower(lang)
	lang = strings.TrimSpace(lang)

	switch lang {
	case "golang", "go":
		return "go"
	case "typescript", "ts", "tsx":
		return "typescript"
	case "javascript", "js", "jsx":
		return "javascript"
	case "python", "py":
		return "python"
	case "java", "kotlin": // Treat Kotlin as Java for cross-lang purposes
		return "java"
	case "rust", "rs":
		return "rust"
	case "ruby":
		return "ruby"
	case "php":
		return "php"
	case "c#", "csharp", ".net":
		return "c#"
	case "c++", "cpp":
		return "c++"
	default:
		return lang
	}
}

// isAdjacentPath checks if two paths are in adjacent directories (siblings or close relatives).
func isAdjacentPath(path1, path2 string) bool {
	dir1 := filepath.Dir(path1)
	dir2 := filepath.Dir(path2)

	// Check if they're siblings (same parent)
	parent1 := filepath.Dir(dir1)
	parent2 := filepath.Dir(dir2)
	if parent1 == parent2 && parent1 != "" && parent1 != "." {
		return true
	}

	// Check if one is directly under the other
	if strings.HasPrefix(dir1, dir2) || strings.HasPrefix(dir2, dir1) {
		return true
	}

	return false
}

// AddCrossLangEdges adds CrossLangDep relationships to the ReferenceModel as C4Relationships.
func AddCrossLangEdges(model *ReferenceModel, result *CrossLangResult) *ReferenceModel {
	if result == nil || len(result.Dependencies) == 0 {
		return model
	}

	for _, dep := range result.Dependencies {
		rel := C4Relationship{
			From:        dep.FromContainer,
			To:          dep.ToContainer,
			Description: "Cross-language dependency via " + dep.BridgeType,
			Type:        "cross_lang",
			Contract:    dep.BridgePath,
		}
		model.Relationships = append(model.Relationships, rel)
	}

	return model
}
