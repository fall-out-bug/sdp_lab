package evals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sdp_dev/internal/kernel"
)

func TestRunCaseTraceAssertionsPass(t *testing.T) {
	root := t.TempDir()
	tracePath := filepath.Join(root, "trace.json")
	if err := os.WriteFile(tracePath, []byte(`{
  "events": [
    {"kind":"tool","payload":{"decision":"ask","tool":"docker"}},
    {"kind":"artifact","payload":{"type":"evidence"}}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := &Case{
		Name:                  "trace-pass",
		InputTrace:            "trace.json",
		ExpectedTraceKinds:    []kernel.TraceEventKind{kernel.TraceEventTool, kernel.TraceEventArtifact},
		ExpectedToolDecisions: []kernel.ToolPolicyDecision{kernel.ToolPolicyAsk},
		ExpectedArtifacts:     []kernel.ArtifactType{kernel.ArtifactEvidence},
	}

	result := RunCase(c, root)
	if !result.Pass {
		t.Fatalf("expected pass, got fail: %s", result.Reason)
	}
}

func TestRunCaseTraceAssertionsFail(t *testing.T) {
	root := t.TempDir()
	tracePath := filepath.Join(root, "trace.jsonl")
	if err := os.WriteFile(tracePath, []byte(`{"kind":"tool","payload":{"decision":"allow","tool":"docker"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := &Case{
		Name:                  "trace-fail",
		InputTrace:            "trace.jsonl",
		ExpectedTraceKinds:    []kernel.TraceEventKind{kernel.TraceEventTool, kernel.TraceEventArtifact},
		ExpectedToolDecisions: []kernel.ToolPolicyDecision{kernel.ToolPolicyAsk},
	}

	result := RunCase(c, root)
	if result.Pass {
		t.Fatal("expected fail")
	}
	if !strings.Contains(result.Reason, "missing trace kinds") || !strings.Contains(result.Reason, "missing tool decisions") {
		t.Fatalf("unexpected reason: %s", result.Reason)
	}
}

func TestRunCaseLegacyFallback(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "good.jsonl"), []byte(`{"role":"assistant","content":"CI GREEN - sdp ci-loop"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := &Case{
		Name:              "legacy",
		InputTranscript:   "good.jsonl",
		RequiredPatterns:  []string{"CI GREEN"},
		ForbiddenPatterns: []string{"Next steps"},
		Verdict:           "PASS",
	}

	result := RunCase(c, root)
	if !result.Pass {
		t.Fatalf("expected legacy pass, got fail: %s", result.Reason)
	}
}
