package index

import (
	"path/filepath"
	"strings"
)

// Language holds configuration for parsing a specific language.
type Language struct {
	Name       string   // e.g. "go", "python", "typescript"
	Extensions []string // e.g. [".go"], [".py"], [".ts", ".tsx"]
	TestSuffix string   // e.g. "_test.go", "_test.py"
}

// SupportedLanguages is the registry of all languages the indexer can chunk.
var SupportedLanguages = []Language{
	{Name: "go", Extensions: []string{".go"}, TestSuffix: "_test.go"},
	{Name: "python", Extensions: []string{".py"}, TestSuffix: "_test.py"},
	{Name: "javascript", Extensions: []string{".js", ".mjs", ".cjs"}, TestSuffix: ".test.js"},
	{Name: "typescript", Extensions: []string{".ts", ".tsx", ".mts", ".cts"}, TestSuffix: ".test.ts"},
	{Name: "java", Extensions: []string{".java"}, TestSuffix: "Test.java"},
	{Name: "rust", Extensions: []string{".rs"}, TestSuffix: "_test.rs"},
	{Name: "ruby", Extensions: []string{".rb"}, TestSuffix: "_test.rb"},
	{Name: "c", Extensions: []string{".c", ".h"}, TestSuffix: "_test.c"},
	{Name: "cpp", Extensions: []string{".cpp", ".cc", ".cxx", ".hpp", ".hxx"}, TestSuffix: "_test.cpp"},
	{Name: "csharp", Extensions: []string{".cs"}, TestSuffix: "Test.cs"},
	{Name: "swift", Extensions: []string{".swift"}, TestSuffix: "Tests.swift"},
	{Name: "kotlin", Extensions: []string{".kt"}, TestSuffix: "Test.kt"},
	{Name: "scala", Extensions: []string{".scala"}, TestSuffix: "Test.scala"},
	{Name: "zig", Extensions: []string{".zig"}, TestSuffix: "_test.zig"},
	{Name: "shell", Extensions: []string{".sh", ".bash"}, TestSuffix: "_test.sh"},
}

// extensionToLanguage maps file extensions to language names.
var extensionToLanguage map[string]string

func init() {
	extensionToLanguage = make(map[string]string)
	for _, lang := range SupportedLanguages {
		for _, ext := range lang.Extensions {
			extensionToLanguage[ext] = lang.Name
		}
	}
}

// DetectLanguage returns the language name for a file path, or empty string.
func DetectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	return extensionToLanguage[ext]
}

// IsTestFile returns true if the file path matches the test pattern for its language.
func IsTestFile(path string) bool {
	lang := DetectLanguage(path)
	if lang == "" {
		return false
	}
	base := filepath.Base(path)
	for _, l := range SupportedLanguages {
		if l.Name == lang && l.TestSuffix != "" {
			return strings.HasSuffix(base, l.TestSuffix) ||
				strings.Contains(base, "_test.") ||
				strings.Contains(base, ".test.") ||
				strings.Contains(base, "Test.")
		}
	}
	return false
}
