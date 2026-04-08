package discovery

import "strings"

var DefaultPlannerModel   = "deepseek/deepseek-v3.2"
var DefaultSynthModel     = "deepseek/deepseek-v3.2"
var DefaultReasonerModel  = "openai/gpt-5.4-mini"  // reserved for Phase 4 VALIDATE
var DefaultOpenRouterBase = "https://openrouter.ai/api/v1"

// stripMarkdownFences removes ```...``` fences that models sometimes add despite instructions.
// Handles trailing prose after the closing fence.
func stripMarkdownFences(content string) string {
	if !strings.HasPrefix(content, "```") {
		return content
	}
	// skip language tag line (e.g. ```json)
	idx := strings.Index(content, "\n")
	if idx == -1 {
		return content
	}
	content = content[idx+1:]
	// strip everything from the last closing fence onward
	if end := strings.LastIndex(content, "\n```"); end != -1 {
		content = content[:end]
	}
	return strings.TrimSpace(content)
}
