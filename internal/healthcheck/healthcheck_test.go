package healthcheck_test

import (
	"context"
	"testing"

	"sdp_dev/internal/healthcheck"
)

func TestNewRunner_EmptyRootReturnsError(t *testing.T) {
	_, err := healthcheck.NewRunner(healthcheck.Config{})
	if err == nil {
		t.Fatal("expected error for empty ProjectRoot, got nil")
	}
}

func TestNewRunner_UnknownCheckReturnsError(t *testing.T) {
	_, err := healthcheck.NewRunner(healthcheck.Config{
		ProjectRoot: t.TempDir(),
		Only:        "nonexistent-check",
	})
	if err == nil {
		t.Fatal("expected error for unknown check name, got nil")
	}
}

func TestNewRunner_KnownChecksAccepted(t *testing.T) {
	for _, name := range []string{"go-build", "beads-ready", "git-clean"} {
		_, err := healthcheck.NewRunner(healthcheck.Config{
			ProjectRoot: t.TempDir(),
			Only:        name,
		})
		if err != nil {
			t.Errorf("NewRunner(Only=%q) unexpected error: %v", name, err)
		}
	}
}

func TestRunner_OnlyFilterSelectsOne(t *testing.T) {
	runner, err := healthcheck.NewRunner(healthcheck.Config{
		ProjectRoot: t.TempDir(),
		Only:        "go-build",
	})
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	results := runner.Run(context.Background())
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "go-build" {
		t.Errorf("expected name go-build, got %q", results[0].Name)
	}
}

func TestCheckResult_StatusConstants(t *testing.T) {
	if string(healthcheck.StatusPass) != "pass" {
		t.Errorf("StatusPass = %q, want %q", healthcheck.StatusPass, "pass")
	}
	if string(healthcheck.StatusFail) != "fail" {
		t.Errorf("StatusFail = %q, want %q", healthcheck.StatusFail, "fail")
	}
	if string(healthcheck.StatusWarn) != "warn" {
		t.Errorf("StatusWarn = %q, want %q", healthcheck.StatusWarn, "warn")
	}
}
