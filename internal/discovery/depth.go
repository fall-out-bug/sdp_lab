package discovery

type Disposition string

const (
	DispositionAdopt   Disposition = "ADOPT"
	DispositionExtract Disposition = "EXTRACT"
	DispositionInspire Disposition = "INSPIRE"
	DispositionMonitor Disposition = "MONITOR"
	DispositionIgnore  Disposition = "IGNORE"
)

// ScanItem represents one candidate from Phase 3 scan.
type ScanItem struct {
	Name                  string      `json:"name"`
	Disposition           Disposition `json:"disposition"`
	DispositionConfidence float64     `json:"disposition_confidence"`
	Stars                 int         `json:"stars"`
	SourceCount           int         `json:"source_count"`
	PrimarySourceRead     bool        `json:"primary_source_read"`
	ArchitectureReviewed  bool        `json:"architecture_reviewed"`
	DescSentences         int         `json:"desc_sentences"`
	MultiSource           bool        `json:"multi_source"`
	AgeMonths             int         `json:"age_months"` // 0 = unknown
	// populated after eval
	CoverageScore float64    `json:"coverage_score"`
	DepthFlag     *DepthFlag `json:"depth_flag,omitempty"`
	// output fields
	KeyStrength  string   `json:"key_strength"`
	KeyGap       string   `json:"key_gap"`
	CoversPhases []string `json:"covers_phases"`
}

type DepthFlag struct {
	Flagged           bool   `json:"flagged"`
	Reason            string `json:"reason"`
	RecommendedAction string `json:"recommended_action"` // deep_dive|proceed_provisional|downgrade
	Blocking          bool   `json:"blocking"`
}

// CoverageScore returns 0.0–1.0. Four equally weighted components.
func CoverageScore(item ScanItem) float64 {
	score := 0.0
	if item.PrimarySourceRead {
		score += 0.25
	}
	if item.ArchitectureReviewed {
		score += 0.25
	}
	if item.MultiSource {
		score += 0.25
	}
	// desc length: 0–20+ sentences → 0–0.25
	sentences := item.DescSentences
	if sentences > 20 {
		sentences = 20
	}
	score += float64(sentences) / 20.0 * 0.25
	return score
}

// EvalDepth applies the 7 heuristics and returns a DepthFlag.
func EvalDepth(item ScanItem) DepthFlag {
	cs := CoverageScore(item)

	// H3: universal stop — non-IGNORE verdict without primary source read
	if item.Disposition != DispositionIgnore && !item.PrimarySourceRead {
		return DepthFlag{
			Flagged:           true,
			Reason:            "no_primary_source",
			RecommendedAction: "deep_dive",
			Blocking:          item.Disposition == DispositionAdopt || item.Disposition == DispositionExtract,
		}
	}

	// H4: ADOPT/EXTRACT requires architecture review
	if (item.Disposition == DispositionAdopt || item.Disposition == DispositionExtract) &&
		!item.ArchitectureReviewed {
		return DepthFlag{
			Flagged:           true,
			Reason:            "architecture_not_reviewed",
			RecommendedAction: "deep_dive",
			Blocking:          true,
		}
	}

	// H7: low confidence on high-stakes verdict
	if (item.Disposition == DispositionAdopt || item.Disposition == DispositionExtract) &&
		item.DispositionConfidence > 0 && item.DispositionConfidence < 0.5 {
		return DepthFlag{
			Flagged:           true,
			Reason:            "low_confidence_high_stakes",
			RecommendedAction: "deep_dive",
		}
	}

	// H1: high stars, thin description
	if item.Stars > 5000 && item.DescSentences < 5 {
		return DepthFlag{
			Flagged:           true,
			Reason:            "high_stars_low_description",
			RecommendedAction: "deep_dive",
		}
	}

	// H2: primary source read but only one source cited — weak corroboration
	// (Note: the !PrimarySourceRead case is already caught by H3 above for non-IGNORE items)
	if item.PrimarySourceRead && item.SourceCount == 1 && item.DescSentences < 8 {
		return DepthFlag{
			Flagged:           true,
			Reason:            "single_source_thin_description",
			RecommendedAction: "proceed_provisional",
		}
	}

	// H5: recently released, sparse data
	if item.AgeMonths > 0 && item.AgeMonths <= 6 && item.DescSentences < 10 {
		return DepthFlag{
			Flagged:           true,
			Reason:            "recent_sparse",
			RecommendedAction: "deep_dive",
		}
	}

	// H6: low coverage on non-trivial verdict
	if cs < 0.4 && item.Disposition != DispositionIgnore && item.Disposition != DispositionMonitor {
		return DepthFlag{
			Flagged:           true,
			Reason:            "low_coverage_score",
			RecommendedAction: "proceed_provisional",
		}
	}

	return DepthFlag{Flagged: false}
}
