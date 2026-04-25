package skills

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const maxSkillFileSize = 10 << 20 // 10MB

// MarkerBlock represents a stack-specific marker block in a skill file.
type MarkerBlock struct {
	Section string // test, build, lint, etc.
	Stack   string // go, python, typescript, etc.
	Content string // The content between BEGIN and END markers
	Start   int    // Byte offset of BEGIN marker
	End     int    // Byte offset of END marker (exclusive)
}

// StackConfig represents the configuration for a technology stack.
type StackConfig struct {
	Stack        string            `json:"stack"`
	Version      string            `json:"version"`
	DisplayName  string            `json:"display_name"`
	Commands     map[string]string `json:"commands"`
	QualityGates []string          `json:"quality_gates"`
	FilePatterns []string          `json:"file_patterns"`
	Sections     map[string]Section `json:"sections"`
}

// Section represents a stack-specific section configuration.
type Section struct {
	Heading string `json:"heading"`
	Content string `json:"content"`
}

// Augmenter handles skill file augmentation with path safety.
type Augmenter struct {
	projectRoot string
}

// NewAugmenter creates a new Augmenter with a validated project root.
func NewAugmenter(projectRoot string) (*Augmenter, error) {
	// Validate projectRoot exists and is a directory
	info, err := os.Stat(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("project root does not exist: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project root is not a directory: %s", projectRoot)
	}

	// Resolve to absolute path
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}

	return &Augmenter{
		projectRoot: absRoot,
	}, nil
}

// safePath validates and sanitizes a file path to prevent path traversal attacks.
// It resolves the path to an absolute path and cleans any ".." or "." components.
// It ALWAYS validates that the resolved path stays within baseDir, regardless of
// whether the input is absolute or relative.
func safePath(baseDir, untrusted string) (string, error) {
	absBase, err := filepath.Abs(filepath.Clean(baseDir))
	if err != nil {
		return "", fmt.Errorf("resolve base: %w", err)
	}

	var resolved string
	if filepath.IsAbs(untrusted) {
		resolved = filepath.Clean(untrusted)
	} else {
		resolved = filepath.Join(absBase, untrusted)
	}

	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	// ALWAYS check containment, even for absolute paths
	if !strings.HasPrefix(resolved, absBase+string(os.PathSeparator)) && resolved != absBase {
		return "", fmt.Errorf("path escapes base directory: %s", untrusted)
	}

	return resolved, nil
}

// readFileWithLimit reads a file with a size limit to prevent excessive memory usage.
func readFileWithLimit(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Use LimitReader to enforce size limit
limitedReader := io.LimitReader(f, maxSkillFileSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	// Check if we hit the limit
	if len(data) > maxSkillFileSize {
		return nil, fmt.Errorf("file size exceeds limit of %d bytes", maxSkillFileSize)
	}

	return data, nil
}

// ParseMarkers parses all STACK_SPECIFIC marker blocks from a file.
func ParseMarkers(content []byte) ([]MarkerBlock, error) {
	var markers []MarkerBlock

	// Pattern: <!-- STACK_SPECIFIC:BEGIN section="..." stack="..." -->
	// Content
	// <!-- STACK_SPECIFIC:END -->
	beginPattern := regexp.MustCompile(`<!-- STACK_SPECIFIC:BEGIN section="([^"]+)" stack="([^"]+)" -->`)
	endPattern := regexp.MustCompile(`<!-- STACK_SPECIFIC:END -->`)

	beginMatches := beginPattern.FindAllSubmatchIndex(content, -1)
	endMatches := endPattern.FindAllIndex(content, -1)

	if len(beginMatches) != len(endMatches) {
		return nil, fmt.Errorf("marker mismatch: found %d BEGIN markers but %d END markers", len(beginMatches), len(endMatches))
	}

	for i, beginMatch := range beginMatches {
		if i >= len(endMatches) {
			break
		}

		// Validate beginMatch indices
		if len(beginMatch) < 6 {
			return nil, fmt.Errorf("invalid BEGIN marker at position %d: insufficient capture groups", i)
		}
		for idx, val := range beginMatch {
			if val < 0 {
				return nil, fmt.Errorf("invalid BEGIN marker at position %d: negative index at capture group %d", i, idx)
			}
		}

		// Extract section and stack from BEGIN marker
		section := string(content[beginMatch[2]:beginMatch[3]])
		stack := string(content[beginMatch[4]:beginMatch[5]])

		// Find content between BEGIN and END markers
		contentStart := beginMatch[1] // End of BEGIN marker
		contentEnd := endMatches[i][0] // Start of END marker

		// Validate marker positions to prevent panics
		if contentStart < 0 || contentEnd < 0 {
			return nil, fmt.Errorf("invalid marker positions: begin=%d end=%d", contentStart, contentEnd)
		}
		if contentEnd < contentStart {
			return nil, fmt.Errorf("malformed markers: END at position %d before BEGIN content at %d", contentEnd, contentStart)
		}
		if contentStart > len(content) || contentEnd > len(content) {
			return nil, fmt.Errorf("marker positions exceed content length: begin=%d end=%d contentLen=%d", contentStart, contentEnd, len(content))
		}

		// Extract content (trim leading/trailing whitespace and newlines)
		markerContent := strings.TrimSpace(string(content[contentStart:contentEnd]))

		markers = append(markers, MarkerBlock{
			Section: section,
			Stack:   stack,
			Content: markerContent,
			Start:   beginMatch[0],
			End:     endMatches[i][1],
		})
	}

	return markers, nil
}

// RenderBlock renders the content for a marker block based on the stack config.
func RenderBlock(section string, stackConfig StackConfig) (string, error) {
	sectionConfig, ok := stackConfig.Sections[section]
	if !ok {
		return "", fmt.Errorf("section %q not found in stack config for %q", section, stackConfig.Stack)
	}

	var buf bytes.Buffer
	buf.WriteString(sectionConfig.Heading)
	buf.WriteString("\n\n")
	buf.WriteString(sectionConfig.Content)

	return buf.String(), nil
}

// AugmentSkill augments a skill file with stack-specific content from the config.
func AugmentSkill(filePath string, stackConfig StackConfig) error {
	// Validate the file path is safe and get the sanitized path
	safeFilePath, err := safePath(".", filePath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Read the file with size limit using the safe path
	content, err := readFileWithLimit(safeFilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse existing markers
	markers, err := ParseMarkers(content)
	if err != nil {
		return fmt.Errorf("failed to parse markers: %w", err)
	}

	// Group markers by section
	markersBySection := make(map[string][]MarkerBlock)
	for _, marker := range markers {
		markersBySection[marker.Section] = append(markersBySection[marker.Section], marker)
	}

	// Build new content
	newContent := string(content)

	// For each section in the stack config, update or add the marker
	for sectionName := range stackConfig.Sections {
		rendered, err := RenderBlock(sectionName, stackConfig)
		if err != nil {
			return fmt.Errorf("failed to render section %q: %w", sectionName, err)
		}

		// Check if this section+stack combination already exists
		found := false
		for _, marker := range markers {
			if marker.Section == sectionName && marker.Stack == stackConfig.Stack {
				// Update existing marker
				beginMarker := fmt.Sprintf(`<!-- STACK_SPECIFIC:BEGIN section="%s" stack="%s" -->`, sectionName, stackConfig.Stack)
				endMarker := "<!-- STACK_SPECIFIC:END -->"

				oldBlock := extractBlock(content, marker.Start, marker.End)
				newBlock := fmt.Sprintf("%s\n%s\n%s", beginMarker, rendered, endMarker)

				newContent = strings.Replace(newContent, oldBlock, newBlock, 1)
				found = true
				break
			}
		}

		if !found {
			// Add new marker after existing markers in the same section
			// or at the end if no markers exist for this section
			insertPos := len(newContent)
			if sectionMarkers, ok := markersBySection[sectionName]; ok && len(sectionMarkers) > 0 {
				// Insert after the last marker in this section
				lastMarker := sectionMarkers[len(sectionMarkers)-1]
				insertPos = lastMarker.End
			} else {
				// Find the best position to insert (before "References" section or at end)
				if idx := strings.Index(newContent, "\n## References"); idx > 0 {
					insertPos = idx
				}
			}

			beginMarker := fmt.Sprintf(`<!-- STACK_SPECIFIC:BEGIN section="%s" stack="%s" -->`, sectionName, stackConfig.Stack)
			newBlock := fmt.Sprintf("\n\n%s\n%s\n<!-- STACK_SPECIFIC:END -->", beginMarker, rendered)

			// Insert at the calculated position
			newContent = newContent[:insertPos] + newBlock + newContent[insertPos:]
		}
	}

	// Write back to file using the safe path
	if err := os.WriteFile(safeFilePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// extractBlock extracts the full block (including markers) from content.
func extractBlock(content []byte, start, end int) string {
	return string(content[start:end])
}

// ValidateMarkers validates that all marker blocks in a file are well-formed.
func ValidateMarkers(filePath string) error {
	// Validate the file path is safe and get the sanitized path
	safeFilePath, err := safePath(".", filePath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	content, err := readFileWithLimit(safeFilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	beginPattern := regexp.MustCompile(`<!-- STACK_SPECIFIC:BEGIN section="([^"]+)" stack="([^"]+)" -->`)
	endPattern := regexp.MustCompile(`<!-- STACK_SPECIFIC:END -->`)

	beginMatches := beginPattern.FindAllSubmatchIndex(content, -1)
	endMatches := endPattern.FindAllIndex(content, -1)

	if len(beginMatches) != len(endMatches) {
		return fmt.Errorf("marker mismatch: found %d BEGIN markers but %d END markers", len(beginMatches), len(endMatches))
	}

	// Validate that markers are properly nested and ordered
	lastEnd := 0
	for i, beginMatch := range beginMatches {
		if beginMatch[0] < lastEnd {
			return fmt.Errorf("overlapping or out-of-order markers at position %d", beginMatch[0])
		}

		if i >= len(endMatches) {
			return fmt.Errorf("missing END marker for BEGIN at position %d", beginMatch[0])
		}

		if endMatches[i][0] < beginMatch[1] {
			return fmt.Errorf("END marker appears before BEGIN marker content at position %d", endMatches[i][0])
		}

		lastEnd = endMatches[i][1]
	}

	return nil
}

// LoadStackConfig loads a stack configuration from a JSON file.
func LoadStackConfig(configPath string) (StackConfig, error) {
	// Validate the config path is safe and get the sanitized path
	safeConfigPath, err := safePath(".", configPath)
	if err != nil {
		return StackConfig{}, fmt.Errorf("invalid config path: %w", err)
	}

	data, err := readFileWithLimit(safeConfigPath)
	if err != nil {
		return StackConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var config StackConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return StackConfig{}, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Validate required fields
	if config.Stack == "" {
		return StackConfig{}, fmt.Errorf("stack field is required")
	}
	if len(config.Sections) == 0 {
		return StackConfig{}, fmt.Errorf("at least one section is required")
	}

	return config, nil
}

// FindSkillFiles finds all skill markdown files in a directory.
func FindSkillFiles(skillsDir string) ([]string, error) {
	var skillFiles []string

	err := filepath.Walk(skillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			skillFiles = append(skillFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk skills directory: %w", err)
	}

	return skillFiles, nil
}

// DryRunAugment performs a dry-run augmentation and returns the diff.
func DryRunAugment(filePath string, stackConfig StackConfig) (string, error) {
	// Validate the file path is safe and get the sanitized path
	safeFilePath, err := safePath(".", filePath)
	if err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}

	// Read the file with size limit using the safe path
	content, err := readFileWithLimit(safeFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Parse existing markers
	markers, err := ParseMarkers(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse markers: %w", err)
	}

	// Group markers by section
	markersBySection := make(map[string][]MarkerBlock)
	for _, marker := range markers {
		markersBySection[marker.Section] = append(markersBySection[marker.Section], marker)
	}

	changes := []string{}

	// For each section in the stack config, check what would change
	for sectionName := range stackConfig.Sections {
		rendered, err := RenderBlock(sectionName, stackConfig)
		if err != nil {
			return "", fmt.Errorf("failed to render section %q: %w", sectionName, err)
		}

		// Check if this section+stack combination already exists
		found := false
		for _, marker := range markers {
			if marker.Section == sectionName && marker.Stack == stackConfig.Stack {
				// Check if content would change
				if strings.TrimSpace(marker.Content) != strings.TrimSpace(rendered) {
					changes = append(changes, fmt.Sprintf("  Section %q: content would be updated", sectionName))
				} else {
					changes = append(changes, fmt.Sprintf("  Section %q: no change (idempotent)", sectionName))
				}
				found = true
				break
			}
		}

		if !found {
			changes = append(changes, fmt.Sprintf("  Section %q: new marker would be added", sectionName))
		}
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("Dry-run for file: %s\n", filePath))
	buf.WriteString(fmt.Sprintf("Stack: %s (%s)\n", stackConfig.Stack, stackConfig.DisplayName))
	if len(changes) > 0 {
		buf.WriteString("Changes:\n")
		for _, change := range changes {
			buf.WriteString(change + "\n")
		}
	} else {
		buf.WriteString("No changes needed.\n")
	}

	return buf.String(), nil
}
