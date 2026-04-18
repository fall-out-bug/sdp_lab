package dispatch

import "strings"

// LocalConfig enables routing of low-complexity coding tasks to a local Ollama model.
// Set Router.LocalConfig to activate; nil = local routing disabled.
type LocalConfig struct {
	BaseURL string // e.g. "http://localhost:11434"
	Model   string // e.g. "qwen2.5-coder:7b"
	// Score is the fixed score assigned to the local profile when a task is low-complexity.
	// Defaults to 0.9 if zero — high enough to win over typical cloud scores when task fits.
	Score float64
}

func (c *LocalConfig) score() float64 {
	if c.Score > 0 {
		return c.Score
	}
	return 0.9
}

// localComplexityKeywords are workstream keywords that signal a function-level task
// that a local model can handle without broader codebase context.
var localComplexityKeywords = []string{
	"stub", "boilerplate", "test", "add_field", "add field",
	"rename", "simple", "implement interface", "docstring",
}

// IsLowComplexity returns true when ws (case-insensitive) matches keywords that
// indicate a self-contained, function-level task suitable for local model execution.
func IsLowComplexity(ws string) bool {
	lower := strings.ToLower(ws)
	for _, kw := range localComplexityKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// localProfile returns a synthetic CapabilityProfile representing the local Ollama model.
func localProfile(cfg *LocalConfig) *CapabilityProfile {
	return &CapabilityProfile{
		Harness:  "opencode",
		Provider: "ollama",
		Model:    cfg.Model,
	}
}
