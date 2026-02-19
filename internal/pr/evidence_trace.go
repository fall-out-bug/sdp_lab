package pr

import (
	"encoding/json"
	"fmt"
	"os"
)

func WritePRURLToEvidence(path, prURL string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return err
	}
	trace, ok := payload["trace"].(map[string]any)
	if !ok {
		return fmt.Errorf("evidence trace section missing or invalid")
	}
	trace["pr_url"] = prURL

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}
