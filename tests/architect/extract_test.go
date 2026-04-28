package architect_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
	"github.com/fall-out-bug/sdp_lab/internal/architect/extract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// writeFile creates a file inside dir (creating intermediate directories) and
// writes content to it.
func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
}

// mkdir creates a directory inside dir.
func mkdir(t *testing.T, dir, rel string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, rel), 0o755))
}

// assertExtractorName is a small helper that checks the Name() method.
func assertExtractorName(t *testing.T, e architect.Extractor, want string) {
	t.Helper()
	assert.Equal(t, want, e.Name())
}

// ---------------------------------------------------------------------------
// FileTreeExtractor
// ---------------------------------------------------------------------------

func TestFileTreeExtractor_Name(t *testing.T) {
	assertExtractorName(t, extract.FileTreeExtractor{}, "filetree")
}

func TestFileTreeExtractor_CountsFilesAndDirs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main")
	writeFile(t, root, "pkg/util.go", "package pkg")
	writeFile(t, root, "pkg/sub/deep.go", "package sub")
	mkdir(t, root, "empty")

	frag, err := extract.FileTreeExtractor{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.FileTree)

	assert.Equal(t, 3, frag.FileTree.TotalFiles)
	// dirs: pkg, pkg/sub, empty = 3
	assert.Equal(t, 3, frag.FileTree.TotalDirs)
	// max depth: pkg/sub = 2
	assert.Equal(t, 2, frag.FileTree.MaxDepth)
}

func TestFileTreeExtractor_SkipsDirs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app.go", "package app")
	writeFile(t, root, ".git/HEAD", "ref")
	writeFile(t, root, "node_modules/foo/index.js", "x")
	writeFile(t, root, "vendor/lib.go", "package vendor")
	writeFile(t, root, "__pycache__/mod.pyc", "x")
	writeFile(t, root, ".sdp/state.json", "{}")

	frag, err := extract.FileTreeExtractor{}.Extract(context.Background(), root)
	require.NoError(t, err)

	// Only app.go should be counted.
	assert.Equal(t, 1, frag.FileTree.TotalFiles)
	assert.Equal(t, 0, frag.FileTree.TotalDirs)
}

func TestFileTreeExtractor_DetectsNamingPatterns(t *testing.T) {
	root := t.TempDir()
	mkdir(t, root, "controllers")
	mkdir(t, root, "services")
	writeFile(t, root, "models/user.go", "package models")
	writeFile(t, root, "handlers/api.go", "package handlers")
	writeFile(t, root, "middleware/auth.go", "package middleware")
	writeFile(t, root, "repository/repo.go", "package repository")
	writeFile(t, root, "entities/order.go", "package entities")

	frag, err := extract.FileTreeExtractor{}.Extract(context.Background(), root)
	require.NoError(t, err)

	patterns := frag.FileTree.Patterns
	assert.Contains(t, patterns, "controller")
	assert.Contains(t, patterns, "service")
	assert.Contains(t, patterns, "model")
	assert.Contains(t, patterns, "handler")
	assert.Contains(t, patterns, "middleware")
	assert.Contains(t, patterns, "repository")
	assert.Contains(t, patterns, "entity")
}

func TestFileTreeExtractor_ExtCounts(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.go", "")
	writeFile(t, root, "b.go", "")
	writeFile(t, root, "c.ts", "")

	frag, err := extract.FileTreeExtractor{}.Extract(context.Background(), root)
	require.NoError(t, err)

	assert.Equal(t, 2, frag.FileTree.ExtCounts[".go"])
	assert.Equal(t, 1, frag.FileTree.ExtCounts[".ts"])
}

// ---------------------------------------------------------------------------
// DependencyManifestParser
// ---------------------------------------------------------------------------

func TestDependencyManifestParser_Name(t *testing.T) {
	assertExtractorName(t, extract.DependencyManifestParser{}, "deps")
}

func TestDependencyManifestParser_GoMod(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", `module example.com/foo

go 1.21

require (
	github.com/stretchr/testify v1.8.0
	github.com/Shopify/sarama v1.38.0
)
`)

	frag, err := extract.DependencyManifestParser{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Dependencies, 1)

	dep := frag.Dependencies[0]
	assert.Equal(t, "go.mod", dep.File)
	assert.Equal(t, "go", dep.Language)
	assert.Equal(t, 2, dep.DepCount)
}

func TestDependencyManifestParser_PackageJSON(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{
  "name": "test",
  "dependencies": {
    "express": "^4.18.0",
    "kafkajs": "^2.0.0"
  },
  "devDependencies": {
    "jest": "^29.0.0"
  }
}`)

	frag, err := extract.DependencyManifestParser{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Dependencies, 1)

	dep := frag.Dependencies[0]
	assert.Equal(t, "package.json", dep.File)
	assert.Equal(t, "javascript", dep.Language)
	assert.Equal(t, 3, dep.DepCount)
	assert.Contains(t, dep.Signals, "event_driven")
}

func TestDependencyManifestParser_RequirementsTxt(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "requirements.txt", `flask==2.3.0
sqlalchemy>=2.0
redis>=4.0
# comment
`)

	frag, err := extract.DependencyManifestParser{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Dependencies, 1)

	dep := frag.Dependencies[0]
	assert.Equal(t, "requirements.txt", dep.File)
	assert.Equal(t, "python", dep.Language)
	assert.Equal(t, 3, dep.DepCount)
	assert.Contains(t, dep.Signals, "orm")
	assert.Contains(t, dep.Signals, "cache")
}

func TestDependencyManifestParser_PomXML(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pom.xml", `<project>
  <dependencies>
    <dependency>
      <groupId>org.apache.kafka</groupId>
      <artifactId>kafka-clients</artifactId>
    </dependency>
    <dependency>
      <groupId>org.hibernate</groupId>
      <artifactId>hibernate-core</artifactId>
    </dependency>
  </dependencies>
</project>`)

	frag, err := extract.DependencyManifestParser{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Dependencies, 1)

	dep := frag.Dependencies[0]
	assert.Equal(t, "pom.xml", dep.File)
	assert.Equal(t, "java", dep.Language)
	assert.Equal(t, 2, dep.DepCount)
	assert.Contains(t, dep.Signals, "event_driven")
	assert.Contains(t, dep.Signals, "orm")
}

func TestDependencyManifestParser_NoManifests(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "# Hello")

	frag, err := extract.DependencyManifestParser{}.Extract(context.Background(), root)
	require.NoError(t, err)
	assert.Empty(t, frag.Dependencies)
}

func TestDependencyManifestParser_MultipleManifests(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", `module x
go 1.21
require github.com/foo/bar v1.0.0
`)
	writeFile(t, root, "package.json", `{
  "dependencies": {
    "react": "^18.0.0"
  }
}`)

	frag, err := extract.DependencyManifestParser{}.Extract(context.Background(), root)
	require.NoError(t, err)
	assert.Len(t, frag.Dependencies, 2)
}

// ---------------------------------------------------------------------------
// SpecInventoryScanner
// ---------------------------------------------------------------------------

func TestSpecInventoryScanner_Name(t *testing.T) {
	assertExtractorName(t, extract.SpecInventoryScanner{}, "specs")
}

func TestSpecInventoryScanner_DetectsOpenAPI(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "openapi.yaml", "openapi: 3.0.0")

	frag, err := extract.SpecInventoryScanner{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Specs, 1)
	assert.Equal(t, "openapi", frag.Specs[0].Kind)
}

func TestSpecInventoryScanner_DetectsProto(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "api/v1/service.proto", `syntax = "proto3";`)

	frag, err := extract.SpecInventoryScanner{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Specs, 1)
	assert.Equal(t, "proto", frag.Specs[0].Kind)
}

func TestSpecInventoryScanner_DetectsDockerfile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Dockerfile", "FROM golang:1.21")
	writeFile(t, root, "docker-compose.yml", "version: '3'")

	frag, err := extract.SpecInventoryScanner{}.Extract(context.Background(), root)
	require.NoError(t, err)
	assert.Len(t, frag.Specs, 2)
	for _, s := range frag.Specs {
		assert.Equal(t, "docker", s.Kind)
	}
}

func TestSpecInventoryScanner_DetectsTerraform(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "infra/main.tf", `provider "aws" {}`)

	frag, err := extract.SpecInventoryScanner{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Specs, 1)
	assert.Equal(t, "terraform", frag.Specs[0].Kind)
}

func TestSpecInventoryScanner_DetectsCI(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".github/workflows/ci.yml", "name: CI")

	frag, err := extract.SpecInventoryScanner{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Specs, 1)
	assert.Equal(t, "ci", frag.Specs[0].Kind)
}

func TestSpecInventoryScanner_DetectsMigrations(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "migrations/001_init.sql", "CREATE TABLE x;")
	writeFile(t, root, "migrations/002_add_col.sql", "ALTER TABLE x;")

	frag, err := extract.SpecInventoryScanner{}.Extract(context.Background(), root)
	require.NoError(t, err)
	assert.Len(t, frag.Specs, 2)
	for _, s := range frag.Specs {
		assert.Equal(t, "migration", s.Kind)
	}
}

func TestSpecInventoryScanner_DetectsGraphQL(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "schema.graphql", "type Query { hello: String }")

	frag, err := extract.SpecInventoryScanner{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Specs, 1)
	assert.Equal(t, "graphql", frag.Specs[0].Kind)
}

func TestSpecInventoryScanner_SkipsDotGit(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".git/hooks/pre-commit.proto", "#!/bin/sh")
	writeFile(t, root, "api.proto", `syntax = "proto3";`)

	frag, err := extract.SpecInventoryScanner{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Specs, 1)
	assert.Equal(t, "api.proto", frag.Specs[0].Path)
}

func TestSpecInventoryScanner_NoSpecs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main")

	frag, err := extract.SpecInventoryScanner{}.Extract(context.Background(), root)
	require.NoError(t, err)
	assert.Empty(t, frag.Specs)
}

// ---------------------------------------------------------------------------
// GeneratedCodeDetector
// ---------------------------------------------------------------------------

func TestGeneratedCodeDetector_Name(t *testing.T) {
	assertExtractorName(t, extract.GeneratedCodeDetector{}, "generated")
}

func TestGeneratedCodeDetector_HeaderMarker_CodeGenerated(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "gen.go", "// Code generated by foo. DO NOT EDIT.\npackage gen\n")

	frag, err := extract.GeneratedCodeDetector{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Generated, 1)
	assert.Equal(t, "gen.go", frag.Generated[0].Path)
	assert.Contains(t, frag.Generated[0].Reason, "header:")
}

func TestGeneratedCodeDetector_HeaderMarker_DoNotEdit(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "types.go", "// DO NOT EDIT - auto-generated\npackage types\n")

	frag, err := extract.GeneratedCodeDetector{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Generated, 1)
	assert.Contains(t, frag.Generated[0].Reason, "DO NOT EDIT")
}

func TestGeneratedCodeDetector_HeaderMarker_AtGenerated(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Foo.java", "// @Generated\npublic class Foo {}\n")

	frag, err := extract.GeneratedCodeDetector{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Generated, 1)
	assert.Contains(t, frag.Generated[0].Reason, "@Generated")
}

func TestGeneratedCodeDetector_FilenamePattern_PbGo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "api/service.pb.go", "package api\n")

	frag, err := extract.GeneratedCodeDetector{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Generated, 1)
	assert.Equal(t, "protobuf_generated", frag.Generated[0].Reason)
}

func TestGeneratedCodeDetector_FilenamePattern_GenGo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "models.gen.go", "package main\n")

	frag, err := extract.GeneratedCodeDetector{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Generated, 1)
	assert.Equal(t, "codegen_generated", frag.Generated[0].Reason)
}

func TestGeneratedCodeDetector_DirectoryPattern(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "__generated__/schema.ts", "export type Foo = {};")

	frag, err := extract.GeneratedCodeDetector{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, frag.Generated, 1)
	assert.Contains(t, frag.Generated[0].Reason, "__generated__")
}

func TestGeneratedCodeDetector_NoGenerated(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n\nfunc main() {}\n")

	frag, err := extract.GeneratedCodeDetector{}.Extract(context.Background(), root)
	require.NoError(t, err)
	assert.Empty(t, frag.Generated)
}

func TestGeneratedCodeDetector_SkipsDotGit(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".git/objects/pack/pack.go", "// Code generated by git\n")
	writeFile(t, root, "real.go", "package main\n")

	frag, err := extract.GeneratedCodeDetector{}.Extract(context.Background(), root)
	require.NoError(t, err)
	assert.Empty(t, frag.Generated)
}

func TestGeneratedCodeDetector_MarkerBeyondLine5(t *testing.T) {
	root := t.TempDir()
	// Marker on line 7 - should NOT be detected.
	content := "line1\nline2\nline3\nline4\nline5\nline6\n// Code generated by foo\n"
	writeFile(t, root, "late.go", content)

	frag, err := extract.GeneratedCodeDetector{}.Extract(context.Background(), root)
	require.NoError(t, err)
	assert.Empty(t, frag.Generated)
}

// ---------------------------------------------------------------------------
// Interface compliance
// ---------------------------------------------------------------------------

func TestExtractorsImplementInterface(t *testing.T) {
	extractors := []architect.Extractor{
		extract.FileTreeExtractor{},
		extract.DependencyManifestParser{},
		extract.SpecInventoryScanner{},
		extract.GeneratedCodeDetector{},
	}
	for _, e := range extractors {
		assert.NotEmpty(t, e.Name())
	}
}
