package evidence

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"sdp_dev/internal/artifact"
)

var requiredSections = []string{"intent", "plan", "execution", "verification", "review", "risk_notes", "boundary", "provenance", "trace"}

type Result struct {
	OK      bool     `json:"ok"`
	Missing []string `json:"missing"`
	Reason  string   `json:"reason"`
}

func ValidateStrictFile(path string, requirePRURL bool) (Result, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return Result{}, err
	}

	missing := make([]string, 0)
	for _, key := range requiredSections {
		if _, ok := payload[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return Result{OK: false, Missing: missing, Reason: "missing strict evidence sections"}, nil
	}

	if !hasBoundaryContract(payload["boundary"]) {
		return Result{OK: false, Reason: "invalid boundary contract"}, nil
	}
	if !hasProvenanceContract(payload["provenance"]) {
		return Result{OK: false, Reason: "invalid provenance contract"}, nil
	}

	if requirePRURL {
		trace, _ := payload["trace"].(map[string]any)
		prURL, _ := trace["pr_url"].(string)
		if strings.TrimSpace(prURL) == "" {
			return Result{OK: false, Reason: "missing trace.pr_url"}, nil
		}
	}

	return Result{OK: true, Reason: "ok"}, nil
}

func hasBoundaryContract(v any) bool {
	b, ok := v.(map[string]any)
	if !ok {
		return false
	}
	declared, ok := b["declared"].(map[string]any)
	if !ok {
		return false
	}
	observed, ok := b["observed"].(map[string]any)
	if !ok {
		return false
	}
	compliance, ok := b["compliance"].(map[string]any)
	if !ok {
		return false
	}
	if _, ok := declared["allowed_path_prefixes"]; !ok {
		return false
	}
	if _, ok := declared["control_path_prefixes"]; !ok {
		return false
	}
	if _, ok := declared["forbidden_path_prefixes"]; !ok {
		return false
	}
	if _, ok := observed["touched_paths"]; !ok {
		return false
	}
	if _, ok := observed["out_of_boundary_paths"]; !ok {
		return false
	}
	if _, ok := compliance["ok"].(bool); !ok {
		return false
	}
	if _, ok := compliance["reason"].(string); !ok {
		return false
	}
	return true
}

func hasProvenanceContract(v any) bool {
	p, ok := v.(map[string]any)
	if !ok {
		return false
	}
	for _, key := range []string{"run_id", "orchestrator", "runtime", "model", "phase", "role", "captured_at", "source_issue_id", "artifact_id", "contract_version", "hash_algorithm", "payload_digest", "hash", "hash_prev"} {
		if _, ok := p[key].(string); !ok {
			return false
		}
	}
	sequence, ok := p["sequence"].(float64)
	if !ok || sequence < 0 {
		return false
	}
	hash, _ := p["hash"].(string)
	if strings.TrimSpace(hash) != "" && !artifact.IsSHA256Hex(hash) {
		return false
	}
	hashPrev, _ := p["hash_prev"].(string)
	if strings.TrimSpace(hashPrev) != "" && !artifact.IsSHA256Hex(hashPrev) {
		return false
	}
	payloadDigest, _ := p["payload_digest"].(string)
	if strings.TrimSpace(payloadDigest) != "" && !artifact.IsSHA256Hex(payloadDigest) {
		return false
	}
	if _, ok := p["gate_results"]; !ok {
		return false
	}
	// Optional prompt provenance (F026): when present, must be valid
	if promptHash, ok := p["prompt_hash"].(string); ok && strings.TrimSpace(promptHash) != "" {
		if !artifact.IsSHA256Hex(promptHash) {
			return false
		}
	}
	if sources, ok := p["context_sources"].([]any); ok && len(sources) > 0 {
		for _, s := range sources {
			src, ok := s.(map[string]any)
			if !ok {
				return false
			}
			t, _ := src["type"].(string)
			path, _ := src["path"].(string)
			h, _ := src["hash"].(string)
			if strings.TrimSpace(t) == "" || strings.TrimSpace(path) == "" || strings.TrimSpace(h) == "" {
				return false
			}
			if !artifact.IsSHA256Hex(h) {
				return false
			}
		}
	}
	return true
}

func FormatMissing(missing []string) string {
	if len(missing) == 0 {
		return ""
	}
	return fmt.Sprintf("missing: %s", strings.Join(missing, ", "))
}
