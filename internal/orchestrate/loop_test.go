package orchestrate_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/orchestrate"
)

func TestLoopExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "context.Canceled returns ExitNeedsHuman (2)",
			err:      context.Canceled,
			expected: orchestrate.ExitNeedsHuman,
		},
		{
			name:     "general error returns ExitFailure (1)",
			err:      errors.New("some error"),
			expected: orchestrate.ExitFailure,
		},
		{
			name:     "nil error should not crash",
			err:      nil,
			expected: orchestrate.ExitFailure, // defaults to 1
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := orchestrate.LoopExitCode(tt.err)
			if got != tt.expected {
				t.Errorf("LoopExitCode(%v) = %d, want %d", tt.err, got, tt.expected)
			}
		})
	}
}

func TestLoopConfig_Defaults(t *testing.T) {
	cfg := orchestrate.LoopConfig{}
	if cfg.NoCommit {
		t.Error("NoCommit should default to false")
	}
	if cfg.OutputDir != "" {
		t.Errorf("OutputDir should default to empty string, got %q", cfg.OutputDir)
	}
}

func TestLoopConfig_Setters(t *testing.T) {
	cfg := orchestrate.LoopConfig{
		NoCommit:  true,
		OutputDir: "/tmp/output",
	}
	if !cfg.NoCommit {
		t.Error("NoCommit should be true")
	}
	if cfg.OutputDir != "/tmp/output" {
		t.Errorf("OutputDir should be /tmp/output, got %q", cfg.OutputDir)
	}
}

func TestFormatProgress(t *testing.T) {
	info := orchestrate.ProgressInfo{
		Done:  2,
		Total: 7,
		WSID:  "00-042-03",
		Phase: "building",
	}
	got := orchestrate.FormatProgress(info)
	expected := "[3/7] building 00-042-03"
	if got != expected {
		t.Errorf("FormatProgress() = %q, want %q", got, expected)
	}
}
