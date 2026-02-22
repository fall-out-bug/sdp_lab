package main

import (
	"fmt"
	"strings"

	"sdp_dev/internal/policy"
)

func modelFromLabels(labels []string) (string, error) {
	for _, label := range labels {
		if strings.HasPrefix(label, "model:") {
			m := strings.TrimPrefix(label, "model:")
			if !policy.AllowedModel(m) {
				return "", fmt.Errorf("model '%s' is not allowed", m)
			}
			return m, nil
		}
	}
	return policy.DefaultModel(), nil
}

func laneFromLabels(labels []string) string {
	for _, label := range labels {
		if strings.HasPrefix(label, "lane:") {
			v := strings.TrimPrefix(label, "lane:")
			if v == "commit" || v == "explore" {
				return v
			}
		}
	}
	return "commit"
}

func allowedPrefixesFromLabels(labels []string) []string {
	restricted := []string{"internal/policy/", "internal/evidence/", "cmd/", "docs/", "specs/", "scripts/"}
	for _, label := range labels {
		switch label {
		case "workstream:policy-slugify-trim", "workstream:model-chain-default-fallback", "workstream:policy-k8s-risk-high", "workstream:handoff-validation":
			return restricted
		case "workstream:generic", "workstream:self-improvement", "workstream:evaluator-recommendation":
			return []string{"internal/", "cmd/", "docs/", "specs/", "scripts/", "deploy/"}
		}
	}
	return []string{"internal/", "cmd/", "docs/", "specs/", "scripts/", "deploy/"}
}
