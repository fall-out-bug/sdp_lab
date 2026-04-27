package decompose

// Status mirrors confidence.Status: gating verdict on a pipeline or stage result.
type Status string

const (
	StatusOK     Status = "ok"
	StatusUnsure Status = "unsure"
	StatusFail   Status = "fail"
)

// aggregateStatus reduces a slice of stage statuses to a single pipeline status:
// any FAIL → FAIL; any UNSURE without FAIL → UNSURE; all OK → OK.
func aggregateStatus(statuses []Status) Status {
	result := StatusOK
	for _, s := range statuses {
		switch s {
		case StatusFail:
			return StatusFail
		case StatusUnsure:
			result = StatusUnsure
		}
	}
	return result
}
