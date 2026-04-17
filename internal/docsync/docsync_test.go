package docsync

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// -----------------------------------------------------------------------
// Link detection tests (CheckConsistency / checkMarkdownLinks)
// -----------------------------------------------------------------------

func TestCheckConsistencyBrokenLinkStrict(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs", "workstreams", "backlog"))
	mkdir(t, filepath.Join(root, "docs", "workstreams"))
	mkdir(t, filepath.Join(root, "docs", "roadmap"))

	write(t, filepath.Join(root, "docs", "workstreams", "INDEX.md"), `# Workstream Index

| Feature | Description | Workstreams | Status |
|---------|-------------|-------------|--------|
| **F100** | Example | 00-100-01 | Backlog |

## Workstream Status

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-100-01 | F100 | Example | Backlog |
`)

	write(t, filepath.Join(root, "docs", "roadmap", "ROADMAP.md"), `# Roadmap

- **F100** — Example
`)

	write(t, filepath.Join(root, "docs", "workstreams", "backlog", "00-100-01.md"), `---
ws_id: 00-100-01
feature_id: F100
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-100-01: Example

## Beads

- sdplab-123: Example

## Acceptance Criteria

- [ ] Example AC
`)

	write(t, filepath.Join(root, "docs", "guide.md"), `See [missing](./missing.md).`)

	report, err := CheckConsistency(root, true)
	if err != nil {
		t.Fatalf("CheckConsistency error: %v", err)
	}
	if !report.HasErrors() {
		t.Fatalf("expected strict broken link error, got: %+v", report.Issues)
	}
}

func TestCheckConsistencyRootLevelBrokenLink(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs", "workstreams", "backlog"))
	mkdir(t, filepath.Join(root, "docs", "workstreams"))
	mkdir(t, filepath.Join(root, "docs", "roadmap"))

	// Minimum workstream/roadmap scaffolding so ValidateProtocol does not fail.
	write(t, filepath.Join(root, "docs", "workstreams", "INDEX.md"), `# Workstream Index

| Feature | Description | Workstreams | Status |
|---------|-------------|-------------|--------|
| **F100** | Example | 00-100-01 | Backlog |

## Workstream Status

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-100-01 | F100 | Example | Backlog |
`)

	write(t, filepath.Join(root, "docs", "roadmap", "ROADMAP.md"), `# Roadmap

- **F100** — Example
`)

	write(t, filepath.Join(root, "docs", "workstreams", "backlog", "00-100-01.md"), `---
ws_id: 00-100-01
feature_id: F100
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-100-01: Example

## Beads

- sdplab-123: Example

## Acceptance Criteria

- [ ] Example AC
`)

	// Root-level README.md with broken link — must be caught.
	write(t, filepath.Join(root, "README.md"), `# Root README

See [broken](./does-not-exist.md) for details.
`)

	report, err := CheckConsistency(root, true)
	if err != nil {
		t.Fatalf("CheckConsistency error: %v", err)
	}
	var sawRoot bool
	for _, i := range report.Issues {
		if i.File == "README.md" && strings.Contains(i.Message, "does-not-exist.md") {
			sawRoot = true
			break
		}
	}
	if !sawRoot {
		t.Fatalf("expected broken link in root README.md to be reported, got: %+v", report.Issues)
	}
}

func TestUpdateChangelog(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	write(t, filepath.Join(root, "docs", "note.md"), "first\n")

	git(t, root, "init")
	git(t, root, "config", "user.email", "test@example.com")
	git(t, root, "config", "user.name", "Test User")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "initial")

	write(t, filepath.Join(root, "docs", "note.md"), "second\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "docs update")

	path, err := UpdateChangelog(root, "HEAD~1..HEAD")
	if err != nil {
		t.Fatalf("UpdateChangelog error: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join("docs", "CHANGELOG.md")) {
		t.Fatalf("unexpected changelog path: %s", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read changelog: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, "### Commits") {
		t.Fatalf("changelog missing commits section: %s", content)
	}
	if !strings.Contains(content, "docs update") {
		t.Fatalf("changelog missing latest commit message: %s", content)
	}
}

func TestCheckMarkdownLinks_BrokenLink(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "guide.md"), `# Guide

See [missing](./absent.md) for details.
Also check [another](../nonexistent.md).
`)

	issues, err := checkMarkdownLinks(root, false)
	if err != nil {
		t.Fatalf("checkMarkdownLinks error: %v", err)
	}

	found := map[string]bool{}
	for _, i := range issues {
		found[i.Message] = true
	}
	if !found["broken local link: ./absent.md"] {
		t.Errorf("expected broken link ./absent.md, got issues: %+v", issues)
	}
	if !found["broken local link: ../nonexistent.md"] {
		t.Errorf("expected broken link ../nonexistent.md, got issues: %+v", issues)
	}
}

func TestCheckMarkdownLinks_BrokenLinkStrict(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "guide.md"), `See [gone](./gone.md).`)

	issues, err := checkMarkdownLinks(root, true)
	if err != nil {
		t.Fatalf("checkMarkdownLinks error: %v", err)
	}
	if len(issues) == 0 {
		t.Fatal("expected at least one issue for broken link")
	}
	if issues[0].Severity != "error" {
		t.Errorf("expected severity 'error' in strict mode, got %q", issues[0].Severity)
	}
}

func TestCheckMarkdownLinks_BrokenLinkWarning(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "guide.md"), `See [gone](./gone.md).`)

	issues, err := checkMarkdownLinks(root, false)
	if err != nil {
		t.Fatalf("checkMarkdownLinks error: %v", err)
	}
	if len(issues) == 0 {
		t.Fatal("expected at least one issue for broken link")
	}
	if issues[0].Severity != "warning" {
		t.Errorf("expected severity 'warning' in non-strict mode, got %q", issues[0].Severity)
	}
}

func TestCheckMarkdownLinks_ValidLink(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "target.md"), "# Target\n")
	write(t, filepath.Join(root, "docs", "guide.md"), `See [target](./target.md) for details.`)

	issues, err := checkMarkdownLinks(root, false)
	if err != nil {
		t.Fatalf("checkMarkdownLinks error: %v", err)
	}
	for _, i := range issues {
		if strings.Contains(i.Message, "target.md") {
			t.Errorf("valid link reported as issue: %+v", i)
		}
	}
}

func TestCheckMarkdownLinks_ValidLinkSubdirectory(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs", "sub"))

	write(t, filepath.Join(root, "docs", "sub", "deep.md"), "# Deep\n")
	write(t, filepath.Join(root, "docs", "guide.md"), `See [deep](./sub/deep.md).`)

	issues, err := checkMarkdownLinks(root, false)
	if err != nil {
		t.Fatalf("checkMarkdownLinks error: %v", err)
	}
	for _, i := range issues {
		if strings.Contains(i.Message, "deep.md") {
			t.Errorf("valid subdirectory link reported as issue: %+v", i)
		}
	}
}

func TestCheckMarkdownLinks_ExternalSkipped(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"https", `[remote](https://example.com/page)`},
		{"http", `[remote](http://example.com/page)`},
		{"mailto", `[email](mailto:user@example.com)`},
		{"mixed", "See [https](https://a.com) and [http](http://b.com) and [mail](mailto:c@d.com)."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			mkdir(t, filepath.Join(root, "docs"))
			write(t, filepath.Join(root, "docs", "ext.md"), tc.input)

			issues, err := checkMarkdownLinks(root, false)
			if err != nil {
				t.Fatalf("checkMarkdownLinks error: %v", err)
			}
			for _, i := range issues {
				if strings.Contains(i.Message, "broken local link") {
					t.Errorf("external link reported as broken: %+v", i)
				}
			}
		})
	}
}

func TestCheckMarkdownLinks_TrailingSlash(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	// filepath.Clean resolves "page.md/" to "page.md", so a trailing slash on
	// a file that exists resolves fine. Test with a directory-style path that
	// doesn't exist instead.
	write(t, filepath.Join(root, "docs", "guide.md"), `See [page](./missing-dir/) for details.`)

	issues, err := checkMarkdownLinks(root, false)
	if err != nil {
		t.Fatalf("checkMarkdownLinks error: %v", err)
	}
	if len(issues) == 0 {
		t.Error("expected broken link for trailing slash directory path, got no issues")
	}
}

func TestCheckMarkdownLinks_AnchorStripped(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "target.md"), "# Target\n\n## Section\n")
	write(t, filepath.Join(root, "docs", "guide.md"), `See [section](./target.md#section).`)

	issues, err := checkMarkdownLinks(root, false)
	if err != nil {
		t.Fatalf("checkMarkdownLinks error: %v", err)
	}
	for _, i := range issues {
		if strings.Contains(i.Message, "target.md") {
			t.Errorf("valid anchor link reported as issue: %+v", i)
		}
	}
}

func TestCheckMarkdownLinks_AnchorOnlySkipped(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "guide.md"), `See [section](#section).`)

	issues, err := checkMarkdownLinks(root, false)
	if err != nil {
		t.Fatalf("checkMarkdownLinks error: %v", err)
	}
	for _, i := range issues {
		if strings.Contains(i.Message, "broken local link") {
			t.Errorf("anchor-only link reported as broken: %+v", i)
		}
	}
}

func TestCheckMarkdownLinks_AnchorWithMissingFile(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	// Link has anchor AND file doesn't exist
	write(t, filepath.Join(root, "docs", "guide.md"), `See [section](./missing.md#section).`)

	issues, err := checkMarkdownLinks(root, false)
	if err != nil {
		t.Fatalf("checkMarkdownLinks error: %v", err)
	}
	found := false
	for _, i := range issues {
		if strings.Contains(i.Message, "missing.md") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected broken link for missing.md with anchor")
	}
}

func TestCheckMarkdownLinks_RelativePathResolution(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs", "a", "b"))
	mkdir(t, filepath.Join(root, "docs", "c"))

	write(t, filepath.Join(root, "docs", "c", "target.md"), "# Target\n")
	// Link from docs/a/b/deep.md to docs/c/target.md using ../../c/target.md
	write(t, filepath.Join(root, "docs", "a", "b", "deep.md"), `See [target](../../c/target.md).`)

	issues, err := checkMarkdownLinks(root, false)
	if err != nil {
		t.Fatalf("checkMarkdownLinks error: %v", err)
	}
	for _, i := range issues {
		if strings.Contains(i.Message, "target.md") {
			t.Errorf("valid relative path reported as issue: %+v", i)
		}
	}
}

func TestCheckMarkdownLinks_RelativePathBroken(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs", "a"))

	write(t, filepath.Join(root, "docs", "a", "guide.md"), `See [missing](../../nope.md).`)

	issues, err := checkMarkdownLinks(root, false)
	if err != nil {
		t.Fatalf("checkMarkdownLinks error: %v", err)
	}
	found := false
	for _, i := range issues {
		if strings.Contains(i.Message, "nope.md") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected broken link for ../../nope.md")
	}
}

func TestCheckMarkdownLinks_RootLevelFile(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "README.md"), `See [docs](./docs/guide.md).`)
	// guide.md does not exist

	issues, err := checkMarkdownLinks(root, false)
	if err != nil {
		t.Fatalf("checkMarkdownLinks error: %v", err)
	}
	found := false
	for _, i := range issues {
		if i.File == "README.md" && strings.Contains(i.Message, "guide.md") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected broken link in README.md, got: %+v", issues)
	}
}

// -----------------------------------------------------------------------
// Fix function tests
// -----------------------------------------------------------------------

func TestFixTrailingSlashes(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	mkdir(t, filepath.Join(root, "docs", "sub"))

	write(t, filepath.Join(root, "docs", "sub", "page.md"), "# Page\n")
	write(t, filepath.Join(root, "docs", "guide.md"), `# Guide

See [page](./sub/page.md/) and [also](./sub/page.md/#section).
External [link](https://example.com/) not touched.
`)

	fixes, issues, err := FixTrailingSlashes(root)
	if err != nil {
		t.Fatalf("FixTrailingSlashes error: %v", err)
	}

	if len(fixes) != 2 {
		t.Fatalf("expected 2 trailing-slash fixes, got %d: %+v", len(fixes), fixes)
	}

	for _, f := range fixes {
		if f.Fix != "trailing-slash" {
			t.Errorf("expected fix type 'trailing-slash', got %q", f.Fix)
		}
		if strings.HasSuffix(f.After, "/") {
			t.Errorf("after should not have trailing slash: %q", f.After)
		}
	}

	// Verify file was actually rewritten
	b, _ := os.ReadFile(filepath.Join(root, "docs", "guide.md"))
	content := string(b)
	if strings.Contains(content, "page.md/") {
		t.Errorf("trailing slash not removed from file: %s", content)
	}
	if !strings.Contains(content, "https://example.com/") {
		t.Error("external link was incorrectly modified")
	}
	if len(issues) > 0 {
		t.Errorf("expected no remaining issues, got: %+v", issues)
	}
}

func TestFixTrailingSlashes_NoTrailingSlash(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "page.md"), "# Page\n")
	write(t, filepath.Join(root, "docs", "guide.md"), `See [page](./page.md).`)

	fixes, _, err := FixTrailingSlashes(root)
	if err != nil {
		t.Fatalf("FixTrailingSlashes error: %v", err)
	}
	if len(fixes) != 0 {
		t.Errorf("expected no fixes for clean links, got %d", len(fixes))
	}
}

func TestFixCodeFenceTags(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "example.md"), "# Example\n\n"+("```")+"\npackage main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"+("```")+"\n")

	fixes, _, err := FixCodeFenceTags(root)
	if err != nil {
		t.Fatalf("FixCodeFenceTags error: %v", err)
	}
	if len(fixes) != 1 {
		t.Fatalf("expected 1 fix for untagged Go fence, got %d: %+v", len(fixes), fixes)
	}
	if fixes[0].Fix != "fence-tag" {
		t.Errorf("expected fix type 'fence-tag', got %q", fixes[0].Fix)
	}
	if fixes[0].After != "```go" {
		t.Errorf("expected After='```go', got %q", fixes[0].After)
	}

	// Verify file was actually rewritten
	b, _ := os.ReadFile(filepath.Join(root, "docs", "example.md"))
	content := string(b)
	if !strings.Contains(content, "```go\n") {
		t.Errorf("expected ```go in file, got: %s", content)
	}
}

func TestFixCodeFenceTags_AlreadyTagged(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "example.md"), "```go\nfmt.Println(\"hello\")\n```\n")

	fixes, _, err := FixCodeFenceTags(root)
	if err != nil {
		t.Fatalf("FixCodeFenceTags error: %v", err)
	}
	if len(fixes) != 0 {
		t.Errorf("expected no fixes for already-tagged fences, got %d", len(fixes))
	}
}

func TestFixCodeFenceTags_BashContent(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "scripts.md"), "# Scripts\n\n"+("```")+"\ngo build ./...\ngit status\n"+("```")+"\n")

	fixes, _, err := FixCodeFenceTags(root)
	if err != nil {
		t.Fatalf("FixCodeFenceTags error: %v", err)
	}
	if len(fixes) != 1 {
		t.Fatalf("expected 1 fix for bash fence, got %d", len(fixes))
	}
	if fixes[0].After != "```bash" {
		t.Errorf("expected After='```bash', got %q", fixes[0].After)
	}
}

func TestFixCodeFenceTags_YAMLContent(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "config.md"), "# Config\n\n"+("```")+"\nname: test\nversion: 1.0\nitems:\n  - foo\n  - bar\n"+("```")+"\n")

	fixes, _, err := FixCodeFenceTags(root)
	if err != nil {
		t.Fatalf("FixCodeFenceTags error: %v", err)
	}
	if len(fixes) != 1 {
		t.Fatalf("expected 1 fix for yaml fence, got %d", len(fixes))
	}
	if fixes[0].After != "```yaml" {
		t.Errorf("expected After='```yaml', got %q", fixes[0].After)
	}
}

func TestFixCodeFenceTags_AmbiguousUntouched(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "misc.md"), "# Misc\n\n"+("```")+"\nsome plain text\nno language hints\n"+("```")+"\n")

	fixes, _, err := FixCodeFenceTags(root)
	if err != nil {
		t.Fatalf("FixCodeFenceTags error: %v", err)
	}
	if len(fixes) != 0 {
		t.Errorf("expected no fixes for ambiguous content, got %d", len(fixes))
	}
}

func TestFixRelativeLinks_MovedFile(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	mkdir(t, filepath.Join(root, "docs", "plans"))

	// Initialize git repo
	git(t, root, "init")
	git(t, root, "config", "user.email", "test@example.com")
	git(t, root, "config", "user.name", "Test User")

	// Create file at original location
	write(t, filepath.Join(root, "docs", "old-name.md"), "# Old Name\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "add original file")

	// Rename file via git mv
	git(t, root, "mv", "docs/old-name.md", "docs/plans/new-name.md")
	git(t, root, "commit", "-m", "rename file")

	// Create a guide that still links to the old location
	write(t, filepath.Join(root, "docs", "guide.md"), `See [old](./old-name.md) for details.`)

	fixes, issues, err := FixRelativeLinks(root)
	if err != nil {
		t.Fatalf("FixRelativeLinks error: %v", err)
	}

	if len(fixes) == 0 {
		t.Fatalf("expected at least one fix for moved file, got none; issues: %+v", issues)
	}

	fix := fixes[0]
	if fix.Fix != "relative-link" {
		t.Errorf("expected fix type 'relative-link', got %q", fix.Fix)
	}
	if fix.Before != "./old-name.md" {
		t.Errorf("expected Before='./old-name.md', got %q", fix.Before)
	}
	if !strings.Contains(fix.After, "plans/new-name.md") {
		t.Errorf("expected After to contain 'plans/new-name.md', got %q", fix.After)
	}

	// Verify the file was rewritten
	b, _ := os.ReadFile(filepath.Join(root, "docs", "guide.md"))
	content := string(b)
	if strings.Contains(content, "./old-name.md") {
		t.Errorf("old link not replaced in file: %s", content)
	}
}

func TestFixRelativeLinks_NoRename(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "target.md"), "# Target\n")
	write(t, filepath.Join(root, "docs", "guide.md"), `See [target](./target.md).`)

	fixes, issues, err := FixRelativeLinks(root)
	if err != nil {
		t.Fatalf("FixRelativeLinks error: %v", err)
	}
	if len(fixes) != 0 {
		t.Errorf("expected no fixes for valid links, got %d", len(fixes))
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues for valid links, got %d", len(issues))
	}
}

func TestFixConsistency_Report(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	mkdir(t, filepath.Join(root, "docs", "workstreams", "backlog"))
	mkdir(t, filepath.Join(root, "docs", "roadmap"))

	// Minimal protocol scaffolding
	write(t, filepath.Join(root, "docs", "workstreams", "INDEX.md"), `# Workstream Index

| Feature | Description | Workstreams | Status |
|---------|-------------|-------------|--------|
| **F100** | Example | 00-100-01 | Backlog |

## Workstream Status

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-100-01 | F100 | Example | Backlog |
`)

	write(t, filepath.Join(root, "docs", "roadmap", "ROADMAP.md"), `# Roadmap

- **F100** — Example
`)

	write(t, filepath.Join(root, "docs", "workstreams", "backlog", "00-100-01.md"), `---
ws_id: 00-100-01
feature_id: F100
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-100-01: Example

## Beads

- sdplab-123: Example

## Acceptance Criteria

- [ ] Example AC
`)

	// A file with trailing slash link and a broken link
	mkdir(t, filepath.Join(root, "docs", "sub"))
	write(t, filepath.Join(root, "docs", "sub", "page.md"), "# Page\n")
	write(t, filepath.Join(root, "docs", "guide.md"), `# Guide

See [page](./sub/page.md/) and [broken](./missing.md).
`)

	report, err := FixConsistency(root, false)
	if err != nil {
		t.Fatalf("FixConsistency error: %v", err)
	}

	// Should have at least one trailing-slash fix
	hasSlashFix := false
	for _, f := range report.Fixed {
		if f.Fix == "trailing-slash" {
			hasSlashFix = true
			break
		}
	}
	if !hasSlashFix {
		t.Errorf("expected at least one trailing-slash fix, got: %+v", report.Fixed)
	}

	// Should have unresolved issues (broken link)
	if len(report.Unresolved) == 0 {
		t.Error("expected unresolved issues for broken link")
	}
}

func TestFixConsistency_ComplexIssues(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	mkdir(t, filepath.Join(root, "docs", "workstreams", "backlog"))
	mkdir(t, filepath.Join(root, "docs", "workstreams"))
	mkdir(t, filepath.Join(root, "docs", "roadmap"))

	// Minimal protocol scaffolding
	write(t, filepath.Join(root, "docs", "workstreams", "INDEX.md"), `# Workstream Index

| Feature | Description | Workstreams | Status |
|---------|-------------|-------------|--------|
| **F100** | Example | 00-100-01 | Backlog |

## Workstream Status

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-100-01 | F100 | Example | Backlog |
`)

	write(t, filepath.Join(root, "docs", "roadmap", "ROADMAP.md"), `# Roadmap

- **F100** — Example
`)

	write(t, filepath.Join(root, "docs", "workstreams", "backlog", "00-100-01.md"), `---
ws_id: 00-100-01
feature_id: F100
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-100-01: Example

## Beads

- sdplab-123: Example

## Acceptance Criteria

- [ ] Example AC
`)

	// Multiple issues in one file
	write(t, filepath.Join(root, "docs", "messy.md"), `# Messy

See [broken1](./a.md/) and [broken2](./b.md/).
Also [gone](./nonexistent.md).
External [ok](https://example.com/) is fine.
`)

	report, err := FixConsistency(root, false)
	if err != nil {
		t.Fatalf("FixConsistency error: %v", err)
	}

	// Trailing slash fixes should be applied
	slashFixes := 0
	for _, f := range report.Fixed {
		if f.Fix == "trailing-slash" {
			slashFixes++
		}
	}
	if slashFixes != 2 {
		t.Errorf("expected 2 trailing-slash fixes, got %d", slashFixes)
	}

	// The nonexistent.md link should remain unresolved
	foundUnresolved := false
	for _, i := range report.Unresolved {
		if strings.Contains(i.Message, "nonexistent.md") {
			foundUnresolved = true
			break
		}
	}
	if !foundUnresolved {
		t.Errorf("expected unresolved issue for nonexistent.md, got: %+v", report.Unresolved)
	}
}

func TestFixConsistency_EmptyDocs(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	mkdir(t, filepath.Join(root, "docs", "workstreams", "backlog"))
	mkdir(t, filepath.Join(root, "docs", "workstreams"))
	mkdir(t, filepath.Join(root, "docs", "roadmap"))

	// Minimal protocol scaffolding
	write(t, filepath.Join(root, "docs", "workstreams", "INDEX.md"), `# Workstream Index

| Feature | Description | Workstreams | Status |
|---------|-------------|-------------|--------|
| **F100** | Example | 00-100-01 | Backlog |

## Workstream Status

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-100-01 | F100 | Example | Backlog |
`)

	write(t, filepath.Join(root, "docs", "roadmap", "ROADMAP.md"), `# Roadmap

- **F100** — Example
`)

	write(t, filepath.Join(root, "docs", "workstreams", "backlog", "00-100-01.md"), `---
ws_id: 00-100-01
feature_id: F100
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-100-01: Example

## Beads

- sdplab-123: Example

## Acceptance Criteria

- [ ] Example AC
`)

	report, err := FixConsistency(root, false)
	if err != nil {
		t.Fatalf("FixConsistency error: %v", err)
	}
	if len(report.Fixed) != 0 {
		t.Errorf("expected no fixes for empty docs, got %d", len(report.Fixed))
	}
}

func TestFixConsistency_AllValid(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	mkdir(t, filepath.Join(root, "docs", "workstreams", "backlog"))
	mkdir(t, filepath.Join(root, "docs", "workstreams"))
	mkdir(t, filepath.Join(root, "docs", "roadmap"))

	// Protocol scaffolding required by CheckConsistency
	write(t, filepath.Join(root, "docs", "workstreams", "INDEX.md"), `# Workstream Index

| Feature | Description | Workstreams | Status |
|---------|-------------|-------------|--------|
| **F100** | Example | 00-100-01 | Backlog |

## Workstream Status

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-100-01 | F100 | Example | Backlog |
`)

	write(t, filepath.Join(root, "docs", "roadmap", "ROADMAP.md"), `# Roadmap

- **F100** — Example
`)

	write(t, filepath.Join(root, "docs", "workstreams", "backlog", "00-100-01.md"), `---
ws_id: 00-100-01
feature_id: F100
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-100-01: Example

## Beads

- sdplab-123: Example

## Acceptance Criteria

- [ ] Example AC
`)

	write(t, filepath.Join(root, "docs", "target.md"), "# Target\n")
	write(t, filepath.Join(root, "docs", "guide.md"), `See [target](./target.md) and [remote](https://example.com).`)

	report, err := FixConsistency(root, false)
	if err != nil {
		t.Fatalf("FixConsistency error: %v", err)
	}
	if len(report.Fixed) != 0 {
		t.Errorf("expected no fixes for valid docs, got %d", len(report.Fixed))
	}
}

// -----------------------------------------------------------------------
// Edge-case tests (F129-06)
// -----------------------------------------------------------------------

func TestFixTrailingSlashes_WithAnchor(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	mkdir(t, filepath.Join(root, "docs", "sub"))

	write(t, filepath.Join(root, "docs", "sub", "page.md"), "# Page\n\n## Intro\n")
	write(t, filepath.Join(root, "docs", "guide.md"), `# Guide

See [intro](./sub/page.md/#intro) for details.
Also [plain](./sub/page.md/) without anchor.
`)

	fixes, _, err := FixTrailingSlashes(root)
	if err != nil {
		t.Fatalf("FixTrailingSlashes error: %v", err)
	}
	if len(fixes) != 2 {
		t.Fatalf("expected 2 fixes, got %d: %+v", len(fixes), fixes)
	}

	// Verify anchors are preserved and slashes are removed.
	b, _ := os.ReadFile(filepath.Join(root, "docs", "guide.md"))
	content := string(b)

	if !strings.Contains(content, "./sub/page.md#intro") {
		t.Errorf("expected anchor preserved as #intro, got: %s", content)
	}
	if strings.Contains(content, "page.md/") {
		t.Errorf("trailing slash not removed: %s", content)
	}
}

func TestFixTrailingSlashes_Subdirectory(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	mkdir(t, filepath.Join(root, "docs", "plans"))

	write(t, filepath.Join(root, "docs", "guide.md"), `See [plans](./plans/) for details.`)

	fixes, _, err := FixTrailingSlashes(root)
	if err != nil {
		t.Fatalf("FixTrailingSlashes error: %v", err)
	}
	if len(fixes) != 1 {
		t.Fatalf("expected 1 fix for subdirectory trailing slash, got %d: %+v", len(fixes), fixes)
	}
	if fixes[0].Before != "./plans/" {
		t.Errorf("expected Before='./plans/', got %q", fixes[0].Before)
	}
	if fixes[0].After != "./plans" {
		t.Errorf("expected After='./plans', got %q", fixes[0].After)
	}

	b, _ := os.ReadFile(filepath.Join(root, "docs", "guide.md"))
	if strings.Contains(string(b), "./plans/") {
		t.Error("trailing slash not removed from subdirectory link")
	}
}

func TestFixCodeFenceTags_MultipleFences(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "multi.md"), `# Multi

`+"```"+`
package main
func main() {}
`+"```"+`

Some text between fences.

`+"```bash"+`
echo "already tagged"
`+"```"+`

More text.

`+"```"+`
go build ./...
git status
`+"```"+`
`)

	fixes, _, err := FixCodeFenceTags(root)
	if err != nil {
		t.Fatalf("FixCodeFenceTags error: %v", err)
	}
	if len(fixes) != 2 {
		t.Fatalf("expected 2 fixes (go + bash), got %d: %+v", len(fixes), fixes)
	}

	// Verify both types were detected.
	kinds := map[string]bool{}
	for _, f := range fixes {
		kinds[f.After] = true
	}
	if !kinds["```go"] {
		t.Error("expected a ```go fix")
	}
	if !kinds["```bash"] {
		t.Error("expected a ```bash fix")
	}

	// Verify file content: both untagged fences should now be tagged.
	b, _ := os.ReadFile(filepath.Join(root, "docs", "multi.md"))
	content := string(b)
	if strings.Count(content, "```go") != 1 {
		t.Errorf("expected exactly 1 ```go occurrence, got: %s", content)
	}
	if strings.Count(content, "```bash") != 2 {
		t.Errorf("expected 2 ```bash occurrences (original + inferred), got: %s", content)
	}
}

func TestFixCodeFenceTags_NestedBackticks(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	write(t, filepath.Join(root, "docs", "nested.md"), "# Nested\n\n"+""+"```"+"\n"+"Use backtick: "+"`cmd`"+" in text.\n"+"Also "+"``double``"+" backticks.\n"+"go test ./...\n"+""+"```"+"\n")

	fixes, _, err := FixCodeFenceTags(root)
	if err != nil {
		t.Fatalf("FixCodeFenceTags error: %v", err)
	}
	if len(fixes) != 1 {
		t.Fatalf("expected 1 fix for bash content with backticks, got %d: %+v", len(fixes), fixes)
	}
	if fixes[0].After != "```bash" {
		t.Errorf("expected ```bash, got %q", fixes[0].After)
	}
}

func TestFixRelativeLinks_MultipleRenames(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	mkdir(t, filepath.Join(root, "docs", "plans"))

	git(t, root, "init")
	git(t, root, "config", "user.email", "test@example.com")
	git(t, root, "config", "user.name", "Test User")

	// Create and commit two files at original locations.
	write(t, filepath.Join(root, "docs", "alpha.md"), "# Alpha\n")
	write(t, filepath.Join(root, "docs", "beta.md"), "# Beta\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "add alpha and beta")

	// Rename both via git mv.
	git(t, root, "mv", "docs/alpha.md", "docs/plans/alpha-renamed.md")
	git(t, root, "mv", "docs/beta.md", "docs/plans/beta-renamed.md")
	git(t, root, "commit", "-m", "rename both files")

	// Guide links to both old locations.
	write(t, filepath.Join(root, "docs", "guide.md"), `See [alpha](./alpha.md) and [beta](./beta.md).`)

	fixes, issues, err := FixRelativeLinks(root)
	if err != nil {
		t.Fatalf("FixRelativeLinks error: %v", err)
	}
	if len(fixes) != 2 {
		t.Fatalf("expected 2 fixes for renamed files, got %d; issues: %+v", len(fixes), issues)
	}

	// Verify both files are referenced correctly in the updated guide.
	b, _ := os.ReadFile(filepath.Join(root, "docs", "guide.md"))
	content := string(b)
	if strings.Contains(content, "./alpha.md") || strings.Contains(content, "./beta.md") {
		t.Errorf("old links not replaced: %s", content)
	}
	if !strings.Contains(content, "alpha-renamed.md") || !strings.Contains(content, "beta-renamed.md") {
		t.Errorf("expected new file names in guide, got: %s", content)
	}
}

func TestFixRelativeLinks_DirectoryMove(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	mkdir(t, filepath.Join(root, "docs", "old-dir"))
	mkdir(t, filepath.Join(root, "docs", "new-dir"))

	git(t, root, "init")
	git(t, root, "config", "user.email", "test@example.com")
	git(t, root, "config", "user.name", "Test User")

	write(t, filepath.Join(root, "docs", "old-dir", "plan.md"), "# Plan\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "add plan in old-dir")

	git(t, root, "mv", "docs/old-dir/plan.md", "docs/new-dir/plan.md")
	git(t, root, "commit", "-m", "move plan to new-dir")

	write(t, filepath.Join(root, "docs", "guide.md"), `See [plan](./old-dir/plan.md).`)

	fixes, _, err := FixRelativeLinks(root)
	if err != nil {
		t.Fatalf("FixRelativeLinks error: %v", err)
	}
	if len(fixes) != 1 {
		t.Fatalf("expected 1 fix for directory move, got %d: %+v", len(fixes), fixes)
	}
	if !strings.Contains(fixes[0].After, "new-dir/plan.md") {
		t.Errorf("expected After to contain 'new-dir/plan.md', got %q", fixes[0].After)
	}

	b, _ := os.ReadFile(filepath.Join(root, "docs", "guide.md"))
	content := string(b)
	if strings.Contains(content, "old-dir") {
		t.Errorf("old directory path not replaced: %s", content)
	}
}

func TestCheckMarkdownLinks_SubmoduleSkip(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	mkdir(t, filepath.Join(root, "docs"))

	// Create an empty sdp/ directory (simulating an uninitialized submodule).
	mkdir(t, filepath.Join(root, "sdp"))

	write(t, filepath.Join(root, "docs", "guide.md"), `See [sdp docs](../sdp/docs/readme.md) for submodule content.`)

	issues, err := checkMarkdownLinks(root, false)
	if err != nil {
		t.Fatalf("checkMarkdownLinks error: %v", err)
	}
	for _, i := range issues {
		if strings.Contains(i.Message, "sdp") && strings.Contains(i.Message, "broken local link") {
			t.Errorf("submodule link should be skipped when uninitialized, got issue: %+v", i)
		}
	}
}

func TestFixConsistency_Idempotent(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	mkdir(t, filepath.Join(root, "docs", "workstreams", "backlog"))
	mkdir(t, filepath.Join(root, "docs", "workstreams"))
	mkdir(t, filepath.Join(root, "docs", "roadmap"))
	mkdir(t, filepath.Join(root, "docs", "sub"))

	// Protocol scaffolding.
	write(t, filepath.Join(root, "docs", "workstreams", "INDEX.md"), `# Workstream Index

| Feature | Description | Workstreams | Status |
|---------|-------------|-------------|--------|
| **F100** | Example | 00-100-01 | Backlog |

## Workstream Status

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-100-01 | F100 | Example | Backlog |
`)

	write(t, filepath.Join(root, "docs", "roadmap", "ROADMAP.md"), `# Roadmap

- **F100** — Example
`)

	write(t, filepath.Join(root, "docs", "workstreams", "backlog", "00-100-01.md"), `---
ws_id: 00-100-01
feature_id: F100
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-100-01: Example

## Beads

- sdplab-123: Example

## Acceptance Criteria

- [ ] Example AC
`)

	// File with a trailing slash link.
	write(t, filepath.Join(root, "docs", "sub", "page.md"), "# Page\n")
	write(t, filepath.Join(root, "docs", "guide.md"), `See [page](./sub/page.md/).`)

	// First run: should apply fixes.
	report1, err := FixConsistency(root, false)
	if err != nil {
		t.Fatalf("first FixConsistency error: %v", err)
	}
	if len(report1.Fixed) == 0 {
		t.Fatal("first run expected at least one fix")
	}

	// Second run: should produce zero fixes (idempotent).
	report2, err := FixConsistency(root, false)
	if err != nil {
		t.Fatalf("second FixConsistency error: %v", err)
	}
	if len(report2.Fixed) != 0 {
		t.Errorf("second run expected 0 fixes (idempotent), got %d: %+v", len(report2.Fixed), report2.Fixed)
	}
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

func mkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
}

func write(t *testing.T, p, content string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v; out=%s", args, err, string(out))
	}
}

// -----------------------------------------------------------------------
// inferCodeLanguage tests
// -----------------------------------------------------------------------

func TestInferCodeLanguage(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected string
	}{
		{
			name: "go package and func",
			lines: []string{
				"package main",
				"",
				"func main() {",
				"\tfmt.Println(\"hi\")",
				"}",
			},
			expected: "go",
		},
		{
			name: "go with :=",
			lines: []string{
				"name := \"world\"",
				"fmt.Println(name)",
			},
			expected: "go",
		},
		{
			name: "bash shebang",
			lines: []string{
				"#!/bin/bash",
				"echo hello",
			},
			expected: "bash",
		},
		{
			name: "bash commands",
			lines: []string{
				"go build ./...",
				"git status",
			},
			expected: "bash",
		},
		{
			name: "bash with $ prompt",
			lines: []string{
				"$ make test",
				"$ go run .",
			},
			expected: "bash",
		},
		{
			name: "bash export",
			lines: []string{
				"export PATH=$PATH:/usr/local/bin",
			},
			expected: "bash",
		},
		{
			name: "yaml key-value",
			lines: []string{
				"name: test",
				"version: 1.0",
				"- item1",
			},
			expected: "yaml",
		},
		{
			name: "yaml with list",
			lines: []string{
				"items:",
				"  - foo",
				"  - bar",
			},
			expected: "yaml",
		},
		{
			name:     "ambiguous plain text",
			lines:    []string{"some random text", "no language hints"},
			expected: "",
		},
		{
			name:     "empty lines only",
			lines:    []string{"", "", ""},
			expected: "",
		},
		{
			name: "bash sudo/chmod",
			lines: []string{
				"sudo apt-get update",
				"chmod +x script.sh",
			},
			expected: "bash",
		},
		{
			name: "bash cd/rm",
			lines: []string{
				"cd /tmp",
				"rm -rf build/",
			},
			expected: "bash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferCodeLanguage(tt.lines)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

// -----------------------------------------------------------------------
// Regression tests for F125 fixes
// -----------------------------------------------------------------------

// TestFixRelativeLinks_SkipsUninitSubmodule verifies that FixRelativeLinks
// does not attempt to resolve links into an uninitialized sdp/ submodule.
func TestFixRelativeLinks_SkipsUninitSubmodule(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	mkdir(t, filepath.Join(root, "sdp")) // empty dir = uninitialized submodule

	write(t, filepath.Join(root, "docs", "guide.md"), `See [sdp docs](../sdp/docs/readme.md).`)

	fixes, issues, err := FixRelativeLinks(root)
	if err != nil {
		t.Fatalf("FixRelativeLinks error: %v", err)
	}
	if len(fixes) != 0 {
		t.Errorf("expected no fixes for submodule links, got %d: %+v", len(fixes), fixes)
	}
	for _, iss := range issues {
		if strings.Contains(iss.Message, "sdp") {
			t.Errorf("submodule link should be skipped, got issue: %+v", iss)
		}
	}
}

// TestDeduplicateIssues verifies that deduplicateIssues removes duplicates
// while preserving order of first occurrence.
func TestDeduplicateIssues(t *testing.T) {
	issues := []Issue{
		{Severity: "warning", File: "a.md", Message: "broken local link: ./x.md"},
		{Severity: "warning", File: "a.md", Message: "broken local link: ./x.md"},
		{Severity: "error", File: "b.md", Message: "broken local link: ./y.md"},
		{Severity: "warning", File: "a.md", Message: "broken local link: ./x.md"},
		{Severity: "error", File: "b.md", Message: "broken local link: ./z.md"},
	}
	deduped := deduplicateIssues(issues)
	if len(deduped) != 3 {
		t.Fatalf("expected 3 unique issues, got %d: %+v", len(deduped), deduped)
	}
	if deduped[0].Message != "broken local link: ./x.md" {
		t.Errorf("first issue unexpected: %+v", deduped[0])
	}
	if deduped[1].Message != "broken local link: ./y.md" {
		t.Errorf("second issue unexpected: %+v", deduped[1])
	}
	if deduped[2].Message != "broken local link: ./z.md" {
		t.Errorf("third issue unexpected: %+v", deduped[2])
	}
}

// TestDeduplicateIssues_Empty verifies dedup on empty and single-element slices.
func TestDeduplicateIssues_Empty(t *testing.T) {
	if got := deduplicateIssues(nil); len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
	if got := deduplicateIssues([]Issue{}); len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
	single := []Issue{{Severity: "warning", File: "a.md", Message: "msg"}}
	if got := deduplicateIssues(single); len(got) != 1 {
		t.Errorf("expected 1, got %d", len(got))
	}
}

// TestFixConsistency_NoDuplicateUnresolved verifies that FixConsistency does not
// produce duplicate unresolved issues for the same broken link.
func TestFixConsistency_NoDuplicateUnresolved(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))
	mkdir(t, filepath.Join(root, "docs", "workstreams", "backlog"))
	mkdir(t, filepath.Join(root, "docs", "workstreams"))
	mkdir(t, filepath.Join(root, "docs", "roadmap"))

	// Minimal protocol scaffolding
	write(t, filepath.Join(root, "docs", "workstreams", "INDEX.md"), `# Workstream Index

| Feature | Description | Workstreams | Status |
|---------|-------------|-------------|--------|
| **F100** | Example | 00-100-01 | Backlog |

## Workstream Status

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-100-01 | F100 | Example | Backlog |
`)

	write(t, filepath.Join(root, "docs", "roadmap", "ROADMAP.md"), `# Roadmap

- **F100** — Example
`)

	write(t, filepath.Join(root, "docs", "workstreams", "backlog", "00-100-01.md"), `---
ws_id: 00-100-01
feature_id: F100
status: backlog
priority: P1
size: S
depends_on: []
---

# 00-100-01: Example

## Beads

- sdplab-123: Example

## Acceptance Criteria

- [ ] Example AC
`)

	// A file with a broken link that will be reported by both FixRelativeLinks and CheckConsistency
	write(t, filepath.Join(root, "docs", "guide.md"), `See [broken](./nonexistent.md).`)

	report, err := FixConsistency(root, false)
	if err != nil {
		t.Fatalf("FixConsistency error: %v", err)
	}

	// Count occurrences of the broken link message
	count := 0
	for _, iss := range report.Unresolved {
		if strings.Contains(iss.Message, "nonexistent.md") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 unresolved issue for nonexistent.md, got %d: %+v", count, report.Unresolved)
	}
}
