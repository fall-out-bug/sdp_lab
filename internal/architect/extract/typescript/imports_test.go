package typescript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFileImports(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		wantEdgeCount  int
		wantExternal   int
		checkKinds     map[TSImportKind]bool
		checkSpecifiers []string
	}{
		{
			name: "ES module default import",
			content: `
import React from 'react';
import { useState } from 'react';
`,
			wantEdgeCount: 2,
			wantExternal:  1,
			checkKinds:     map[TSImportKind]bool{TSImportESModule: true},
			checkSpecifiers: []string{"react"},
		},
		{
			name: "CommonJS require",
			content: `
const express = require('express');
const router = require('./router');
`,
			wantEdgeCount: 2,
			wantExternal:  1,
			checkKinds:     map[TSImportKind]bool{TSImportCommonJS: true},
			checkSpecifiers: []string{"express"},
		},
		{
			name: "Side-effect import",
			content: `
import 'polyfills';
import './styles.css';
`,
			wantEdgeCount: 2,
			wantExternal:  1,
			checkKinds:     map[TSImportKind]bool{TSImportSideEffect: true},
		},
		{
			name: "Dynamic import",
			content: `
const module = await import('lodash');
import('./utils').then(utils => {});
`,
			wantEdgeCount: 2,
			wantExternal:  1,
			checkKinds:     map[TSImportKind]bool{TSImportDynamic: true},
		},
		{
			name: "Re-export",
			content: `
export { Component } from './Component';
export * from 'lib';
export { default } from './index';
`,
			wantEdgeCount: 3,
			wantExternal:  1,
			checkKinds:     map[TSImportKind]bool{TSImportReExport: true},
		},
		{
			name: "Mixed imports",
			content: `
import React from 'react';
import express from 'express';
const _ = require('lodash');
import './styles.css';
export { Button } from './Button';
`,
			wantEdgeCount: 5,
			wantExternal:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file.
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.ts")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			edges, externals := extractFileImports(testFile, "test.ts", tmpDir)

			if len(edges) != tt.wantEdgeCount {
				t.Errorf("got %d edges, want %d", len(edges), tt.wantEdgeCount)
			}

			if len(externals) != tt.wantExternal {
				t.Errorf("got %d externals, want %d", len(externals), tt.wantExternal)
			}

			// Check kinds if specified.
			if tt.checkKinds != nil {
				foundKinds := make(map[TSImportKind]bool)
				for _, e := range edges {
					foundKinds[e.Kind] = true
				}
				for kind := range tt.checkKinds {
					if !foundKinds[kind] {
						t.Errorf("missing import kind %v", kind)
					}
				}
			}

			// Check specifiers if specified.
			if tt.checkSpecifiers != nil {
				specs := make(map[string]bool)
				for _, e := range edges {
					specs[e.To] = true
				}
				for _, spec := range tt.checkSpecifiers {
					if !specs[spec] {
						t.Errorf("missing specifier %q", spec)
					}
				}
			}
		})
	}
}

func TestIsLocalSpecifier(t *testing.T) {
	tests := []struct {
		spec string
		want bool
	}{
		{"./local", true},
		{"../parent", true},
		{"/absolute", true},
		{"package", false},
		{"@scope/package", false},
		{"@scope/package/sub", false},
		{"package/sub", false},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			if got := isLocalSpecifier(tt.spec); got != tt.want {
				t.Errorf("isLocalSpecifier(%q) = %v, want %v", tt.spec, got, tt.want)
			}
		})
	}
}

func TestResolveSpecifier(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		fromRel    string
		wantTarget string
	}{
		{
			name:       "local relative import",
			spec:       "./utils",
			fromRel:    "src/index.ts",
			wantTarget: "src/utils",
		},
		{
			name:       "local parent import",
			spec:       "../config",
			fromRel:    "src/index.ts",
			wantTarget: "config",
		},
		{
			name:       "package import",
			spec:       "react",
			fromRel:    "src/index.ts",
			wantTarget: "react",
		},
		{
			name:       "scoped package import",
			spec:       "@nestjs/common",
			fromRel:    "src/index.ts",
			wantTarget: "@nestjs/common",
		},
		{
			name:       "scoped package with subpath",
			spec:       "@nestjs/common/decorators",
			fromRel:    "src/index.ts",
			wantTarget: "@nestjs/common",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSpecifier(tt.spec, tt.fromRel, "/root")
			if got != tt.wantTarget {
				t.Errorf("resolveSpecifier(%q, %q) = %q, want %q", tt.spec, tt.fromRel, got, tt.wantTarget)
			}
		})
	}
}

func TestIsBarrelFile(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		content    string
		wantBarrel bool
	}{
		{
			name:       "index.ts with re-exports",
			filename:   "index.ts",
			content:    "export * from './foo';",
			wantBarrel: true,
		},
		{
			name:       "index.ts without re-exports",
			filename:   "index.ts",
			content:    "export const x = 1;",
			wantBarrel: false,
		},
		{
			name:       "non-index file with re-exports",
			filename:   "utils.ts",
			content:    "export * from './foo';",
			wantBarrel: false,
		},
		{
			name:       "index.js with re-exports",
			filename:   "index.js",
			content:    "export { foo } from './foo';",
			wantBarrel: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			got := isBarrelFile(testFile, tt.filename)
			if got != tt.wantBarrel {
				t.Errorf("isBarrelFile() = %v, want %v", got, tt.wantBarrel)
			}
		})
	}
}
