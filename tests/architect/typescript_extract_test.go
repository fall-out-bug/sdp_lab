package architect_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"sdp_dev/internal/architect/extract"
)

// helper creates a temp dir tree from a map of relative-path → content.
func setupProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestTypeScriptExtractor_ESModuleImports(t *testing.T) {
	root := setupProject(t, map[string]string{
		"package.json": `{"name":"test"}`,
		"src/app.ts": `import React from 'react'
import { useState } from 'react'
import type { FC } from 'react'
import * as path from 'path'
`,
		"src/utils.tsx": `import { helper } from './helper'
export { helper } from './helper'
`,
	})

	ext := extract.NewTypeScriptExtractor()
	if !ext.Detect(root) {
		t.Fatal("expected Detect to return true for TS project")
	}

	result, err := ext.Extract(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Language != "typescript" {
		t.Errorf("expected language typescript, got %s", result.Language)
	}
	if result.ExtractionMethod != "regex" {
		t.Errorf("expected method regex, got %s", result.ExtractionMethod)
	}
	if result.AccuracyEstimate != 0.60 {
		t.Errorf("expected accuracy 0.60, got %f", result.AccuracyEstimate)
	}

	appImports := result.Imports["src/app.ts"]
	if len(appImports) < 3 {
		t.Fatalf("expected at least 3 imports in app.ts, got %d", len(appImports))
	}

	// Verify ES module imports.
	specifiers := make(map[string]extract.ImportKind)
	for _, imp := range appImports {
		specifiers[imp.Specifier] = imp.Kind
	}
	if kind, ok := specifiers["react"]; !ok || kind != extract.ImportESModule {
		t.Error("expected ES module import of 'react'")
	}
	if kind, ok := specifiers["path"]; !ok || kind != extract.ImportESModule {
		t.Error("expected ES module import of 'path'")
	}

	// Verify re-exports in utils.tsx.
	utilsImports := result.Imports["src/utils.tsx"]
	hasReExport := false
	for _, imp := range utilsImports {
		if imp.Kind == extract.ImportReExport && imp.Specifier == "./helper" {
			hasReExport = true
		}
	}
	if !hasReExport {
		t.Error("expected re-export of './helper' in utils.tsx")
	}
}

func TestTypeScriptExtractor_CommonJS(t *testing.T) {
	root := setupProject(t, map[string]string{
		"package.json": `{"name":"cjs-test"}`,
		"index.js": `const express = require('express')
const path = require('path')
const myModule = require('./lib/my-module')
`,
	})

	ext := extract.NewTypeScriptExtractor()
	result, err := ext.Extract(root)
	if err != nil {
		t.Fatal(err)
	}

	imports := result.Imports["index.js"]
	if len(imports) != 3 {
		t.Fatalf("expected 3 CommonJS imports, got %d", len(imports))
	}

	for _, imp := range imports {
		if imp.Kind != extract.ImportCommonJS {
			t.Errorf("expected CommonJS kind for %q, got %s", imp.Specifier, imp.Kind)
		}
	}

	specifiers := make(map[string]bool)
	for _, imp := range imports {
		specifiers[imp.Specifier] = true
	}
	for _, expected := range []string{"express", "path", "./lib/my-module"} {
		if !specifiers[expected] {
			t.Errorf("missing CommonJS import %q", expected)
		}
	}
}

func TestTypeScriptExtractor_PackageJson(t *testing.T) {
	root := setupProject(t, map[string]string{
		"package.json": `{
  "name": "my-app",
  "dependencies": {
    "react": "^18.2.0",
    "next": "14.0.0"
  },
  "devDependencies": {
    "typescript": "^5.0.0",
    "@types/react": "^18.0.0"
  },
  "workspaces": ["packages/*", "apps/*"]
}`,
		"src/index.ts": `import React from 'react'
`,
	})

	ext := extract.NewTypeScriptExtractor()
	result, err := ext.Extract(root)
	if err != nil {
		t.Fatal(err)
	}

	// Verify dependencies.
	if len(result.Dependencies) != 4 {
		t.Fatalf("expected 4 dependencies, got %d", len(result.Dependencies))
	}

	depMap := make(map[string]extract.TSDependency)
	for _, d := range result.Dependencies {
		depMap[d.Name] = d
	}

	if d, ok := depMap["react"]; !ok {
		t.Error("missing dependency react")
	} else if d.Version != "^18.2.0" || d.Dev {
		t.Errorf("react: version=%q dev=%v", d.Version, d.Dev)
	}

	if d, ok := depMap["typescript"]; !ok {
		t.Error("missing devDependency typescript")
	} else if !d.Dev {
		t.Error("typescript should be dev dependency")
	}

	// Verify workspaces.
	sort.Strings(result.Workspaces)
	if len(result.Workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(result.Workspaces))
	}
	if result.Workspaces[0] != "apps/*" || result.Workspaces[1] != "packages/*" {
		t.Errorf("unexpected workspaces: %v", result.Workspaces)
	}
}

func TestTypeScriptExtractor_NextJsDetection(t *testing.T) {
	t.Run("high confidence with config and app dir", func(t *testing.T) {
		root := setupProject(t, map[string]string{
			"package.json":  `{"name":"next-app","dependencies":{"next":"14.0.0","react":"^18.0.0"}}`,
			"next.config.js": `module.exports = { reactStrictMode: true }`,
			"app/page.tsx":  `export default function Home() { return <h1>Hi</h1> }`,
		})

		ext := extract.NewTypeScriptExtractor()
		result, err := ext.Extract(root)
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, fw := range result.Frameworks {
			if fw.Name == "Next.js" {
				found = true
				if fw.Confidence != "high" {
					t.Errorf("expected high confidence for Next.js, got %s", fw.Confidence)
				}
			}
		}
		if !found {
			t.Error("Next.js framework not detected")
		}
	})

	t.Run("high confidence with config and pages dir", func(t *testing.T) {
		root := setupProject(t, map[string]string{
			"package.json":   `{"name":"next-app","dependencies":{"next":"14.0.0"}}`,
			"next.config.mjs": `export default { reactStrictMode: true }`,
			"pages/index.tsx": `export default function Home() { return <h1>Hi</h1> }`,
		})

		ext := extract.NewTypeScriptExtractor()
		result, err := ext.Extract(root)
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, fw := range result.Frameworks {
			if fw.Name == "Next.js" {
				found = true
				if fw.Confidence != "high" {
					t.Errorf("expected high confidence for Next.js, got %s", fw.Confidence)
				}
			}
		}
		if !found {
			t.Error("Next.js framework not detected with pages dir")
		}
	})

	t.Run("medium confidence with dep only", func(t *testing.T) {
		root := setupProject(t, map[string]string{
			"package.json": `{"name":"next-maybe","dependencies":{"next":"14.0.0"}}`,
			"src/index.ts": `console.log("hello")`,
		})

		ext := extract.NewTypeScriptExtractor()
		result, err := ext.Extract(root)
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, fw := range result.Frameworks {
			if fw.Name == "Next.js" {
				found = true
				if fw.Confidence != "medium" {
					t.Errorf("expected medium confidence, got %s", fw.Confidence)
				}
			}
		}
		if !found {
			t.Error("Next.js not detected from dependency alone")
		}
	})
}

func TestTypeScriptExtractor_PathAliases(t *testing.T) {
	root := setupProject(t, map[string]string{
		"tsconfig.json": `{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"],
      "@components/*": ["src/components/*"],
      "@utils/*": ["lib/utils/*"]
    }
  }
}`,
		"package.json": `{"name":"alias-test"}`,
		"src/app.ts": `import { Button } from '@/components/Button'
import { format } from '@utils/format'
`,
	})

	ext := extract.NewTypeScriptExtractor()
	result, err := ext.Extract(root)
	if err != nil {
		t.Fatal(err)
	}

	// Verify path aliases were parsed.
	if len(result.PathAliases) != 3 {
		t.Fatalf("expected 3 path aliases, got %d: %v", len(result.PathAliases), result.PathAliases)
	}

	// "@/" -> "src/"
	if v, ok := result.PathAliases["@/"]; !ok || v != "src/" {
		t.Errorf("expected @/ -> src/, got %q", v)
	}

	// "@components/" -> "src/components/"
	if v, ok := result.PathAliases["@components/"]; !ok || v != "src/components/" {
		t.Errorf("expected @components/ -> src/components/, got %q", v)
	}

	// "@utils/" -> "lib/utils/"
	if v, ok := result.PathAliases["@utils/"]; !ok || v != "lib/utils/" {
		t.Errorf("expected @utils/ -> lib/utils/, got %q", v)
	}

	// Verify that imports using aliases are captured.
	appImports := result.Imports["src/app.ts"]
	if len(appImports) != 2 {
		t.Fatalf("expected 2 imports in app.ts, got %d", len(appImports))
	}

	specifiers := make(map[string]bool)
	for _, imp := range appImports {
		specifiers[imp.Specifier] = true
	}
	if !specifiers["@/components/Button"] {
		t.Error("missing aliased import @/components/Button")
	}
	if !specifiers["@utils/format"] {
		t.Error("missing aliased import @utils/format")
	}
}

func TestTypeScriptExtractor_NoTSFiles(t *testing.T) {
	root := setupProject(t, map[string]string{
		"README.md":  "# Not a TS project",
		"main.py":    "print('hello')",
		"Makefile":   "all:\n\techo hi",
	})

	ext := extract.NewTypeScriptExtractor()
	if ext.Detect(root) {
		t.Error("Detect should return false when no TS/JS files or config exist")
	}
}

// Additional coverage: skip node_modules, framework detection for NestJS/Express/Vue/Angular.

func TestTypeScriptExtractor_SkipNodeModules(t *testing.T) {
	root := setupProject(t, map[string]string{
		"package.json":                `{"name":"test"}`,
		"src/app.ts":                  `import { foo } from './foo'`,
		"node_modules/react/index.js": `export default {}`,
	})

	ext := extract.NewTypeScriptExtractor()
	result, err := ext.Extract(root)
	if err != nil {
		t.Fatal(err)
	}

	for file := range result.Imports {
		if filepath.Base(filepath.Dir(file)) == "node_modules" || file == "node_modules/react/index.js" {
			t.Errorf("should not have imports from node_modules: %s", file)
		}
	}
}

func TestTypeScriptExtractor_SideEffectImports(t *testing.T) {
	root := setupProject(t, map[string]string{
		"package.json": `{"name":"test"}`,
		"src/main.ts": `import 'reflect-metadata'
import './polyfills'
`,
	})

	ext := extract.NewTypeScriptExtractor()
	result, err := ext.Extract(root)
	if err != nil {
		t.Fatal(err)
	}

	imports := result.Imports["src/main.ts"]
	if len(imports) != 2 {
		t.Fatalf("expected 2 side-effect imports, got %d", len(imports))
	}
	for _, imp := range imports {
		if imp.Kind != extract.ImportSideEffect {
			t.Errorf("expected side_effect kind for %q, got %s", imp.Specifier, imp.Kind)
		}
	}
}

func TestTypeScriptExtractor_NestJSDetection(t *testing.T) {
	root := setupProject(t, map[string]string{
		"package.json": `{"name":"nest-app","dependencies":{"@nestjs/core":"^10.0.0"}}`,
		"src/app.module.ts": `import { Module } from '@nestjs/common'

@Module({
  imports: [],
})
export class AppModule {}
`,
	})

	ext := extract.NewTypeScriptExtractor()
	result, err := ext.Extract(root)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, fw := range result.Frameworks {
		if fw.Name == "NestJS" {
			found = true
			if fw.Confidence != "high" {
				t.Errorf("expected high confidence, got %s", fw.Confidence)
			}
		}
	}
	if !found {
		t.Error("NestJS not detected")
	}
}

func TestTypeScriptExtractor_ExpressDetection(t *testing.T) {
	root := setupProject(t, map[string]string{
		"package.json": `{"name":"express-app","dependencies":{"express":"^4.18.0"}}`,
		"server.js": `const express = require('express')
const app = express()
app.get('/', (req, res) => res.send('ok'))
app.post('/api', handler)
`,
	})

	ext := extract.NewTypeScriptExtractor()
	result, err := ext.Extract(root)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, fw := range result.Frameworks {
		if fw.Name == "Express" {
			found = true
			if fw.Confidence != "high" {
				t.Errorf("expected high confidence, got %s", fw.Confidence)
			}
		}
	}
	if !found {
		t.Error("Express not detected")
	}
}

func TestTypeScriptExtractor_VueDetection(t *testing.T) {
	root := setupProject(t, map[string]string{
		"package.json":          `{"name":"vue-app","dependencies":{"vue":"^3.3.0"}}`,
		"src/App.vue":           `<template><div>Hello</div></template>`,
		"src/components/Btn.vue": `<template><button>Click</button></template>`,
	})

	ext := extract.NewTypeScriptExtractor()
	result, err := ext.Extract(root)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, fw := range result.Frameworks {
		if fw.Name == "Vue" {
			found = true
			if fw.Confidence != "high" {
				t.Errorf("expected high confidence, got %s", fw.Confidence)
			}
		}
	}
	if !found {
		t.Error("Vue not detected")
	}
}

func TestTypeScriptExtractor_AngularDetection(t *testing.T) {
	root := setupProject(t, map[string]string{
		"package.json": `{"name":"ng-app","dependencies":{"@angular/core":"^17.0.0"}}`,
		"angular.json": `{"version": 1}`,
		"src/app.ts":   `import { Component } from '@angular/core'`,
	})

	ext := extract.NewTypeScriptExtractor()
	result, err := ext.Extract(root)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, fw := range result.Frameworks {
		if fw.Name == "Angular" {
			found = true
			if fw.Confidence != "high" {
				t.Errorf("expected high confidence, got %s", fw.Confidence)
			}
		}
	}
	if !found {
		t.Error("Angular not detected")
	}
}
