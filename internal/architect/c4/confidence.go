package c4

import (
	"github.com/fall-out-bug/sdp_lab/internal/architect"
)

// confidence thresholds per the C4 spec Section 6.
const (
	confidenceHigh   = 0.80
	confidenceMedium = 0.60
)

// scoreModel computes confidence scores for all model elements.
func scoreModel(model *architect.ReferenceModel, profile *architect.CodebaseProfile) {
	for i := range model.Containers {
		c := &model.Containers[i]
		cScore := containerConfidence(c)
		c.Confidence = cScore

		// Score components.
		for j := range c.Components {
			comp := &c.Components[j]
			if comp.Confidence == 0 {
				comp.Confidence = 0.50
			}
		}
	}
}

// containerConfidence returns a confidence score for a container based on how
// it was detected.
func containerConfidence(c *architect.C4Container) float64 {
	switch {
	case c.Deploy == "docker" || c.Deploy == "docker-compose":
		return 0.95
	case c.Deploy == "kubernetes":
		return 0.90
	case c.Source == "go":
		// Go cmd/ directories
		return 0.85
	case c.Deploy == "npm":
		return 0.85
	case c.Deploy == "maven":
		return 0.80
	case c.Deploy == "gradle":
		return 0.80
	case c.Deploy == "inferred":
		return 0.50
	default:
		return 0.60
	}
}

// confidenceMarker returns the prefix marker for a given confidence level.
// Per spec Section 6.4:
//   - >= 0.80: no marker (high confidence)
//   - 0.60-0.80: "[AUTO?]" (medium confidence)
//   - < 0.60: "[AUTO]" (low confidence)
func confidenceMarker(confidence float64) string {
	if confidence >= confidenceHigh {
		return ""
	}
	if confidence >= confidenceMedium {
		return "[AUTO?] "
	}
	return "[AUTO] "
}

// modelConfidence computes an overall confidence score for the model.
func modelConfidence(model *architect.ReferenceModel) float64 {
	var totalNodes float64
	var nodeConfSum float64
	var totalEdges float64
	var edgeConfSum float64

	for _, c := range model.Containers {
		cc := containerConfidence(&c)
		totalNodes++
		nodeConfSum += cc

		for _, comp := range c.Components {
			totalNodes++
			nodeConfSum += comp.Confidence
		}
	}

	// Edges have implicit confidence from their sources.
	for range model.Relationships {
		totalEdges++
		edgeConfSum += 0.70 // default edge confidence
	}

	if totalNodes == 0 {
		return 0
	}

	nodeAvg := nodeConfSum / totalNodes
	edgeAvg := 0.70
	if totalEdges > 0 {
		edgeAvg = edgeConfSum / totalEdges
	}

	// Weighted: 60% nodes, 40% edges (per spec Section 6.3).
	if totalEdges == 0 {
		return nodeAvg
	}
	return 0.6*nodeAvg + 0.4*edgeAvg
}

// ReviewReport lists low-confidence elements needing human review.
type ReviewReport struct {
	ReviewRequired []ReviewItem `json:"review_required,omitempty"`
	Stats          ReviewStats  `json:"stats"`
}

// ReviewItem describes a single element that needs human review.
type ReviewItem struct {
	ElementID   string  `json:"element_id"`
	ElementType string  `json:"element_type"`
	Confidence  float64 `json:"confidence"`
	Reason      string  `json:"reason"`
	Suggestion  string  `json:"suggestion,omitempty"`
}

// ReviewStats provides confidence distribution stats.
type ReviewStats struct {
	TotalNodes       int     `json:"total_nodes"`
	HighConfidence   int     `json:"high_confidence"`
	MediumConfidence int     `json:"medium_confidence"`
	LowConfidence    int     `json:"low_confidence"`
	OverallConfidence float64 `json:"overall_confidence"`
}

// GenerateReviewReport creates a review report for low-confidence elements.
func GenerateReviewReport(model *architect.ReferenceModel) *ReviewReport {
	report := &ReviewReport{}
	report.Stats.OverallConfidence = modelConfidence(model)

	for _, c := range model.Containers {
		cc := containerConfidence(&c)
		report.Stats.TotalNodes++
		classifyConfidence(cc, &report.Stats)

		if cc < confidenceHigh {
			report.ReviewRequired = append(report.ReviewRequired, ReviewItem{
				ElementID:   c.ID,
				ElementType: "Container",
				Confidence:  cc,
				Reason:      containerReviewReason(c),
				Suggestion:  "Add a Dockerfile or docker-compose entry to confirm this deploy unit",
			})
		}

		for _, comp := range c.Components {
			report.Stats.TotalNodes++
			classifyConfidence(comp.Confidence, &report.Stats)

			if comp.Confidence < confidenceHigh {
				report.ReviewRequired = append(report.ReviewRequired, ReviewItem{
					ElementID:   comp.ID,
					ElementType: "Component",
					Confidence:  comp.Confidence,
					Reason:      componentReviewReason(comp),
					Suggestion:  "Verify component boundary; consider splitting or merging with adjacent components",
				})
			}
		}
	}

	return report
}

func classifyConfidence(confidence float64, stats *ReviewStats) {
	switch {
	case confidence >= confidenceHigh:
		stats.HighConfidence++
	case confidence >= confidenceMedium:
		stats.MediumConfidence++
	default:
		stats.LowConfidence++
	}
}

func containerReviewReason(c architect.C4Container) string {
	if c.Deploy == "inferred" {
		return "No explicit deploy configuration found; inferred from directory structure"
	}
	return "Container detected with moderate confidence"
}

func componentReviewReason(comp architect.C4Component) string {
	if comp.Confidence < confidenceMedium {
		return "Component boundary inferred from sparse import graph data"
	}
	return "Component boundary may need human verification"
}
