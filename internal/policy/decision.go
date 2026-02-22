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

// allowedProviderModels: provider/model pairs. OpenRouter uses provider=openai|anthropic etc.
// GLM via zhipuai-coding-plan (coding plan), not OpenRouter.
var allowedProviderModels = map[string]struct{}{
	"zhipuai-coding-plan/glm-5":   {},
	"zhipuai-coding-plan/glm-4.7": {},
	"openai/gpt-5.2-codex":        {},
	"anthropic/claude-sonnet-4.6": {},
	"anthropic/claude-opus-4.6":   {},
	"minimax/minimax-m2.5":        {},
	"moonshotai/kimi-k2.5":        {},
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
	regexp.MustCompile(`k8s`),
}

type DecisionRequest struct {
	IssueID        string   `json:"issue_id"`
	Title          string   `json:"title"`
	Lane           string   `json:"lane"`
	Role           string   `json:"role"` // For 3-tier per-role fallback (WS-012-02)
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
	model, reasons, escalation := chooseModel(req.PreferredModel, req.Role)
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
	fallbackChain := ResolveFallbackSequence(model)
	if req.Role != "" {
		fallbackChain = ResolveFallbackSequenceFromRole(req.Role, model)
	}

	return DecisionResponse{
		PolicyVerdict:      verdict,
		RiskClass:          risk,
		SelectedModel:      model,
		FallbackChain:      fallbackChain,
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

// TierForModel returns primary, fallback, or economy based on model position in chain.
func TierForModel(chain []string, model string) string {
	for i, m := range chain {
		if m == model {
			switch i {
			case 0:
				return "primary"
			case 1:
				return "fallback"
			case 2:
				return "economy"
			default:
				return "economy"
			}
		}
	}
	return "unknown"
}

func chooseModel(preferred, role string) (string, []string, bool) {
	defaultModel := DefaultModel()
	if role != "" {
		defaultModel = RoleDefaultModel(role)
	}
	if preferred == "" {
		return defaultModel, nil, false
	}
	if AllowedModel(preferred) {
		return preferred, nil, false
	}
	return defaultModel, []string{"preferred_model '" + preferred + "' not in allowlist"}, true
}

// AllowedModel checks if model is in allowlist. Uses config when loaded, else built-in.
func AllowedModel(model string) bool {
	return AllowedModelFromConfig(model)
}

func DefaultModel() string { return "glm-5" }
