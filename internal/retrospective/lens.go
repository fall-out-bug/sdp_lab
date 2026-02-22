package retrospective

// LensID identifies an analysis lens.
type LensID string

const (
	LensProtocol    LensID = "protocol"
	LensInfra       LensID = "infra"
	LensCodeQuality LensID = "code-quality"
	LensOperator    LensID = "operator"
	LensDX          LensID = "dx"
)

// Lens defines analysis focus.
type Lens struct {
	ID          LensID
	Description string
	Focus       []string
}

// DefaultLenses returns the 5 analysis lenses.
func DefaultLenses() []Lens {
	return []Lens{
		{ID: LensProtocol, Description: "FSM transition issues, gate gaps, evidence schema", Focus: []string{"protocol", "fsm", "evidence"}},
		{ID: LensInfra, Description: "Timeouts, retries, flakiness, resource limits", Focus: []string{"infra", "timeout", "retry"}},
		{ID: LensCodeQuality, Description: "Boundary violations, test failures, complexity", Focus: []string{"code", "boundary", "test"}},
		{ID: LensOperator, Description: "Scheduling bottlenecks, lock contention, utilization", Focus: []string{"operator", "scheduling"}},
		{ID: LensDX, Description: "Error messages, recovery rates, runbook gaps", Focus: []string{"dx", "error", "runbook"}},
	}
}
