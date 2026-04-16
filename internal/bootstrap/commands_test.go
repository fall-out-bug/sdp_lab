package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectBuildCommands_GoMakefile(t *testing.T) {
	dir := t.TempDir()
	mf := `build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Makefile"), []byte(mf), 0o644))

	cmds := DetectBuildCommands(dir, "go")
	assert.Equal(t, "make build", cmds.Build)
	assert.Equal(t, "make test", cmds.Test)
	assert.Equal(t, "make lint", cmds.Lint)
}

func TestDetectBuildCommands_MakefilePartial(t *testing.T) {
	dir := t.TempDir()
	mf := `build:
	go build ./...

test:
	go test ./...
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Makefile"), []byte(mf), 0o644))

	cmds := DetectBuildCommands(dir, "go")
	assert.Equal(t, "make build", cmds.Build)
	assert.Equal(t, "make test", cmds.Test)
	assert.Equal(t, "", cmds.Lint) // not in Makefile, no fallback since Makefile was found
}

func TestDetectBuildCommands_PackageJSON(t *testing.T) {
	dir := t.TempDir()
	pkg := `{
  "name": "my-app",
  "scripts": {
    "build": "tsc",
    "test": "jest",
    "lint": "eslint ."
  }
}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644))

	cmds := DetectBuildCommands(dir, "javascript")
	assert.Equal(t, "npm run build", cmds.Build)
	assert.Equal(t, "npm test", cmds.Test)
	assert.Equal(t, "npm run lint", cmds.Lint)
}

func TestDetectBuildCommands_PackageJSONPartial(t *testing.T) {
	dir := t.TempDir()
	pkg := `{
  "name": "my-app",
  "scripts": {
    "build": "tsc",
    "test": "jest"
  }
}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644))

	cmds := DetectBuildCommands(dir, "javascript")
	assert.Equal(t, "npm run build", cmds.Build)
	assert.Equal(t, "npm test", cmds.Test)
	assert.Equal(t, "", cmds.Lint)
}

func TestDetectBuildCommands_GitHubActions(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0o755))

	ci := `name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: go build ./...
      - run: go test ./...
      - run: golangci-lint run
`
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".github", "workflows", "ci.yml"), []byte(ci), 0o644))

	cmds := DetectBuildCommands(dir, "go")
	assert.Equal(t, "go build ./...", cmds.Build)
	assert.Equal(t, "go test ./...", cmds.Test)
	assert.Equal(t, "golangci-lint run", cmds.Lint)
}

func TestDetectBuildCommands_GitLabCI(t *testing.T) {
	dir := t.TempDir()
	ci := `stages:
  - build
  - test
build:
  script: go build ./...
test:
  script: go test ./...
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitlab-ci.yml"), []byte(ci), 0o644))

	cmds := DetectBuildCommands(dir, "go")
	assert.Equal(t, "go build ./...", cmds.Build)
	assert.Equal(t, "go test ./...", cmds.Test)
}

func TestDetectBuildCommands_DefaultGo(t *testing.T) {
	dir := t.TempDir()
	cmds := DetectBuildCommands(dir, "go")
	assert.Equal(t, "go build ./...", cmds.Build)
	assert.Equal(t, "go test ./...", cmds.Test)
	assert.Equal(t, "golangci-lint run", cmds.Lint)
}

func TestDetectBuildCommands_DefaultRust(t *testing.T) {
	dir := t.TempDir()
	cmds := DetectBuildCommands(dir, "rust")
	assert.Equal(t, "cargo build", cmds.Build)
	assert.Equal(t, "cargo test", cmds.Test)
	assert.Equal(t, "cargo clippy", cmds.Lint)
}

func TestDetectBuildCommands_DefaultNode(t *testing.T) {
	dir := t.TempDir()
	cmds := DetectBuildCommands(dir, "javascript")
	assert.Equal(t, "npm run build", cmds.Build)
	assert.Equal(t, "npm test", cmds.Test)
	assert.Equal(t, "npm run lint", cmds.Lint)
}

func TestDetectBuildCommands_DefaultPython(t *testing.T) {
	dir := t.TempDir()
	cmds := DetectBuildCommands(dir, "python")
	assert.Equal(t, "python -m build", cmds.Build)
	assert.Equal(t, "pytest", cmds.Test)
	assert.Equal(t, "ruff check", cmds.Lint)
}

func TestDetectBuildCommands_DefaultUnknown(t *testing.T) {
	dir := t.TempDir()
	cmds := DetectBuildCommands(dir, "unknown-lang")
	assert.Equal(t, "make build", cmds.Build)
	assert.Equal(t, "make test", cmds.Test)
	assert.Equal(t, "make lint", cmds.Lint)
}

func TestDetectBuildCommands_MakefilePriority(t *testing.T) {
	dir := t.TempDir()

	// Both Makefile and package.json exist — Makefile should win.
	mf := `build:
	make build
test:
	make test
lint:
	make lint
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Makefile"), []byte(mf), 0o644))

	pkg := `{"scripts": {"build": "tsc", "test": "jest", "lint": "eslint"}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644))

	cmds := DetectBuildCommands(dir, "javascript")
	assert.Equal(t, "make build", cmds.Build)
	assert.Equal(t, "make test", cmds.Test)
	assert.Equal(t, "make lint", cmds.Lint)
}

func TestDetectBuildCommands_PackageJSONPriorityOverCI(t *testing.T) {
	dir := t.TempDir()

	// package.json but no Makefile — package.json wins over CI.
	pkg := `{"scripts": {"build": "tsc", "test": "jest"}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644))

	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0o755))
	ci := `jobs: { build: { script: npm run build } }`
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".github", "workflows", "ci.yml"), []byte(ci), 0o644))

	cmds := DetectBuildCommands(dir, "javascript")
	assert.Equal(t, "npm run build", cmds.Build)
	assert.Equal(t, "npm test", cmds.Test)
}
