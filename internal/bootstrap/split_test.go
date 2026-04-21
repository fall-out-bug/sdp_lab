package bootstrap

import (
	"strings"
	"testing"
)

func TestSplitContent_PrinciplesOnly(t *testing.T) {
	input := `# Philosophy

This project values simplicity because complexity kills productivity.
The reason we prefer explicit code is readability.
Our philosophy centers on small functions.
The value of determinism cannot be overstated.`

	result := SplitContent(input)
	if result.Principles == "" {
		t.Fatal("expected non-empty principles")
	}
	if result.Rules != "" {
		t.Fatalf("expected empty rules, got: %q", result.Rules)
	}
	if !strings.Contains(result.Principles, "simplicity") {
		t.Error("principles should contain rationale text")
	}
}

func TestSplitContent_RulesOnly(t *testing.T) {
	input := `# Conventions

Always run tests before committing.
Never commit directly to main.
Must use conventional commits.
Should keep functions under 60 lines.
Use table-driven tests.
Avoid global state.
Prefer composition over inheritance.`

	result := SplitContent(input)
	if result.Rules == "" {
		t.Fatal("expected non-empty rules")
	}
	if result.Principles != "" {
		t.Fatalf("expected empty principles, got: %q", result.Principles)
	}
	if !strings.Contains(result.Rules, "Always run tests") {
		t.Error("rules should contain directive text")
	}
}

func TestSplitContent_Mixed(t *testing.T) {
	input := `# Architecture

We value simplicity because it reduces cognitive load.

# Rules

Always write tests first.
Never skip code review.`

	result := SplitContent(input)
	if result.Principles == "" {
		t.Fatal("expected non-empty principles for mixed input")
	}
	if result.Rules == "" {
		t.Fatal("expected non-empty rules for mixed input")
	}
	if !strings.Contains(result.Principles, "simplicity") {
		t.Error("principles should contain rationale lines")
	}
	if !strings.Contains(result.Rules, "Always write tests") {
		t.Error("rules should contain directive lines")
	}
}

func TestSplitContent_Empty(t *testing.T) {
	result := SplitContent("")
	if result == nil {
		t.Fatal("expected non-nil SplitResult for empty input")
	}
	if result.Principles != "" {
		t.Fatalf("expected empty principles for empty input, got: %q", result.Principles)
	}
	if result.Rules != "" {
		t.Fatalf("expected empty rules for empty input, got: %q", result.Rules)
	}
}

func TestSplitContent_Deterministic(t *testing.T) {
	input := `# Philosophy

We value explicit code because implicit behavior hides bugs.

# Rules

Always use explicit error handling.
Never ignore errors.`

	first := SplitContent(input)
	second := SplitContent(input)

	if first.Principles != second.Principles {
		t.Fatalf("principles not deterministic:\nfirst:  %q\nsecond: %q", first.Principles, second.Principles)
	}
	if first.Rules != second.Rules {
		t.Fatalf("rules not deterministic:\nfirst:  %q\nsecond: %q", first.Rules, second.Rules)
	}
}

func TestRenderPrinciplesFile_ContainsSections(t *testing.T) {
	principles := "We value simplicity because it reduces cognitive load."
	output := RenderPrinciplesFile(principles)

	expectedSections := []string{
		"# PRINCIPLES.md",
		"## Values",
		"## Architecture Philosophy",
		"## Quality Standards",
	}
	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("output missing expected section %q", section)
		}
	}
	if !strings.Contains(output, principles) {
		t.Error("output should contain the provided principles content")
	}
}

func TestRenderRulesSection_ReferencesPrinciples(t *testing.T) {
	rules := "Always write tests first."
	output := RenderRulesSection(rules)

	if !strings.Contains(output, "PRINCIPLES.md") {
		t.Error("rules section should reference PRINCIPLES.md for rationale")
	}
	if !strings.Contains(output, rules) {
		t.Error("output should contain the provided rules content")
	}
}

func TestSplitContent_PreservesHeaders(t *testing.T) {
	input := `# Philosophy

This matters because quality compounds over time.

## Rationale

The reason we test is confidence.

# Rules

Always run linters before push.`

	result := SplitContent(input)

	// Headers should appear in whichever section they contextualize
	combined := result.Principles + "\n" + result.Rules
	if !strings.Contains(combined, "# Philosophy") {
		t.Error("expected # Philosophy header to be preserved")
	}
	if !strings.Contains(combined, "## Rationale") {
		t.Error("expected ## Rationale header to be preserved")
	}
	if !strings.Contains(combined, "# Rules") {
		t.Error("expected # Rules header to be preserved")
	}
}

func TestSplitContent_BlankLinesDontSwitchSection(t *testing.T) {
	input := `We value simplicity because it reduces cognitive load.

This line is still principles because blank lines continue the section.

Always run tests before committing.`

	result := SplitContent(input)
	if !strings.Contains(result.Principles, "simplicity") {
		t.Error("first principle line should be in principles")
	}
	if !strings.Contains(result.Principles, "still principles") {
		t.Error("blank line should not switch section; following line should stay in principles")
	}
	if !strings.Contains(result.Rules, "Always run tests") {
		t.Error("directive line should be in rules")
	}
}
