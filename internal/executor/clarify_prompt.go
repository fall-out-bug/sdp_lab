package executor

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"sdp_dev/internal/augmentation"
	"sdp_dev/internal/prompt"
)

const clarifySystemPrompt = `You are an SDP intent clarifier. Your job is to normalize a raw development intent into a structured FeatureCard.

Given the raw intent and project context:
1. Analyze what the user actually wants
2. Determine scope (which files/directories are involved)
3. Identify what should NOT be touched
4. Assess risk and complexity
5. If the intent is ambiguous or could be interpreted multiple ways, set clarification_needed=true and provide specific questions

Rules:
- Be specific about scope. Use file paths and glob patterns.
- If intent says "fix X" but doesn't specify which X or where, ask.
- If intent is clear and specific, don't ask unnecessary questions.
- Consider existing project structure (listed files) when determining scope.

Output JSON only, no markdown:
{
  "normalized_intent": "...",
  "scope_in": ["..."],
  "scope_out": ["..."],
  "phase": "build|fix|refactor|feature|research",
  "risk_level": "low|medium|high",
  "clarification_needed": false,
  "questions": [],
  "estimated_complexity": "low|medium|high"
}`

func BuildClarificationPrompt(projectRoot, rawIntent string) string {
	var sections []string
	sections = append(sections, clarifySystemPrompt)
	if packSection := prompt.ContextSegmentsSection("Pack Context", augmentation.MustResolveDefaultPromptContext("planner.pack")); packSection != "" {
		sections = append(sections, packSection)
	}
	sections = append(sections, "Raw intent:\n"+strings.TrimSpace(rawIntent))
	sections = append(sections, "Project context:\n"+collectProjectContext(projectRoot))
	return strings.Join(sections, "\n\n") + "\n"
}

func collectProjectContext(projectRoot string) string {
	var sections []string
	sections = append(sections, fmt.Sprintf("Project root: %s", projectRoot))
	sections = append(sections, "Top-level ls -la:\n"+formatDirListing(projectRoot))
	sections = append(sections, "Relevant directory structure:\n"+formatTree(projectRoot, 0, 2))
	return strings.Join(sections, "\n\n")
}

func formatDirListing(root string) string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Sprintf("<error reading root: %v>", err)
	}
	lines := []string{".", ".."}
	for _, entry := range entries {
		kind := "file"
		if entry.IsDir() {
			kind = "dir"
		}
		lines = append(lines, fmt.Sprintf("%s\t%s", kind, entry.Name()))
	}
	sort.Strings(lines[2:])
	return strings.Join(lines, "\n")
}

func formatTree(root string, depth, maxDepth int) string {
	if depth > maxDepth {
		return ""
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Sprintf("<error reading %s: %v>", root, err)
	}
	slices.SortFunc(entries, func(a, b os.DirEntry) int { return cmp.Compare(a.Name(), b.Name()) })
	var lines []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".git") || name == "node_modules" || name == "vendor" {
			continue
		}
		rel := name
		if root != "" {
			if projectRel, err := filepath.Rel(root, filepath.Join(root, name)); err == nil {
				rel = projectRel
			}
		}
		prefix := strings.Repeat("  ", depth)
		if entry.IsDir() {
			lines = append(lines, prefix+name+"/")
			if depth < maxDepth {
				sub := formatTree(filepath.Join(root, name), depth+1, maxDepth)
				if strings.TrimSpace(sub) != "" {
					for _, line := range strings.Split(sub, "\n") {
						if strings.TrimSpace(line) != "" {
							lines = append(lines, line)
						}
					}
				}
			}
		} else if likelyRelevantFile(name) {
			_ = rel
			lines = append(lines, prefix+name)
		}
	}
	if len(lines) == 0 {
		return "<no relevant files>"
	}
	return strings.Join(lines, "\n")
}

func likelyRelevantFile(name string) bool {
	for _, suffix := range []string{".go", ".md", ".yaml", ".yml", ".json", ".toml", ".sh"} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}
