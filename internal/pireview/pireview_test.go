package pireview

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRunner records calls and returns canned responses.
type fakeRunner struct {
	responses map[string][]byte
	errors    map[string]error
	calls     []call
}

type call struct {
	dir  string
	name string
	args []string
}

func (f *fakeRunner) Output(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, call{dir, name, args})
	if err, ok := f.errors[key]; ok {
		return f.responses[key], err
	}
	if resp, ok := f.responses[key]; ok {
		return resp, nil
	}
	return nil, nil
}

func (f *fakeRunner) CombinedOutput(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, call{dir, name, args})
	if err, ok := f.errors[key]; ok {
		return f.responses[key], err
	}
	if resp, ok := f.responses[key]; ok {
		return resp, nil
	}
	return nil, nil
}

func (f *fakeRunner) Run(ctx context.Context, dir, name string, args ...string) error {
	key := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, call{dir, name, args})
	if err, ok := f.errors[key]; ok {
		return err
	}
	return nil
}

func TestConfig_Validate(t *testing.T) {
	runner := &fakeRunner{}

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid auto scope",
			cfg: Config{
				ProjectRoot: "/tmp/test",
				Scope:       ScopeAuto,
				BaseRef:     "main",
				Runner:      runner,
			},
			wantErr: false,
		},
		{
			name: "valid branch scope with base",
			cfg: Config{
				ProjectRoot: "/tmp/test",
				Scope:       ScopeBranch,
				BaseRef:     "main",
				Runner:      runner,
			},
			wantErr: false,
		},
		{
			name: "missing project root",
			cfg: Config{
				Scope:  ScopeAuto,
				Runner: runner,
			},
			wantErr: true,
		},
		{
			name: "invalid scope",
			cfg: Config{
				ProjectRoot: "/tmp/test",
				Scope:       "invalid",
				Runner:      runner,
			},
			wantErr: true,
		},
		{
			name: "branch scope without base",
			cfg: Config{
				ProjectRoot: "/tmp/test",
				Scope:       ScopeBranch,
				Runner:      runner,
			},
			wantErr: true,
		},
		{
			name: "missing runner",
			cfg: Config{
				ProjectRoot: "/tmp/test",
				Scope:       ScopeAuto,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestScopeMode_Constants(t *testing.T) {
	if ScopeAuto != "auto" {
		t.Errorf("ScopeAuto = %q, want %q", ScopeAuto, "auto")
	}
	if ScopeWorkingTree != "working-tree" {
		t.Errorf("ScopeWorkingTree = %q, want %q", ScopeWorkingTree, "working-tree")
	}
	if ScopeBranch != "branch" {
		t.Errorf("ScopeBranch = %q, want %q", ScopeBranch, "branch")
	}
}

func TestParseUntrackedFromPorcelain(t *testing.T) {
	input := " M modified.go\n?? new_file.go\n?? another.go\nD  deleted.go"
	files := parseUntrackedFromPorcelain(input)

	if len(files) != 2 {
		t.Fatalf("expected 2 untracked files, got %d", len(files))
	}
	if files[0] != "new_file.go" {
		t.Errorf("files[0] = %q, want %q", files[0], "new_file.go")
	}
	if files[1] != "another.go" {
		t.Errorf("files[1] = %q, want %q", files[1], "another.go")
	}
}

func TestShouldSkipFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"foo.go", false},
		{"vendor/pkg/main.go", true},
		{".git/config", true},
		{".worktrees/F161/main.go", true},
		{"image.png", true},
		{"binary.exe", true},
		{"src/main.go", false},
		{"package-lock.json", true},
		{".sdp/config.yml", true},
		{".sdp/config.yaml", true},
		{".sdp/runs/pi-review/run.json", true},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := shouldSkipFile(tc.path)
			if got != tc.want {
				t.Errorf("shouldSkipFile(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestLoadProjectRulesExcludesSensitiveSDPConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("agent rules"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".sdp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".sdp", "config.yml"), []byte("token: secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	rules := loadProjectRules(root)
	if _, ok := rules["AGENTS.md"]; !ok {
		t.Fatalf("expected AGENTS.md to be loaded")
	}
	if _, ok := rules[".sdp/config.yml"]; ok {
		t.Fatalf(".sdp/config.yml content must not be loaded into provider-bound rules")
	}
}

func TestWorkingTreeFiles(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   []string
	}{
		{
			name:   "empty status",
			status: "",
			want:   nil,
		},
		{
			name:   "modified file",
			status: " M foo.go",
			want:   []string{"foo.go"},
		},
		{
			name:   "multiple changes",
			status: " M foo.go\nA  bar.go\n?? baz.go",
			want:   []string{"bar.go", "baz.go", "foo.go"},
		},
		{
			name:   "renamed file",
			status: "R  old.go -> new.go",
			want:   []string{"new.go"},
		},
		{
			name:   "skip vendor",
			status: " M vendor/pkg/main.go\n M real.go",
			want:   []string{"real.go"},
		},
		{
			name:   "dedup same file",
			status: "MM staged_and_unstaged.go",
			want:   []string{"staged_and_unstaged.go"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := workingTreeFiles(tc.status)
			if err != nil {
				t.Fatalf("workingTreeFiles() error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("workingTreeFiles() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestBuildContextPacket_ValidatesConfig(t *testing.T) {
	_, err := BuildContextPacket(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error for empty config")
	}
	if !strings.Contains(err.Error(), "ProjectRoot") {
		t.Errorf("error should mention ProjectRoot: %v", err)
	}
}

func TestBuildContextPacket_BranchScope(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"git rev-parse --abbrev-ref HEAD":              []byte("feature/F161-test\n"),
			"git rev-parse HEAD":                           []byte("abc123def456\n"),
			"git status --porcelain --untracked-files=all": []byte(""),
			"git diff --name-only main...HEAD":             []byte("foo.go\nbar.go\n"),
			"git diff main...HEAD -- bar.go foo.go":        []byte("diff content here"),
		},
	}

	cfg := Config{
		ProjectRoot: t.TempDir(),
		Scope:       ScopeBranch,
		BaseRef:     "main",
		Runner:      runner,
	}

	pkt, err := BuildContextPacket(context.Background(), cfg)
	if err != nil {
		t.Fatalf("BuildContextPacket() error: %v", err)
	}
	if pkt.Branch != "feature/F161-test" {
		t.Errorf("Branch = %q, want %q", pkt.Branch, "feature/F161-test")
	}
	if pkt.HeadSHA != "abc123def456" {
		t.Errorf("HeadSHA = %q, want %q", pkt.HeadSHA, "abc123def456")
	}
	if len(pkt.ReviewedFiles) != 2 {
		t.Errorf("ReviewedFiles = %v, want 2 files", pkt.ReviewedFiles)
	}
	if pkt.UnifiedDiff != "diff content here" {
		t.Errorf("UnifiedDiff = %q, want %q", pkt.UnifiedDiff, "diff content here")
	}
}

func TestBuildContextPacket_WorkingTreeScope(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"git rev-parse --abbrev-ref HEAD":                                             []byte("main\n"),
			"git rev-parse HEAD":                                                          []byte("deadbeef\n"),
			"git status --porcelain --untracked-files=all":                                []byte(" M internal/pireview/context.go\nA  internal/pireview/evidence.go\n"),
			"git diff HEAD -- internal/pireview/context.go internal/pireview/evidence.go": []byte("diff --git a/context.go b/context.go\n+new line\n"),
		},
	}

	cfg := Config{
		ProjectRoot: t.TempDir(),
		Scope:       ScopeWorkingTree,
		Runner:      runner,
	}

	pkt, err := BuildContextPacket(context.Background(), cfg)
	if err != nil {
		t.Fatalf("BuildContextPacket() error: %v", err)
	}
	if len(pkt.ReviewedFiles) != 2 {
		t.Fatalf("ReviewedFiles = %v, want 2 files", pkt.ReviewedFiles)
	}
	if pkt.ReviewedFiles[0] != "internal/pireview/context.go" {
		t.Errorf("ReviewedFiles[0] = %q", pkt.ReviewedFiles[0])
	}
	if pkt.ReviewedFiles[1] != "internal/pireview/evidence.go" {
		t.Errorf("ReviewedFiles[1] = %q", pkt.ReviewedFiles[1])
	}
}

func TestBuildContextPacket_FiltersSkippedFilesFromDiff(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"git rev-parse --abbrev-ref HEAD":               []byte("main\n"),
			"git rev-parse HEAD":                            []byte("deadbeef\n"),
			"git status --porcelain --untracked-files=all":  []byte(" M internal/pireview/context.go\n M .beads/issues.jsonl\n"),
			"git diff HEAD -- internal/pireview/context.go": []byte("diff --git a/internal/pireview/context.go b/internal/pireview/context.go\n+new line\n"),
		},
	}

	cfg := Config{
		ProjectRoot: t.TempDir(),
		Scope:       ScopeWorkingTree,
		Runner:      runner,
	}

	pkt, err := BuildContextPacket(context.Background(), cfg)
	if err != nil {
		t.Fatalf("BuildContextPacket() error: %v", err)
	}
	if strings.Contains(pkt.UnifiedDiff, ".beads/issues.jsonl") {
		t.Fatalf("UnifiedDiff includes skipped beads file: %s", pkt.UnifiedDiff)
	}
	if len(pkt.ReviewedFiles) != 1 || pkt.ReviewedFiles[0] != "internal/pireview/context.go" {
		t.Fatalf("ReviewedFiles = %v, want only internal/pireview/context.go", pkt.ReviewedFiles)
	}
	for _, call := range runner.calls {
		if call.name == "git" && strings.Join(call.args, " ") == "diff HEAD" {
			t.Fatalf("unfiltered git diff was called: %#v", call)
		}
	}
}

func TestBuildContextPacket_BranchScopeFiltersSkippedFilesFromDiff(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"git rev-parse --abbrev-ref HEAD":                      []byte("feature/F161-test\n"),
			"git rev-parse HEAD":                                   []byte("abc123def456\n"),
			"git status --porcelain --untracked-files=all":         []byte(""),
			"git diff --name-only main...HEAD":                     []byte("internal/pireview/context.go\n.beads/issues.jsonl\n"),
			"git diff main...HEAD -- internal/pireview/context.go": []byte("diff --git a/internal/pireview/context.go b/internal/pireview/context.go\n+new line\n"),
		},
	}

	cfg := Config{
		ProjectRoot: t.TempDir(),
		Scope:       ScopeBranch,
		BaseRef:     "main",
		Runner:      runner,
	}

	pkt, err := BuildContextPacket(context.Background(), cfg)
	if err != nil {
		t.Fatalf("BuildContextPacket() error: %v", err)
	}
	if strings.Contains(pkt.UnifiedDiff, ".beads/issues.jsonl") {
		t.Fatalf("UnifiedDiff includes skipped beads file: %s", pkt.UnifiedDiff)
	}
	if pkt.UnifiedDiff != "diff --git a/internal/pireview/context.go b/internal/pireview/context.go\n+new line" {
		t.Fatalf("UnifiedDiff = %q, want expected filtered diff", pkt.UnifiedDiff)
	}
	if len(pkt.ReviewedFiles) != 1 || pkt.ReviewedFiles[0] != "internal/pireview/context.go" {
		t.Fatalf("ReviewedFiles = %v, want only internal/pireview/context.go", pkt.ReviewedFiles)
	}
	for _, call := range runner.calls {
		if call.name == "git" && strings.Join(call.args, " ") == "diff main...HEAD" {
			t.Fatalf("unfiltered git diff was called: %#v", call)
		}
	}
}

func TestBuildContextPacket_AllFilesSkippedProducesEmptyDiff(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"git rev-parse --abbrev-ref HEAD":              []byte("main\n"),
			"git rev-parse HEAD":                           []byte("deadbeef\n"),
			"git status --porcelain --untracked-files=all": []byte(" M .beads/issues.jsonl\n"),
		},
	}

	cfg := Config{
		ProjectRoot: t.TempDir(),
		Scope:       ScopeWorkingTree,
		Runner:      runner,
	}

	pkt, err := BuildContextPacket(context.Background(), cfg)
	if err != nil {
		t.Fatalf("BuildContextPacket() error: %v", err)
	}
	if len(pkt.ReviewedFiles) != 0 {
		t.Fatalf("ReviewedFiles = %v, want empty", pkt.ReviewedFiles)
	}
	if pkt.UnifiedDiff != "" {
		t.Fatalf("UnifiedDiff = %q, want empty", pkt.UnifiedDiff)
	}
	for _, call := range runner.calls {
		if call.name == "git" && strings.HasPrefix(strings.Join(call.args, " "), "diff ") {
			t.Fatalf("git diff should not be called when all files are skipped: %#v", call)
		}
	}
}

func TestBuildContextPacket_AutoScopeFailsWhenOnlyIgnoredTelemetryChanged(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"git rev-parse --abbrev-ref HEAD":              []byte("feature/F168\n"),
			"git rev-parse HEAD":                           []byte("deadbeef\n"),
			"git status --porcelain --untracked-files=all": []byte("?? .sdp/runs/pi-review/run/models/zai.json\n"),
		},
	}

	cfg := Config{
		ProjectRoot: t.TempDir(),
		Scope:       ScopeAuto,
		Runner:      runner,
	}

	_, err := BuildContextPacket(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected auto scope to fail when only ignored telemetry changed")
	}
	if !strings.Contains(err.Error(), "no reviewable files") {
		t.Fatalf("error = %v, want clear no reviewable files error", err)
	}
}

func TestBuildContextPacket_AutoScopeFallsBackToBranchDiffWhenTelemetryIgnored(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"git rev-parse --abbrev-ref HEAD":                     []byte("feature/F168\n"),
			"git rev-parse HEAD":                                  []byte("deadbeef\n"),
			"git status --porcelain --untracked-files=all":        []byte("?? .sdp/runs/pi-review/run/models/zai.json\n"),
			"git diff --name-only main...HEAD":                    []byte("internal/pireview/runner.go\n"),
			"git diff main...HEAD -- internal/pireview/runner.go": []byte("diff --git a/internal/pireview/runner.go b/internal/pireview/runner.go\n+new line\n"),
		},
	}

	cfg := Config{
		ProjectRoot: t.TempDir(),
		Scope:       ScopeAuto,
		BaseRef:     "main",
		Runner:      runner,
	}

	pkt, err := BuildContextPacket(context.Background(), cfg)
	if err != nil {
		t.Fatalf("BuildContextPacket() error: %v", err)
	}
	if len(pkt.ReviewedFiles) != 1 || pkt.ReviewedFiles[0] != "internal/pireview/runner.go" {
		t.Fatalf("ReviewedFiles = %v, want branch diff file", pkt.ReviewedFiles)
	}
	if !strings.Contains(pkt.UnifiedDiff, "internal/pireview/runner.go") {
		t.Fatalf("UnifiedDiff = %q, want branch diff", pkt.UnifiedDiff)
	}
}
