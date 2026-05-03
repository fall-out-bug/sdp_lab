//go:build sdp_experimental

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainMissingSkillExits(t *testing.T) {
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
	bin := filepath.Join(t.TempDir(), "sdp-eval")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/sdp-eval")
	cmd.Dir = modRoot
	if err := cmd.Run(); err != nil {
		t.Skipf("build failed: %v", err)
	}
	out, err := exec.Command(bin).CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit when --skill and --all are missing")
	}
	s := string(out)
	if !strings.Contains(s, "skill") && !strings.Contains(s, "error") {
		t.Errorf("stderr should mention skill or error, got: %s", out)
	}
}

func TestMainRunsBehaviorSuite(t *testing.T) {
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
	bin := filepath.Join(t.TempDir(), "sdp-eval")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/sdp-eval")
	cmd.Dir = modRoot
	if err := cmd.Run(); err != nil {
		t.Skipf("build failed: %v", err)
	}

	run := exec.Command(bin, "--skill", "behavior", "--project-root", ".")
	run.Dir = modRoot
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("expected behavior suite to pass, got %v: %s", err, out)
	}
	if !strings.Contains(string(out), "behavior: 3/3 passed") {
		t.Fatalf("unexpected behavior output: %s", out)
	}
}

func TestMainIndirectPIReportText(t *testing.T) {
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
	bin := filepath.Join(t.TempDir(), "sdp-eval")
	cmd := exec.Command("go", "build", "-tags", "sdp_experimental", "-o", bin, "./cmd/sdp-eval")
	cmd.Dir = modRoot
	if err := cmd.Run(); err != nil {
		t.Skipf("build failed: %v", err)
	}

	run := exec.Command(bin, "--indirect-pi-report", "--project-root", ".")
	run.Dir = modRoot
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("expected indirect-pi report to pass, got %v: %s", err, out)
	}
	s := string(out)
	if !strings.Contains(s, "F165 Indirect Prompt Injection Report") {
		t.Fatalf("report missing header: %s", s)
	}
	if !strings.Contains(s, "F165-VEC-001") {
		t.Fatalf("report missing case F165-VEC-001: %s", s)
	}
	if !strings.Contains(s, "advisory demo report") {
		t.Fatalf("report missing advisory disclaimer: %s", s)
	}
}

func TestMainIndirectPIReportJSON(t *testing.T) {
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
	bin := filepath.Join(t.TempDir(), "sdp-eval")
	cmd := exec.Command("go", "build", "-tags", "sdp_experimental", "-o", bin, "./cmd/sdp-eval")
	cmd.Dir = modRoot
	if err := cmd.Run(); err != nil {
		t.Skipf("build failed: %v", err)
	}

	run := exec.Command(bin, "--indirect-pi-report", "--indirect-pi-json", "--project-root", ".")
	run.Dir = modRoot
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("expected indirect-pi json report to pass, got %v: %s", err, out)
	}
	s := string(out)
	if !strings.Contains(s, `"feature_id": "F165"`) {
		t.Fatalf("json report missing feature_id: %s", s)
	}
	if !strings.Contains(s, `"case_id": "F165-VEC-001"`) {
		t.Fatalf("json report missing case_id: %s", s)
	}
}

func TestMainPromptInjectionReportSkipsLiveByDefault(t *testing.T) {
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
	bin := filepath.Join(t.TempDir(), "sdp-eval")
	cmd := exec.Command("go", "build", "-tags", "sdp_experimental", "-o", bin, "./cmd/sdp-eval")
	cmd.Dir = modRoot
	if err := cmd.Run(); err != nil {
		t.Skipf("build failed: %v", err)
	}

	run := exec.Command(bin, "--prompt-injection-report", "--project-root", ".")
	run.Dir = modRoot
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("expected prompt-injection report to pass without live credentials, got %v: %s", err, out)
	}
	s := string(out)
	if !strings.Contains(s, "PI-013 supply-chain case present") {
		t.Fatalf("report should include PI-013 supply-chain status: %s", s)
	}
	if !strings.Contains(s, "ADVISORY_DEGRADED") {
		t.Fatalf("live-provider status should be advisory-degraded by default: %s", s)
	}
}
