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

type Trace struct {
	ID              string
	SourceEntityID  string
	TargetEntityID  string
	Relation        TraceRelation
	Confidence      float64
	Justification   string
	Direction       TraceDirection
	CreatedAt       string
}
