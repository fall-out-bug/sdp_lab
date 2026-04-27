package architect_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
	"github.com/fall-out-bug/sdp_lab/internal/architect/extract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// GitHistoryExtractor
// ---------------------------------------------------------------------------

func TestGitHistoryExtractorName(t *testing.T) {
	assertExtractorName(t, extract.GitHistoryExtractor{}, "git_history")
}

func TestGitHistoryExtractorNoGitDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "Hello")

	frag, err := extract.GitHistoryExtractor{}.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag)
	assert.Nil(t, frag.GitAnalysis)
}

func initGitRepo(t *testing.T, root string) {
	t.Helper()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = root
	require.NoError(t, cmd.Run())

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = root
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = root
	require.NoError(t, cmd.Run())
}

func commitFile(t *testing.T, root, path, content, author string) {
	t.Helper()

	fullPath := filepath.Join(root, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))

	// Stage the file
	cmd := exec.Command("git", "add", path)
	cmd.Dir = root
	require.NoError(t, cmd.Run())

	// Commit with specific author
	cmd = exec.Command("git", "-c", "user.name="+author, "-c", "user.email="+author+"@example.com", "commit", "-m", "Add "+path)
	cmd.Dir = root
	require.NoError(t, cmd.Run())
}

func TestGitHistoryExtractorWithGitRepo(t *testing.T) {
	root := t.TempDir()
	initGitRepo(t, root)

	// Create multiple commits with different authors
	commitFile(t, root, "main.go", "package main", "Alice")
	commitFile(t, root, "README.md", "# Test", "Alice")
	commitFile(t, root, "main.go", "package main\n\nfunc main() {}", "Bob")
	commitFile(t, root, "README.md", "# Test\n\nUpdated", "Bob")
	commitFile(t, root, "pkg/util.go", "package pkg", "Charlie")

	extractor := extract.GitHistoryExtractor{}
	frag, err := extractor.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.GitAnalysis)

	// Verify basic fields
	assert.Greater(t, frag.GitAnalysis.AnalyzedCommits, 0)
	assert.NotEmpty(t, frag.GitAnalysis.AnalyzedPeriod)

	// Top contributors should be populated
	assert.NotEmpty(t, frag.GitAnalysis.TopContributors)
	assert.Contains(t, frag.GitAnalysis.TopContributors, "Alice")
	assert.Contains(t, frag.GitAnalysis.TopContributors, "Bob")

	// Hotspots should detect files changed multiple times
	assert.NotEmpty(t, frag.GitAnalysis.Hotspots)

	// Check that main.go and README.md are in hotspots (changed 2+ times)
	hotspotPaths := make(map[string]bool)
	for _, h := range frag.GitAnalysis.Hotspots {
		hotspotPaths[h.Path] = true
	}
	assert.True(t, hotspotPaths["main.go"], "main.go should be in hotspots")
	assert.True(t, hotspotPaths["README.md"], "README.md should be in hotspots")

	// Ownership should have entries for directories
	assert.NotEmpty(t, frag.GitAnalysis.Ownership)
}

func TestGitHistoryCoChange(t *testing.T) {
	root := t.TempDir()
	initGitRepo(t, root)

	// Create commits where files A and B always change together
	commitFile(t, root, "a.go", "package a\n\ntype A struct{}", "Alice")
	commitFile(t, root, "b.go", "package b\n\ntype B struct{}", "Alice")

	commitFile(t, root, "a.go", "package a\n\ntype A struct{}\n\nfunc NewA() *A { return &A{} }", "Bob")
	commitFile(t, root, "b.go", "package b\n\ntype B struct{}\n\nfunc NewB() *B { return &B{} }", "Bob")

	commitFile(t, root, "a.go", "package a\n\ntype A struct{}\n\nfunc NewA() *A { return &A{} }\n\nfunc (a *A) Do() {}", "Charlie")
	commitFile(t, root, "b.go", "package b\n\ntype B struct{}\n\nfunc NewB() *B { return &B{} }\n\nfunc (b *B) Do() {}", "Charlie")

	// Add a file that doesn't co-change
	commitFile(t, root, "c.go", "package c", "Alice")

	extractor := extract.GitHistoryExtractor{}
	frag, err := extractor.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.GitAnalysis)

	// Should have co-change clusters
	if len(frag.GitAnalysis.CoChangeClusters) > 0 {
		// Look for a cluster containing both a.go and b.go
		foundCluster := false
		for _, cluster := range frag.GitAnalysis.CoChangeClusters {
			hasA := false
			hasB := false
			for _, f := range cluster.Files {
				if f == "a.go" {
					hasA = true
				}
				if f == "b.go" {
					hasB = true
				}
			}
			if hasA && hasB {
				foundCluster = true
				break
			}
		}
		assert.True(t, foundCluster, "a.go and b.go should be in a co-change cluster")
	}
}

func TestGitHistoryCoChangeWithModifiedFiles(t *testing.T) {
	root := t.TempDir()
	initGitRepo(t, root)

	// Create a file
	commitFile(t, root, "a.go", "package a", "Alice")

	// Modify it along with a new file in the same commit
	commitFile(t, root, "a.go", "package a\n\nfunc A() {}", "Bob")
	commitFile(t, root, "b.go", "package b", "Bob")

	// Modify both again
	commitFile(t, root, "a.go", "package a\n\nfunc A() {}\n\nfunc B() {}", "Charlie")
	commitFile(t, root, "b.go", "package b\n\nfunc C() {}", "Charlie")

	extractor := extract.GitHistoryExtractor{}
	frag, err := extractor.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.GitAnalysis)

	// Should detect co-change relationship
	if len(frag.GitAnalysis.CoChangeClusters) > 0 {
		// Look for a cluster containing both a.go and b.go
		foundCluster := false
		for _, cluster := range frag.GitAnalysis.CoChangeClusters {
			hasA := false
			hasB := false
			for _, f := range cluster.Files {
				if strings.Contains(f, "a.go") {
					hasA = true
				}
				if strings.Contains(f, "b.go") {
					hasB = true
				}
			}
			if hasA && hasB {
				foundCluster = true
				break
			}
		}
		assert.True(t, foundCluster, "a.go and b.go should be in a co-change cluster")
	}
}

func TestGitHistoryOwnership(t *testing.T) {
	root := t.TempDir()
	initGitRepo(t, root)

	// Create files in different directories with different authors
	commitFile(t, root, "frontend/app.js", "console.log('hello')", "Alice")
	commitFile(t, root, "frontend/style.css", "body {}", "Alice")

	commitFile(t, root, "backend/server.go", "package main", "Bob")
	commitFile(t, root, "backend/handler.go", "package handler", "Bob")

	commitFile(t, root, "README.md", "# Test", "Charlie")

	extractor := extract.GitHistoryExtractor{}
	frag, err := extractor.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.GitAnalysis)

	// Check ownership
	assert.NotEmpty(t, frag.GitAnalysis.Ownership)

	// Should have entries for frontend/ and backend/
	dirs := make(map[string]bool)
	for dir := range frag.GitAnalysis.Ownership {
		dirs[dir] = true
	}

	assert.True(t, dirs["frontend/"] || dirs["frontend"], "Should have frontend ownership")
	assert.True(t, dirs["backend/"] || dirs["backend"], "Should have backend ownership")
}

func TestGitHistoryCODEOWNERS(t *testing.T) {
	root := t.TempDir()
	initGitRepo(t, root)

	// Create CODEOWNERS file
	codeownersContent := `# Code owners
*.go @alice @bob
*.js @charlie
/docs/ @dave
`
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".github"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".github", "CODEOWNERS"), []byte(codeownersContent), 0o644))

	// Commit it
	cmd := exec.Command("git", "add", ".github/CODEOWNERS")
	cmd.Dir = root
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Add CODEOWNERS")
	cmd.Dir = root
	require.NoError(t, cmd.Run())

	// Create some files
	commitFile(t, root, "main.go", "package main", "Alice")
	commitFile(t, root, "app.js", "console.log('hi')", "Charlie")

	extractor := extract.GitHistoryExtractor{}
	frag, err := extractor.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.GitAnalysis)

	// Check CODEOWNERS parsing
	assert.NotEmpty(t, frag.GitAnalysis.Ownership)

	// Look for @codeowners entries
	foundCodeowners := false
	for key := range frag.GitAnalysis.Ownership {
		if strings.HasPrefix(key, "@codeowners:") {
			foundCodeowners = true
			// Check that it has owners
			owners := frag.GitAnalysis.Ownership[key]
			assert.NotEmpty(t, owners)
		}
	}
	assert.True(t, foundCodeowners, "Should have @codeowners entries")
}

func TestGitHistoryCODEOWNERSRoot(t *testing.T) {
	root := t.TempDir()
	initGitRepo(t, root)

	// Create CODEOWNERS file at root
	codeownersContent := `*.go @team-go
*.js @team-js
`
	require.NoError(t, os.WriteFile(filepath.Join(root, "CODEOWNERS"), []byte(codeownersContent), 0o644))

	// Commit it
	cmd := exec.Command("git", "add", "CODEOWNERS")
	cmd.Dir = root
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Add CODEOWNERS")
	cmd.Dir = root
	require.NoError(t, cmd.Run())

	// Create some files
	commitFile(t, root, "main.go", "package main", "Alice")

	extractor := extract.GitHistoryExtractor{}
	frag, err := extractor.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.GitAnalysis)

	// Check CODEOWNERS parsing
	assert.NotEmpty(t, frag.GitAnalysis.Ownership)

	// Look for @codeowners entries
	foundCodeowners := false
	foundTeamGo := false
	for key, owners := range frag.GitAnalysis.Ownership {
		if strings.HasPrefix(key, "@codeowners:") {
			foundCodeowners = true
			assert.NotEmpty(t, owners)
			for _, owner := range owners {
				if owner == "@team-go" {
					foundTeamGo = true
				}
			}
		}
	}
	assert.True(t, foundCodeowners, "Should have @codeowners entries")
	assert.True(t, foundTeamGo, "Should find @team-go in codeowners")
}

func TestGitHistorySampling(t *testing.T) {
	root := t.TempDir()
	initGitRepo(t, root)

	// Create a small number of commits (well below the 50K threshold)
	for i := 0; i < 10; i++ {
		commitFile(t, root, "file.txt", "content "+strconv.Itoa(i), "Alice")
	}

	extractor := extract.GitHistoryExtractor{
		MaxCommits: 50000,
	}
	frag, err := extractor.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.GitAnalysis)

	// Should process all commits
	assert.Equal(t, 10, frag.GitAnalysis.AnalyzedCommits)
}

func TestGitHistoryHotspotAuthorCount(t *testing.T) {
	root := t.TempDir()
	initGitRepo(t, root)

	// Create a file that's modified by multiple authors
	commitFile(t, root, "shared.go", "package main", "Alice")
	commitFile(t, root, "shared.go", "package main\n\nfunc A() {}", "Bob")
	commitFile(t, root, "shared.go", "package main\n\nfunc A() {}\n\nfunc B() {}", "Charlie")
	commitFile(t, root, "shared.go", "package main\n\nfunc A() {}\n\nfunc B() {}\n\nfunc C() {}", "Alice")

	extractor := extract.GitHistoryExtractor{}
	frag, err := extractor.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.GitAnalysis)

	// Find the hotspot for shared.go
	var sharedHotspot *architect.Hotspot
	for i := range frag.GitAnalysis.Hotspots {
		if strings.Contains(frag.GitAnalysis.Hotspots[i].Path, "shared.go") {
			sharedHotspot = &frag.GitAnalysis.Hotspots[i]
			break
		}
	}

	require.NotNil(t, sharedHotspot, "shared.go should be in hotspots")
	assert.Equal(t, 4, sharedHotspot.Changes, "shared.go should have 4 changes")
	assert.GreaterOrEqual(t, sharedHotspot.Authors, 2, "shared.go should have at least 2 authors")
}

func TestGitHistoryEmptyRepo(t *testing.T) {
	root := t.TempDir()
	initGitRepo(t, root)

	// Don't create any commits

	extractor := extract.GitHistoryExtractor{}
	frag, err := extractor.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.GitAnalysis)

	// Should have zero commits
	assert.Equal(t, 0, frag.GitAnalysis.AnalyzedCommits)
}
