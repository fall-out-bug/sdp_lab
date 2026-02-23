package evidence

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Inspect reads an evidence file, validates it, and returns a human-readable summary.
// If invalid, returns error and empty string.
func Inspect(path string, requirePRURL bool) (string, Result, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", Result{}, err
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return "", Result{}, err
	}
	res, err := ValidateStrictFile(path, requirePRURL)
	if err != nil {
		return "", res, err
	}
	if !res.OK {
		return "", res, nil
	}
	return formatSummary(payload), res, nil
}

func formatSummary(p map[string]any) string {
	var sb strings.Builder

	// intent
	if intent, ok := p["intent"].(map[string]any); ok {
		sb.WriteString("intent:\n")
		if id, _ := intent["issue_id"].(string); id != "" {
			sb.WriteString(fmt.Sprintf("  issue_id: %s\n", id))
		}
		if rc, _ := intent["risk_class"].(string); rc != "" {
			sb.WriteString(fmt.Sprintf("  risk_class: %s\n", rc))
		}
		if acc, ok := intent["acceptance"].([]any); ok && len(acc) > 0 {
			sb.WriteString(fmt.Sprintf("  acceptance: %d items\n", len(acc)))
		}
	}

	// plan
	if plan, ok := p["plan"].(map[string]any); ok {
		sb.WriteString("plan:\n")
		if ws, ok := plan["workstreams"].([]any); ok {
			sb.WriteString(fmt.Sprintf("  workstreams: %v\n", ws))
		}
		if r, _ := plan["ordering_rationale"].(string); r != "" {
			sb.WriteString(fmt.Sprintf("  ordering_rationale: %s\n", r))
		}
	}

	// execution
	if exec, ok := p["execution"].(map[string]any); ok {
		sb.WriteString("execution:\n")
		if branch, _ := exec["branch"].(string); branch != "" {
			sb.WriteString(fmt.Sprintf("  branch: %s\n", branch))
		}
		if cf, ok := exec["changed_files"].([]any); ok && len(cf) > 0 {
			sb.WriteString(fmt.Sprintf("  changed_files: %d\n", len(cf)))
			for i, f := range cf {
				if i >= 5 {
					sb.WriteString(fmt.Sprintf("    ... and %d more\n", len(cf)-5))
					break
				}
				sb.WriteString(fmt.Sprintf("    - %v\n", f))
			}
		}
	}

	// verification
	if ver, ok := p["verification"].(map[string]any); ok {
		sb.WriteString("verification:\n")
		if cov, ok := ver["coverage"].(map[string]any); ok {
			if v, ok := cov["value"].(float64); ok {
				sb.WriteString(fmt.Sprintf("  coverage: %.0f%%\n", v))
			}
		}
		if tests, ok := ver["tests"].([]any); ok && len(tests) > 0 {
			sb.WriteString(fmt.Sprintf("  tests: %d\n", len(tests)))
		}
	}

	// review
	if rev, ok := p["review"].(map[string]any); ok {
		sb.WriteString("review:\n")
		if sr, ok := rev["self_review"].([]any); ok && len(sr) > 0 {
			sb.WriteString(fmt.Sprintf("  self_review: %d items\n", len(sr)))
		}
		if ar, ok := rev["adversarial_review"].([]any); ok && len(ar) > 0 {
			sb.WriteString(fmt.Sprintf("  adversarial_review: %d items\n", len(ar)))
		}
	}

	// boundary compliance
	if bnd, ok := p["boundary"].(map[string]any); ok {
		if comp, ok := bnd["compliance"].(map[string]any); ok {
			okVal, _ := comp["ok"].(bool)
			reason, _ := comp["reason"].(string)
			sb.WriteString(fmt.Sprintf("boundary_compliance: ok=%v", okVal))
			if reason != "" {
				sb.WriteString(fmt.Sprintf(" reason=%s", reason))
			}
			sb.WriteString("\n")
		}
	}

	// provenance chain
	if prov, ok := p["provenance"].(map[string]any); ok {
		sb.WriteString("provenance:\n")
		if runID, _ := prov["run_id"].(string); runID != "" {
			sb.WriteString(fmt.Sprintf("  run_id: %s\n", runID))
		}
		if orch, _ := prov["orchestrator"].(string); orch != "" {
			sb.WriteString(fmt.Sprintf("  orchestrator: %s\n", orch))
		}
		if h, _ := prov["hash"].(string); h != "" {
			sb.WriteString("  hash_chain: present\n")
		} else {
			sb.WriteString("  hash_chain: empty\n")
		}
		// prompt provenance (F026)
		if promptHash, _ := prov["prompt_hash"].(string); promptHash != "" {
			sb.WriteString(fmt.Sprintf("  prompt_hash: %s\n", promptHash))
		}
		if sources, ok := prov["context_sources"].([]any); ok && len(sources) > 0 {
			sb.WriteString(fmt.Sprintf("  context_sources: %d items\n", len(sources)))
			for i, s := range sources {
				if i >= 3 {
					sb.WriteString(fmt.Sprintf("    ... and %d more\n", len(sources)-3))
					break
				}
				if src, ok := s.(map[string]any); ok {
					t, _ := src["type"].(string)
					path, _ := src["path"].(string)
					sb.WriteString(fmt.Sprintf("    - %s: %s\n", t, path))
				}
			}
		}
	}

	return strings.TrimSuffix(sb.String(), "\n")
}
