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
	// Test 1: credentials.go is a legitimate source file, NOT a secret
	// (Bug 683 fix: overly broad substring matching)
	legitimateFile := filepath.Join(dir, "credentials.go")
	content := `package main

var DBPassword = "super-secret"
`
	require.NoError(t, os.WriteFile(legitimateFile, []byte(content), 0o644))
	assert.False(t, IsSecretFile(legitimateFile), "credentials.go should NOT be classified as a secret")

	// Test 2: credentials.json IS a secret file
	secretFile := filepath.Join(dir, "credentials.json")
	require.NoError(t, os.WriteFile(secretFile, []byte(`{"password": "secret"}`), 0o644))
	assert.True(t, IsSecretFile(secretFile), "credentials.json should be classified as a secret")

	// The parser should still parse both files (exclusion is at builder level)
	chunks, _, err := ParseFile(legitimateFile, "go")
	require.NoError(t, err)
	assert.NotEmpty(t, chunks, "parser should parse credentials.go")
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

// TestIsSecretFile_Bug683_OverlyBroadSubstringMatching tests that legitimate
// source files with "token" or "password" in their names are NOT classified as secrets.
// This fixes Bug sdplab-683.
func TestIsSecretFile_Bug683_OverlyBroadSubstringMatching(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		isSecret bool
	}{
		// Legitimate source files that should NOT be classified as secrets
		{name: "tokenizer.py", path: "tokenizer.py", isSecret: false},
		{name: "token_handler.go", path: "token_handler.go", isSecret: false},
		{name: "tokenize.rs", path: "tokenize.rs", isSecret: false},
		{name: "password_validator.go", path: "password_validator.go", isSecret: false},
		{name: "password_utils.ts", path: "password_utils.ts", isSecret: false},
		{name: "token_manager.go", path: "token_manager.go", isSecret: false},
		{name: "auth_token.go", path: "auth_token.go", isSecret: false},
		{name: "secret_scanner.go", path: "secret_scanner.go", isSecret: false},
		{name: "secrets_manager.go", path: "secrets_manager.go", isSecret: false},

		// Actual secret files that SHOULD be classified as secrets
		{name: ".env", path: ".env", isSecret: true},
		{name: ".env.local", path: ".env.local", isSecret: true},
		{name: ".env.production", path: ".env.production", isSecret: true},
		{name: "credentials.json", path: "credentials.json", isSecret: true},
		{name: "credentials.yaml", path: "credentials.yaml", isSecret: true},
		{name: "secret.env", path: "secret.env", isSecret: true},
		{name: "secret.config", path: "secret.config", isSecret: true},
		{name: "password.txt", path: "password.txt", isSecret: true},
		{name: "password.file", path: "password.file", isSecret: true},
		{name: "config.secret", path: "config.secret", isSecret: true},
		{name: "db.password", path: "db.password", isSecret: true},
		{name: "service.credentials", path: "service.credentials", isSecret: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSecretFile(tt.path)
			assert.Equal(t, tt.isSecret, result, "IsSecretFile(%q) = %v, want %v", tt.path, result, tt.isSecret)
		})
	}
}

// TestIsSecretFile_BugAuo_IncompletePatternCoverage tests that all important
// secret file patterns are detected.
// This fixes Bug sdplab-auo.
func TestIsSecretFile_BugAuo_IncompletePatternCoverage(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		isSecret bool
	}{
		// New secret patterns that should be detected
		{name: "keystore.jks", path: "keystore.jks", isSecret: true},
		{name: "app.jks", path: "app.jks", isSecret: true},
		{name: "config.keystore", path: "config.keystore", isSecret: true},
		{name: "id_rsa", path: "id_rsa", isSecret: true},
		{name: "id_rsa.pub", path: "id_rsa.pub", isSecret: true},
		{name: "id_ed25519", path: "id_ed25519", isSecret: true},
		{name: "id_ed25519.pub", path: "id_ed25519.pub", isSecret: true},
		{name: "id_dsa", path: "id_dsa", isSecret: true},
		{name: "id_ecdsa", path: "id_ecdsa", isSecret: true},
		{name: ".netrc", path: ".netrc", isSecret: true},
		{name: ".kubeconfig", path: ".kubeconfig", isSecret: true},
		{name: "config.kubeconfig", path: "config.kubeconfig", isSecret: true},

		// Existing secret patterns (regression test)
		{name: "cert.pem", path: "cert.pem", isSecret: true},
		{name: "private.key", path: "private.key", isSecret: true},
		{name: "cert.p12", path: "cert.p12", isSecret: true},
		{name: "cert.pfx", path: "cert.pfx", isSecret: true},

		// Legitimate files that should NOT be detected as secrets
		{name: "jks_utils.go", path: "jks_utils.go", isSecret: false},
		{name: "keystore_manager.go", path: "keystore_manager.go", isSecret: false},
		{name: "rsa_wrapper.go", path: "rsa_wrapper.go", isSecret: false},
		{name: "ed25519_test.go", path: "ed25519_test.go", isSecret: false},
		{name: "netrc_reader.go", path: "netrc_reader.go", isSecret: false},
		{name: "kubeconfig_parser.go", path: "kubeconfig_parser.go", isSecret: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSecretFile(tt.path)
			assert.Equal(t, tt.isSecret, result, "IsSecretFile(%q) = %v, want %v", tt.path, result, tt.isSecret)
		})
	}
}

// TestIsSecretFile_EdgeCases tests edge cases and boundary conditions.
func TestIsSecretFile_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		isSecret bool
	}{
		// Empty and edge cases
		{name: "empty string", path: "", isSecret: false},
		{name: "just extension", path: ".pem", isSecret: true},
		{name: "multiple extensions", path: "file.tar.gz", isSecret: false},
		{name: "case insensitive", path: "CERT.PEM", isSecret: true},
		{name: "case insensitive mixed", path: "Cert.Key", isSecret: true},

		// Paths with directories
		{name: "hidden dir .ssh/id_rsa", path: ".ssh/id_rsa", isSecret: true},
		{name: "config/.env", path: "config/.env", isSecret: true},
		{name: "src/credentials.json", path: "src/credentials.json", isSecret: true},

		// Similar but legitimate filenames
		{name: "tokenize.go", path: "tokenize.go", isSecret: false},
		{name: "tokenizer.rs", path: "tokenizer.rs", isSecret: false},
		{name: "tokens.go", path: "tokens.go", isSecret: false},
		{name: "password.go", path: "password.go", isSecret: false}, // No prefix/suffix pattern
		{name: "secret.go", path: "secret.go", isSecret: false},     // No prefix/suffix pattern

		// Private files (should be detected)
		{name: "private.key", path: "private.key", isSecret: true},
		{name: "config.private", path: "config.private", isSecret: true},
		{name: "private_config.go", path: "private_config.go", isSecret: false}, // Underscore is not a separator
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSecretFile(tt.path)
			assert.Equal(t, tt.isSecret, result, "IsSecretFile(%q) = %v, want %v", tt.path, result, tt.isSecret)
		})
	}
}

// Test countBraces with edge cases
func TestCountBraces(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected int
	}{
		{
			name:     "simple open brace",
			line:     "func test() {",
			expected: 1,
		},
		{
			name:     "simple close brace",
			line:     "}",
			expected: -1,
		},
		{
			name:     "braces in string should be ignored",
			line:     `x := "value { }"`,
			expected: 0,
		},
		{
			name:     "braces in raw string should be ignored",
			line:     "x := `raw { string }`",
			expected: 0,
		},
		{
			name:     "multiline raw string with braces",
			line:     "x := `multiline { string } here`",
			expected: 0,
		},
		{
			name:     "escaped quote in string",
			line:     `x := "escaped \" quote { }"`,
			expected: 0,
		},
		{
			name:     "real brace after string",
			line:     `x := "test" {`,
			expected: 1,
		},
		{
			name:     "braces in rune literal",
			line:     `x := '{'`,
			expected: 0,
		},
		{
			name:     "escaped backslash then quote",
			line:     `x := "\\{test\\""`,
			expected: 0,
		},
		{
			name:     "comment should ignore braces",
			line:     "// x := { }",
			expected: 0,
		},
		{
			name:     "code then comment",
			line:     "x := 1 // comment {",
			expected: 0,
		},
		{
			name:     "multiple braces",
			line:     "func test() {{ return }}",
			expected: 0,
		},
		{
			name:     "nested braces in strings only",
			line:     `s := "{}{}{}"`,
			expected: 0,
		},
		{
			name:     "raw string with backticks",
			line:     "s := `{\\n\\t}`",
			expected: 0,
		},
		{
			name:     "escaped quote keeps string open",
			line:     `s := "test \"`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := countBraces(tt.line, braceCountState{})
			assert.Equal(t, tt.expected, result, "countBraces(%q) = %d, want %d", tt.line, result, tt.expected)
		})
	}
}

// Test countBraces with multi-line state
func TestCountBraces_MultiLine(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected int
	}{
		{
			name:     "multiline raw string",
			lines:    []string{"return `SELECT", "    FROM users`"},
			expected: 0,
		},
		{
			name:     "multiline raw string with brace then real brace",
			lines:    []string{"s := `{", "    value`", "}"},
			expected: -1, // -1 from the closing brace
		},
		{
			name:     "function with multiline string",
			lines:    []string{"func test() {", "    return `test", "    value`", "}"},
			expected: 0, // { then }
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := braceCountState{}
			total := 0
			for _, line := range tt.lines {
				delta, newState := countBraces(line, state)
				total += delta
				state = newState
			}
			assert.Equal(t, tt.expected, total, "total brace count for %v = %d, want %d", tt.lines, total, tt.expected)
		})
	}
}

// Test Go parser with multiline strings and escaped quotes
func TestParseGoFile_MultilineStrings(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "multiline.go")
	content := `package multiline

import "fmt"

// Function with multiline string
func Query() string {
	return ` + "`" + `SELECT * FROM users WHERE active = true
	AND created_at > '2020-01-01'
	ORDER BY name` + "`" + `
}

// Function with escaped quotes
func JSONExample() string {
	return "{\"key\": \"value\", \"nested\": {\"a\": 1}}"
}

// Function with raw string containing braces
func RegexExample() string {
	return ` + "`" + `^test{1,3}\d+` + "`" + `
}

// Function with mixed content
func ComplexExample() map[string]interface{} {
	data := ` + "`" + `{"users": [{id: 1}, {id: 2}]}` + "`" + `
	result := make(map[string]interface{})
	return result
}
`
	require.NoError(t, os.WriteFile(goFile, []byte(content), 0o644))

	chunks, _, err := ParseFile(goFile, "go")
	require.NoError(t, err)

	// Should find at least 4 functions
	assert.GreaterOrEqual(t, len(chunks), 4, "should extract functions with multiline strings")

	// Verify each chunk has proper content
	for _, c := range chunks {
		assert.Equal(t, "go", c.Language)
		assert.NotEmpty(t, c.Content)
		// Content should include the multiline strings
		assert.NotContains(t, c.Content, "... truncated", "should not truncate multiline strings")
	}
}

// Test edge case: escaped backslash before quote
func TestParseGoFile_EscapedBackslash(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "escaped.go")
	content := `package escaped

func EscapedBackslash() string {
	// This is an escaped backslash followed by a quote: \"
	s := "\\\""
	return s
}

func DoubleEscape() string {
	// Two escaped backslashes
	s := "\\\\"
	return s
}
`
	require.NoError(t, os.WriteFile(goFile, []byte(content), 0o644))

	chunks, _, err := ParseFile(goFile, "go")
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(chunks), 2, "should extract functions with escaped backslashes")
}
