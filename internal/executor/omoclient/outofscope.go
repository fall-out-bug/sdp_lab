package omoclient

import (
	"context"
	"fmt"
	"path/filepath"
)

// OutOfScopeReport contains scope check results
type OutOfScopeReport struct {
	Violations []string `json:"violations"`
	Clean      bool     `json:"clean"`
}

// OutOfScopeChecker validates files against allowed/denied glob patterns
type OutOfScopeChecker struct {
	AllowedFiles []string
	DeniedFiles  []string
}

// NewOutOfScopeChecker creates a new scope checker
func NewOutOfScopeChecker(allowed, denied []string) *OutOfScopeChecker {
	return &OutOfScopeChecker{
		AllowedFiles: allowed,
		DeniedFiles:  denied,
	}
}

// globMatch tries pattern against both full path and basename.
func globMatch(pattern, path string) bool {
	normalized := filepath.ToSlash(path)
	if m, err := filepath.Match(pattern, normalized); err == nil && m {
		return true
	}
	base := filepath.Base(path)
	if m, err := filepath.Match(pattern, base); err == nil && m {
		return true
	}
	return false
}

// Check validates actual files against allowed/denied patterns
func (c *OutOfScopeChecker) Check(ctx context.Context, actualFiles []string) OutOfScopeReport {
	report := OutOfScopeReport{
		Violations: []string{},
		Clean:      true,
	}

	if ctx.Err() != nil {
		return report
	}

	for _, file := range actualFiles {
		denied := false
		for _, pattern := range c.DeniedFiles {
			if globMatch(pattern, file) {
				report.Violations = append(report.Violations, fmt.Sprintf("%s matches denied pattern: %s", file, pattern))
				denied = true
				report.Clean = false
				break
			}
		}

		if denied {
			continue
		}

		if len(c.AllowedFiles) == 0 {
			// Empty allowed list = deny all
			report.Violations = append(report.Violations, fmt.Sprintf("%s not in allowed patterns (empty allow list)", file))
			report.Clean = false
		} else {
			allowed := false
			for _, pattern := range c.AllowedFiles {
				if globMatch(pattern, file) {
					allowed = true
					break
				}
			}

			if !allowed {
				report.Violations = append(report.Violations, fmt.Sprintf("%s not in allowed patterns", file))
				report.Clean = false
			}
		}
	}

	return report
}
