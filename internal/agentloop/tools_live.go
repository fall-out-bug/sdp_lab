package agentloop

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"sdp_dev/internal/control"
)

// BuildLiveTools returns all concrete tools for DefaultPhaseMap.
// web_search is excluded (out of scope); bd_* tools accept nil store (return stub errors).
func BuildLiveTools(projectRoot string, store *control.Store) []Tool {
	return []Tool{
		BashTool(projectRoot),
		ReadFileTool(projectRoot),
		EditFileTool(projectRoot),
		GlobTool(projectRoot),
		GrepTool(projectRoot),
		BdSearchTool(store),
		BdCreateTool(store),
		BdCommentTool(store),
	}
}

// ---------------------------------------------------------------------------
// BashTool
// ---------------------------------------------------------------------------

// BashTool executes shell commands in the working directory.
// Output containing PASS/ok/FAIL is compatible with EvidenceAccumulator (OnToolResult).
func BashTool(workdir string) Tool {
	return Tool{
		Name: "bash",
		Description: "Execute a bash command in the project working directory. " +
			"Returns combined stdout+stderr. Dangerous commands are blocked.",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"command": {"type": "string", "description": "The bash command to execute"},
				"timeout": {"type": "integer", "description": "Timeout in seconds (default 60, max 300)", "default": 60}
			},
			"required": ["command"]
		}`),
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			var a struct {
				Command string `json:"command"`
				Timeout int    `json:"timeout"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", fmt.Errorf("bash: invalid arguments: %w", err)
			}
			if a.Command == "" {
				return "", fmt.Errorf("bash: command is required")
			}
			if err := validateBashCommand(a.Command); err != nil {
				return "", err
			}

			timeoutSec := a.Timeout
			if timeoutSec <= 0 {
				timeoutSec = 60
			}
			if timeoutSec > 300 {
				timeoutSec = 300
			}

			ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, "bash", "-c", a.Command)
			cmd.Dir = workdir

			out, err := cmd.CombinedOutput()
			output := string(out)
			if err != nil {
				// Include the error but still return output so EvidenceAccumulator can inspect it.
				if ctx.Err() == context.DeadlineExceeded {
					return output + "\nTIMEOUT: command exceeded " + fmt.Sprintf("%d", timeoutSec) + "s limit",
						fmt.Errorf("bash: timeout after %ds", timeoutSec)
				}
				return output, fmt.Errorf("bash: %w", err)
			}
			return output, nil
		},
	}
}

// validateBashCommand blocks dangerous shell commands.
//
// NOTE: A regex deny-list is NOT a sandbox. Arbitrary code execution via
// interpreters (python -c, node -e, perl -e) can bypass these checks.
// This is an MVP mitigation; a proper sandbox (containers/seccomp) is
// tracked as Phase 2 (bead sdplab-lm6).
func validateBashCommand(cmd string) error {
	dangerous := []struct {
		pattern string
		reason  string
	}{
		{`rm\s+-rf\s+/(?:\s|$)`, "rm -rf / is not allowed"},
		{`rm\s+-rf\s+~(?:\s|$)`, "rm -rf ~ is not allowed"},
		{`rm\s+-rf\s+\$HOME(?:\s|$)`, "rm -rf $HOME is not allowed"},
		{`chmod\s+777\s+/(?:\s|$)`, "chmod 777 / is not allowed"},
		{`dd\s+if=/dev/zero`, "dd if=/dev/zero is not allowed"},
		{`:\(\)\{.*:\|:`, "fork bomb is not allowed"},
		{`\bmkfs\b`, "mkfs is not allowed"},
		{`>\s*/dev/sda`, "writing to /dev/sda is not allowed"},
		{`(?i)cd\s+/`, "cd to absolute path is not allowed (workdir escape)"},
		{`(?i)cd\s+~`, "cd to home directory is not allowed (workdir escape)"},
		{`(?i)cd\s+\$HOME`, "cd to $HOME is not allowed (workdir escape)"},
		// P0-1 fix: block interpreters that bypass regex deny-lists.
		{`\bpython3?\s+-c\b`, "python -c is not allowed (use bash directly)"},
		{`\bperl\s+-e\b`, "perl -e is not allowed (use bash directly)"},
		{`\bnode\s+-e\b`, "node -e is not allowed (use bash directly)"},
		{`\bruby\s+-e\b`, "ruby -e is not allowed (use bash directly)"},
		// P0-1 fix: block dangerous git commands.
		{`\bgit\s+clean\b`, "git clean is not allowed"},
		{`\bgit\s+reset\s+--hard\b`, "git reset --hard is not allowed"},
		// P0-1 fix: block pipe-to-shell patterns.
		{`curl\b.*\|\s*(ba)?sh`, "curl | sh is not allowed"},
		{`wget\b.*\|\s*(ba)?sh`, "wget | sh is not allowed"},
	}

	for _, d := range dangerous {
		re := regexp.MustCompile(d.pattern)
		if re.MatchString(cmd) {
			return fmt.Errorf("bash: blocked — %s", d.reason)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// ReadFileTool
// ---------------------------------------------------------------------------

// ReadFileTool reads file contents relative to root. Path escape is blocked.
func ReadFileTool(root string) Tool {
	return Tool{
		Name:        "read_file",
		Description: "Read the contents of a file relative to the project root.",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string", "description": "File path relative to project root"}
			},
			"required": ["path"]
		}`),
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			var a struct {
				Path string `json:"path"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", fmt.Errorf("read_file: invalid arguments: %w", err)
			}
			cleanPath, err := safePath(root, a.Path)
			if err != nil {
				return "", err
			}

			data, err := os.ReadFile(cleanPath)
			if err != nil {
				return "", fmt.Errorf("read_file: %w", err)
			}
			return string(data), nil
		},
	}
}

// ---------------------------------------------------------------------------
// EditFileTool
// ---------------------------------------------------------------------------

// EditFileTool writes content to a file relative to root.
// Output format contract: MUST produce exactly "edited: <path>" where <path> is the
// user-supplied relative path (not the cleaned absolute path). The EvidenceAccumulator
// in evidence.go parses this via extractFilePath which takes the last space-delimited
// token. Changing the output prefix or adding extra fields after <path> will silently
// break evidence collection. See extractFilePath in evidence.go for the consumer.
func EditFileTool(root string) Tool {
	return Tool{
		Name:        "edit_file",
		Description: "Write or overwrite a file relative to the project root. Creates parent directories as needed.",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string", "description": "File path relative to project root"},
				"content": {"type": "string", "description": "File content to write"}
			},
			"required": ["path", "content"]
		}`),
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			var a struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", fmt.Errorf("edit_file: invalid arguments: %w", err)
			}
			cleanPath, err := safePath(root, a.Path)
			if err != nil {
				return "", err
			}

			// Ensure parent directories exist.
			if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
				return "", fmt.Errorf("edit_file: mkdir: %w", err)
			}

			if err := os.WriteFile(cleanPath, []byte(a.Content), 0o644); err != nil {
				return "", fmt.Errorf("edit_file: %w", err)
			}

			// EVIDENCE CONTRACT: output must be "edited: <path>" where <path> is a
			// cleaned, slash-normalized relative path. extractFilePath in evidence.go
			// trims the "edited: " prefix. Equivalent paths (./foo, foo, sub/../foo)
			// must produce the same evidence token, so we clean before emitting.
			cleanRel := filepath.ToSlash(filepath.Clean(a.Path))
			return "edited: " + cleanRel, nil
		},
	}
}

// ---------------------------------------------------------------------------
// GlobTool
// ---------------------------------------------------------------------------

// GlobTool returns files matching a glob pattern under root.
func GlobTool(root string) Tool {
	return Tool{
		Name:        "glob",
		Description: "Find files matching a glob pattern under the project root.",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"pattern": {"type": "string", "description": "Glob pattern (e.g. **/*.go)"}
			},
			"required": ["pattern"]
		}`),
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			var a struct {
				Pattern string `json:"pattern"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", fmt.Errorf("glob: invalid arguments: %w", err)
			}
			if a.Pattern == "" {
				return "", fmt.Errorf("glob: pattern is required")
			}

			// Reject absolute paths.
			if filepath.IsAbs(a.Pattern) {
				return "", fmt.Errorf("glob: absolute paths are not allowed: %s", a.Pattern)
			}
			// Reject patterns containing ".." path components.
			for _, component := range strings.Split(a.Pattern, string(os.PathSeparator)) {
				if component == ".." {
					return "", fmt.Errorf("glob: pattern must not contain '..' components: %s", a.Pattern)
				}
			}

			// Resolve pattern relative to root.
			fullPattern := filepath.Join(root, a.Pattern)
			matches, err := filepath.Glob(fullPattern)
			if err != nil {
				return "", fmt.Errorf("glob: %w", err)
			}

			// If no literal matches, try walking with a doublestar approximation.
			if len(matches) == 0 && strings.Contains(a.Pattern, "**") {
				matches, err = globWalk(ctx, root, a.Pattern)
				if err != nil {
					return "", fmt.Errorf("glob walk: %w", err)
				}
			}

			// Filter matches through symlink resolution to prevent escapes.
			canonicalRoot, rootErr := filepath.EvalSymlinks(filepath.Clean(root))
			if rootErr != nil {
				return "", fmt.Errorf("glob: cannot resolve root: %w", rootErr)
			}

			var relPaths []string
			for _, m := range matches {
				resolved, resErr := filepath.EvalSymlinks(m)
				if resErr != nil {
					resolved = m // path may not exist; skip filtering
				}
				if !isUnderRoot(resolved, canonicalRoot) && resolved != canonicalRoot {
					continue // skip paths that escape root via symlink
				}
				rel, err := filepath.Rel(root, m)
				if err != nil {
					rel = m
				}
				relPaths = append(relPaths, rel)
			}

			if len(relPaths) == 0 {
				return "(no matches)", nil
			}
			return strings.Join(relPaths, "\n"), nil
		},
	}
}

// globWalk implements ** support by walking the filesystem.
// It respects context cancellation to stop long traversals early.
func globWalk(ctx context.Context, root, pattern string) ([]string, error) {
	// Convert glob pattern to a regex for ** support.
	// Replace ** with a marker, then * with [^/]*, then restore **.
	seg := strings.ReplaceAll(pattern, "**", "\x00STARSTAR\x00")
	seg = regexp.QuoteMeta(seg)
	seg = strings.ReplaceAll(seg, "\x00STARSTAR\x00", ".*")
	seg = strings.ReplaceAll(seg, `\*`, `[^/]*`)
	seg = "^" + seg + "$"

	re, err := regexp.Compile(seg)
	if err != nil {
		return nil, err
	}

	var matches []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		if re.MatchString(rel) {
			matches = append(matches, path)
		}
		return nil
	})
	return matches, err
}

// ---------------------------------------------------------------------------
// GrepTool
// ---------------------------------------------------------------------------

// GrepTool searches files for a pattern, returning matching lines with file:line prefix.
func GrepTool(root string) Tool {
	return Tool{
		Name:        "grep",
		Description: "Search files under the project root for a text pattern. Returns matching lines with file:line:content format.",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"pattern": {"type": "string", "description": "Text or regex pattern to search for"},
				"path": {"type": "string", "description": "Subdirectory to search in (relative to root). Default: root"}
			},
			"required": ["pattern"]
		}`),
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			var a struct {
				Pattern string `json:"pattern"`
				Path    string `json:"path"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", fmt.Errorf("grep: invalid arguments: %w", err)
			}
			if a.Pattern == "" {
				return "", fmt.Errorf("grep: pattern is required")
			}

			searchDir := root
			if a.Path != "" {
				cleaned, err := safePath(root, a.Path)
				if err != nil {
					return "", err
				}
				searchDir = cleaned
			}
			canonicalRoot, rootErr := filepath.EvalSymlinks(filepath.Clean(root))
			if rootErr != nil {
				return "", fmt.Errorf("grep: cannot resolve root: %w", rootErr)
			}

			re, err := regexp.Compile(a.Pattern)
			if err != nil {
				return "", fmt.Errorf("grep: invalid regex: %w", err)
			}

			var lines []string
			err = filepath.WalkDir(searchDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if d.IsDir() {
					// Skip hidden and vendor directories.
					name := d.Name()
					if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
						return filepath.SkipDir
					}
					return nil
				}
				// Skip binary-ish files.
				if isBinaryExtension(filepath.Ext(path)) {
					return nil
				}
				resolved, resErr := filepath.EvalSymlinks(path)
				if resErr == nil && !isUnderRoot(resolved, canonicalRoot) && resolved != canonicalRoot {
					return nil
				}

				f, err := os.Open(path)
				if err != nil {
					return nil
				}
				// Explicit close — NOT defer. WalkDir callbacks stack defers for the
				// entire walk, so defer f.Close() would leak file descriptors on large
				// codebases. Closing eagerly after scanning prevents FD exhaustion.
				rel, _ := filepath.Rel(root, path)
				scanner := bufio.NewScanner(f)
				lineNum := 0
				for scanner.Scan() {
					lineNum++
					if re.MatchString(scanner.Text()) {
						lines = append(lines, fmt.Sprintf("%s:%d:%s", rel, lineNum, scanner.Text()))
					}
				}
				f.Close()
				return nil
			})
			if err != nil {
				return "", fmt.Errorf("grep: %w", err)
			}

			if len(lines) == 0 {
				return "(no matches)", nil
			}
			return strings.Join(lines, "\n"), nil
		},
	}
}

// isBinaryExtension returns true for file extensions that should be skipped by grep.
func isBinaryExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico", ".webp",
		".zip", ".tar", ".gz", ".bz2", ".xz", ".zst",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".exe", ".dll", ".so", ".dylib", ".o", ".a",
		".wasm", ".class", ".jar":
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// BdSearchTool
// ---------------------------------------------------------------------------

// BdSearchTool searches for cards in the control store.
func BdSearchTool(store *control.Store) Tool {
	return Tool{
		Name:        "bd_search",
		Description: "Search for feature cards/issues in the SDP control store. Returns matching card summaries.",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {"type": "string", "description": "Search query (keyword or card ID)"},
				"project": {"type": "string", "description": "Project ID to scope the search"},
				"limit": {"type": "integer", "description": "Maximum results (default 20)", "default": 20}
			},
			"required": ["query"]
		}`),
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			if store == nil {
				return "", fmt.Errorf("bd_search: control store not available (store is nil)")
			}
			var a struct {
				Query   string `json:"query"`
				Project string `json:"project"`
				Limit   int    `json:"limit"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", fmt.Errorf("bd_search: invalid arguments: %w", err)
			}
			if a.Limit <= 0 {
				a.Limit = 20
			}

			// If the query looks like an ID, try direct lookup.
			if len(a.Query) > 3 && strings.Contains(a.Query, "-") {
				card, err := store.LoadCardByID(a.Query)
				if err == nil && card != nil {
					return formatCard(card), nil
				}
			}

			// Otherwise, load cards for the project and filter by keyword.
			projectID := a.Project
			if projectID == "" {
				projectID = "" // LoadCards with empty projectID returns all for file mode
			}
			cards, err := store.LoadCards(projectID)
			if err != nil {
				return "", fmt.Errorf("bd_search: %w", err)
			}

			var results []string
			lower := strings.ToLower(a.Query)
			for _, c := range cards {
				if strings.Contains(strings.ToLower(c.Title), lower) ||
					strings.Contains(strings.ToLower(c.ID), lower) ||
					strings.Contains(strings.ToLower(c.NormalizedIntent), lower) {
					results = append(results, formatCard(&c))
					if len(results) >= a.Limit {
						break
					}
				}
			}

			if len(results) == 0 {
				return "(no matching cards)", nil
			}
			return strings.Join(results, "\n\n"), nil
		},
	}
}

// ---------------------------------------------------------------------------
// BdCreateTool
// ---------------------------------------------------------------------------

// BdCreateTool creates a new card in the control store.
// Output format: "card_created:<id>" — compatible with EvidenceAccumulator.
func BdCreateTool(store *control.Store) Tool {
	return Tool{
		Name:        "bd_create",
		Description: "Create a new feature card in the SDP control store.",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"project": {"type": "string", "description": "Project ID"},
				"title": {"type": "string", "description": "Card title"},
				"description": {"type": "string", "description": "Raw request / description"}
			},
			"required": ["project", "title"]
		}`),
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			if store == nil {
				return "", fmt.Errorf("bd_create: control store not available (store is nil)")
			}
			var a struct {
				Project     string `json:"project"`
				Title       string `json:"title"`
				Description string `json:"description"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", fmt.Errorf("bd_create: invalid arguments: %w", err)
			}

			card, err := store.CreateCard(a.Project, a.Title, a.Description)
			if err != nil {
				return "", fmt.Errorf("bd_create: %w", err)
			}

			// Output format compatible with extractCardID in evidence.go.
			return "card_created:" + card.ID, nil
		},
	}
}

// ---------------------------------------------------------------------------
// BdCommentTool
// ---------------------------------------------------------------------------

// BdCommentTool adds a comment to an existing card.
func BdCommentTool(store *control.Store) Tool {
	return Tool{
		Name:        "bd_comment",
		Description: "Add a comment to an existing feature card in the SDP control store.",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"card_id": {"type": "string", "description": "Card ID to comment on"},
				"comment": {"type": "string", "description": "Comment text"}
			},
			"required": ["card_id", "comment"]
		}`),
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			if store == nil {
				return "", fmt.Errorf("bd_comment: control store not available (store is nil)")
			}
			var a struct {
				CardID  string `json:"card_id"`
				Comment string `json:"comment"`
			}
			if err := json.Unmarshal(args, &a); err != nil {
				return "", fmt.Errorf("bd_comment: invalid arguments: %w", err)
			}
			if a.CardID == "" {
				return "", fmt.Errorf("bd_comment: card_id is required")
			}
			if a.Comment == "" {
				return "", fmt.Errorf("bd_comment: comment is required")
			}

			// Verify card exists.
			card, err := store.LoadCardByID(a.CardID)
			if err != nil {
				return "", fmt.Errorf("bd_comment: card %s not found: %w", a.CardID, err)
			}

			// Append comment to ReviewSummary field (lightweight comment storage).
			// A full comment system would use beads comments, but this is sufficient
			// for the agent loop's needs.
			existing := card.ReviewSummary
			if existing != "" {
				existing += "\n"
			}
			card.ReviewSummary = existing + a.Comment
			if err := store.SaveCard(card); err != nil {
				return "", fmt.Errorf("bd_comment: save failed: %w", err)
			}

			return "comment_added:" + a.CardID, nil
		},
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// safePath validates that path does not escape root. Returns the cleaned absolute path.
// It resolves symlinks on both root and the candidate path so that symlinks inside
// the project cannot be used to read or write files outside the root.
func safePath(root, relPath string) (string, error) {
	if relPath == "" {
		return "", fmt.Errorf("path is required")
	}
	// First check for ".." components that would escape.
	cleanRel := filepath.Clean(relPath)
	if strings.HasPrefix(cleanRel, "..") {
		return "", fmt.Errorf("path escapes root: %s", relPath)
	}

	// Resolve root to its canonical form (follow symlinks).
	canonicalRoot, err := filepath.EvalSymlinks(filepath.Clean(root))
	if err != nil {
		return "", fmt.Errorf("path escapes root: cannot resolve root: %w", err)
	}

	candidate := filepath.Join(canonicalRoot, cleanRel)

	// For existing paths, resolve symlinks.
	resolved, err := filepath.EvalSymlinks(candidate)
	if err == nil {
		// Path exists — check resolved path is under canonical root.
		if !isUnderRoot(resolved, canonicalRoot) {
			return "", fmt.Errorf("path escapes root: %s resolves outside root", relPath)
		}
		return resolved, nil
	}

	// Path doesn't exist yet — walk up to find the first existing ancestor.
	parent := filepath.Dir(candidate)
	for {
		resolvedParent, pErr := filepath.EvalSymlinks(parent)
		if pErr == nil {
			// Found an existing ancestor — verify it is under root.
			if !isUnderRoot(resolvedParent, canonicalRoot) {
				return "", fmt.Errorf("path escapes root: parent of %s resolves outside root", relPath)
			}
			break
		}
		// Parent doesn't exist either — keep walking up.
		nextParent := filepath.Dir(parent)
		if nextParent == parent {
			// Reached filesystem root without finding anything.
			return "", fmt.Errorf("path escapes root: no existing ancestor for %s", relPath)
		}
		// If we've walked above the canonical root, bail out.
		if !strings.HasPrefix(nextParent, canonicalRoot) && nextParent != canonicalRoot {
			return "", fmt.Errorf("path escapes root: parent of %s resolves outside root", relPath)
		}
		parent = nextParent
	}
	// Return the candidate path (not yet existing) — it is safe because its
	// parent is verified to be under the canonical root.
	return candidate, nil
}

// isUnderRoot returns true if path is equal to root or is a descendant of root.
func isUnderRoot(path, root string) bool {
	return strings.HasPrefix(path, root+string(os.PathSeparator)) || path == root
}

// formatCard formats a FeatureCard for tool output.
func formatCard(c *control.FeatureCard) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("ID: %s", c.ID))
	b.WriteString(fmt.Sprintf("\nTitle: %s", c.Title))
	b.WriteString(fmt.Sprintf("\nStatus: %s", c.Status))
	if c.NormalizedIntent != "" {
		b.WriteString(fmt.Sprintf("\nDescription: %s", c.NormalizedIntent))
	}
	if c.RiskLevel != "" {
		b.WriteString(fmt.Sprintf("\nRisk: %s", c.RiskLevel))
	}
	if c.RecommendedNextAction != "" {
		b.WriteString(fmt.Sprintf("\nNext: %s", c.RecommendedNextAction))
	}
	return b.String()
}
