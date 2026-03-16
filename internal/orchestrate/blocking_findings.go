package orchestrate

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type BlockingFinding struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Status string   `json:"status"`
	Labels []string `json:"labels,omitempty"`
}

func ListBlockingFindings(ctx context.Context, featureID string) ([]BlockingFinding, error) {
	args := []string{"list", "--all", "--json", "-n", "0", "-l", "sdp-finding", "-l", "blocking"}
	if strings.TrimSpace(featureID) != "" {
		args = append(args, "-l", featureID)
	}
	cmd := exec.CommandContext(ctx, "bd", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd list blocking findings: %w", err)
	}
	var findings []BlockingFinding
	if err := json.Unmarshal(out, &findings); err != nil {
		return nil, fmt.Errorf("parse blocking findings: %w", err)
	}
	open := findings[:0]
	for _, finding := range findings {
		if normalizeFindingStatus(finding.Status) == "closed" {
			continue
		}
		open = append(open, finding)
	}
	return open, nil
}

func HasBlockingFindings(ctx context.Context, featureID string) (bool, []BlockingFinding, error) {
	findings, err := ListBlockingFindings(ctx, featureID)
	if err != nil {
		return false, nil, err
	}
	return len(findings) > 0, findings, nil
}

func normalizeFindingStatus(status string) string {
	return strings.TrimSpace(strings.ToLower(status))
}
