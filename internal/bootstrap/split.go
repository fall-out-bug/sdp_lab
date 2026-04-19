package bootstrap

import (
	"strings"
)

// SplitResult holds the separated principles and rules extracted from mixed
// content. Principles capture "why" (rationale, philosophy) while rules
// capture "what" (directives, conventions).
type SplitResult struct {
	Principles string `json:"principles"` // "why" content
	Rules      string `json:"rules"`      // "what" content
}

// SplitContent separates principles from rules in mixed content using a
// keyword heuristic. Output is deterministic: identical input always produces
// an identical split.
//
// Heuristic:
//   - Lines containing rationale keywords ("because", "why", "rationale",
//     "reason", "philosophy", "value") are classified as principles.
//   - Lines starting with directive keywords ("always", "never", "must",
//     "should", "use", "avoid", "prefer") are classified as rules.
//   - Blank lines continue the current section context.
//   - Header lines (starting with #) start a new section context.
//   - All other lines default to rules.
func SplitContent(content string) *SplitResult {
	if content == "" {
		return &SplitResult{}
	}

	var principleLines, ruleLines []string
	currentIsPrinciple := false

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			// Blank line: continue current section, add to both to preserve spacing
			if currentIsPrinciple {
				principleLines = append(principleLines, line)
			} else {
				ruleLines = append(ruleLines, line)
			}
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			// Header: start new context, classify by what follows
			// Headers themselves go to the section implied by their content
			if isPrincipleHeader(trimmed) {
				currentIsPrinciple = true
				principleLines = append(principleLines, line)
			} else if isRuleHeader(trimmed) {
				currentIsPrinciple = false
				ruleLines = append(ruleLines, line)
			} else {
				// Neutral header: assign to current context
				if currentIsPrinciple {
					principleLines = append(principleLines, line)
				} else {
					ruleLines = append(ruleLines, line)
				}
			}
			continue
		}

		if isPrincipleLine(trimmed) {
			currentIsPrinciple = true
			principleLines = append(principleLines, line)
		} else {
			currentIsPrinciple = false
			ruleLines = append(ruleLines, line)
		}
	}

	return &SplitResult{
		Principles: strings.TrimRight(strings.Join(principleLines, "\n"), "\n"),
		Rules:      strings.TrimRight(strings.Join(ruleLines, "\n"), "\n"),
	}
}

// RenderPrinciplesFile renders the principles content as a PRINCIPLES.md
// template with standard section headers for Values, Architecture Philosophy,
// and Quality Standards.
func RenderPrinciplesFile(principles string) string {
	var b strings.Builder
	b.Grow(len(principles) + 256)

	b.WriteString("# PRINCIPLES.md\n\n")
	b.WriteString("## Values\n\n")
	b.WriteString(principles)
	b.WriteString("\n\n")
	b.WriteString("## Architecture Philosophy\n\n")
	b.WriteString("<!-- Add architecture rationale here -->\n\n")
	b.WriteString("## Quality Standards\n\n")
	b.WriteString("<!-- Add quality rationale here -->\n")

	return b.String()
}

// RenderRulesSection renders rules content with a reference to PRINCIPLES.md
// so readers know where to find the underlying rationale.
func RenderRulesSection(rules string) string {
	var b strings.Builder
	b.Grow(len(rules) + 128)

	b.WriteString("## Rules\n\n")
	b.WriteString("<!-- For rationale behind these rules, see PRINCIPLES.md -->\n\n")
	b.WriteString(rules)
	b.WriteString("\n")

	return b.String()
}

// isPrincipleLine checks whether a trimmed line contains rationale keywords
// that indicate "why" content.
func isPrincipleLine(line string) bool {
	lower := strings.ToLower(line)
	principleKeywords := []string{
		"because", "why ", "why,", "why.",
		"rationale", "reason", "philosophy", "value",
	}
	for _, kw := range principleKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// isPrincipleHeader checks whether a header line belongs to the principles
// section based on its content.
func isPrincipleHeader(header string) bool {
	lower := strings.ToLower(header)
	return strings.Contains(lower, "philosophy") ||
		strings.Contains(lower, "value") ||
		strings.Contains(lower, "rationale") ||
		strings.Contains(lower, "why")
}

// isRuleHeader checks whether a header line belongs to the rules section
// based on its content.
func isRuleHeader(header string) bool {
	lower := strings.ToLower(header)
	return strings.Contains(lower, "rule") ||
		strings.Contains(lower, "convention") ||
		strings.Contains(lower, "directive") ||
		strings.Contains(lower, "standard")
}
