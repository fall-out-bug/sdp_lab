package federation

import (
	"encoding/json"
	"testing"
)

func TestIntakeBatchPayload_JSONRoundtrip(t *testing.T) {
	p := IntakeBatchPayload{
		ProjectID: "proj1",
		Feature:   IntakeBatchItem{Title: "F", Description: "D"},
		Subtasks: []IntakeBatchItem{
			{Title: "S1", Description: "D1", Acceptance: "AC1"},
			{Title: "S2", Description: "D2"},
		},
		DepEdges: []IntakeDepEdge{{Blocked: 1, Blocker: 0}},
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var q IntakeBatchPayload
	if err := json.Unmarshal(b, &q); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if q.ProjectID != p.ProjectID || len(q.Subtasks) != 2 || len(q.DepEdges) != 1 {
		t.Errorf("roundtrip: got %+v", q)
	}
}
