package observability

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestBuildIntakeRecordsValidatorCompatible(t *testing.T) {
	input := IntakeEventInput{
		RunID:               "run-emit-1",
		IssueID:             "sdp_dev-2aq.20.2",
		Phase:               "execute",
		Status:              "success",
		Component:           "swarm-worker",
		AgentRole:           "worker",
		ModelName:           "glm-4.7",
		Elapsed:             175 * time.Millisecond,
		RetryCount:          1,
		FallbackUsed:        true,
		Escalated:           false,
		EvidenceContextLink: ".sdp/evidence/sdp_dev-2aq.20.2.json",
		PRURL:               "https://example.invalid/org/repo/pull/99",
	}

	records := BuildIntakeRecords(input)
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	event, ok := records[0]["event"].(map[string]any)
	if !ok {
		t.Fatalf("expected event payload in first record, got %#v", records[0])
	}

	errList := ValidateUnifiedMetricsTraceEvent(event)
	if len(errList) != 0 {
		t.Fatalf("expected validator-compatible event, got errors: %v", errList)
	}
}

func TestEmitIntakeRecordsWritesJSONLines(t *testing.T) {
	input := IntakeEventInput{
		RunID:               "run-emit-2",
		IssueID:             "sdp_dev-2aq.20.2",
		Phase:               "review",
		Status:              "running",
		Component:           "swarm-reviewer",
		AgentRole:           "reviewer",
		ModelName:           "glm-5",
		EvidenceContextLink: ".sdp/evidence/sdp_dev-2aq.20.2.json",
		PRURL:               "unknown",
	}

	var buf bytes.Buffer
	if err := EmitIntakeRecords(&buf, input); err != nil {
		t.Fatalf("emit records: %v", err)
	}

	dec := json.NewDecoder(&buf)
	count := 0
	for dec.More() {
		var rec map[string]any
		if err := dec.Decode(&rec); err != nil {
			t.Fatalf("decode record: %v", err)
		}
		count++
	}
	if count != 2 {
		t.Fatalf("expected 2 emitted records, got %d", count)
	}
}
