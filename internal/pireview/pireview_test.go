package pireview

import (
	"context"
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
				Scope:   ScopeAuto,
				Runner:  runner,
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
			"git rev-parse --abbrev-ref HEAD": []byte("feature/F161-test\n"),
			"git rev-parse HEAD":              []byte("abc123def456\n"),
			"git status --porcelain --untracked-files=all":          []byte(""),
			"git diff --name-only main...HEAD": []byte("foo.go\nbar.go\n"),
			"git diff main...HEAD":            []byte("diff content here"),
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
			"git rev-parse --abbrev-ref HEAD": []byte("main\n"),
			"git rev-parse HEAD":              []byte("deadbeef\n"),
			"git status --porcelain --untracked-files=all":          []byte(" M internal/pireview/context.go\nA  internal/pireview/evidence.go\n"),
			"git diff HEAD":                   []byte("diff --git a/context.go b/context.go\n+new line\n"),
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
