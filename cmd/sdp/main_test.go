package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildTestBinary(t *testing.T) string {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "sdp-test")
	cmd := exec.Command("go", "build", "-tags", "sqlite_fts5", "-o", binPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build test binary: %v\n%s", err, out)
	}
	return binPath
}

func TestMainUsage(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error exit, got nil")
	}
	output := out.String()
	if !strings.Contains(output, "usage: sdp") {
		t.Fatalf("expected usage message in output, got: %s", output)
	}
	if !strings.Contains(output, "Card commands:") {
		t.Fatalf("expected command sections in output, got: %s", output)
	}
}

func TestMainHelpExitsZeroAndListsInstallCommands(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath, "--help")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("expected --help to exit 0, got %v\n%s", err, out.String())
	}
	output := out.String()
	for _, want := range []string{
		"sdp init",
		"sdp manifest validate",
		"sdp generate-adapters",
		"sdp doctor adapters",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help to contain %q, got:\n%s", want, output)
		}
	}
}

func TestMainHelpListsImplementedProductCommands(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath, "--help")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("expected --help to exit 0, got %v\n%s", err, out.String())
	}
	output := out.String()
	for _, want := range []string{
		"sdp metrics",
		"sdp architect analyze",
		"sdp coverage-scan",
		"sdp skills augment",
		"sdp tower",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help to contain implemented command %q, got:\n%s", want, output)
		}
	}
}

func TestCardUsage(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath, "card")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error exit, got nil")
	}
	output := out.String()
	if !strings.Contains(output, "usage: sdp card") {
		t.Fatalf("expected card usage in output, got: %s", output)
	}
	if !strings.Contains(output, "heartbeat") {
		t.Fatalf("expected heartbeat subcommand in card usage, got: %s", output)
	}
}

func TestBoardUsage(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath, "board")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error exit, got nil")
	}
	output := out.String()
	if !strings.Contains(output, "usage: sdp board") {
		t.Fatalf("expected board usage in output, got: %s", output)
	}
}

func TestDoctorControlUsage(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath, "doctor")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	_ = cmd.Run()
	output := out.String()
	if strings.Contains(output, "usage: sdp doctor") {
		t.Fatalf("plain doctor should run doctor control, got usage:\n%s", output)
	}
	if !strings.Contains(output, "DOCTOR CONTROL") {
		t.Fatalf("expected doctor control output, got:\n%s", output)
	}
}

func TestDispatchUsage(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath, "dispatch")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error exit, got nil")
	}
	output := out.String()
	if !strings.Contains(output, "usage: sdp dispatch card") {
		t.Fatalf("expected dispatch usage in output, got: %s", output)
	}
}

func TestResultUsage(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath, "result")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error exit, got nil")
	}
	output := out.String()
	if !strings.Contains(output, "usage: sdp result ingest") {
		t.Fatalf("expected result usage in output, got: %s", output)
	}
}

func TestUnknownCommand(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath, "unknown")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error exit, got nil")
	}
}
