package ciloop

import "strings"

// Classification describes how a failing CI check should be handled.
type Classification string

const (
	ClassAutoFixable Classification = "auto-fixable"
	ClassEscalate    Classification = "escalate"
)

// autoFixablePatterns are substrings (lowercased) that map to auto-fixable checks.
var autoFixablePatterns = []string{
	"go-test",
	"go-build",
	"go test",
	"go build",
	"k8s-validate",
	"k8s validate",
}

// Classify returns the classification for a failing CI check by name.
// Unknown checks default to Escalate (fail-safe).
func Classify(checkName string) Classification {
	lower := strings.ToLower(checkName)
	for _, p := range autoFixablePatterns {
		if strings.Contains(lower, p) {
			return ClassAutoFixable
		}
	}
	return ClassEscalate
}
