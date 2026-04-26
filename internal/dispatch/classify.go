package dispatch

import (
	"path/filepath"
	"strings"
)

// ContextPacketSummary is a lightweight summary of the active context packet.
type ContextPacketSummary struct {
	Phase      string
	Workstream string
	ScopeFiles []string
	Risk       string
}

// TaskClassification describes the inferred characteristics of a task.
type TaskClassification struct {
	Phase       string // "discovery", "design", "build", "review", "qa"
	TaskType    string // "research", "architecture", "refactor", "feature", "bugfix", "analysis"
	Language    string // "go", "typescript", etc.
	Complexity  string // "low", "medium", "high"
	Risk        string // "low", "medium", "high"
	RequiredCap string // "reasoning", "coding", "review"
}

// extToLang maps a file extension (without dot) to a language name.
var extToLang = map[string]string{
	"go":  "go",
	"ts":  "typescript",
	"tsx": "typescript",
	"js":  "javascript",
	"jsx": "javascript",
	"py":  "python",
	"rs":  "rust",
}

// Classify infers a TaskClassification from a ContextPacketSummary.
func Classify(pkt ContextPacketSummary) TaskClassification {
	// Start with medium complexity, then check for low-complexity indicators.
	complexity := "medium"
	if IsLowComplexity(pkt.Workstream) {
		complexity = "low"
	}

	// Safety override: high-risk tasks never get low complexity.
	if pkt.Risk == "high" && complexity == "low" {
		complexity = "medium"
	}

	return TaskClassification{
		Phase:       pkt.Phase,
		TaskType:    inferTaskType(pkt.Phase, pkt.Workstream),
		Language:    inferLanguage(pkt.ScopeFiles),
		Complexity:  complexity,
		Risk:        pkt.Risk,
		RequiredCap: inferRequiredCap(pkt.Phase),
	}
}

// inferRequiredCap maps phase to the required capability.
func inferRequiredCap(phase string) string {
	switch phase {
	case "discovery", "design":
		return "reasoning"
	case "review", "qa":
		return "review"
	default:
		return "coding"
	}
}

// inferTaskType derives a task type from the phase and workstream text.
func inferTaskType(phase, workstream string) string {
	// Phase-based overrides come first.
	switch phase {
	case "discovery":
		return "research"
	case "design":
		return "architecture"
	case "review", "qa":
		return "analysis"
	}

	// For other phases, inspect workstream keywords.
	lower := strings.ToLower(workstream)
	if strings.Contains(lower, "refactor") {
		return "refactor"
	}
	if strings.Contains(lower, "fix") || strings.Contains(lower, "bug") {
		return "bugfix"
	}
	return "feature"
}

// inferLanguage picks the most common language from scope file extensions.
func inferLanguage(files []string) string {
	counts := make(map[string]int)
	for _, f := range files {
		ext := strings.TrimPrefix(filepath.Ext(f), ".")
		if lang, ok := extToLang[ext]; ok {
			counts[lang]++
		}
	}

	var best string
	var bestCount int
	for lang, count := range counts {
		if count > bestCount || (count == bestCount && lang < best) {
			best = lang
			bestCount = count
		}
	}
	return best
}

// IsLowComplexity checks if a workstream ID or name indicates low complexity.
// Low-complexity workstreams are typically function-level, self-contained tasks.
func IsLowComplexity(workstream string) bool {
	lower := strings.ToLower(workstream)
	// Keywords that signal low-complexity, function-level tasks suitable for local model execution
	indicators := []string{
		"stub", "boilerplate", "test", "add_field", "add field",
		"rename", "simple", "implement interface", "docstring",
	}
	for _, indicator := range indicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}
