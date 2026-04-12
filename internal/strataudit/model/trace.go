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
	TraceVerificationModeLLMEvidence TraceVerificationMode = "llm_evidence"
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
