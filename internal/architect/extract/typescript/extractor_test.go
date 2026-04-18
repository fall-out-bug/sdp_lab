package typescript

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestTSExtractorDetect(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		wantTrue bool
	}{
		{
			name: "tsconfig.json present",
			files: map[string]string{
				"tsconfig.json": "{}",
			},
			wantTrue: true,
		},
		{
			name: "package.json present",
			files: map[string]string{
				"package.json": `{"name": "test"}`,
			},
			wantTrue: true,
		},
		{
			name: "jsconfig.json present",
			files: map[string]string{
				"jsconfig.json": "{}",
			},
			wantTrue: true,
		},
		{
			name: "TS file present",
			files: map[string]string{
				"src/index.ts": "console.log('hello');",
			},
			wantTrue: true,
		},
		{
			name:     "no TS/JS markers",
			files:    map[string]string{},
			wantTrue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create test files.
			for path, content := range tt.files {
				fullPath := filepath.Join(tmpDir, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("failed to create dir: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
			}

			extractor := NewTSExtractor()
			got := extractor.detect(tmpDir)

			if got != tt.wantTrue {
				t.Errorf("TSExtractor.detect() = %v, want %v", got, tt.wantTrue)
			}
		})
	}
}

func TestTSExtractorExtract(t *testing.T) {
	tests := []struct {
		name          string
		files         map[string]string
		wantLang      string
		wantNodeCount int
		wantEdgeCount int
		wantClusters  int
	}{
		{
			name: "simple TypeScript project",
			files: map[string]string{
				"tsconfig.json":    `{}`,
				"package.json":     `{"name": "test", "dependencies": {"react": "^18.0.0"}}`,
				"src/index.ts":     "import { App } from './App';",
				"src/App.tsx":      "export default function App() { return <div>Hello</div>; }",
				"src/utils.ts":     "export const foo = () => {};",
			},
			wantLang:      "typescript",
			wantNodeCount: 3,
			wantEdgeCount: 1,
			wantClusters:  1,
		},
		{
			name: "no TS/JS markers",
			files: map[string]string{
				"README.md": "# Test",
			},
			wantLang:      "",
			wantNodeCount: 0,
			wantEdgeCount: 0,
			wantClusters:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create test files.
			for path, content := range tt.files {
				fullPath := filepath.Join(tmpDir, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("failed to create dir: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
			}

			extractor := NewTSExtractor()
			ctx := context.Background()
			frag, err := extractor.Extract(ctx, tmpDir)

			if err != nil {
				t.Fatalf("Extract() error = %v", err)
			}

			// Check if we got an empty fragment for non-TS projects.
			if tt.wantLang == "" {
				if frag.Languages != nil && len(frag.Languages) > 0 {
					t.Errorf("Extract() got languages %v, want empty", frag.Languages)
				}
				return
			}

			// Check language.
			if len(frag.Languages) == 0 {
				t.Errorf("Extract() got no languages, want at least one")
			} else if frag.Languages[0].Primary != tt.wantLang {
				t.Errorf("Extract() primary language = %q, want %q", frag.Languages[0].Primary, tt.wantLang)
			}

			// Check import graph.
			if frag.ImportGraph == nil {
				t.Errorf("Extract() got nil ImportGraph, want non-nil")
				return
			}

			if frag.ImportGraph.Nodes != tt.wantNodeCount {
				t.Errorf("Extract() node count = %d, want %d", frag.ImportGraph.Nodes, tt.wantNodeCount)
			}

			if frag.ImportGraph.Edges != tt.wantEdgeCount {
				t.Errorf("Extract() edge count = %d, want %d", frag.ImportGraph.Edges, tt.wantEdgeCount)
			}

			if len(frag.ImportGraph.Clusters) != tt.wantClusters {
				t.Errorf("Extract() cluster count = %d, want %d", len(frag.ImportGraph.Clusters), tt.wantClusters)
			}

			// Check extraction method.
			if frag.ImportGraph.ExtractionMethod != "regex" {
				t.Errorf("Extract() extraction method = %q, want 'regex'", frag.ImportGraph.ExtractionMethod)
			}

			// Check accuracy estimate.
			if frag.ImportGraph.AccuracyEstimate < 0.6 || frag.ImportGraph.AccuracyEstimate > 0.75 {
				t.Errorf("Extract() accuracy = %f, want 0.6-0.75", frag.ImportGraph.AccuracyEstimate)
			}
		})
	}
}

func TestTSExtractorName(t *testing.T) {
	extractor := NewTSExtractor()
	if name := extractor.Name(); name != "typescript" {
		t.Errorf("TSExtractor.Name() = %q, want 'typescript'", name)
	}
}

func TestComputeTSCluster(t *testing.T) {
	tests := []struct {
		path        string
		wantCluster string
	}{
		{"src/index.ts", "src"},
		{"src/components/Button.tsx", "src/components"},
		{"pages/index.tsx", "pages"},
		{"utils.ts", ""},
		{"index.ts", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := computeTSCluster(tt.path)
			if got != tt.wantCluster {
				t.Errorf("computeTSCluster(%q) = %q, want %q", tt.path, got, tt.wantCluster)
			}
		})
	}
}

func TestIsTSGenerated(t *testing.T) {
	tests := []struct {
		path    string
		wantGen bool
	}{
		{"src/utils.generated.ts", true},
		{"src/utils.gen.ts", true},
		{"src/api.pb.ts", true},
		{"src/types.d.ts", true},
		{"src/__generated__/types.ts", true},
		{"src/generated/utils.ts", true},
		{"src/utils.ts", false},
		{"src/index.ts", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isTSGenerated(tt.path)
			if got != tt.wantGen {
				t.Errorf("isTSGenerated(%q) = %v, want %v", tt.path, got, tt.wantGen)
			}
		})
	}
}

func TestConvertTSImportGraph(t *testing.T) {
	graph := &TSImportGraph{
		Nodes: []TSPackageNode{
			{RelPath: "src/index.ts", Cluster: "src"},
			{RelPath: "src/utils.ts", Cluster: "src"},
		},
		Edges: []TSImportEdge{
			{From: "src/index.ts", To: "src/utils.ts", Resolved: true},
		},
		Clusters:         []string{"src"},
		Dependencies:     []TSDependencyEntry{{Name: "react", Version: "^18.0.0", Dev: false}},
		ExtractionMethod: "regex",
		AccuracyEstimate: 0.65,
	}

	frag := convertTSImportGraph(graph, "/root")

	if frag == nil {
		t.Fatal("convertTSImportGraph() returned nil")
	}

	// Check languages.
	if len(frag.Languages) == 0 {
		t.Errorf("convertTSImportGraph() got no languages")
	} else if frag.Languages[0].Primary != "typescript" {
		t.Errorf("convertTSImportGraph() primary = %q, want 'typescript'", frag.Languages[0].Primary)
	}

	// Check import graph.
	if frag.ImportGraph == nil {
		t.Fatal("convertTSImportGraph() got nil ImportGraph")
	}

	if frag.ImportGraph.Nodes != 2 {
		t.Errorf("convertTSImportGraph() nodes = %d, want 2", frag.ImportGraph.Nodes)
	}

	if frag.ImportGraph.Edges != 1 {
		t.Errorf("convertTSImportGraph() edges = %d, want 1", frag.ImportGraph.Edges)
	}

	// Check dependencies.
	if len(frag.Dependencies) == 0 {
		t.Errorf("convertTSImportGraph() got no dependencies")
	} else if len(frag.Dependencies[0].NotableDeps) != 1 {
		t.Errorf("convertTSImportGraph() notable deps = %d, want 1", len(frag.Dependencies[0].NotableDeps))
	}
}

func TestDetectTSDepSignal(t *testing.T) {
	tests := []struct {
		name   string
		signal string
	}{
		{"react", "ui_framework"},
		{"next", "ssr_framework"},
		{"express", "web_framework"},
		{"@nestjs/core", "web_framework"},
		{"prisma", "orm"},
		{"jest", "testing"},
		{"webpack", "bundler"},
		{"unknown-package", "dependency"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectTSDepSignal(tt.name)
			if got != tt.signal {
				t.Errorf("detectTSDepSignal(%q) = %q, want %q", tt.name, got, tt.signal)
			}
		})
	}
}

func TestDetectTSCircularDeps(t *testing.T) {
	tests := []struct {
		name       string
		edges      []TSImportEdge
		wantCycles int
	}{
		{
			name: "no cycles",
			edges: []TSImportEdge{
				{From: "a.ts", To: "b.ts", Resolved: true},
				{From: "b.ts", To: "c.ts", Resolved: true},
			},
			wantCycles: 0,
		},
		{
			name: "simple cycle",
			edges: []TSImportEdge{
				{From: "a.ts", To: "b.ts", Resolved: true},
				{From: "b.ts", To: "a.ts", Resolved: true},
			},
			wantCycles: 1,
		},
		{
			name: "three-way cycle",
			edges: []TSImportEdge{
				{From: "a.ts", To: "b.ts", Resolved: true},
				{From: "b.ts", To: "c.ts", Resolved: true},
				{From: "c.ts", To: "a.ts", Resolved: true},
			},
			wantCycles: 1,
		},
		{
			name: "external edges ignored",
			edges: []TSImportEdge{
				{From: "a.ts", To: "react", Resolved: false},
				{From: "b.ts", To: "lodash", Resolved: false},
			},
			wantCycles: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cycles := detectTSCircularDeps(tt.edges)
			if len(cycles) != tt.wantCycles {
				t.Errorf("detectTSCircularDeps() got %d cycles, want %d", len(cycles), tt.wantCycles)
			}
		})
	}
}
