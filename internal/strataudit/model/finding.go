package model

type FindingType string

const (
	FindingAlignment        FindingType = "alignment"
	FindingStrongTrace      FindingType = "strong_trace"
	FindingCoverage         FindingType = "coverage"
	FindingGap              FindingType = "gap"
	FindingOrphan           FindingType = "orphan"
	FindingUnknownRationale FindingType = "unknown_rationale"
	FindingAmbiguousTrace   FindingType = "ambiguous_trace"
	FindingConflict         FindingType = "conflict"
	FindingWeakLink         FindingType = "weak_link"
	FindingStale            FindingType = "stale"
	FindingInferredStrategy FindingType = "inferred_strategy"
	FindingShadowStrategy   FindingType = "shadow_strategy"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarn     Severity = "warn"
	SeverityCritical Severity = "critical"
)

type LLMScore string

const (
	LLMScoreHigh    LLMScore = "HIGH"
	LLMScoreMedium  LLMScore = "MEDIUM"
	LLMScoreLow     LLMScore = "LOW"
	LLMScoreAbstain LLMScore = "ABSTAIN"
)

type CrossModelStatus string

const (
	CrossModelConfirmed CrossModelStatus = "confirmed"
	CrossModelDisputed  CrossModelStatus = "disputed"
	CrossModelRefuted   CrossModelStatus = "refuted"
)

type ConfidenceTier string

const (
	TierHigh   ConfidenceTier = "high"
	TierMedium ConfidenceTier = "medium"
	TierLow    ConfidenceTier = "low"
)

type Finding struct {
	ID                 string
	Type               FindingType
	Severity           Severity
	EntityIDs          []string
	Title              string
	Description        string
	Recommendation     string
	Suppressed         bool
	LLMScore           LLMScore
	EvidenceQuotes     []string
	EvidenceVerified   bool
	EvidenceCount      int
	SupportRatio       float64
	CrossModelStatus   CrossModelStatus
	VerificationPassed bool
	ConfidenceScore    float64
	Ephemeral          bool
	CreatedAt          string
}

func (f *Finding) Tier() ConfidenceTier {
	switch {
	case f.ConfidenceScore >= 0.7:
		return TierHigh
	case f.ConfidenceScore >= 0.4:
		return TierMedium
	default:
		return TierLow
	}
}

// ComputeConfidence calculates composite confidence from 4 factors.
// Structural findings (gap, orphan, stale, coverage) get 1.0 automatically.
func (f *Finding) ComputeConfidence() float64 {
	switch f.Type {
	case FindingGap, FindingOrphan, FindingStale, FindingCoverage, FindingAmbiguousTrace:
		f.ConfidenceScore = 1.0
		return 1.0
	}

	score := 0.0

	// Factor 1: LLM self-assessment (0-30 points)
	switch f.LLMScore {
	case LLMScoreHigh:
		score += 30
	case LLMScoreMedium:
		score += 20
	case LLMScoreLow:
		score += 10
	case LLMScoreAbstain:
		f.ConfidenceScore = 0.0
		return 0.0
	}

	// Factor 2: Evidence grounding (0-30 points)
	if f.EvidenceVerified && f.EvidenceCount > 0 {
		bonus := float64(f.EvidenceCount) * 10
		if bonus > 30 {
			bonus = 30
		}
		score += bonus
	}

	// Factor 3: Support ratio (0-20 points)
	score += f.SupportRatio * 20

	// Factor 4: Adversarial verification (0-20 points)
	switch f.CrossModelStatus {
	case CrossModelConfirmed:
		score += 20
	case CrossModelDisputed:
		score += 5
	}

	result := score / 100

	// Apply confidence caps for high-risk types
	switch f.Type {
	case FindingUnknownRationale, FindingInferredStrategy, FindingShadowStrategy:
		if result > 0.7 {
			result = 0.7
		}
	case FindingConflict:
		if result > 0.9 {
			result = 0.9
		}
	}

	f.ConfidenceScore = result
	return result
}
