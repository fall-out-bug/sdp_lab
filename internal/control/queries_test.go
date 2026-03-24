package control

import (
	"bytes"
	"log"
	"testing"
)

func TestMissingInfo_Struct(t *testing.T) {
	m := MissingInfo{
		ID:                  "F-001",
		Title:               "Test",
		MissingEvidence:     true,
		MissingDispatch:     false,
		MissingExecutorState: true,
	}

	if !m.MissingEvidence {
		t.Error("expected missing evidence")
	}
	if m.MissingDispatch {
		t.Error("expected dispatch present")
	}
}

func TestPrintMissing_Empty(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	PrintMissing(nil, logger)

	if buf.String() == "" {
		t.Error("expected output for nil")
	}
}

func TestPrintMissing_WithItems(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	missing := []MissingInfo{
		{ID: "F-001", Title: "Test", MissingEvidence: true},
	}
	PrintMissing(missing, logger)

	output := buf.String()
	if output == "" {
		t.Error("expected output")
	}
}

func TestPrintWhyBlocked_Empty(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	PrintWhyBlocked(nil, logger)

	if buf.String() == "" {
		t.Error("expected output for nil")
	}
}

func TestBlockerInfo_Struct(t *testing.T) {
	b := BlockerInfo{ID: "bd-1", Title: "Blocker", Notes: "reason"}
	if b.ID != "bd-1" {
		t.Error("ID mismatch")
	}
}

func TestFeatureTrace_Struct(t *testing.T) {
	trace := FeatureTrace{
		Root:     "F-001",
		Children: []CardSummary{{ID: "F-001.1"}},
	}
	if trace.Root != "F-001" {
		t.Error("root mismatch")
	}
	if len(trace.Children) != 1 {
		t.Error("children count mismatch")
	}
}
