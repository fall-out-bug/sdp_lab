package testwriter

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCoverGaps(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		threshold float64
		want      []CoverageGap
		wantErr   bool
	}{
		{
			name: "functions below threshold",
			output: `internal/foo/bar.go:Bar 100.0%
internal/foo/bar.go:Uncovered 0.0%
internal/baz/qux.go:Qux 50.0%
total:                                                  (statements) 50.0%
`,
			threshold: 80.0,
			want: []CoverageGap{
				{File: "internal/foo/bar.go", Function: "Uncovered", Coverage: 0.0, Line: 0},
				{File: "internal/baz/qux.go", Function: "Qux", Coverage: 50.0, Line: 0},
			},
			wantErr: false,
		},
		{
			name: "all above threshold",
			output: `internal/foo/bar.go:Bar 100.0%
internal/foo/bar.go:Also 90.0%
total:                                                  95.0%
`,
			threshold: 80.0,
			want:      []CoverageGap{},
			wantErr:   false,
		},
		{
			name: "exact threshold excluded",
			output: `internal/foo/bar.go:Bar 80.0%
total:                                                  80.0%
`,
			threshold: 80.0,
			want:      []CoverageGap{},
			wantErr:   false,
		},
		{
			name: "zero threshold returns nothing (nothing below 0%)",
			output: `internal/foo/bar.go:Bar 100.0%
internal/baz/qux.go:Qux 50.0%
total:                                                  75.0%
`,
			threshold: 0.0,
			want:      []CoverageGap{},
			wantErr:   false,
		},
		{
			name:      "empty output returns error",
			output:    "",
			threshold: 80.0,
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "only malformed lines returns error",
			output:    "garbage line here\nanother bad line\n",
			threshold: 80.0,
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "only total line returns error",
			output:    "total:                                                  75.0%\n",
			threshold: 80.0,
			want:      nil,
			wantErr:   true,
		},
		{
			name: "malformed line skipped",
			output: `internal/foo/bar.go:Bar 100.0%
garbage line here
internal/baz/qux.go:Qux 30.0%
total:                                                  65.0%
`,
			threshold: 80.0,
			want: []CoverageGap{
				{File: "internal/baz/qux.go", Function: "Qux", Coverage: 30.0, Line: 0},
			},
			wantErr: false,
		},
		{
			name: "function with parentheses in name",
			output: `internal/foo/bar.go:NewFoo 0.0%
total:                                                  0.0%
`,
			threshold: 50.0,
			want: []CoverageGap{
				{File: "internal/foo/bar.go", Function: "NewFoo", Coverage: 0.0, Line: 0},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCoverGaps(tt.output, tt.threshold)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCoverGaps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("ParseCoverGaps() returned %d gaps, want %d", len(got), len(tt.want))
				for i, g := range got {
					t.Logf("  got[%d]: %+v", i, g)
				}
				return
			}
			for i, w := range tt.want {
				if got[i] != w {
					t.Errorf("gap[%d] = %+v, want %+v", i, got[i], w)
				}
			}
		})
	}
}

func TestParseCoverGaps_WithLineNumbers(t *testing.T) {
	output := `internal/foo/bar.go:25:  Bar 100.0%
internal/foo/bar.go:40:  Uncovered 0.0%
total:                                                  50.0%
`
	got, err := ParseCoverGaps(output, 80.0)
	if err != nil {
		t.Fatalf("ParseCoverGaps: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(got))
	}
	if got[0].Function != "Uncovered" {
		t.Errorf("Function = %q, want %q", got[0].Function, "Uncovered")
	}
}

func TestGenerateTestStub(t *testing.T) {
	tests := []struct {
		name        string
		funcName    string
		packageName string
		cases       []TestCase
		wantContain []string
	}{
		{
			name:        "single test case",
			funcName:    "Add",
			packageName: "math",
			cases: []TestCase{
				{Name: "positive numbers", Input: "1, 2", Expected: "3"},
			},
			wantContain: []string{
				"func TestAdd(t *testing.T) {",
				`"positive numbers"`,
				"// Input: tt.Input",
				"// Expected: tt.Expected",
				"for _, tt := range tests",
				"t.Run(tt.Name",
				"// TestAdd covers math.Add",
			},
		},
		{
			name:        "multiple test cases",
			funcName:    "Parse",
			packageName: "parser",
			cases: []TestCase{
				{Name: "valid input", Input: `"hello"`, Expected: `"hello", nil`},
				{Name: "empty input", Input: `""`, Expected: `"", nil`},
			},
			wantContain: []string{
				"func TestParse(t *testing.T) {",
				`"valid input"`,
				`"empty input"`,
				"// TestParse covers parser.Parse",
			},
		},
		{
			name:        "empty cases",
			funcName:    "Foo",
			packageName: "bar",
			cases:       []TestCase{},
			wantContain: []string{
				"func TestFoo(t *testing.T) {",
				"// TODO: Add test cases",
			},
		},
		{
			name:        "case with special chars",
			funcName:    "Sanitize",
			packageName: "util",
			cases: []TestCase{
				{Name: `input with "quotes"`, Input: `"hello"`, Expected: `"hello"`},
			},
			wantContain: []string{
				"func TestSanitize",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateTestStub(tt.funcName, tt.packageName, tt.cases)

			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("GenerateTestStub() missing expected substring %q\nGot:\n%s", want, got)
				}
			}

			// Must always contain the function name
			if !strings.Contains(got, "func Test"+tt.funcName) {
				t.Errorf("GenerateTestStub() missing test function Test%s", tt.funcName)
			}
		})
	}
}

func TestGenerateTestStub_TableDrivenFormat(t *testing.T) {
	cases := []TestCase{
		{Name: "case one", Input: "a", Expected: "b"},
		{Name: "case two", Input: "c", Expected: "d"},
	}
	got := GenerateTestStub("MyFunc", "pkg", cases)

	// Verify table-driven structure
	if !strings.Contains(got, "var tests = []testScenario{") {
		t.Error("missing var tests declaration")
	}
	if !strings.Contains(got, "for _, tt := range tests") {
		t.Error("missing range iteration")
	}
	if !strings.Contains(got, "t.Run(tt.Name") {
		t.Error("missing t.Run")
	}
	if !strings.Contains(got, "tt.Input") {
		t.Error("missing tt.Input reference")
	}
	if !strings.Contains(got, "tt.Expected") {
		t.Error("missing tt.Expected reference")
	}
}

func TestReadFuncSource(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		line        int
		wantBody    string
		wantErr     bool
	}{
		{
			name: "simple function at declaration line",
			fileContent: `package example

func Hello() string {
	return "hello"
}
`,
			line:     3,
			wantBody: `func Hello() string {`,
			wantErr:  false,
		},
		{
			name: "function with body — line inside function",
			fileContent: `package main

func Add(a, b int) int {
	return a + b
}

func main() {}
`,
			line:     4,
			wantBody: "func Add",
			wantErr:  false,
		},
		{
			name: "line beyond any function",
			fileContent: `package main

func Foo() {}
`,
			line:     99,
			wantBody: "",
			wantErr:  true,
		},
		{
			name:        "line zero returns error",
			fileContent: `package main`,
			line:        0,
			wantBody:    "",
			wantErr:     true,
		},
		{
			name: "line in package clause — no function there",
			fileContent: `package main

func Foo() {}
`,
			line:     1,
			wantBody: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			filePath := filepath.Join(tmp, "test.go")
			if err := os.WriteFile(filePath, []byte(tt.fileContent), 0o644); err != nil {
				t.Fatalf("write temp file: %v", err)
			}

			got, err := ReadFuncSource(filePath, tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFuncSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(got, tt.wantBody) {
				t.Errorf("ReadFuncSource() = %q, want to contain %q", got, tt.wantBody)
			}
		})
	}
}

func TestReadFuncSource_ExtractsFullFunction(t *testing.T) {
	source := `package example

import "fmt"

// Greet returns a greeting.
func Greet(name string) string {
	if name == "" {
		name = "world"
	}
	return fmt.Sprintf("hello %s", name)
}

func Helper() int { return 42 }
`
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "example.go")
	if err := os.WriteFile(filePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ReadFuncSource(filePath, 6) // Greet starts at line 6
	if err != nil {
		t.Fatalf("ReadFuncSource: %v", err)
	}

	// Must contain the full function
	if !strings.Contains(got, "func Greet") {
		t.Error("result should contain func Greet")
	}
	if !strings.Contains(got, `return fmt.Sprintf`) {
		t.Error("result should contain the return statement")
	}
	// Must NOT contain the next function
	if strings.Contains(got, "func Helper") {
		t.Error("result should not contain func Helper")
	}
}

func TestReadFuncSource_BraceInStringLiteral(t *testing.T) {
	// This is the primary edge case that the old brace-counting implementation
	// got wrong: string literals containing } would cause premature termination.
	source := `package example

import "errors"

// ReturnBrace returns a string containing braces.
func ReturnBrace() string {
	return "}"
}

func HasJSON() string {
	return ` + "`{`" + `
}

func MultiBrace() string {
	s := "{{}}"
	if s == "{{}}" {
		return s
	}
	return ""
}

func CommentBrace() string {
	// This comment has a brace: }
	return "ok"
}
`
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "brace.go")
	if err := os.WriteFile(filePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	tests := []struct {
		line int
		want string
	}{
		{6, `func ReturnBrace()`},
		{10, `func HasJSON()`},
		{14, `func MultiBrace()`},
		{22, `func CommentBrace()`},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got, err := ReadFuncSource(filePath, tt.line)
			if err != nil {
				t.Fatalf("ReadFuncSource line %d: %v", tt.line, err)
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("line %d: result should contain %q, got:\n%s", tt.line, tt.want, got)
			}
			// Must not bleed into the next function
			if strings.Contains(got, "func ReturnBrace") && tt.want != "func ReturnBrace()" {
				t.Errorf("line %d: result should not contain func ReturnBrace", tt.line)
			}
			if strings.Contains(got, "func HasJSON") && tt.want != "func HasJSON()" {
				t.Errorf("line %d: result should not contain func HasJSON", tt.line)
			}
			if strings.Contains(got, "func MultiBrace") && tt.want != "func MultiBrace()" {
				t.Errorf("line %d: result should not contain func MultiBrace", tt.line)
			}
			if strings.Contains(got, "func CommentBrace") && tt.want != "func CommentBrace()" {
				t.Errorf("line %d: result should not contain func CommentBrace", tt.line)
			}
		})
	}
}

func TestReadFuncSource_NonexistentFile(t *testing.T) {
	_, err := ReadFuncSource("/nonexistent/file.go", 1)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestFormatTestFile(t *testing.T) {
	tests := []struct {
		name        string
		packageName string
		imports     []string
		testFuncs   []string
		wantContain []string
	}{
		{
			name:        "basic test file",
			packageName: "mypackage",
			imports:     []string{"testing"},
			testFuncs:   []string{"func TestFoo(t *testing.T) { t.Log(\"foo\") }"},
			wantContain: []string{
				"package mypackage",
				`import (`,
				`"testing"`,
				`)`,
				"func TestFoo",
			},
		},
		{
			name:        "multiple imports",
			packageName: "mypackage",
			imports:     []string{"testing", "os"},
			testFuncs:   []string{"func TestBar(t *testing.T) {}"},
			wantContain: []string{
				`"testing"`,
				`"os"`,
			},
		},
		{
			name:        "no imports",
			packageName: "mypackage",
			imports:     []string{},
			testFuncs:   []string{"func TestBaz(t *testing.T) {}"},
			wantContain: []string{
				"package mypackage",
				"func TestBaz",
			},
		},
		{
			name:        "multiple test functions",
			packageName: "pkg",
			imports:     []string{"testing"},
			testFuncs: []string{
				"func TestA(t *testing.T) {}",
				"func TestB(t *testing.T) {}",
			},
			wantContain: []string{
				"func TestA",
				"func TestB",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTestFile(tt.packageName, tt.imports, tt.testFuncs)

			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("FormatTestFile() missing %q\nGot:\n%s", want, got)
				}
			}

			// Package declaration must be first
			lines := strings.Split(strings.TrimSpace(got), "\n")
			if len(lines) == 0 {
				t.Fatal("FormatTestFile returned empty output")
			}
			if !strings.HasPrefix(lines[0], "package ") {
				t.Errorf("first line should start with 'package ', got %q", lines[0])
			}
		})
	}
}

func TestFormatTestFile_NoDuplicateImports(t *testing.T) {
	got := FormatTestFile("pkg", []string{"testing", "testing"}, []string{"func TestA(t *testing.T) {}"})

	// Count occurrences of "testing" in import block
	count := strings.Count(got, `"testing"`)
	if count != 1 {
		t.Errorf("expected 1 occurrence of \"testing\" import, got %d", count)
	}
}

func TestFormatTestFile_ValidGoStructure(t *testing.T) {
	got := FormatTestFile("mypackage", []string{"testing"}, []string{
		"func TestExample(t *testing.T) {\n\tt.Log(\"hello\")\n}",
	})

	// Verify structure order: package, imports, test funcs
	packageIdx := strings.Index(got, "package mypackage")
	importIdx := strings.Index(got, "import")
	testIdx := strings.Index(got, "func TestExample")

	if packageIdx == -1 || importIdx == -1 || testIdx == -1 {
		t.Fatal("missing required sections in output")
	}
	if !(packageIdx < importIdx && importIdx < testIdx) {
		t.Errorf("wrong order: package(%d) < import(%d) < test(%d)", packageIdx, importIdx, testIdx)
	}
}

func TestFormatTestFile_ValidGoStructure_WithTestCase(t *testing.T) {
	got := FormatTestFile("mypackage", []string{"testing"}, []string{
		"func TestExample(t *testing.T) {\n\tvar tests = []testScenario{{Name: \"a\"}}\n\t_ = tests\n}",
	})

	// Verify structure order: package, imports, testScenario struct, test funcs
	packageIdx := strings.Index(got, "package mypackage")
	importIdx := strings.Index(got, "import")
	tcIdx := strings.Index(got, "type testScenario struct")
	testIdx := strings.Index(got, "func TestExample")

	if packageIdx == -1 || importIdx == -1 || tcIdx == -1 || testIdx == -1 {
		t.Fatal("missing required sections in output")
	}
	if !(packageIdx < importIdx && importIdx < tcIdx && tcIdx < testIdx) {
		t.Errorf("wrong order: package(%d) < import(%d) < tc(%d) < test(%d)",
			packageIdx, importIdx, tcIdx, testIdx)
	}
}

func TestCoverageGap_Fields(t *testing.T) {
	gap := CoverageGap{
		File:     "internal/foo/bar.go",
		Function: "Bar",
		Coverage: 45.5,
		Line:     10,
	}
	if gap.File != "internal/foo/bar.go" {
		t.Errorf("File = %q, want %q", gap.File, "internal/foo/bar.go")
	}
	if gap.Function != "Bar" {
		t.Errorf("Function = %q, want %q", gap.Function, "Bar")
	}
	if gap.Coverage != 45.5 {
		t.Errorf("Coverage = %.1f, want 45.5", gap.Coverage)
	}
	if gap.Line != 10 {
		t.Errorf("Line = %d, want 10", gap.Line)
	}
}

func TestTestCase_Fields(t *testing.T) {
	tc := TestCase{
		Name:     "positive",
		Input:    "1, 2",
		Expected: "3",
	}
	if tc.Name != "positive" {
		t.Errorf("Name = %q, want %q", tc.Name, "positive")
	}
	if tc.Input != "1, 2" {
		t.Errorf("Input = %q, want %q", tc.Input, "1, 2")
	}
	if tc.Expected != "3" {
		t.Errorf("Expected = %q, want %q", tc.Expected, "3")
	}
}

func TestFormatTestFile_IncludesTestCaseStruct(t *testing.T) {
	testBody := GenerateTestStub("Add", "math", []TestCase{
		{Name: "positive", Input: "1, 2", Expected: "3"},
	})

	got := FormatTestFile("math", []string{"testing"}, []string{testBody})

	// Must contain the testScenario type definition.
	if !strings.Contains(got, "type testScenario struct") {
		t.Error("FormatTestFile() missing 'type testScenario struct' definition")
	}
	if !strings.Contains(got, "Name") || !strings.Contains(got, "Input") || !strings.Contains(got, "Expected") {
		t.Error("FormatTestFile() testScenario struct missing required fields")
	}

	// Verify the output parses as valid Go.
	_, err := parser.ParseFile(token.NewFileSet(), "", got, parser.ParseComments)
	if err != nil {
		t.Errorf("FormatTestFile() output does not parse as valid Go: %v\nGot:\n%s", err, got)
	}
}

func TestGeneratedCode_Compiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compilation test in short mode")
	}

	tmp := t.TempDir()

	// Initialize a Go module so `go build' can resolve packages.
	initCmd := exec.Command("go", "mod", "init", "example.com/math")
	initCmd.Dir = tmp
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}

	// Write a simple source file.
	src := `package math

func Add(a, b int) int { return a + b }
`
	if err := os.WriteFile(filepath.Join(tmp, "add.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	// Generate a test file that references testScenario.
	testBody := GenerateTestStub("Add", "math", []TestCase{
		{Name: "positive", Input: "1, 2", Expected: "3"},
	})
	testFile := FormatTestFile("math", []string{"testing"}, []string{testBody})

	if err := os.WriteFile(filepath.Join(tmp, "add_test.go"), []byte(testFile), 0o644); err != nil {
		t.Fatalf("write test: %v", err)
	}

	// Verify the generated file compiles.
	cmd := exec.Command("go", "test", "-run", "^$", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("generated test file does not compile: %v\nOutput:\n%s\nGenerated:\n%s", err, out, testFile)
	}
}
