package selfimprove

// WeaknessPattern represents a detected weakness.
type WeaknessPattern struct {
	ID          string
	Description string
	IssueIDs    []string
	Count       int
	Class       FailureClass
}

// WeaknessDetector detects patterns (repeated failures, boundary violations).
type WeaknessDetector struct {
	classifier *FailureClassifier
}

// NewWeaknessDetector returns a new detector.
func NewWeaknessDetector() *WeaknessDetector {
	return &WeaknessDetector{classifier: NewFailureClassifier()}
}

// Detect analyzes run docs and telemetry to find weakness patterns.
func (d *WeaknessDetector) Detect(runs []RunDoc, telemetry []TelemetryRecord) []WeaknessPattern {
	var patterns []WeaknessPattern

	// Count failures by class
	byClass := make(map[FailureClass][]string)
	for _, doc := range runs {
		cf := d.classifier.ClassifyRun(doc)
		if cf != nil {
			byClass[cf.Class] = append(byClass[cf.Class], cf.IssueID)
		}
	}

	for class, ids := range byClass {
		if len(ids) >= 2 {
			patterns = append(patterns, WeaknessPattern{
				ID:          "repeated-" + string(class),
				Description: "Repeated " + string(class) + " failures across runs",
				IssueIDs:    ids,
				Count:       len(ids),
				Class:       class,
			})
		}
	}

	// Escalation pattern
	escalated := 0
	for _, rec := range telemetry {
		if rec.Escalated {
			escalated++
		}
	}
	if escalated >= 2 {
		patterns = append(patterns, WeaknessPattern{
			ID:          "escalation-spike",
			Description: "Multiple escalations in telemetry window",
			Count:       escalated,
			Class:       ClassPolicyConflict,
		})
	}

	return patterns
}

// SuggestImprovement returns a suggested improvement title for a pattern.
func SuggestImprovement(p WeaknessPattern) string {
	switch p.Class {
	case ClassTransient:
		return "Harden retry/backoff for transient failures"
	case ClassToolFlake:
		return "Add infra-flaky detection and circuit breaker"
	case ClassVerificationFail:
		return "Improve verification gate diagnostics"
	case ClassPolicyConflict:
		return "Review policy allowlist and escalation path"
	case ClassSecuritySensitive:
		return "Document security-sensitive escalation flow"
	default:
		return "Investigate " + p.ID + " pattern"
	}
}
