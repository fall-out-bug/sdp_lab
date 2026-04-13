package model

type TraceRelation string

const (
	RelationDecomposesInto TraceRelation = "decomposes_into"
	RelationContributesTo  TraceRelation = "contributes_to"
	RelationMeasures       TraceRelation = "measures"
	RelationEnables        TraceRelation = "enables"
	RelationConflictsWith  TraceRelation = "conflicts_with"
	RelationDuplicates     TraceRelation = "duplicates"
	RelationDependsOn      TraceRelation = "depends_on"
	RelationNone           TraceRelation = "none"
)

type TraceDirection string

const (
	DirectionUp            TraceDirection = "up"
	DirectionDown          TraceDirection = "down"
	DirectionBidirectional TraceDirection = "bidirectional"
)

type TraceVerificationMode string

const (
	TraceVerificationModeLLMEvidence     TraceVerificationMode = "llm_evidence"
	TraceVerificationModeCandidateSearch TraceVerificationMode = "candidate_search"
)

type TraceGapStage string

const (
	TraceGapStageCandidateSearch TraceGapStage = "candidate_search"
	TraceGapStageVerification    TraceGapStage = "verification"
	TraceGapStageAdmission       TraceGapStage = "admission"
	TraceGapStageUpstreamMissing TraceGapStage = "upstream_missing"
)

type TraceGapType string

const (
	TraceGapTypeNoCandidates                TraceGapType = "no_candidates"
	TraceGapTypeAllCandidatesRejected       TraceGapType = "all_candidates_rejected"
	TraceGapTypeLowConfidence               TraceGapType = "low_confidence"
	TraceGapTypeMissingUpstreamEntities     TraceGapType = "missing_upstream_entities"
	TraceGapTypeQuoteEvidenceMissing        TraceGapType = "quote_evidence_missing"
	TraceGapTypeVerificationUnavailable     TraceGapType = "verification_unavailable"
	TraceGapTypeVerificationBudgetExhausted TraceGapType = "verification_budget_exhausted"
)

type TraceCandidateDiagnostic string

const (
	TraceCandidateDiagnosticEmbeddingSimilarityCandidate TraceCandidateDiagnostic = "embedding_similarity_candidate"
	TraceCandidateDiagnosticLLMVerified                  TraceCandidateDiagnostic = "llm_verified"
	TraceCandidateDiagnosticLLMVerificationRejected      TraceCandidateDiagnostic = "llm_verification_rejected"
	TraceCandidateDiagnosticQuoteEvidenceMissing         TraceCandidateDiagnostic = "quote_evidence_missing"
	TraceCandidateDiagnosticBelowTraceConfidence         TraceCandidateDiagnostic = "below_trace_confidence"
	TraceCandidateDiagnosticVerificationUnavailable      TraceCandidateDiagnostic = "verification_unavailable"
	TraceCandidateDiagnosticVerificationBudgetExhausted  TraceCandidateDiagnostic = "verification_budget_exhausted"
)

type TraceEdgeStatus string

const (
	TraceEdgeStatusVerified  TraceEdgeStatus = "verified"
	TraceEdgeStatusCandidate TraceEdgeStatus = "candidate"
	TraceEdgeStatusRejected  TraceEdgeStatus = "rejected"
)

type Trace struct {
	ID                     string
	SourceEntityID         string
	TargetEntityID         string
	Relation               TraceRelation
	Confidence             float64
	SimilarityScore        float64
	Justification          string
	Direction              TraceDirection
	VerificationMode       TraceVerificationMode
	TrustGrade             TrustGrade
	SourceSectionID        string
	TargetSectionID        string
	SourceQuoteStartOffset *int
	SourceQuoteEndOffset   *int
	TargetQuoteStartOffset *int
	TargetQuoteEndOffset   *int
	CreatedAt              string
}
