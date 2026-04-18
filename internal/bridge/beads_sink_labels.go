package bridge

import "strings"

func (s *BeadsSink) buildLabels(source FindingSourceType, severity, category, featureID, wsID string, blocking bool, findingHash, payloadHash string) []string {
	labels := []string{"sdp-finding", sourceFindingLabel(source)}
	if severity != "" {
		labels = append(labels, normalizeValue(severity))
	}
	if blocking {
		labels = append(labels, "blocking")
	} else {
		labels = append(labels, "non-blocking")
	}

	if category != "" {
		labels = append(labels, normalizeValue(category))
	}

	if featureID != "" {
		labels = append(labels, featureID)
	}

	if wsID != "" {
		labels = append(labels, wsID)
	}

	if findingHash != "" {
		labels = append(labels, findingHashLabel(findingHash))
	}

	if payloadHash != "" {
		labels = append(labels, payloadHashLabel(payloadHash))
	}

	labels = append(labels, s.labels...)

	return uniqueLabels(labels)
}

func sourceFindingLabel(source FindingSourceType) string {
	switch source {
	case FindingSourceReview:
		return "review-finding"
	case FindingSourceDrift:
		return "drift-finding"
	case FindingSourceQA:
		return "qa-finding"
	case FindingSourceCI:
		return "ci-finding"
	case FindingSourceGitHub:
		return "github-finding"
	default:
		return "ci-finding"
	}
}

func uniqueLabels(labels []string) []string {
	seen := make(map[string]struct{}, len(labels))
	result := make([]string, 0, len(labels))
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		result = append(result, label)
	}
	return result
}
