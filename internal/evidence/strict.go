package evidence

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var requiredSections = []string{"intent", "plan", "execution", "verification", "review", "risk_notes", "trace"}

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

	if requirePRURL {
		trace, _ := payload["trace"].(map[string]any)
		prURL, _ := trace["pr_url"].(string)
		if strings.TrimSpace(prURL) == "" {
			return Result{OK: false, Reason: "missing trace.pr_url"}, nil
		}
	}

	return Result{OK: true, Reason: "ok"}, nil
}

func FormatMissing(missing []string) string {
	if len(missing) == 0 {
		return ""
	}
	return fmt.Sprintf("missing: %s", strings.Join(missing, ", "))
}
