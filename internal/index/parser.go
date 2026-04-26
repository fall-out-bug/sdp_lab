package index

import (
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ParseFile extracts chunks and edges from a source file.
// language should be the detected language name (e.g. "go", "python", "typescript").
// If language is empty, a file-level chunk is produced.
func ParseFile(filePath, language string) ([]Chunk, []Edge, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}
	content := string(data)

	if language == "" || len(strings.TrimSpace(content)) == 0 {
		return parseFallback(filePath, content, language)
	}

	// Dispatch to language-specific parser
	switch language {
	case "go":
		return parseGo(filePath, content)
	case "python":
		return parsePython(filePath, content)
	case "typescript", "javascript":
		return parseTypeScript(filePath, content)
	case "java":
		return parseJava(filePath, content)
	case "rust":
		return parseRust(filePath, content)
	case "ruby":
		return parseRuby(filePath, content)
	case "c", "cpp":
		return parseCFamily(filePath, content)
	default:
		return parseFallback(filePath, content, language)
	}
}

// contentHash computes a SHA256 hash of the given content.
func contentHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:16])
}

// makeDescription creates a NL description for a chunk.
func makeDescription(kind, symbolName, filePath, language string) string {
	pkg := extractPackage(filePath)
	switch kind {
	case "function":
		if pkg != "" {
			return fmt.Sprintf("%s in %s", symbolName, pkg)
		}
		return symbolName
	case "method":
		if pkg != "" {
			return fmt.Sprintf("%s in %s", symbolName, pkg)
		}
		return symbolName
	case "type", "class", "interface", "struct":
		return fmt.Sprintf("%s (%s)", symbolName, kind)
	case "const", "var":
		return fmt.Sprintf("%s in %s", symbolName, pkg)
	case "file":
		return fmt.Sprintf("%s (%s)", filePath, language)
	default:
		return symbolName
	}
}

func extractPackage(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		return strings.Join(parts[:len(parts)-1], "/")
	}
	return ""
}

// parseFallback creates a single file-level chunk.
func parseFallback(filePath, content, language string) ([]Chunk, []Edge, error) {
	if strings.TrimSpace(content) == "" {
		return nil, nil, nil
	}
	lines := strings.Split(content, "\n")
	chunk := Chunk{
		FilePath:    filePath,
		Kind:        "file",
		Language:    language,
		LineStart:   1,
		LineEnd:     len(lines),
		Content:     content,
		Hash:        contentHash(content),
		SymbolName:  filepathBase(filePath),
		Description: makeDescription("file", filepathBase(filePath), filePath, language),
	}
	return []Chunk{chunk}, nil, nil
}

func filepathBase(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// --- Go Parser ---

var (
	goFuncRe     = regexp.MustCompile(`(?m)^func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(`)
	goTypeRe     = regexp.MustCompile(`(?m)^type\s+(\w+)\s+(struct|interface)`)
	goMethodRe   = regexp.MustCompile(`(?m)^func\s+\(\s*\w+\s+\*?(\w+)\s*\)\s*(\w+)\s*\(`)
	goVarRe      = regexp.MustCompile(`(?m)^var\s+(\w+)\s+`)
	goConstRe    = regexp.MustCompile(`(?m)^const\s+(\w+)\s*=`)
	goPackageRe  = regexp.MustCompile(`(?m)^package\s+(\w+)`)
)

func parseGo(filePath, content string) ([]Chunk, []Edge, error) {
	var chunks []Chunk
	var edges []Edge
	lines := strings.Split(content, "\n")
	pkgName := extractGoPackage(content)

	// Extract top-level blocks using brace matching
	blocks := extractTopLevelBlocks(content)

	for _, block := range blocks {
		var kind, symbolName string

		line := strings.TrimSpace(block.content)
		if line == "" {
			continue
		}

		// Classify the block
		switch {
		case strings.HasPrefix(line, "func ("):
			// Method
			matches := goMethodRe.FindStringSubmatch(line)
			if len(matches) >= 3 {
				kind = "method"
				symbolName = fmt.Sprintf("%s.%s", matches[1], matches[2])
			}
		case strings.HasPrefix(line, "func "):
			// Function
			matches := goFuncRe.FindStringSubmatch(line)
			if len(matches) >= 2 {
				kind = "function"
				symbolName = matches[1]
			}
		case strings.HasPrefix(line, "type ") && strings.Contains(line, " struct"):
			matches := goTypeRe.FindStringSubmatch(line)
			if len(matches) >= 2 {
				kind = "type"
				symbolName = matches[1]
			}
		case strings.HasPrefix(line, "type ") && strings.Contains(line, " interface"):
			matches := goTypeRe.FindStringSubmatch(line)
			if len(matches) >= 2 {
				kind = "interface"
				symbolName = matches[1]
			}
		case strings.HasPrefix(line, "var "):
			matches := goVarRe.FindStringSubmatch(line)
			if len(matches) >= 2 {
				kind = "var"
				symbolName = matches[1]
			}
		case strings.HasPrefix(line, "const "):
			matches := goConstRe.FindStringSubmatch(line)
			if len(matches) >= 2 {
				kind = "const"
				symbolName = matches[1]
			}
		default:
			// Skip import blocks and other non-semantic top-level code
			continue
		}

		if kind == "" {
			continue
		}

		scope := filePath
		if pkgName != "" {
			scope = fmt.Sprintf("%s > %s", filePath, pkgName)
		}

		chunkContent := block.content
		// Limit chunk size to ~1024 lines (roughly 1024 tokens)
		chunkLines := strings.Split(chunkContent, "\n")
		if len(chunkLines) > 1024 {
			chunkContent = strings.Join(chunkLines[:1024], "\n") + "\n// ... truncated"
		}

		chunk := Chunk{
			FilePath:    filePath,
			SymbolName:  symbolName,
			Kind:        kind,
			Scope:       scope,
			Language:    "go",
			LineStart:   block.startLine + 1, // 0-indexed to 1-indexed
			LineEnd:     block.endLine + 1,
			Content:     chunkContent,
			Hash:        contentHash(chunkContent),
			Description: makeDescription(kind, symbolName, filePath, "go"),
		}
		chunks = append(chunks, chunk)
	}

	// If no chunks found, create file-level chunk
	if len(chunks) == 0 && strings.TrimSpace(content) != "" {
		chunks = append(chunks, Chunk{
			FilePath:    filePath,
			Kind:        "file",
			Language:    "go",
			LineStart:   1,
			LineEnd:     len(lines),
			Content:     content,
			Hash:        contentHash(content),
			SymbolName:  filepathBase(filePath),
			Description: makeDescription("file", filepathBase(filePath), filePath, "go"),
		})
	}

	return chunks, edges, nil
}

func extractGoPackage(content string) string {
	matches := goPackageRe.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// topBlock represents a top-level code block.
type topBlock struct {
	content   string
	startLine int
	endLine   int
}

// extractTopLevelBlocks splits content into top-level blocks by matching braces.
func extractTopLevelBlocks(content string) []topBlock {
	var blocks []topBlock
	lines := strings.Split(content, "\n")
	depth := 0
	blockStart := 0
	blockLines := []string{}
	inBlock := false
	var state braceCountState

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect start of a top-level declaration
		if depth == 0 && !inBlock {
			if isTopLevelDecl(trimmed) {
				blockStart = i
				blockLines = []string{line}
				// Check if line opens a brace
				depth, state = countBraces(line, state)
				if depth > 0 {
					inBlock = true
				} else {
					// Single-line declaration (e.g., "const X = 5")
					// Continue to next line to see if there's a block
					depth = 0
					state = braceCountState{}
					inBlock = false
					blocks = append(blocks, topBlock{
						content:   strings.Join(blockLines, "\n"),
						startLine: blockStart,
						endLine:   i,
					})
					blockLines = nil
				}
				continue
			}
			continue
		}

		if inBlock {
			blockLines = append(blockLines, line)
			lineDelta, newState := countBraces(line, state)
			depth += lineDelta
			state = newState

			if depth <= 0 {
				blocks = append(blocks, topBlock{
					content:   strings.Join(blockLines, "\n"),
					startLine: blockStart,
					endLine:   i,
				})
				blockLines = nil
				inBlock = false
				depth = 0
				state = braceCountState{}
			}
		}
	}

	// Handle trailing block
	if len(blockLines) > 0 {
		blocks = append(blocks, topBlock{
			content:   strings.Join(blockLines, "\n"),
			startLine: blockStart,
			endLine:   len(lines) - 1,
		})
	}

	return blocks
}

func isTopLevelDecl(line string) bool {
	return strings.HasPrefix(line, "func ") ||
		strings.HasPrefix(line, "type ") ||
		strings.HasPrefix(line, "var ") ||
		strings.HasPrefix(line, "const ") ||
		strings.HasPrefix(line, "import ")
}

// braceCountState tracks the state for counting braces across lines.
type braceCountState struct {
	inString    bool   // Track if we're inside a double-quoted string
	inRawString bool   // Track if we're inside a raw string (backtick)
	inRune      bool   // Track if we're inside a rune literal
	// Note: inComment is NOT tracked here because line comments (//)
	// are line-local and should not persist across lines.
}

// countBraces counts the brace depth change in a line, handling multi-line strings,
// raw strings, and escaped quotes. It returns the depth delta and updated state.
// Note: Line comments (//) are handled but do NOT persist in the state since they
// are line-local in Go.
func countBraces(line string, state braceCountState) (int, braceCountState) {
	count := 0
	// Work with a mutable copy of the state
	s := state
	// inComment is line-local, so we track it separately
	inComment := false

	for i := 0; i < len(line); i++ {
		ch := rune(line[i])

		// Skip everything if we're in a comment
		if inComment {
			continue
		}

		// Check for line comment start
		if ch == '/' && i+1 < len(line) && line[i+1] == '/' {
			inComment = true
			continue
		}

		// Handle raw string literals (backtick)
		// Raw strings cannot contain escaped backticks
		if ch == '`' {
			s.inRawString = !s.inRawString
			continue
		}

		// If we're in a raw string, skip all other processing
		if s.inRawString {
			continue
		}

		// Handle double-quoted strings
		if ch == '"' {
			// Check if the quote is escaped
			// We need to count consecutive backslashes to determine if the quote is escaped
			backslashCount := 0
			for j := i - 1; j >= 0 && line[j] == '\\'; j-- {
				backslashCount++
			}
			// An odd number of backslashes means the quote is escaped
			if backslashCount%2 == 0 {
				s.inString = !s.inString
			}
			continue
		}

		// Handle rune literals (single quotes)
		if ch == '\'' {
			// Similar escape logic as double quotes
			backslashCount := 0
			for j := i - 1; j >= 0 && line[j] == '\\'; j-- {
				backslashCount++
			}
			if backslashCount%2 == 0 {
				s.inRune = !s.inRune
			}
			continue
		}

		// Count braces only when not in a string, rune, or comment
		if !s.inString && !s.inRune {
			if ch == '{' {
				count++
			} else if ch == '}' {
				count--
			}
		}
	}

	return count, s
}

// --- Python Parser ---

var (
	pyClassRe    = regexp.MustCompile(`(?m)^class\s+(\w+)`)
	pyFuncRe     = regexp.MustCompile(`(?m)^def\s+(\w+)`)
	pyAsyncFuncRe = regexp.MustCompile(`(?m)^async\s+def\s+(\w+)`)
)

func parsePython(filePath, content string) ([]Chunk, []Edge, error) {
	var chunks []Chunk
	lines := strings.Split(content, "\n")

	// Find class and function definitions
	type block struct {
		name      string
		kind      string
		startLine int
		indent    int
	}
	var blocks []block

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		if matches := pyClassRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			blocks = append(blocks, block{name: matches[1], kind: "class", startLine: i, indent: indent})
		} else if matches := pyAsyncFuncRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			blocks = append(blocks, block{name: matches[1], kind: "function", startLine: i, indent: indent})
		} else if matches := pyFuncRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			blocks = append(blocks, block{name: matches[1], kind: "function", startLine: i, indent: indent})
		}
	}

	// Extract content for each block
	for i, b := range blocks {
		endLine := len(lines) - 1
		if i+1 < len(blocks) && blocks[i+1].indent <= b.indent {
			endLine = blocks[i+1].startLine - 1
		} else if i+1 < len(blocks) {
			endLine = blocks[i+1].startLine - 1
		}

		// For blocks with children, find the real end by checking indentation
		if b.kind == "class" {
			for j := b.startLine + 1; j < len(lines); j++ {
				lineIndent := len(lines[j]) - len(strings.TrimLeft(lines[j], " \t"))
				if strings.TrimSpace(lines[j]) == "" {
					continue
				}
				if lineIndent <= b.indent && j > b.startLine+1 {
					endLine = j - 1
					break
				}
			}
		}

		if endLine >= len(lines) {
			endLine = len(lines) - 1
		}

		blockContent := strings.Join(lines[b.startLine:endLine+1], "\n")
		if len(blockContent) > 32*1024 { // ~32KB limit
			blockContent = blockContent[:32*1024] + "\n# ... truncated"
		}

		chunk := Chunk{
			FilePath:    filePath,
			SymbolName:  b.name,
			Kind:        b.kind,
			Scope:       filePath,
			Language:    "python",
			LineStart:   b.startLine + 1,
			LineEnd:     endLine + 1,
			Content:     blockContent,
			Hash:        contentHash(blockContent),
			Description: makeDescription(b.kind, b.name, filePath, "python"),
		}
		chunks = append(chunks, chunk)
	}

	// File-level fallback
	if len(chunks) == 0 && strings.TrimSpace(content) != "" {
		chunks = append(chunks, Chunk{
			FilePath:    filePath,
			Kind:        "file",
			Language:    "python",
			LineStart:   1,
			LineEnd:     len(lines),
			Content:     content,
			Hash:        contentHash(content),
			SymbolName:  filepathBase(filePath),
			Description: makeDescription("file", filepathBase(filePath), filePath, "python"),
		})
	}

	return chunks, nil, nil
}

// --- TypeScript/JavaScript Parser ---

var (
	tsFuncRe      = regexp.MustCompile(`(?m)(?:export\s+)?(?:async\s+)?function\s+(\w+)`)
	tsClassRe     = regexp.MustCompile(`(?m)(?:export\s+)?(?:default\s+)?class\s+(\w+)`)
	tsInterfaceRe = regexp.MustCompile(`(?m)(?:export\s+)?interface\s+(\w+)`)
	tsMethodRe    = regexp.MustCompile(`(?m)\s+(?:async\s+)?(\w+)\s*\([^)]*\)\s*(?::\s*\w+)?\s*\{`)
	tsArrowRe     = regexp.MustCompile(`(?m)(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\([^)]*\)\s*=>`)
)

func parseTypeScript(filePath, content string) ([]Chunk, []Edge, error) {
	var chunks []Chunk
	lines := strings.Split(content, "\n")

	type decl struct {
		name      string
		kind      string
		startLine int
	}
	var decls []decl

	// Find top-level declarations
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if matches := tsInterfaceRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			decls = append(decls, decl{name: matches[1], kind: "interface", startLine: i})
		} else if matches := tsClassRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			decls = append(decls, decl{name: matches[1], kind: "class", startLine: i})
		} else if matches := tsFuncRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			decls = append(decls, decl{name: matches[1], kind: "function", startLine: i})
		} else if matches := tsArrowRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			decls = append(decls, decl{name: matches[1], kind: "function", startLine: i})
		}
	}

	// Extract content for each declaration
	for i, d := range decls {
		endLine := len(lines) - 1

		// Use brace matching to find end
		if i+1 < len(decls) {
			endLine = decls[i+1].startLine - 1
		}

		// Better: use brace counting from startLine
		braceEnd := findBraceEnd(lines, d.startLine)
		if braceEnd > 0 && braceEnd < endLine {
			endLine = braceEnd
		}

		blockContent := strings.Join(lines[d.startLine:endLine+1], "\n")
		if len(blockContent) > 32*1024 {
			blockContent = blockContent[:32*1024] + "\n// ... truncated"
		}

		chunk := Chunk{
			FilePath:    filePath,
			SymbolName:  d.name,
			Kind:        d.kind,
			Scope:       filePath,
			Language:    "typescript",
			LineStart:   d.startLine + 1,
			LineEnd:     endLine + 1,
			Content:     blockContent,
			Hash:        contentHash(blockContent),
			Description: makeDescription(d.kind, d.name, filePath, "typescript"),
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 && strings.TrimSpace(content) != "" {
		chunks = append(chunks, Chunk{
			FilePath:    filePath,
			Kind:        "file",
			Language:    "typescript",
			LineStart:   1,
			LineEnd:     len(lines),
			Content:     content,
			Hash:        contentHash(content),
			SymbolName:  filepathBase(filePath),
			Description: makeDescription("file", filepathBase(filePath), filePath, "typescript"),
		})
	}

	return chunks, nil, nil
}

// findBraceEnd finds the line where the brace block started at startLine ends.
// It handles string literals (single-quoted, double-quoted, template literals),
// and line/block comments for TypeScript/JavaScript, Java, Rust, and C/C++.
func findBraceEnd(lines []string, startLine int) int {
	type braceState struct {
		inSingleQuote  bool
		inDoubleQuote  bool
		inTemplate     bool // Template literals (backticks)
		inLineComment  bool
		inBlockComment bool
	}
	state := braceState{}
	depth := 0
	started := false

	for i := startLine; i < len(lines); i++ {
		line := lines[i]
		for j := 0; j < len(line); j++ {
			ch := line[j]
			nextCh := byte(0)
			if j+1 < len(line) {
				nextCh = line[j+1]
			}

			// Handle line comments (//) - skip rest of line
			if !state.inBlockComment && !state.inSingleQuote && !state.inDoubleQuote && !state.inTemplate {
				if ch == '/' && nextCh == '/' {
					state.inLineComment = true
					break // Skip rest of line
				}
			}

			// Handle block comments (/* */)
			if !state.inLineComment && !state.inSingleQuote && !state.inDoubleQuote && !state.inTemplate {
				if ch == '/' && nextCh == '*' && !state.inBlockComment {
					state.inBlockComment = true
					j++ // Skip next char
					continue
				}
				if ch == '*' && nextCh == '/' && state.inBlockComment {
					state.inBlockComment = false
					j++ // Skip next char
					continue
				}
			}

			// Skip all processing if in comments
			if state.inLineComment || state.inBlockComment {
				continue
			}

			// Handle escape sequences
			if ch == '\\' && j+1 < len(line) {
				j++ // Skip escaped character
				continue
			}

			// Handle template literals (backticks) - for TypeScript/JavaScript
			if ch == '`' {
				state.inTemplate = !state.inTemplate
				continue
			}

			// Handle single quotes
			if ch == '\'' && !state.inDoubleQuote && !state.inTemplate {
				state.inSingleQuote = !state.inSingleQuote
				continue
			}

			// Handle double quotes
			if ch == '"' && !state.inSingleQuote && !state.inTemplate {
				state.inDoubleQuote = !state.inDoubleQuote
				continue
			}

			// Count braces only when not in strings or comments
			if !state.inSingleQuote && !state.inDoubleQuote && !state.inTemplate {
				if ch == '{' {
					depth++
					started = true
				} else if ch == '}' {
					depth--
				}
			}
		}

		// Reset line comment state at end of line
		state.inLineComment = false

		if started && depth <= 0 {
			return i
		}
	}
	return -1
}

// --- Java Parser ---

var (
	javaClassRe     = regexp.MustCompile(`(?m)(?:public|private|protected)?\s*(?:static\s+)?(?:abstract\s+)?(?:class|interface|enum)\s+(\w+)`)
	javaMethodRe    = regexp.MustCompile(`(?m)\s+(?:public|private|protected)?\s*(?:static\s+)?(?:\w+(?:<[^>]+>)?)\s+(\w+)\s*\([^)]*\)\s*(?:throws\s+[\w,\s]+)?\s*\{`)
	javaConstructorRe = regexp.MustCompile(`(?m)\s+(?:public|private|protected)?\s+(\w+)\s*\([^)]*\)\s*\{`)
)

func parseJava(filePath, content string) ([]Chunk, []Edge, error) {
	var chunks []Chunk
	lines := strings.Split(content, "\n")

	type decl struct {
		name      string
		kind      string
		startLine int
	}
	var decls []decl

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if matches := javaClassRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			kind := "class"
			if strings.Contains(trimmed, "interface ") {
				kind = "interface"
			} else if strings.Contains(trimmed, "enum ") {
				kind = "class"
			}
			decls = append(decls, decl{name: matches[1], kind: kind, startLine: i})
		}
	}

	// Extract content for each class/interface
	for i, d := range decls {
		endLine := len(lines) - 1
		if i+1 < len(decls) {
			endLine = decls[i+1].startLine - 1
		}

		braceEnd := findBraceEnd(lines, d.startLine)
		if braceEnd > 0 && braceEnd < endLine {
			endLine = braceEnd
		}

		blockContent := strings.Join(lines[d.startLine:endLine+1], "\n")
		if len(blockContent) > 32*1024 {
			blockContent = blockContent[:32*1024] + "\n// ... truncated"
		}

		chunk := Chunk{
			FilePath:    filePath,
			SymbolName:  d.name,
			Kind:        d.kind,
			Scope:       filePath,
			Language:    "java",
			LineStart:   d.startLine + 1,
			LineEnd:     endLine + 1,
			Content:     blockContent,
			Hash:        contentHash(blockContent),
			Description: makeDescription(d.kind, d.name, filePath, "java"),
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 && strings.TrimSpace(content) != "" {
		chunks = append(chunks, Chunk{
			FilePath:    filePath,
			Kind:        "file",
			Language:    "java",
			LineStart:   1,
			LineEnd:     len(lines),
			Content:     content,
			Hash:        contentHash(content),
			SymbolName:  filepathBase(filePath),
			Description: makeDescription("file", filepathBase(filePath), filePath, "java"),
		})
	}

	return chunks, nil, nil
}

// --- Rust Parser ---

var (
	rustFnRe   = regexp.MustCompile(`(?m)(?:pub\s+)?(?:async\s+)?fn\s+(\w+)`)
	rustStructRe = regexp.MustCompile(`(?m)(?:pub\s+)?struct\s+(\w+)`)
	rustImplRe  = regexp.MustCompile(`(?m)impl\s+(?:<[^>]+>\s*)?(\w+)`)
	rustEnumRe  = regexp.MustCompile(`(?m)(?:pub\s+)?enum\s+(\w+)`)
)

func parseRust(filePath, content string) ([]Chunk, []Edge, error) {
	var chunks []Chunk
	lines := strings.Split(content, "\n")

	type decl struct {
		name      string
		kind      string
		startLine int
	}
	var decls []decl

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if matches := rustFnRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			kind := "function"
			decls = append(decls, decl{name: matches[1], kind: kind, startLine: i})
		} else if matches := rustStructRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			decls = append(decls, decl{name: matches[1], kind: "type", startLine: i})
		} else if matches := rustImplRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			decls = append(decls, decl{name: matches[1], kind: "impl", startLine: i})
		} else if matches := rustEnumRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			decls = append(decls, decl{name: matches[1], kind: "type", startLine: i})
		}
	}

	for i, d := range decls {
		endLine := len(lines) - 1
		if i+1 < len(decls) {
			endLine = decls[i+1].startLine - 1
		}

		braceEnd := findBraceEnd(lines, d.startLine)
		if braceEnd > 0 && braceEnd < endLine {
			endLine = braceEnd
		}

		blockContent := strings.Join(lines[d.startLine:endLine+1], "\n")
		if len(blockContent) > 32*1024 {
			blockContent = blockContent[:32*1024] + "\n// ... truncated"
		}

		chunk := Chunk{
			FilePath:    filePath,
			SymbolName:  d.name,
			Kind:        d.kind,
			Scope:       filePath,
			Language:    "rust",
			LineStart:   d.startLine + 1,
			LineEnd:     endLine + 1,
			Content:     blockContent,
			Hash:        contentHash(blockContent),
			Description: makeDescription(d.kind, d.name, filePath, "rust"),
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 && strings.TrimSpace(content) != "" {
		chunks = append(chunks, Chunk{
			FilePath:    filePath,
			Kind:        "file",
			Language:    "rust",
			LineStart:   1,
			LineEnd:     len(lines),
			Content:     content,
			Hash:        contentHash(content),
			SymbolName:  filepathBase(filePath),
			Description: makeDescription("file", filepathBase(filePath), filePath, "rust"),
		})
	}

	return chunks, nil, nil
}

// --- Ruby Parser ---

var (
	rubyClassRe = regexp.MustCompile(`(?m)class\s+(\w+)`)
	rubyFuncRe  = regexp.MustCompile(`(?m)def\s+(\w+)`)
)

func parseRuby(filePath, content string) ([]Chunk, []Edge, error) {
	var chunks []Chunk
	lines := strings.Split(content, "\n")

	type decl struct {
		name      string
		kind      string
		startLine int
	}
	var decls []decl

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if matches := rubyClassRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			decls = append(decls, decl{name: matches[1], kind: "class", startLine: i})
		} else if matches := rubyFuncRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			decls = append(decls, decl{name: matches[1], kind: "function", startLine: i})
		}
	}

	for i, d := range decls {
		endLine := len(lines) - 1
		if i+1 < len(decls) {
			endLine = decls[i+1].startLine - 1
		}

		blockContent := strings.Join(lines[d.startLine:endLine+1], "\n")
		if len(blockContent) > 32*1024 {
			blockContent = blockContent[:32*1024] + "\n# ... truncated"
		}

		chunk := Chunk{
			FilePath:    filePath,
			SymbolName:  d.name,
			Kind:        d.kind,
			Scope:       filePath,
			Language:    "ruby",
			LineStart:   d.startLine + 1,
			LineEnd:     endLine + 1,
			Content:     blockContent,
			Hash:        contentHash(blockContent),
			Description: makeDescription(d.kind, d.name, filePath, "ruby"),
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 && strings.TrimSpace(content) != "" {
		chunks = append(chunks, Chunk{
			FilePath:    filePath,
			Kind:        "file",
			Language:    "ruby",
			LineStart:   1,
			LineEnd:     len(lines),
			Content:     content,
			Hash:        contentHash(content),
			SymbolName:  filepathBase(filePath),
			Description: makeDescription("file", filepathBase(filePath), filePath, "ruby"),
		})
	}

	return chunks, nil, nil
}

// --- C/C++ Parser ---

var (
	cFuncRe   = regexp.MustCompile(`(?m)^(?:[\w*]+\s+)+(\w+)\s*\([^)]*\)\s*\{`)
	cStructRe = regexp.MustCompile(`(?m)(?:typedef\s+)?struct\s+(\w+)`)
)

func parseCFamily(filePath, content string) ([]Chunk, []Edge, error) {
	var chunks []Chunk
	lines := strings.Split(content, "\n")

	type decl struct {
		name      string
		kind      string
		startLine int
	}
	var decls []decl

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if matches := cStructRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			decls = append(decls, decl{name: matches[1], kind: "type", startLine: i})
		} else if matches := cFuncRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
			name := matches[1]
			if name != "if" && name != "while" && name != "for" && name != "switch" {
				decls = append(decls, decl{name: name, kind: "function", startLine: i})
			}
		}
	}

	for i, d := range decls {
		endLine := len(lines) - 1
		if i+1 < len(decls) {
			endLine = decls[i+1].startLine - 1
		}

		braceEnd := findBraceEnd(lines, d.startLine)
		if braceEnd > 0 && braceEnd < endLine {
			endLine = braceEnd
		}

		blockContent := strings.Join(lines[d.startLine:endLine+1], "\n")
		if len(blockContent) > 32*1024 {
			blockContent = blockContent[:32*1024] + "\n// ... truncated"
		}

		lang := "c"
		if strings.Contains(filePath, ".cpp") || strings.Contains(filePath, ".cc") ||
			strings.Contains(filePath, ".cxx") || strings.Contains(filePath, ".hpp") {
			lang = "cpp"
		}

		chunk := Chunk{
			FilePath:    filePath,
			SymbolName:  d.name,
			Kind:        d.kind,
			Scope:       filePath,
			Language:    lang,
			LineStart:   d.startLine + 1,
			LineEnd:     endLine + 1,
			Content:     blockContent,
			Hash:        contentHash(blockContent),
			Description: makeDescription(d.kind, d.name, filePath, lang),
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 && strings.TrimSpace(content) != "" {
		lang := "c"
		if strings.Contains(filePath, ".cpp") || strings.Contains(filePath, ".cc") {
			lang = "cpp"
		}
		chunks = append(chunks, Chunk{
			FilePath:    filePath,
			Kind:        "file",
			Language:    lang,
			LineStart:   1,
			LineEnd:     len(lines),
			Content:     content,
			Hash:        contentHash(content),
			SymbolName:  filepathBase(filePath),
			Description: makeDescription("file", filepathBase(filePath), filePath, lang),
		})
	}

	return chunks, nil, nil
}
