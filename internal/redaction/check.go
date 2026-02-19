package redaction

import (
	"regexp"
	"strings"
)

var forbiddenPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)pricing`),
	regexp.MustCompile(`(?i)margin`),
	regexp.MustCompile(`(?i)sales assumptions`),
	regexp.MustCompile(`(?i)internal hostnames?`),
	regexp.MustCompile(`(?i)data residency exceptions?`),
	regexp.MustCompile(`(?i)private policy`),
	regexp.MustCompile(`(?i)customer-specific`),
	regexp.MustCompile(`(?i)secrets?`),
	regexp.MustCompile(`(?i)credential`),
}

type Violation struct {
	Pattern string `json:"pattern"`
	Line    int    `json:"line"`
	Text    string `json:"text"`
}

func CheckContent(content string) []Violation {
	lines := strings.Split(content, "\n")
	violations := make([]Violation, 0)
	for i, line := range lines {
		for _, pattern := range forbiddenPatterns {
			if pattern.MatchString(line) {
				violations = append(violations, Violation{
					Pattern: pattern.String(),
					Line:    i + 1,
					Text:    strings.TrimSpace(line),
				})
			}
		}
	}
	return violations
}
