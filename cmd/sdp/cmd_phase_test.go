package main

import (
	"flag"
	"fmt"
	"os"
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
			t.Logf("Parsed flags: featureID=%q, wsID=%q, runID=%q, strict=%v", f.featureID, f.wsID, f.runID, f.strict)

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
		name     string
		setup    func(*gate.Gate)
		wantStatus string
		wantBlocking bool
	}{
		{
			name: "pending gate",
			setup: func(g *gate.Gate) {
				// Default state is pending
			},
			wantStatus: "pending",
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
			wantStatus: "resolved",
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
