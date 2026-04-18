package extract

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"

	"sdp_dev/internal/architect"
)

// specGlob defines a glob pattern and the kind label assigned to matches.
type specGlob struct {
	Pattern string
	Kind    string
}

// specGlobs lists the patterns we scan for.  Patterns are matched against
// paths relative to the repo root.
var specGlobs = []specGlob{
	// OpenAPI / Swagger
	{Pattern: "openapi.yaml", Kind: "openapi"},
	{Pattern: "openapi.yml", Kind: "openapi"},
	{Pattern: "openapi.json", Kind: "openapi"},
	{Pattern: "swagger.yaml", Kind: "openapi"},
	{Pattern: "swagger.yml", Kind: "openapi"},
	{Pattern: "swagger.json", Kind: "openapi"},

	// AsyncAPI
	{Pattern: "asyncapi.yaml", Kind: "asyncapi"},
	{Pattern: "asyncapi.yml", Kind: "asyncapi"},
	{Pattern: "asyncapi.json", Kind: "asyncapi"},

	// Protocol Buffers
	{Pattern: "*.proto", Kind: "proto"},

	// GraphQL
	{Pattern: "*.graphql", Kind: "graphql"},
	{Pattern: "*.gql", Kind: "graphql"},

	// Architecture Decision Records
	{Pattern: "adr/*.md", Kind: "adr"},
	{Pattern: "docs/adr/*.md", Kind: "adr"},
	{Pattern: "doc/adr/*.md", Kind: "adr"},
	{Pattern: "ADR-*.md", Kind: "adr"},

	// Docker
	{Pattern: "Dockerfile", Kind: "docker"},
	{Pattern: "Dockerfile.*", Kind: "docker"},
	{Pattern: "docker-compose*.yml", Kind: "docker"},
	{Pattern: "docker-compose*.yaml", Kind: "docker"},

	// Terraform
	{Pattern: "*.tf", Kind: "terraform"},

	// CI
	{Pattern: ".github/workflows/*.yml", Kind: "ci"},
	{Pattern: ".github/workflows/*.yaml", Kind: "ci"},
	{Pattern: ".gitlab-ci.yml", Kind: "ci"},
	{Pattern: "Jenkinsfile", Kind: "ci"},
	{Pattern: ".circleci/config.yml", Kind: "ci"},

	// Kubernetes
	{Pattern: "k8s/*.yaml", Kind: "k8s"},
	{Pattern: "k8s/*.yml", Kind: "k8s"},
	{Pattern: "kubernetes/*.yaml", Kind: "k8s"},
	{Pattern: "kubernetes/*.yml", Kind: "k8s"},
	{Pattern: "deploy/*.yaml", Kind: "k8s"},
	{Pattern: "deploy/*.yml", Kind: "k8s"},

	// Migrations
	{Pattern: "migrations/*.sql", Kind: "migration"},
	{Pattern: "db/migrations/*.sql", Kind: "migration"},
	{Pattern: "migrate/*.sql", Kind: "migration"},
}

// SpecInventoryScanner globs for well-known specification and infrastructure
// files and returns each as a SpecArtifact.
type SpecInventoryScanner struct{}

// Name implements architect.Extractor.
func (SpecInventoryScanner) Name() string { return "specs" }

// Extract implements architect.Extractor.
func (SpecInventoryScanner) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
	seen := make(map[string]bool)
	var specs []architect.SpecArtifact

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() {
			if skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}

		rel, _ := filepath.Rel(repoRoot, path)
		// Normalise to forward slashes for consistent matching.
		relSlash := filepath.ToSlash(rel)

		for _, sg := range specGlobs {
			matched, _ := matchSpecGlob(relSlash, sg.Pattern)
			if matched && !seen[rel] {
				seen[rel] = true
				specs = append(specs, architect.SpecArtifact{
					Path: rel,
					Kind: sg.Kind,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &architect.ProfileFragment{
		Specs: specs,
	}, nil
}

// matchSpecGlob matches relPath (forward-slash normalised) against a pattern.
// The pattern may contain a directory prefix (e.g. "k8s/*.yaml"), a basename
// with wildcard (e.g. "*.proto"), or a literal name.
func matchSpecGlob(relPath, pattern string) (bool, error) {
	// If the pattern contains a slash it's path-qualified: match the full
	// relative path.
	if strings.Contains(pattern, "/") {
		return filepath.Match(pattern, relPath)
	}

	// Otherwise match against the file basename only.
	base := filepath.Base(relPath)
	return filepath.Match(pattern, base)
}
