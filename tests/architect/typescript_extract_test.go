package architect_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/architect/extract"

	"github.com/stretchr/testify/require"
)

// setupTSProject creates a temp dir tree from a map of relative-path -> content.
func setupTSProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
		require.NoError(t, os.WriteFile(abs, []byte(content), 0o644))
	}
	return root
}

func TestTSExtractor_ESModuleImports(t *testing.T) {
	root := setupTSProject(t, map[string]string{
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

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)

	// Verify language detection.
	require.NotEmpty(t, frag.Languages)
	require.Equal(t, "typescript", frag.Languages[0].Primary)

	// Verify import graph was populated.
	require.NotNil(t, frag.ImportGraph)
	require.NotZero(t, frag.ImportGraph.Nodes)
	require.NotZero(t, frag.ImportGraph.Edges)
	require.Equal(t, "regex", frag.ImportGraph.ExtractionMethod)
	require.InDelta(t, 0.65, frag.ImportGraph.AccuracyEstimate, 0.01)
}

func TestTSExtractor_CommonJS(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"package.json": `{"name":"cjs-test"}`,
		"index.js": `const express = require('express')
const path = require('path')
const myModule = require('./lib/my-module')
`,
	})

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)

	require.NotNil(t, frag.ImportGraph)
	require.GreaterOrEqual(t, frag.ImportGraph.Edges, 3)
}

func TestTSExtractor_PackageJsonDependencies(t *testing.T) {
	root := setupTSProject(t, map[string]string{
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

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)

	require.NotEmpty(t, frag.Dependencies)
	depInfo := frag.Dependencies[0]
	require.Equal(t, 4, depInfo.DepCount)

	notableNames := make(map[string]bool)
	for _, nd := range depInfo.NotableDeps {
		notableNames[nd.Name] = true
	}
	require.True(t, notableNames["react"], "expected react in notable deps")
	require.True(t, notableNames["next"], "expected next in notable deps")
}

func TestTSExtractor_NextJsDetection(t *testing.T) {
	t.Run("high confidence with config and app dir", func(t *testing.T) {
		root := setupTSProject(t, map[string]string{
			"package.json":   `{"name":"next-app","dependencies":{"next":"14.0.0","react":"^18.0.0"}}`,
			"next.config.js": `module.exports = { reactStrictMode: true }`,
			"app/page.tsx":   `export default function Home() { return <h1>Hi</h1> }`,
		})

		e := extract.NewTSExtractor()
		frag, err := e.Extract(context.Background(), root)
		require.NoError(t, err)
		require.NotNil(t, frag.ImportGraph)

		found := false
		for _, di := range frag.Dependencies {
			for _, nd := range di.NotableDeps {
				if nd.Name == "next" {
					found = true
				}
			}
		}
		require.True(t, found, "Next.js dependency not found in notable deps")
	})

	t.Run("with dep only", func(t *testing.T) {
		root := setupTSProject(t, map[string]string{
			"package.json": `{"name":"next-maybe","dependencies":{"next":"14.0.0"}}`,
			"src/index.ts": `console.log("hello")`,
		})

		e := extract.NewTSExtractor()
		frag, err := e.Extract(context.Background(), root)
		require.NoError(t, err)

		found := false
		for _, di := range frag.Dependencies {
			for _, nd := range di.NotableDeps {
				if nd.Name == "next" {
					found = true
				}
			}
		}
		require.True(t, found, "Next.js not detected from dependency alone")
	})
}

func TestTSExtractor_PathAliases(t *testing.T) {
	root := setupTSProject(t, map[string]string{
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

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)

	require.NotNil(t, frag.ImportGraph)
	require.GreaterOrEqual(t, frag.ImportGraph.Edges, 2)
}

func TestTSExtractor_NoTSFiles(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"README.md": "# Not a TS project",
		"main.py":   "print('hello')",
		"Makefile":  "all:\n\techo hi",
	})

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag)
	require.Nil(t, frag.ImportGraph)
}

func TestTSExtractor_SkipNodeModules(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"package.json":                `{"name":"test"}`,
		"src/app.ts":                  `import { foo } from './foo'`,
		"node_modules/react/index.js": `export default {}`,
	})

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)

	if frag.ImportGraph != nil {
		for _, cluster := range frag.ImportGraph.Clusters {
			for _, pkg := range cluster.Packages {
				require.NotContains(t, pkg, "node_modules",
					"should not have imports from node_modules")
			}
		}
	}
}

func TestTSExtractor_BarrelFileDetection(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"package.json": `{"name":"barrel-test"}`,
		"src/index.ts": `export { Button } from './components/Button'
export { Input } from './components/Input'
`,
		"src/components/Button.ts": `export function Button() {}`,
		"src/components/Input.ts":  `export function Input() {}`,
	})

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)

	require.NotNil(t, frag.ImportGraph)
	require.NotEmpty(t, frag.ImportGraph.Clusters)
}

func TestTSExtractor_NestJSDetection(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"package.json": `{"name":"nest-app","dependencies":{"@nestjs/core":"^10.0.0"}}`,
		"src/app.module.ts": `import { Module } from '@nestjs/common'

@Module({
  imports: [],
})
export class AppModule {}
`,
	})

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)

	found := false
	for _, di := range frag.Dependencies {
		for _, nd := range di.NotableDeps {
			if nd.Name == "@nestjs/core" {
				found = true
			}
		}
	}
	require.True(t, found, "NestJS (@nestjs/core) not found in dependencies")
}

func TestTSExtractor_ExpressDetection(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"package.json": `{"name":"express-app","dependencies":{"express":"^4.18.0"}}`,
		"server.js": `const express = require('express')
const app = express()
app.get('/', (req, res) => res.send('ok'))
app.post('/api', handler)
`,
	})

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)

	found := false
	for _, di := range frag.Dependencies {
		for _, nd := range di.NotableDeps {
			if nd.Name == "express" {
				found = true
			}
		}
	}
	require.True(t, found, "Express not found in dependencies")
}

func TestTSExtractor_VueDetection(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"package.json":           `{"name":"vue-app","dependencies":{"vue":"^3.3.0"}}`,
		"src/App.vue":            `<template><div>Hello</div></template>`,
		"src/components/Btn.vue": `<template><button>Click</button></template>`,
	})

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)

	found := false
	for _, di := range frag.Dependencies {
		for _, nd := range di.NotableDeps {
			if nd.Name == "vue" {
				found = true
			}
		}
	}
	require.True(t, found, "Vue not found in dependencies")
}

func TestTSExtractor_AngularDetection(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"package.json": `{"name":"ng-app","dependencies":{"@angular/core":"^17.0.0"}}`,
		"angular.json": `{"version": 1}`,
		"src/app.ts":   `import { Component } from '@angular/core'`,
	})

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)

	found := false
	for _, di := range frag.Dependencies {
		for _, nd := range di.NotableDeps {
			if nd.Name == "@angular/core" {
				found = true
			}
		}
	}
	require.True(t, found, "Angular (@angular/core) not found in dependencies")
}

func TestTSExtractor_SvelteDetection(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"package.json":     `{"name":"svelte-app","dependencies":{"svelte":"^4.0.0"}}`,
		"svelte.config.js": `export default {}`,
		"src/App.svelte":   `<script>let name = 'world'</script><h1>Hello {name}</h1>`,
	})

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)

	found := false
	for _, di := range frag.Dependencies {
		for _, nd := range di.NotableDeps {
			if nd.Name == "svelte" {
				found = true
			}
		}
	}
	require.True(t, found, "Svelte not found in dependencies")
}

func TestTSExtractor_MonorepoDetection(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"package.json": `{
  "name": "monorepo",
  "workspaces": ["packages/*"]
}`,
		"pnpm-workspace.yaml": `packages:
  - 'packages/*'
`,
		"lerna.json":              `{"version": "1.0.0"}`,
		"packages/a/package.json": `{"name": "@mono/a"}`,
		"packages/a/index.ts":     `export const a = 1`,
		"packages/b/package.json": `{"name": "@mono/b"}`,
		"packages/b/index.ts":     `import { a } from '@mono/a'`,
	})

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)

	require.NotNil(t, frag.ImportGraph)

	clusterIDs := make(map[string]bool)
	for _, c := range frag.ImportGraph.Clusters {
		clusterIDs[c.ID] = true
	}
	require.True(t, clusterIDs["packages/a"], "expected cluster packages/a")
	require.True(t, clusterIDs["packages/b"], "expected cluster packages/b")
}

func TestTSExtractor_ContextCancellation(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"package.json": `{"name":"test"}`,
		"src/app.ts":   `import React from 'react'`,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	e := extract.NewTSExtractor()
	_, err := e.Extract(ctx, root)
	// Either succeeds (fast path before cancellation check) or returns context error.
	if err != nil && ctx.Err() != nil {
		require.ErrorIs(t, err, context.Canceled)
	}
}

func TestTSExtractor_SideEffectImports(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"package.json": `{"name":"test"}`,
		"src/main.ts": `import 'reflect-metadata'
import './polyfills'
`,
	})

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)

	require.NotNil(t, frag.ImportGraph)
	require.GreaterOrEqual(t, frag.ImportGraph.Edges, 2)
}

func TestTSExtractor_DynamicImports(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"package.json": `{"name":"test"}`,
		"src/app.ts": `const mod = import('./heavy-module')
async function load() {
  const lib = import('some-lib')
}
`,
	})

	e := extract.NewTSExtractor()
	frag, err := e.Extract(context.Background(), root)
	require.NoError(t, err)

	require.NotNil(t, frag.ImportGraph)
	require.GreaterOrEqual(t, frag.ImportGraph.Edges, 2)
}

func TestTypeScriptAdapter_BasicProject(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"package.json": `{"name":"test","dependencies":{"express":"^4.18.0"}}`,
		"server.js":    `const express = require('express')`,
	})

	adapter := extract.TypeScriptAdapter{}
	frag, err := adapter.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag)
	require.NotEmpty(t, frag.Languages)
	require.Equal(t, "typescript", frag.Languages[0].Primary)
}

func TestTypeScriptAdapter_NonTSProject(t *testing.T) {
	root := setupTSProject(t, map[string]string{
		"main.py":   "print('hello')",
		"README.md": "# Python project",
	})

	adapter := extract.TypeScriptAdapter{}
	frag, err := adapter.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag)
	require.Nil(t, frag.ImportGraph)
}
