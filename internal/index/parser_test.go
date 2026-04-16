package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGoFile(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "example.go")
	content := `package example

import "fmt"

// Config holds configuration.
type Config struct {
	Name string
	Port int
}

func NewConfig() *Config {
	return &Config{Name: "default", Port: 8080}
}

func (c *Config) Validate() error {
	if c.Port < 0 {
		return fmt.Errorf("invalid port")
	}
	return nil
}

const DefaultTimeout = 30
`
	require.NoError(t, os.WriteFile(goFile, []byte(content), 0o644))

	chunks, _, err := ParseFile(goFile, "go")
	require.NoError(t, err)

	// Should find: type Config, func NewConfig, method Validate, const DefaultTimeout
	assert.GreaterOrEqual(t, len(chunks), 3, "should extract at least 3 chunks from Go file")

	// Check that we got specific kinds
	kinds := map[string]bool{}
	for _, c := range chunks {
		kinds[c.Kind] = true
		assert.Equal(t, "go", c.Language)
		assert.Equal(t, "example.go", filepath.Base(c.FilePath))
		assert.GreaterOrEqual(t, c.LineEnd, c.LineStart)
		assert.NotEmpty(t, c.Content)
		assert.NotEmpty(t, c.Hash)
	}

	// Should have at least function and type chunks
	assert.True(t, kinds["function"] || kinds["method"], "should have function or method chunks")
}

func TestParsePythonFile(t *testing.T) {
	dir := t.TempDir()
	pyFile := filepath.Join(dir, "service.py")
	content := `"""Service module."""

import os

class Service:
    """A service class."""

    def __init__(self, name):
        self.name = name

    def process(self, data):
        """Process data."""
        return data.upper()

def helper():
    return True

CONSTANT = 42
`
	require.NoError(t, os.WriteFile(pyFile, []byte(content), 0o644))

	chunks, _, err := ParseFile(pyFile, "python")
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(chunks), 2, "should extract chunks from Python file")

	for _, c := range chunks {
		assert.Equal(t, "python", c.Language)
		assert.NotEmpty(t, c.Hash)
	}
}

func TestParseTypeScriptFile(t *testing.T) {
	dir := t.TempDir()
	tsFile := filepath.Join(dir, "router.ts")
	content := `export interface Route {
  path: string;
  handler: () => void;
}

export function createRouter(): Route[] {
  return [];
}

export class Router {
  private routes: Route[] = [];

  add(route: Route): void {
    this.routes.push(route);
  }
}
`
	require.NoError(t, os.WriteFile(tsFile, []byte(content), 0o644))

	chunks, _, err := ParseFile(tsFile, "typescript")
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(chunks), 2, "should extract chunks from TypeScript file")

	for _, c := range chunks {
		assert.Equal(t, "typescript", c.Language)
	}
}

func TestParseUnknownLanguage(t *testing.T) {
	dir := t.TempDir()
	txtFile := filepath.Join(dir, "notes.txt")
	content := "This is just some text.\nLine 2.\n"
	require.NoError(t, os.WriteFile(txtFile, []byte(content), 0o644))

	chunks, _, err := ParseFile(txtFile, "")
	require.NoError(t, err)
	// Should still produce a file-level chunk
	assert.GreaterOrEqual(t, len(chunks), 1)
	assert.Equal(t, "file", chunks[0].Kind)
}

func TestParseEmptyFile(t *testing.T) {
	dir := t.TempDir()
	emptyFile := filepath.Join(dir, "empty.go")
	require.NoError(t, os.WriteFile(emptyFile, []byte(""), 0o644))

	chunks, _, err := ParseFile(emptyFile, "go")
	require.NoError(t, err)
	_ = chunks // Empty file may produce no chunks or a single file chunk
}

func TestParseFile_HashStability(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "stable.go")
	content := `package stable

func Hello() string {
	return "world"
}
`
	require.NoError(t, os.WriteFile(goFile, []byte(content), 0o644))

	chunks1, _, err := ParseFile(goFile, "go")
	require.NoError(t, err)

	chunks2, _, err := ParseFile(goFile, "go")
	require.NoError(t, err)

	require.Equal(t, len(chunks1), len(chunks2), "same file should produce same number of chunks")
	for i := range chunks1 {
		assert.Equal(t, chunks1[i].Hash, chunks2[i].Hash, "hashes should be stable across parses")
	}
}

func TestParseGoFile_ExtractionDetail(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "detail.go")
	content := `package detail

import "errors"

var ErrNotFound = errors.New("not found")

type Handler struct {
	Name string
}

func NewHandler(name string) *Handler {
	return &Handler{Name: name}
}

func (h *Handler) Serve() error {
	return nil
}
`
	require.NoError(t, os.WriteFile(goFile, []byte(content), 0o644))

	chunks, _, err := ParseFile(goFile, "go")
	require.NoError(t, err)

	// Check symbol names
	names := map[string]string{} // name -> kind
	for _, c := range chunks {
		if c.SymbolName != "" {
			names[c.SymbolName] = c.Kind
		}
	}

	// Should find at least the function and method
	assert.Contains(t, names, "NewHandler")
	assert.Contains(t, names, "Handler.Serve")
}

func TestParseGoFile_SecretsExcluded(t *testing.T) {
	dir := t.TempDir()
	// Simulate a secrets file that happens to be parseable Go
	secretFile := filepath.Join(dir, "credentials.go")
	content := `package main

var DBPassword = "super-secret"
`
	require.NoError(t, os.WriteFile(secretFile, []byte(content), 0o644))

	// The parser should still parse the file (exclusion is at builder level),
	// but the IsSecretFile function should detect it
	assert.True(t, IsSecretFile(secretFile))
}

func TestParseRustFile(t *testing.T) {
	dir := t.TempDir()
	rsFile := filepath.Join(dir, "main.rs")
	content := `use std::io;

fn main() {
    println!("hello");
}

struct Config {
    name: String,
}

impl Config {
    fn new() -> Self {
        Config { name: "default".into() }
    }
}
`
	require.NoError(t, os.WriteFile(rsFile, []byte(content), 0o644))

	chunks, _, err := ParseFile(rsFile, "rust")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(chunks), 2, "should extract chunks from Rust file")
}

func TestParseJavaFile(t *testing.T) {
	dir := t.TempDir()
	javaFile := filepath.Join(dir, "Service.java")
	content := `package com.example;

public class Service {
    private String name;

    public Service(String name) {
        this.name = name;
    }

    public String process(String input) {
        return input.toUpperCase();
    }
}
`
	require.NoError(t, os.WriteFile(javaFile, []byte(content), 0o644))

	chunks, _, err := ParseFile(javaFile, "java")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(chunks), 1, "should extract at least the class from Java file")

	// Verify we got the class
	for _, c := range chunks {
		assert.Equal(t, "java", c.Language)
		assert.NotEmpty(t, c.Hash)
	}
}
