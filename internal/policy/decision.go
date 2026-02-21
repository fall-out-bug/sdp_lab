package policy

import (
	"regexp"
	"strings"
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

var allowedModels = map[string]struct{}{
	"glm-5":   {},
	"glm-4.7": {},
}

// allowedProviderModels: provider/model pairs. OpenRouter uses provider=openai|anthropic|google etc.
// See https://openrouter.ai/models for current model IDs.
var allowedProviderModels = map[string]struct{}{
	"zhipuai-coding-plan/glm-5":   {},
	"zhipuai-coding-plan/glm-4.7": {},
	"openai/gpt-4o":              {},
	"openai/gpt-4o-mini":         {},
	"anthropic/claude-sonnet-4.6": {},
	"anthropic/claude-opus-4.6":   {},
	"google/gemini-2.5-pro":      {},
	"z-ai/glm-5":                 {},
}

var criticalPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(^|/)security(/|$)`),
	regexp.MustCompile(`(^|/)auth(/|$)`),
	regexp.MustCompile(`(^|/)secrets?(/|$)`),
	regexp.MustCompile(`policy`),
}

var highPatterns = []*regexp.Regexp{
	regexp.MustCompile(`orchestr`),
	regexp.MustCompile(`beads`),
	regexp.MustCompile(`evidence`),
	regexp.MustCompile(`git`),
}

type DecisionRequest struct {
	IssueID        string   `json:"issue_id"`
	Title          string   `json:"title"`
	Lane           string   `json:"lane"`
	PreferredModel string   `json:"preferred_model"`
	ChangedPaths   []string `json:"changed_paths"`
}

type DecisionResponse struct {
	PolicyVerdict      string   `json:"policy_verdict"`
	RiskClass          string   `json:"risk_class"`
	SelectedModel      string   `json:"selected_model"`
	FallbackChain      []string `json:"fallback_chain"`
	BranchName         string   `json:"branch_name"`
	EscalationRequired bool     `json:"escalation_required"`
	Reasons            []string `json:"reasons"`
	Lane               string   `json:"lane"`
}

func Decide(req DecisionRequest) DecisionResponse {
	risk := classifyRisk(req.ChangedPaths)
	model, reasons, escalation := chooseModel(req.PreferredModel)
	if risk == "critical" {
		escalation = true
		reasons = append(reasons, "critical-risk change requires human security gate")
	}
	verdict := "allow"
	if escalation {
		verdict = "escalate"
	}
	lane := req.Lane
	if lane == "" {
		lane = "commit"
	}

	return DecisionResponse{
		PolicyVerdict:      verdict,
		RiskClass:          risk,
		SelectedModel:      model,
		FallbackChain:      ResolveFallbackSequence(model),
		BranchName:         BuildBranchName(req.IssueID, req.Title),
		EscalationRequired: escalation,
		Reasons:            reasons,
		Lane:               lane,
	}
}

func BuildBranchName(issueID, title string) string {
	return "feat/" + issueID + "-" + slugify(title)
}

func slugify(text string) string {
	t := strings.ToLower(text)
	t = nonAlnum.ReplaceAllString(t, "-")
	t = strings.Trim(t, "-")
	if t == "" {
		return "task"
	}
	if len(t) > 48 {
		t = t[:48]
		t = strings.Trim(t, "-")
	}
	if t == "" {
		return "task"
	}
	return t
}

func classifyRisk(paths []string) string {
	for _, p := range paths {
		pp := strings.ToLower(p)
		for _, pattern := range criticalPatterns {
			if pattern.MatchString(pp) {
				return "critical"
			}
		}
	}
	for _, p := range paths {
		pp := strings.ToLower(p)
		for _, pattern := range highPatterns {
			if pattern.MatchString(pp) {
				return "high"
			}
		}
	}
	if len(paths) == 0 {
		return "low"
	}
	return "medium"
}

func chooseModel(preferred string) (string, []string, bool) {
	if preferred == "" {
		return "glm-5", nil, false
	}
	if _, ok := allowedModels[preferred]; ok {
		return preferred, nil, false
	}
	if _, ok := allowedProviderModels[preferred]; ok {
		return preferred, nil, false
	}
	return "glm-5", []string{"preferred_model '" + preferred + "' not in allowlist"}, true
}

func AllowedModel(model string) bool {
	if _, ok := allowedProviderModels[model]; ok {
		return true
	}
	_, modelID := ParseProviderModel(model)
	if modelID != "" {
		model = modelID
	}
	_, ok := allowedModels[model]
	return ok
}

func DefaultModel() string { return "glm-5" }
