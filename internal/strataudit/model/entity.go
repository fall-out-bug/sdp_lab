package model

type EntityType string

type TrustGrade string

const (
	TrustGradeVerified TrustGrade = "verified"
	TrustGradeSuspect  TrustGrade = "suspect"
	TrustGradeRejected TrustGrade = "rejected"
)

const (
	EntityGoal        EntityType = "goal"
	EntityObjective   EntityType = "objective"
	EntityKPI         EntityType = "kpi"
	EntityInitiative  EntityType = "initiative"
	EntityTask        EntityType = "task"
	EntityPrinciple   EntityType = "principle"
	EntityStakeholder EntityType = "stakeholder"
	EntityCapability  EntityType = "capability"
)

func ValidEntityTypes() []EntityType {
	return []EntityType{
		EntityGoal, EntityObjective, EntityKPI, EntityInitiative,
		EntityTask, EntityPrinciple, EntityStakeholder, EntityCapability,
	}
}

func IsValidEntityType(t EntityType) bool {
	for _, v := range ValidEntityTypes() {
		if v == t {
			return true
		}
	}
	return false
}

type Entity struct {
	ID              string
	DocumentID      string
	LevelID         string
	Type            EntityType
	Title           string
	Description     string
	SourceQuote     string
	TrustGrade      TrustGrade
	QualityFlags    []string
	PageNumber      int
	Embedding       []float32
	EmbeddingModel  string
	EmbeddingDims   int
	ExtractionModel string
	Metadata        map[string]string
	CreatedAt       string
}
