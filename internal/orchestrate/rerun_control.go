package orchestrate

import (
	"fmt"
	"strings"
)

func RedirectToBuildForBlockingFindings(cp *Checkpoint, phase string, findings []BlockingFinding) (string, []string, error) {
	if cp == nil {
		return "", nil, fmt.Errorf("checkpoint is required")
	}
	if len(findings) == 0 {
		return "", nil, fmt.Errorf("blocking findings are required")
	}

	targetWS := checkpointFindingWSID(cp)
	for _, finding := range findings {
		if wsid := findingWorkstreamID(finding); wsid != "" {
			targetWS = wsid
			break
		}
	}
	if targetWS == "" {
		return "", nil, fmt.Errorf("no workstream found for blocking findings")
	}

	findingIDs := make([]string, 0, len(findings))
	foundTarget := false
	for i := range cp.Workstreams {
		if cp.Workstreams[i].ID == targetWS {
			cp.Workstreams[i].Status = "pending"
			foundTarget = true
		}
	}
	if !foundTarget {
		return "", nil, fmt.Errorf("target workstream %s not found in checkpoint", targetWS)
	}

	cp.Phase = PhaseBuild
	if cp.Review != nil && (phase == PhaseReview || phase == PhaseQA) {
		cp.Review.Status = "pending"
	}
	if cp.QA != nil && phase == PhaseQA {
		cp.QA.Status = "pending"
	}

	for _, finding := range findings {
		findingIDs = append(findingIDs, finding.ID)
	}
	return targetWS, findingIDs, nil
}

func findingWorkstreamID(finding BlockingFinding) string {
	for _, label := range finding.Labels {
		label = strings.TrimSpace(label)
		if strings.HasPrefix(label, "00-") {
			return label
		}
	}
	return ""
}
