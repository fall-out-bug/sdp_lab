// Package testwriter generates Go unit test stubs from coverage gap reports.
// It parses the output of `go tool cover -func` to identify uncovered functions
// and produces table-driven test code following Go conventions.
package testwriter

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
)

// CoverageGap represents a function with coverage below the threshold.
type CoverageGap struct {
	File     string  `json:"file"`
	Function string  `json:"function"`
	Coverage float64 `json:"coverage"`
	Line     int     `json:"line"`
}

// TestCase represents a single row in a table-driven test.
type TestCase struct {
	Name     string `json:"name"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
}

// ParseCoverGaps parses the output of `go tool cover -func=cov.out` and returns
// functions whose coverage is strictly below the given threshold (in percent).
// The total: line is skipped. Malformed lines are silently ignored.
//
// Input format (each line):
//
//	path/file.go:LineNumber:  FunctionName  Percentage%
//	total:                    (statements)  Percentage%
func ParseCoverGaps(coverOutput string, threshold float64) ([]CoverageGap, error) {
	var gaps []CoverageGap

	scanner := bufio.NewScanner(strings.NewReader(coverOutput))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Skip the total: aggregate line
		if strings.HasPrefix(line, "total:") {
			continue
		}

		gap, ok := parseFuncLine(line)
		if !ok {
			continue // malformed line, skip
		}

		if gap.Coverage < threshold {
			gaps = append(gaps, gap)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("testwriter: scan cover output: %w", err)
	}

	if gaps == nil {
		gaps = []CoverageGap{}
	}

	return gaps, nil
}

// parseFuncLine parses a single line from `go tool cover -func` output.
// Expected format: "file.go:FuncName percentage%" or "file.go:line: FuncName percentage%"
func parseFuncLine(line string) (CoverageGap, bool) {
	// Fields are tab/space separated. Typical format:
	//   internal/foo/bar.go:FuncName  100.0%
	//   internal/foo/bar.go:25:       FuncName  100.0%
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return CoverageGap{}, false
	}

	// Last field should be the percentage (e.g., "100.0%")
	pctStr := fields[len(fields)-1]
	pctStr = strings.TrimSuffix(pctStr, "%")
	coverage, err := strconv.ParseFloat(pctStr, 64)
	if err != nil {
		return CoverageGap{}, false
	}

	// First field contains file:function or file:line:function
	first := fields[0]

	// Check if first field has "file:line:" pattern (with trailing colon after line number)
	// vs "file:function" pattern
	filePath, funcName, lineNum := splitFileFunc(first)

	// If we have file:line: pattern, the function name might be in the second field
	if lineNum > 0 && len(fields) >= 3 {
		funcName = fields[1]
	} else if funcName == "" && len(fields) >= 3 {
		// Sometimes format is "file:func  pct" with func already extracted
		funcName = fields[1]
	}

	if filePath == "" || funcName == "" {
		return CoverageGap{}, false
	}

	return CoverageGap{
		File:     filePath,
		Function: funcName,
		Coverage: coverage,
		Line:     lineNum,
	}, true
}

// splitFileFunc splits "path/file.go:FuncName" or "path/file.go:42:  FuncName"
// into file path, function name, and line number.
func splitFileFunc(s string) (file, function string, line int) {
	// Try "file:line:" pattern first
	lastColon := strings.LastIndex(s, ":")
	if lastColon < 0 {
		return s, "", 0
	}

	// Check if what's after the last colon is a number (line number)
	afterColon := s[lastColon+1:]
	if afterColon == "" {
		// Trailing colon: "file:42:" — extract line number
		beforeColon := s[:lastColon]
		prevColon := strings.LastIndex(beforeColon, ":")
		if prevColon >= 0 {
			lineStr := beforeColon[prevColon+1:]
			if n, err := strconv.Atoi(lineStr); err == nil {
				return beforeColon[:prevColon], "", n
			}
		}
		return beforeColon, "", 0
	}

	// Check if afterColon is a number (line number)
	if n, err := strconv.Atoi(afterColon); err == nil {
		return s[:lastColon], "", n
	}

	// Standard pattern: "file:FuncName"
	return s[:lastColon], afterColon, 0
}

// GenerateTestStub produces a table-driven test function for the given function name.
// The generated test follows Go conventions:
//   - Function named Test{FuncName}
//   - Table of TestCase structs
//   - t.Run subtests for each case
//   - Comments indicating where to add actual logic
func GenerateTestStub(funcName, packageName string, cases []TestCase) string {
	var b strings.Builder

	fmt.Fprintf(&b, "// Test%s covers %s.%s\n", funcName, packageName, funcName)
	fmt.Fprintf(&b, "func Test%s(t *testing.T) {\n", funcName)
	fmt.Fprintf(&b, "\tvar tests = []TestCase{\n")

	for _, tc := range cases {
		fmt.Fprintf(&b, "\t\t{Name: %q, Input: %q, Expected: %q},\n", tc.Name, tc.Input, tc.Expected)
	}

	if len(cases) == 0 {
		fmt.Fprintf(&b, "\t\t// TODO: Add test cases\n")
	}

	fmt.Fprintf(&b, "\t}\n\n")
	fmt.Fprintf(&b, "\tfor _, tt := range tests {\n")
	fmt.Fprintf(&b, "\t\tt.Run(tt.Name, func(t *testing.T) {\n")
	fmt.Fprintf(&b, "\t\t\t// Input: %s\n", "tt.Input")
	fmt.Fprintf(&b, "\t\t\t// Expected: %s\n", "tt.Expected")
	fmt.Fprintf(&b, "\t\t\t// TODO: call %s() and assert result\n", funcName)
	fmt.Fprintf(&b, "\t\t})\n")
	fmt.Fprintf(&b, "\t}\n")
	fmt.Fprintf(&b, "}\n")

	return b.String()
}

// ReadFuncSource reads a Go source file and extracts the function containing the
// given line number (1-based). It uses go/ast for accurate function boundary
// detection, correctly handling braces inside string literals, comments, and
// other non-code contexts.
func ReadFuncSource(file string, line int) (string, error) {
	if line <= 0 {
		return "", fmt.Errorf("testwriter: line must be > 0, got %d", line)
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("testwriter: parse %s: %w", file, err)
	}

	// Find the function declaration whose body spans the given line.
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Body == nil {
			continue
		}
		fnStart := fset.Position(fn.Pos()).Line
		fnEnd := fset.Position(fn.Body.End()).Line
		if line >= fnStart && line <= fnEnd {
			data, err := os.ReadFile(file)
			if err != nil {
				return "", fmt.Errorf("testwriter: read %s: %w", file, err)
			}
			lines := splitLines(string(data))
			if fnStart > len(lines) {
				return "", fmt.Errorf("testwriter: function start line %d beyond file", fnStart)
			}
			end := fnEnd
			if end > len(lines) {
				end = len(lines)
			}
			selected := lines[fnStart-1 : end]
			return strings.Join(selected, "\n"), nil
		}
	}

	return "", fmt.Errorf("testwriter: no function found at line %d in %s", line, file)
}

// FormatTestFile assembles a complete Go test file from package name, imports,
// and test function bodies. It deduplicates imports and ensures proper Go file
// structure: package declaration, import block, test functions.
func FormatTestFile(packageName string, imports []string, tests []string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "package %s\n", packageName)

	// Deduplicate imports
	seen := make(map[string]bool)
	var uniqueImports []string
	for _, imp := range imports {
		if !seen[imp] {
			seen[imp] = true
			uniqueImports = append(uniqueImports, imp)
		}
	}

	if len(uniqueImports) > 0 {
		fmt.Fprintf(&b, "\nimport (\n")
		for _, imp := range uniqueImports {
			fmt.Fprintf(&b, "\t%q\n", imp)
		}
		fmt.Fprintf(&b, ")\n")
	}

	for _, test := range tests {
		fmt.Fprintf(&b, "\n%s\n", test)
	}

	return b.String()
}

// splitLines splits a string into lines without removing empty lines.
func splitLines(s string) []string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	// If the last character was a newline, scanner ignores the trailing empty line
	// but for our purposes that's fine since we're doing 1-based line lookups.
	return lines
}
