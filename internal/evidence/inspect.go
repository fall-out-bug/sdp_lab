package evidence

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func Inspect(path string, requirePRURL bool) (string, Result, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", Result{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return "", Result{}, err
	}

	if t, _ := raw["_type"].(string); t == StatementType {
		return inspectAttestation(path, requirePRURL)
	}
	return inspectLegacy(path, raw, requirePRURL)
}

func inspectAttestation(path string, requirePRURL bool) (string, Result, error) {
	stmt, err := ReadAttestation(path)
	if err != nil {
		return "", Result{}, err
	}
	res := ValidateAttestation(stmt, requirePRURL)
	if !res.OK {
		return "", res, nil
	}
	return formatAttestationSummary(stmt), res, nil
}

func inspectLegacy(path string, payload map[string]any, requirePRURL bool) (string, Result, error) {
	res := validateLegacyPayload(payload, requirePRURL)
	if !res.OK {
		return "", res, nil
	}
	return formatLegacySummary(payload), res, nil
}

func formatAttestationSummary(stmt CodingWorkflowStatement) string {
	var sb strings.Builder
	p := stmt.Predicate

	fmt.Fprintf(&sb, "format: in-toto attestation (%s)\n", PredicateTypeCodingWorkflow)
	if len(stmt.Subject) > 0 {
		fmt.Fprintf(&sb, "subject: %s\n", stmt.Subject[0].Name)
	}

	sb.WriteString("intent:\n")
	fmt.Fprintf(&sb, "  issue_id: %s\n", p.Intent.IssueID)
	fmt.Fprintf(&sb, "  risk_class: %s\n", p.Intent.RiskClass)
	if len(p.Intent.AcceptanceCriteria) > 0 {
		fmt.Fprintf(&sb, "  acceptance_criteria: %d items\n", len(p.Intent.AcceptanceCriteria))
	}

	sb.WriteString("plan:\n")
	fmt.Fprintf(&sb, "  workstreams: %v\n", p.Plan.Workstreams)

	sb.WriteString("execution:\n")
	fmt.Fprintf(&sb, "  branch: %s\n", p.Execution.Branch)
	fmt.Fprintf(&sb, "  changed_files: %d\n", len(p.Execution.ChangedFiles))

	sb.WriteString("verification:\n")
	fmt.Fprintf(&sb, "  tests: %d\n", len(p.Verification.Tests))
	if p.Verification.Coverage != nil {
		fmt.Fprintf(&sb, "  coverage: %.0f%%\n", p.Verification.Coverage.Value)
	}

	fmt.Fprintf(&sb, "boundary_compliance: ok=%v reason=%s\n", p.Boundary.Compliance.OK, p.Boundary.Compliance.Reason)

	sb.WriteString("provenance:\n")
	fmt.Fprintf(&sb, "  run_id: %s\n", p.Provenance.RunID)
	fmt.Fprintf(&sb, "  orchestrator: %s\n", p.Provenance.Orchestrator)
	if p.Provenance.PromptHash != "" {
		fmt.Fprintf(&sb, "  prompt_hash: %s\n", p.Provenance.PromptHash)
	}
	if len(p.Provenance.ContextSources) > 0 {
		fmt.Fprintf(&sb, "  context_sources: %d items\n", len(p.Provenance.ContextSources))
	}

	sb.WriteString("trace:\n")
	fmt.Fprintf(&sb, "  branch: %s\n", p.Trace.Branch)
	fmt.Fprintf(&sb, "  commits: %d\n", len(p.Trace.Commits))
	if p.Trace.PRURL != "" {
		fmt.Fprintf(&sb, "  pr_url: %s\n", p.Trace.PRURL)
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

func formatLegacySummary(p map[string]any) string {
	var sb strings.Builder

	sb.WriteString("format: legacy evidence envelope\n")

	if intent, ok := p["intent"].(map[string]any); ok {
		sb.WriteString("intent:\n")
		if id, _ := intent["issue_id"].(string); id != "" {
			fmt.Fprintf(&sb, "  issue_id: %s\n", id)
		}
		if rc, _ := intent["risk_class"].(string); rc != "" {
			fmt.Fprintf(&sb, "  risk_class: %s\n", rc)
		}
		if acc, ok := intent["acceptance"].([]any); ok && len(acc) > 0 {
			fmt.Fprintf(&sb, "  acceptance: %d items\n", len(acc))
		}
	}

	if plan, ok := p["plan"].(map[string]any); ok {
		sb.WriteString("plan:\n")
		if ws, ok := plan["workstreams"].([]any); ok {
			fmt.Fprintf(&sb, "  workstreams: %v\n", ws)
		}
	}

	if exec, ok := p["execution"].(map[string]any); ok {
		sb.WriteString("execution:\n")
		if branch, _ := exec["branch"].(string); branch != "" {
			fmt.Fprintf(&sb, "  branch: %s\n", branch)
		}
		if cf, ok := exec["changed_files"].([]any); ok {
			fmt.Fprintf(&sb, "  changed_files: %d\n", len(cf))
		}
	}

	if ver, ok := p["verification"].(map[string]any); ok {
		sb.WriteString("verification:\n")
		if cov, ok := ver["coverage"].(map[string]any); ok {
			if v, ok := cov["value"].(float64); ok {
				fmt.Fprintf(&sb, "  coverage: %.0f%%\n", v)
			}
		}
		if tests, ok := ver["tests"].([]any); ok {
			fmt.Fprintf(&sb, "  tests: %d\n", len(tests))
		}
	}

	if bnd, ok := p["boundary"].(map[string]any); ok {
		if comp, ok := bnd["compliance"].(map[string]any); ok {
			okVal, _ := comp["ok"].(bool)
			reason, _ := comp["reason"].(string)
			fmt.Fprintf(&sb, "boundary_compliance: ok=%v reason=%s\n", okVal, reason)
		}
	}

	if prov, ok := p["provenance"].(map[string]any); ok {
		sb.WriteString("provenance:\n")
		if runID, _ := prov["run_id"].(string); runID != "" {
			fmt.Fprintf(&sb, "  run_id: %s\n", runID)
		}
		if orch, _ := prov["orchestrator"].(string); orch != "" {
			fmt.Fprintf(&sb, "  orchestrator: %s\n", orch)
		}
		if promptHash, _ := prov["prompt_hash"].(string); promptHash != "" {
			fmt.Fprintf(&sb, "  prompt_hash: %s\n", promptHash)
		}
		if sources, ok := prov["context_sources"].([]any); ok && len(sources) > 0 {
			fmt.Fprintf(&sb, "  context_sources: %d items\n", len(sources))
			for i, s := range sources {
				if i >= 3 {
					fmt.Fprintf(&sb, "    ... and %d more\n", len(sources)-3)
					break
				}
				if src, ok := s.(map[string]any); ok {
					t, _ := src["type"].(string)
					path, _ := src["path"].(string)
					fmt.Fprintf(&sb, "    - %s: %s\n", t, path)
				}
			}
		}
	}

	return strings.TrimSuffix(sb.String(), "\n")
}
