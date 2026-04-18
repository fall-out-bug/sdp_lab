package typescript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectMonorepo(t *testing.T) {
	tests := []struct {
		name          string
		files         map[string]string
		wantMonorepo  bool
		wantTool      string
		wantWorkspace int
	}{
		{
			name: "npm workspaces",
			files: map[string]string{
				"package.json": `{
					"name": "monorepo",
					"workspaces": ["packages/*"]
				}`,
				"packages/pkg1/package.json": `{"name": "pkg1"}`,
				"packages/pkg2/package.json": `{"name": "pkg2"}`,
			},
			wantMonorepo:  true,
			wantTool:      "npm",
			wantWorkspace: 2,
		},
		{
			name: "yarn workspaces",
			files: map[string]string{
				"package.json": `{
					"name": "monorepo",
					"workspaces": ["packages/*"]
				}`,
				"yarn.lock":                    "",
				"packages/pkg1/package.json":  `{"name": "pkg1"}`,
				"packages/pkg2/package.json":  `{"name": "pkg2"}`,
			},
			wantMonorepo:  true,
			wantTool:      "yarn",
			wantWorkspace: 2,
		},
		{
			name: "pnpm workspaces",
			files: map[string]string{
				"pnpm-workspace.yaml": "packages:\n  - 'packages/*'\n",
				"pnpm-lock.yaml":      "",
				"packages/pkg1/package.json": `{"name": "pkg1"}`,
				"packages/pkg2/package.json": `{"name": "pkg2"}`,
			},
			wantMonorepo:  true,
			wantTool:      "pnpm",
			wantWorkspace: 2,
		},
		{
			name: "turborepo",
			files: map[string]string{
				"package.json": `{
					"name": "monorepo",
					"workspaces": ["apps/*", "packages/*"]
				}`,
				"turbo.json":    "{}",
				"apps/web/package.json": `{"name": "web"}`,
				"packages/ui/package.json": `{"name": "ui"}`,
			},
			wantMonorepo:  true,
			wantTool:      "turborepo",
			wantWorkspace: 2,
		},
		{
			name: "lerna",
			files: map[string]string{
				"lerna.json":   `{"version": "independent"}`,
				"package.json": `{"name": "root"}`,
				"packages/pkg1/package.json": `{"name": "pkg1"}`,
			},
			wantMonorepo:  true,
			wantTool:      "lerna",
			wantWorkspace: 0,
		},
		{
			name: "no monorepo",
			files: map[string]string{
				"package.json": `{"name": "single-package"}`,
			},
			wantMonorepo:  false,
			wantTool:      "",
			wantWorkspace: 0,
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

			workspaces, isMonorepo, tool := detectMonorepo(tmpDir)

			if isMonorepo != tt.wantMonorepo {
				t.Errorf("detectMonorepo() isMonorepo = %v, want %v", isMonorepo, tt.wantMonorepo)
			}

			if tool != tt.wantTool {
				t.Errorf("detectMonorepo() tool = %q, want %q", tool, tt.wantTool)
			}

			if len(workspaces) != tt.wantWorkspace {
				t.Errorf("detectMonorepo() got %d workspaces, want %d", len(workspaces), tt.wantWorkspace)
			}
		})
	}
}

func TestParsePackageJSONWorkspaces(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantLen  int
		wantPats []string
	}{
		{
			name: "array form",
			content: `{
				"workspaces": ["packages/*", "apps/*"]
			}`,
			wantLen:  2,
			wantPats: []string{"packages/*", "apps/*"},
		},
		{
			name: "object form",
			content: `{
				"workspaces": {
					"packages": ["packages/*", "apps/*"]
				}
			}`,
			wantLen:  2,
			wantPats: []string{"packages/*", "apps/*"},
		},
		{
			name:     "no workspaces",
			content:  `{"name": "pkg"}`,
			wantLen:  0,
			wantPats: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			pkgPath := filepath.Join(tmpDir, "package.json")
			if err := os.WriteFile(pkgPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write package.json: %v", err)
			}

			patterns := parsePackageJSONWorkspaces(pkgPath)

			if len(patterns) != tt.wantLen {
				t.Errorf("parsePackageJSONWorkspaces() got %d patterns, want %d", len(patterns), tt.wantLen)
			}

			if tt.wantPats != nil {
				for i, want := range tt.wantPats {
					if i >= len(patterns) || patterns[i] != want {
						t.Errorf("parsePackageJSONWorkspaces()[%d] = %q, want %q", i, patterns[i], want)
					}
				}
			}
		})
	}
}

func TestParsePnpmWorkspace(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantLen  int
		wantPats []string
	}{
		{
			name: "simple patterns",
			content: `
packages:
  - 'packages/*'
  - 'apps/*'
`,
			wantLen:  2,
			wantPats: []string{"packages/*", "apps/*"},
		},
		{
			name: "no packages",
			content: `
# empty workspace
`,
			wantLen:  0,
			wantPats: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			wsPath := filepath.Join(tmpDir, "pnpm-workspace.yaml")
			if err := os.WriteFile(wsPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write pnpm-workspace.yaml: %v", err)
			}

			patterns := parsePnpmWorkspace(wsPath)

			if len(patterns) != tt.wantLen {
				t.Errorf("parsePnpmWorkspace() got %d patterns, want %d", len(patterns), tt.wantLen)
			}

			if tt.wantPats != nil {
				for i, want := range tt.wantPats {
					if i >= len(patterns) || patterns[i] != want {
						t.Errorf("parsePnpmWorkspace()[%d] = %q, want %q", i, patterns[i], want)
					}
				}
			}
		})
	}
}

func TestParseTSConfigAliases(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantLen    int
		wantAlias  []string
		wantTarget []string
	}{
		{
			name: "basic path aliases",
			content: `{
				"compilerOptions": {
					"baseUrl": ".",
					"paths": {
						"@/*": ["src/*"],
						"@components/*": ["src/components/*"]
					}
				}
			}`,
			wantLen:    2,
			wantAlias:  []string{"@/", "@components/"},
			wantTarget: []string{"src/", "src/components/"},
		},
		{
			name: "no paths",
			content: `{
				"compilerOptions": {
					"target": "es5"
				}
			}`,
			wantLen:    0,
			wantAlias:  nil,
			wantTarget: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tsconfigPath := filepath.Join(tmpDir, "tsconfig.json")
			if err := os.WriteFile(tsconfigPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write tsconfig.json: %v", err)
			}

			aliases := parseTSConfigAliases(tmpDir)

			if len(aliases) != tt.wantLen {
				t.Errorf("parseTSConfigAliases() got %d aliases, want %d", len(aliases), tt.wantLen)
			}

			if tt.wantAlias != nil {
				for i, wantAlias := range tt.wantAlias {
					if i >= len(aliases) || aliases[i].Alias != wantAlias {
						t.Errorf("parseTSConfigAliases()[%d].Alias = %q, want %q", i, aliases[i].Alias, wantAlias)
					}
					if aliases[i].Target != tt.wantTarget[i] {
						t.Errorf("parseTSConfigAliases()[%d].Target = %q, want %q", i, aliases[i].Target, tt.wantTarget[i])
					}
				}
			}
		})
	}
}

func TestDetectTSFrameworksV2(t *testing.T) {
	tests := []struct {
		name            string
		pkgJSON         string
		files           map[string]string
		wantFrameworks  []string
		wantMinConf     float64
	}{
		{
			name: "React only",
			pkgJSON: `{
				"dependencies": {
					"react": "^18.0.0"
				}
			}`,
			files: map[string]string{
				"src/App.tsx": "export default function App() { return <div>Hello</div>; }",
			},
			wantFrameworks: []string{"React"},
			wantMinConf:    0.7,
		},
		{
			name: "Next.js",
			pkgJSON: `{
				"dependencies": {
					"next": "^14.0.0",
					"react": "^18.0.0"
				}
			}`,
			files: map[string]string{
				"pages/index.tsx": "export default function Page() { return <div>Home</div>; }",
				"next.config.js":  "module.exports = {}",
			},
			wantFrameworks: []string{"Next.js", "React"},
			wantMinConf:    1.0,
		},
		{
			name: "NestJS",
			pkgJSON: `{
				"dependencies": {
					"@nestjs/core": "^10.0.0",
					"@nestjs/common": "^10.0.0"
				}
			}`,
			files: map[string]string{
				"src/app.module.ts": "@Module({})\nexport class AppModule {}",
			},
			wantFrameworks: []string{"NestJS"},
			wantMinConf:    0.8,
		},
		{
			name: "Express",
			pkgJSON: `{
				"dependencies": {
					"express": "^4.18.0"
				}
			}`,
			files: map[string]string{
				"src/server.ts": "app.get('/', (req, res) => res.send('Hello'));",
			},
			wantFrameworks: []string{"Express"},
			wantMinConf:    0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create package.json
			pkgPath := filepath.Join(tmpDir, "package.json")
			if err := os.WriteFile(pkgPath, []byte(tt.pkgJSON), 0644); err != nil {
				t.Fatalf("failed to write package.json: %v", err)
			}

			// Create test files
			for path, content := range tt.files {
				fullPath := filepath.Join(tmpDir, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("failed to create dir: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
			}

			deps := parsePackageJSONDeps(tmpDir)
			frameworks := detectTSFrameworksV2(tmpDir, deps, nil)

			if len(frameworks) != len(tt.wantFrameworks) {
				t.Errorf("detectTSFrameworksV2() got %d frameworks, want %d", len(frameworks), len(tt.wantFrameworks))
			}

			for i, wantName := range tt.wantFrameworks {
				if i >= len(frameworks) {
					t.Errorf("detectTSFrameworksV2()[%d] missing, want %q", i, wantName)
					continue
				}
				if frameworks[i].Name != wantName {
					t.Errorf("detectTSFrameworksV2()[%d].Name = %q, want %q", i, frameworks[i].Name, wantName)
				}
				// Only check min confidence for the first framework (primary)
				if i == 0 && frameworks[i].Confidence < tt.wantMinConf {
					t.Errorf("detectTSFrameworksV2()[%d].Confidence = %f, want >= %f", i, frameworks[i].Confidence, tt.wantMinConf)
				}
			}
		})
	}
}
