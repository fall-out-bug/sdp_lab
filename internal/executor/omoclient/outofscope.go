package omoclient

import (
	"context"
	"fmt"

	"sdp_dev/internal/glob"
)

// OutOfScopeReport contains scope check results
type OutOfScopeReport struct {
	Violations []string `json:"violations"`
	Clean      bool     `json:"clean"`
}

// OutOfScopeChecker validates files against allowed/denied glob patterns
type OutOfScopeChecker struct {
	allowedMatcher *glob.Matcher
	deniedMatcher  *glob.Matcher
}

// NewOutOfScopeChecker creates a new scope checker with pre-compiled matchers
func NewOutOfScopeChecker(allowed, denied []string) *OutOfScopeChecker {
	return &OutOfScopeChecker{
		allowedMatcher: glob.NewMatcher(allowed),
		deniedMatcher:  glob.NewMatcher(denied),
	}
}

// Check validates actual files against allowed/denied patterns using optimized matchers
func (c *OutOfScopeChecker) Check(ctx context.Context, actualFiles []string) OutOfScopeReport {
	report := OutOfScopeReport{
		Violations: []string{},
		Clean:      true,
	}

	if ctx.Err() != nil {
		return report
	}

	for _, file := range actualFiles {
		// Check denied patterns first
		if c.deniedMatcher != nil {
			if matched := c.deniedMatcher.MatchAnyPattern(file); matched != "" {
				report.Violations = append(report.Violations, fmt.Sprintf("%s matches denied pattern: %s", file, matched))
				report.Clean = false
				continue
			}
		}

		// Check allowed patterns
		if c.allowedMatcher == nil || len(c.allowedMatcher.Patterns()) == 0 {
			// Empty allowed list = deny all
			report.Violations = append(report.Violations, fmt.Sprintf("%s not in allowed patterns (empty allow list)", file))
			report.Clean = false
		} else if !c.allowedMatcher.MatchAny(file) {
			report.Violations = append(report.Violations, fmt.Sprintf("%s not in allowed patterns", file))
			report.Clean = false
		}
	}

	return report
}
