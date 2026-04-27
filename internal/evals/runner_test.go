package evals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/kernel"
)

func TestRunCaseTraceAssertionsPass(t *testing.T) {
	root := t.TempDir()
	tracePath := filepath.Join(root, "trace.json")
	toolEvt, err := NewToolDecisionTraceEvent(
		kernel.ToolCallRequest{Tool: "docker"},
		kernel.ToolCallDecision{Decision: kernel.ToolPolicyAsk},
	)
	if err != nil {
		t.Fatal(err)
	}
	artifactEvt := kernel.TraceEvent{
		Kind:    kernel.TraceEventArtifact,
		Payload: []byte(`{"type":"evidence"}`),
	}
	if err := WriteTraceFixture(tracePath, toolEvt, artifactEvt); err != nil {
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

func TestRunCaseRoutingProviderAssertions(t *testing.T) {
	root := t.TempDir()
	tracePath := filepath.Join(root, "routing.json")
	routingEvt, err := NewRoutingTraceEvent(
		kernel.RoutingDecision{
			SelectedProvider: "selfhosted",
			SelectedModel:    "llama3",
			DecisionReason:   "restricted data",
			EvaluatedAt:      time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		},
		kernel.RoutingInput{
			TaskClass:   kernel.TaskClassCode,
			Sensitivity: kernel.SensitivityRestricted,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteTraceFixture(tracePath, routingEvt); err != nil {
		t.Fatal(err)
	}

	c := &Case{
		Name:                     "routing-provider",
		InputTrace:               "routing.json",
		ExpectedTraceKinds:       []kernel.TraceEventKind{kernel.TraceEventRouting},
		ExpectedRoutingProviders: []kernel.ProviderID{"selfhosted"},
	}

	result := RunCase(c, root)
	if !result.Pass {
		t.Fatalf("expected routing pass, got fail: %s", result.Reason)
	}
}

func TestRunBehaviorCatalogCases(t *testing.T) {
	modRoot, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(modRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(modRoot)
		if parent == modRoot {
			t.Skip("no go.mod found")
		}
		modRoot = parent
	}

	results, err := Run(modRoot, filepath.Join(modRoot, "internal", "eval", "cases"), "behavior")
	if err != nil {
		t.Fatalf("run behavior catalog: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 behavior results, got %d", len(results))
	}
	for _, result := range results {
		if !result.Pass {
			t.Fatalf("expected behavior case %s to pass, got: %s", result.Case, result.Reason)
		}
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
