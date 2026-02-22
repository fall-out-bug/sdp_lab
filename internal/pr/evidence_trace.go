package pr

import (
	"encoding/json"
	"fmt"
	"os"
)

func WritePRURLToEvidence(path, prURL string) error {
	return WritePublishTraceToEvidence(path, prURL, "", "")
}

func WritePublishTraceToEvidence(path, prURL, runContextLink, evidenceContextLink string) error {
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
	if runContextLink != "" {
		trace["run_context_link"] = runContextLink
	}
	if evidenceContextLink != "" {
		trace["evidence_context_link"] = evidenceContextLink
	}

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}

func ReadRunIDFromEvidence(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return "", err
	}
	provenance, ok := payload["provenance"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("evidence provenance section missing or invalid")
	}
	runID, _ := provenance["run_id"].(string)
	if runID == "" {
		return "", fmt.Errorf("evidence run_id missing")
	}
	return runID, nil
}
