package harness

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ClassifyClarificationText(text string) ClarificationDecision {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return ClarificationDecision{Classification: ClarificationNoImpact}
	}

	policyKeywords := []string{
		"security", "secret", "token", "credential", "billing", "production", "compliance", "pii",
		"безопас", "секрет", "токен", "креденш", "прод", "комплаенс", "персональ",
	}
	for _, kw := range policyKeywords {
		if strings.Contains(normalized, kw) {
			return ClarificationDecision{
				Classification:   ClarificationPolicySensitive,
				RequiresApproval: true,
				Blocking:         true,
				Reasons:          []string{"policy-sensitive clarification text detected"},
			}
		}
	}

	reductiveKeywords := []string{
		"remove", "drop", "skip", "disable", "less", "relax", "weaker", "ignore",
		"убери", "удали", "отключ", "пропуст", "ослаб", "меньше", "игнор",
	}
	for _, kw := range reductiveKeywords {
		if strings.Contains(normalized, kw) {
			return ClarificationDecision{
				Classification:   ClarificationReductive,
				RequiresApproval: true,
				Blocking:         true,
				Reasons:          []string{"reductive clarification text detected"},
			}
		}
	}

	additiveKeywords := []string{
		"add", "also", "plus", "extend", "tighten", "stricter", "extra",
		"добав", "еще", "плюс", "усиль", "строже", "дополн",
	}
	for _, kw := range additiveKeywords {
		if strings.Contains(normalized, kw) {
			return ClarificationDecision{
				Classification: ClarificationAdditive,
				Reasons:        []string{"additive clarification text detected"},
			}
		}
	}

	return ClarificationDecision{Classification: ClarificationNoImpact, Reasons: []string{"no-impact clarification text"}}
}

func ClassifyClarification(change *ClarificationChange) ClarificationDecision {
	if change == nil {
		return ClarificationDecision{Classification: ClarificationNoImpact}
	}

	decision := ClarificationDecision{Classification: ClarificationNoImpact}

	if change.PolicySensitive {
		decision.Classification = ClarificationPolicySensitive
		decision.RequiresApproval = true
		decision.Blocking = true
		decision.Reasons = append(decision.Reasons, "policy_sensitive flag is set")
		return decision
	}

	hasAdditions := len(change.AddAcceptanceCriteria) > 0 ||
		len(change.AddMetrics) > 0 ||
		len(change.AddEvidence) > 0 ||
		len(change.EnableQualityGates) > 0

	hasReductions := len(change.RemoveAcceptanceCriteria) > 0 ||
		len(change.RemoveMetrics) > 0 ||
		len(change.RemoveEvidence) > 0 ||
		len(change.DisableQualityGates) > 0

	if hasReductions {
		decision.Classification = ClarificationReductive
		decision.RequiresApproval = true
		decision.Blocking = true
		decision.Reasons = append(decision.Reasons, "clarification reduces AC/metrics/evidence/quality gates")
	}

	if hasAdditions && decision.Classification == ClarificationNoImpact {
		decision.Classification = ClarificationAdditive
		decision.Reasons = append(decision.Reasons, "clarification adds scope or stricter checks")
	}

	if hasAdditions && hasReductions {
		decision.Reasons = append(decision.Reasons, "contains both additive and reductive changes")
	}

	return decision
}

func ApplyClarification(contract *TaskContract, change *ClarificationChange, approvedBy string, approvedAt time.Time) (ClarificationDecision, error) {
	if contract == nil {
		return ClarificationDecision{}, fmt.Errorf("contract is required")
	}
	decision := ClassifyClarification(change)
	if change == nil || decision.Classification == ClarificationNoImpact {
		return decision, nil
	}

	if decision.RequiresApproval && strings.TrimSpace(approvedBy) == "" {
		return decision, fmt.Errorf("approval is required for %s clarification", decision.Classification)
	}

	if approvedAt.IsZero() {
		approvedAt = time.Now().UTC()
	}

	addAcceptanceCriteria(contract, change.AddAcceptanceCriteria)
	removeAcceptanceCriteria(contract, change.RemoveAcceptanceCriteria)
	addMetrics(contract, change.AddMetrics)
	removeMetrics(contract, change.RemoveMetrics)
	addEvidence(contract, change.AddEvidence)
	removeEvidence(contract, change.RemoveEvidence)
	setQualityGates(contract, change.EnableQualityGates, true)
	setQualityGates(contract, change.DisableQualityGates, false)

	contract.Version = bumpVersion(contract.Version)
	appendChangeRequest(contract, change, decision, approvedBy, approvedAt)

	return decision, nil
}

func addAcceptanceCriteria(contract *TaskContract, add []AcceptanceCriterion) {
	existing := make(map[string]struct{}, len(contract.AcceptanceCriteria))
	for _, ac := range contract.AcceptanceCriteria {
		existing[ac.ID] = struct{}{}
	}
	for _, ac := range add {
		if ac.ID == "" {
			continue
		}
		if _, ok := existing[ac.ID]; ok {
			continue
		}
		contract.AcceptanceCriteria = append(contract.AcceptanceCriteria, ac)
		existing[ac.ID] = struct{}{}
	}
}

func removeAcceptanceCriteria(contract *TaskContract, remove []string) {
	if len(remove) == 0 {
		return
	}
	drop := make(map[string]struct{}, len(remove))
	for _, id := range remove {
		drop[id] = struct{}{}
	}
	filtered := contract.AcceptanceCriteria[:0]
	for _, ac := range contract.AcceptanceCriteria {
		if _, ok := drop[ac.ID]; ok {
			continue
		}
		filtered = append(filtered, ac)
	}
	contract.AcceptanceCriteria = filtered
}

func addMetrics(contract *TaskContract, add []RequiredMetric) {
	existing := make(map[string]struct{}, len(contract.RequiredMetrics))
	for _, metric := range contract.RequiredMetrics {
		existing[metric.Name] = struct{}{}
	}
	for _, metric := range add {
		if metric.Name == "" {
			continue
		}
		if _, ok := existing[metric.Name]; ok {
			continue
		}
		contract.RequiredMetrics = append(contract.RequiredMetrics, metric)
		existing[metric.Name] = struct{}{}
	}
}

func removeMetrics(contract *TaskContract, remove []string) {
	if len(remove) == 0 {
		return
	}
	drop := make(map[string]struct{}, len(remove))
	for _, name := range remove {
		drop[name] = struct{}{}
	}
	filtered := contract.RequiredMetrics[:0]
	for _, metric := range contract.RequiredMetrics {
		if _, ok := drop[metric.Name]; ok {
			continue
		}
		filtered = append(filtered, metric)
	}
	contract.RequiredMetrics = filtered
}

func addEvidence(contract *TaskContract, add []string) {
	existing := make(map[string]struct{}, len(contract.RequiredEvidence))
	for _, ev := range contract.RequiredEvidence {
		existing[ev] = struct{}{}
	}
	for _, ev := range add {
		if ev == "" {
			continue
		}
		if _, ok := existing[ev]; ok {
			continue
		}
		contract.RequiredEvidence = append(contract.RequiredEvidence, ev)
		existing[ev] = struct{}{}
	}
}

func removeEvidence(contract *TaskContract, remove []string) {
	if len(remove) == 0 {
		return
	}
	drop := make(map[string]struct{}, len(remove))
	for _, ev := range remove {
		drop[ev] = struct{}{}
	}
	filtered := contract.RequiredEvidence[:0]
	for _, ev := range contract.RequiredEvidence {
		if _, ok := drop[ev]; ok {
			continue
		}
		filtered = append(filtered, ev)
	}
	contract.RequiredEvidence = filtered
}

func setQualityGates(contract *TaskContract, gates []string, value bool) {
	for _, gate := range gates {
		switch strings.ToLower(gate) {
		case "build":
			contract.QualityGates.Build = value
		case "test":
			contract.QualityGates.Test = value
		case "lint":
			contract.QualityGates.Lint = value
		case "typecheck":
			contract.QualityGates.Typecheck = value
		}
	}
}

func appendChangeRequest(contract *TaskContract, change *ClarificationChange, decision ClarificationDecision, approvedBy string, approvedAt time.Time) {
	crID := strings.TrimSpace(change.ID)
	if crID == "" {
		crID = fmt.Sprintf("CR-%d", len(contract.ChangeRequests)+1)
	}
	reason := strings.TrimSpace(change.Reason)
	if reason == "" {
		reason = strings.Join(decision.Reasons, "; ")
	}
	approver := approvedBy
	if strings.TrimSpace(approver) == "" {
		approver = "system:auto"
	}
	contract.ChangeRequests = append(contract.ChangeRequests, ChangeRequest{
		ID:         crID,
		Reason:     reason,
		ApprovedBy: approver,
		ApprovedAt: approvedAt.UTC().Format(time.RFC3339),
	})
}

func bumpVersion(version string) string {
	if !strings.HasPrefix(version, "v") {
		return "v1"
	}
	n, err := strconv.Atoi(strings.TrimPrefix(version, "v"))
	if err != nil {
		return "v1"
	}
	return fmt.Sprintf("v%d", n+1)
}
