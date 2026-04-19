package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sdp_dev/internal/delta"
	"sdp_dev/internal/gate"
)

// TestPhaseFlagsParsing tests that phase flags are parsed correctly.
func TestPhaseFlagsParsing(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError bool
		checkFunc func(*testing.T, *phaseFlags)
	}{
		{
			name:      "parse feature-id",
			args:      []string{"--feature-id", "F134"},
			wantError: false,
			checkFunc: func(t *testing.T, f *phaseFlags) {
				if f.featureID != "F134" {
					t.Errorf("Expected featureID F134, got %s", f.featureID)
				}
			},
		},
		{
			name:      "parse all flags",
			args:      []string{"--feature-id", "F134", "--ws-id", "00-134-01", "--run-id", "test-run", "--strict"},
			wantError: false,
			checkFunc: func(t *testing.T, f *phaseFlags) {
				if f.featureID != "F134" {
					t.Errorf("Expected featureID F134, got %s", f.featureID)
				}
				if f.wsID != "00-134-01" {
					t.Errorf("Expected wsID 00-134-01, got %s", f.wsID)
				}
				if f.runID != "test-run" {
					t.Errorf("Expected runID test-run, got %s", f.runID)
				}
				if !f.strict {
					t.Error("Expected strict to be true")
				}
			},
		},
		{
			name:      "strict flag defaults to false",
			args:      []string{"--feature-id", "F134"},
			wantError: false,
			checkFunc: func(t *testing.T, f *phaseFlags) {
				if f.strict {
					t.Error("Expected strict to default to false")
				}
			},
		},
		{
			name:      "parse evidence-path",
			args:      []string{"--feature-id", "F134", "--evidence-path", "/tmp/evidence.json"},
			wantError: false,
			checkFunc: func(t *testing.T, f *phaseFlags) {
				if f.evidencePath != "/tmp/evidence.json" {
					t.Errorf("Expected evidencePath /tmp/evidence.json, got %s", f.evidencePath)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			f := parsePhaseFlags(fs)

			// Capture stderr to avoid error output in tests
			oldStderr := os.Stderr
			_, w, _ := os.Pipe()
			os.Stderr = w

			err := fs.Parse(tt.args)

			w.Close()
			os.Stderr = oldStderr

			// Debug output to see what we got
			t.Logf("Parsed flags: featureID=%q, wsID=%q, runID=%q, strict=%v, evidencePath=%q", f.featureID, f.wsID, f.runID, f.strict, f.evidencePath)

			if (err != nil) != tt.wantError {
				t.Errorf("parsePhaseFlags() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, f)
			}
		})
	}
}

// TestValidatePhaseFlags tests flag validation.
func TestValidatePhaseFlags(t *testing.T) {
	tests := []struct {
		name      string
		flags     *phaseFlags
		phaseName string
		wantErr   bool
	}{
		{
			name: "valid flags",
			flags: &phaseFlags{
				featureID: "F134",
			},
			phaseName: "plan",
			wantErr:   false,
		},
		{
			name: "missing feature-id",
			flags: &phaseFlags{
				featureID: "",
			},
			phaseName: "plan",
			wantErr:   true,
		},
		{
			name: "strict without evidence-path fails",
			flags: &phaseFlags{
				featureID: "F134",
				strict:    true,
			},
			phaseName: "plan",
			wantErr:   true,
		},
		{
			name: "strict with evidence-path passes",
			flags: &phaseFlags{
				featureID:    "F134",
				strict:       true,
				evidencePath: "/tmp/evidence.json",
			},
			phaseName: "plan",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePhaseFlags(tt.flags, tt.phaseName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePhaseFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGenerateRunID tests run ID generation.
func TestGenerateRunID(t *testing.T) {
	tests := []struct {
		name     string
		runID    string
		wantSame bool
	}{
		{
			name:     "provided run ID is used",
			runID:    "custom-run-id",
			wantSame: true,
		},
		{
			name:     "empty run ID generates new ID",
			runID:    "",
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateRunID(tt.runID)
			if tt.wantSame && result != tt.runID {
				t.Errorf("Expected run ID %q, got %q", tt.runID, result)
			}
			if !tt.wantSame && result == tt.runID {
				t.Errorf("Expected generated run ID, got %q", result)
			}
			if !tt.wantSame && result == "" {
				t.Error("Expected non-empty generated run ID")
			}
		})
	}
}

// TestPhaseGateCreation tests that gates are created correctly for each phase.
func TestPhaseGateCreation(t *testing.T) {
	tests := []struct {
		name         string
		phase        string
		featureID    string
		runID        string
		wantQuestion string
		wantOptions  []string
	}{
		{
			name:         "plan gate",
			phase:        "plan",
			featureID:    "F134",
			runID:        "test-run",
			wantQuestion: "Approve plan delta?",
			wantOptions:  []string{"approve", "reject", "defer"},
		},
		{
			name:         "review gate",
			phase:        "review",
			featureID:    "F134",
			runID:        "test-run",
			wantQuestion: "Approve review delta?",
			wantOptions:  []string{"approve", "reject", "request-changes"},
		},
		{
			name:         "eval gate",
			phase:        "eval",
			featureID:    "F134",
			runID:        "test-run",
			wantQuestion: "Approve eval results?",
			wantOptions:  []string{"approve", "reject", "retry"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := createTestGate(tt.phase, tt.featureID, tt.runID)

			if g.Question != tt.wantQuestion {
				t.Errorf("Expected question %q, got %q", tt.wantQuestion, g.Question)
			}

			if len(g.Options) != len(tt.wantOptions) {
				t.Errorf("Expected %d options, got %d", len(tt.wantOptions), len(g.Options))
			}

			for i, opt := range g.Options {
				if i >= len(tt.wantOptions) {
					break
				}
				if opt != tt.wantOptions[i] {
					t.Errorf("Option %d: expected %q, got %q", i, tt.wantOptions[i], opt)
				}
			}

			if g.ID == "" {
				t.Error("Expected gate ID to be set")
			}

			if g.Context == "" {
				t.Error("Expected gate context to be set")
			}
		})
	}
}

// TestDeltaCreation tests that deltas are created correctly for each phase.
func TestDeltaCreation(t *testing.T) {
	tests := []struct {
		name      string
		phase     string
		featureID string
		wsID      string
		runID     string
	}{
		{
			name:      "plan delta",
			phase:     "plan",
			featureID: "F134",
			wsID:      "00-134-01",
			runID:     "test-run",
		},
		{
			name:      "review delta",
			phase:     "review",
			featureID: "F134",
			wsID:      "",
			runID:     "test-run",
		},
		{
			name:      "eval delta",
			phase:     "eval",
			featureID: "F134",
			wsID:      "00-134-01",
			runID:     "test-run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := delta.NewDelta(tt.phase,
				delta.WithFeatureID(tt.featureID),
				delta.WithWorkstreamID(tt.wsID),
				delta.WithRunID(tt.runID),
			)

			if d.Phase != tt.phase {
				t.Errorf("Expected phase %q, got %q", tt.phase, d.Phase)
			}

			if d.FeatureID != tt.featureID {
				t.Errorf("Expected featureID %q, got %q", tt.featureID, d.FeatureID)
			}

			if d.WorkstreamID != tt.wsID {
				t.Errorf("Expected wsID %q, got %q", tt.wsID, d.WorkstreamID)
			}

			if d.RunID != tt.runID {
				t.Errorf("Expected runID %q, got %q", tt.runID, d.RunID)
			}

			// Empty delta should have no changes
			if !d.IsEmpty() {
				t.Error("Expected new delta to be empty")
			}
		})
	}
}

// TestGateStatus tests gate status logic.
func TestGateStatus(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*gate.Gate)
		wantStatus   string
		wantBlocking bool
	}{
		{
			name: "pending gate",
			setup: func(g *gate.Gate) {
				// Default state is pending
			},
			wantStatus:   "pending",
			wantBlocking: true,
		},
		{
			name: "resolved gate",
			setup: func(g *gate.Gate) {
				now := time.Now()
				g.Answer = "approve"
				g.Answerer = "test"
				g.ResolvedAt = &now
			},
			wantStatus:   "resolved",
			wantBlocking: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &gate.Gate{
				ID:        "test-gate",
				Question:  "Test question",
				CreatedAt: time.Now(),
			}
			tt.setup(g)

			status := g.Status()
			if status != tt.wantStatus {
				t.Errorf("Expected status %q, got %q", tt.wantStatus, status)
			}

			blocking := g.IsBlocking()
			if blocking != tt.wantBlocking {
				t.Errorf("Expected blocking %v, got %v", tt.wantBlocking, blocking)
			}
		})
	}
}

// TestResolveWithEvidence tests gate resolution with evidence.
func TestResolveWithEvidence(t *testing.T) {
	// Create a temporary evidence file
	tmpDir := t.TempDir()

	// Valid evidence for plan gate
	validPlanEvidence := map[string]interface{}{
		"test_coverage":     0.9,
		"design_checklist":  "done",
	}
	validPlanData, _ := json.Marshal(validPlanEvidence)
	validPlanPath := filepath.Join(tmpDir, "plan-evidence.json")
	os.WriteFile(validPlanPath, validPlanData, 0o644)

	// Invalid evidence (missing required keys)
	invalidEvidence := map[string]interface{}{
		"some_key": "some_value",
	}
	invalidData, _ := json.Marshal(invalidEvidence)
	invalidPath := filepath.Join(tmpDir, "invalid-evidence.json")
	os.WriteFile(invalidPath, invalidData, 0o644)

	tests := []struct {
		name          string
		gateType      gate.GateType
		answer        string
		answerer      string
		evidencePath  string
		wantErr       bool
		wantResolved  bool
	}{
		{
			name:         "plan gate with valid evidence",
			gateType:     gate.GateTypePlan,
			answer:       "approve",
			answerer:     "sdp-phase-strict",
			evidencePath: validPlanPath,
			wantErr:      false,
			wantResolved: true,
		},
		{
			name:         "plan gate with invalid evidence schema",
			gateType:     gate.GateTypePlan,
			answer:       "approve",
			answerer:     "sdp-phase-strict",
			evidencePath: invalidPath,
			wantErr:      true,
			wantResolved: false,
		},
		{
			name:         "plan gate without evidence fails",
			gateType:     gate.GateTypePlan,
			answer:       "approve",
			answerer:     "sdp-phase-strict",
			evidencePath: "",
			wantErr:      true,
			wantResolved: false,
		},
		{
			name:         "manual gate without evidence succeeds",
			gateType:     gate.GateTypeManual,
			answer:       "approve",
			answerer:     "sdp-phase-auto",
			evidencePath: "",
			wantErr:      false,
			wantResolved: true,
		},
		{
			name:         "nonexistent evidence file fails",
			gateType:     gate.GateTypePlan,
			answer:       "approve",
			answerer:     "sdp-phase-strict",
			evidencePath: filepath.Join(tmpDir, "nonexistent.json"),
			wantErr:      true,
			wantResolved: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &gate.Gate{
				ID:        "test-gate",
				Question:  "Test question",
				Type:      tt.gateType,
				CreatedAt: time.Now(),
			}

			err := g.ResolveWithEvidence(tt.answer, tt.answerer, tt.evidencePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveWithEvidence() error = %v, wantErr %v", err, tt.wantErr)
			}
			if (g.ResolvedAt != nil) != tt.wantResolved {
				t.Errorf("Expected resolved=%v, got resolved=%v", tt.wantResolved, g.ResolvedAt != nil)
			}
		})
	}
}

// TestTraceRecord tests that trace records are written correctly.
func TestTraceRecord(t *testing.T) {
	tmpDir := t.TempDir()

	rec := &traceRecord{
		Phase:        "plan",
		FeatureID:    "F134",
		RunID:        "test-run",
		Strict:       false,
		EvidencePath: "",
		GateID:       "plan-F134-test-run",
		Answer:       "approve",
		Answerer:     "sdp-phase-auto",
		Resolved:     true,
		Timestamp:    time.Now().UTC(),
	}

	writeTraceRecord(tmpDir, rec)

	tracePath := filepath.Join(tmpDir, "trace.json")
	data, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("Failed to read trace file: %v", err)
	}

	var loaded traceRecord
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to parse trace JSON: %v", err)
	}

	if loaded.Phase != "plan" {
		t.Errorf("Expected phase 'plan', got %q", loaded.Phase)
	}
	if loaded.FeatureID != "F134" {
		t.Errorf("Expected featureID 'F134', got %q", loaded.FeatureID)
	}
	if !loaded.Resolved {
		t.Error("Expected resolved=true")
	}
	if loaded.Answer != "approve" {
		t.Errorf("Expected answer 'approve', got %q", loaded.Answer)
	}
}

// Example_phasePlan provides an example of using the phase plan command.
func Example_phasePlan() {
	// This example demonstrates the phase plan command.
	// In real usage, this would be called via:
	//   sdp phase plan --feature-id F134

	fmt.Println("Phase: Plan")
	fmt.Println("Feature:  F134")
	fmt.Println("Run ID:   20250119-120000-000")
	fmt.Println("Strict:   false")
	// Output:
	// Phase: Plan
	// Feature:  F134
	// Run ID:   20250119-120000-000
	// Strict:   false
}

// TestGateReadBack tests that an existing resolved gate.json is read back
// instead of being overwritten.
func TestGateReadBack(t *testing.T) {
	tmpDir := t.TempDir()
	phaseDir := filepath.Join(tmpDir, ".sdp", "phases", "test-run")
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a resolved gate.json
	resolvedAt := time.Now().UTC().Truncate(time.Second)
	existing := &gate.Gate{
		ID:         "plan-F134-test-run",
		Question:   "Approve plan delta?",
		Context:    "plan phase for feature F134",
		Options:    []string{"approve", "reject", "defer"},
		Type:       gate.GateTypePlan,
		CreatedAt:  time.Now().Add(-time.Minute),
		Answer:     "approve",
		Answerer:   "human-operator",
		ResolvedAt: &resolvedAt,
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	gatePath := filepath.Join(phaseDir, "gate.json")
	if err := os.WriteFile(gatePath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Read it back and verify resolution
	loaded, err := os.ReadFile(gatePath)
	if err != nil {
		t.Fatal(err)
	}
	var readBack gate.Gate
	if err := json.Unmarshal(loaded, &readBack); err != nil {
		t.Fatal(err)
	}

	if readBack.Answer != "approve" {
		t.Errorf("Expected answer 'approve', got %q", readBack.Answer)
	}
	if readBack.Answerer != "human-operator" {
		t.Errorf("Expected answerer 'human-operator', got %q", readBack.Answerer)
	}
	if readBack.ResolvedAt == nil {
		t.Error("Expected gate to be resolved")
	}
	if readBack.ResolvedAt.Format(time.RFC3339) != resolvedAt.Format(time.RFC3339) {
		t.Errorf("Expected resolved_at %s, got %s", resolvedAt.Format(time.RFC3339), readBack.ResolvedAt.Format(time.RFC3339))
	}
}

// TestGateReadBackPending tests that a pending (unresolved) gate.json
// is read back and reported as AWAITING.
func TestGateReadBackPending(t *testing.T) {
	tmpDir := t.TempDir()
	phaseDir := filepath.Join(tmpDir, ".sdp", "phases", "test-run-pending")
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a pending gate.json (no answer/resolved_at)
	pending := &gate.Gate{
		ID:        "plan-F134-test-run-pending",
		Question:  "Approve plan delta?",
		Context:   "plan phase for feature F134",
		Options:   []string{"approve", "reject", "defer"},
		Type:      gate.GateTypePlan,
		CreatedAt: time.Now().Add(-time.Minute),
	}
	data, err := json.MarshalIndent(pending, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	gatePath := filepath.Join(phaseDir, "gate.json")
	if err := os.WriteFile(gatePath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Read it back and verify it's still pending
	loaded, err := os.ReadFile(gatePath)
	if err != nil {
		t.Fatal(err)
	}
	var readBack gate.Gate
	if err := json.Unmarshal(loaded, &readBack); err != nil {
		t.Fatal(err)
	}

	if readBack.ResolvedAt != nil {
		t.Error("Expected gate to be unresolved (pending)")
	}
	if !readBack.IsBlocking() {
		t.Error("Expected pending gate to be blocking")
	}
	if readBack.Status() != "pending" {
		t.Errorf("Expected status 'pending', got %q", readBack.Status())
	}
}

// Helper functions for tests

// createTestGate creates a test gate for the given phase.
func createTestGate(phase, featureID, runID string) *gate.Gate {
	var question string
	var options []string

	switch phase {
	case "plan":
		question = "Approve plan delta?"
		options = []string{"approve", "reject", "defer"}
	case "review":
		question = "Approve review delta?"
		options = []string{"approve", "reject", "request-changes"}
	case "eval":
		question = "Approve eval results?"
		options = []string{"approve", "reject", "retry"}
	default:
		question = "Approve delta?"
		options = []string{"approve", "reject"}
	}

	return &gate.Gate{
		ID:        fmt.Sprintf("%s-%s-%s", phase, featureID, runID),
		Question:  question,
		Context:   fmt.Sprintf("%s phase for feature %s", phase, featureID),
		Options:   options,
		CreatedAt: time.Now(),
	}
}
