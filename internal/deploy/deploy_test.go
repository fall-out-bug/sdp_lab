package deploy

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShortHash(t *testing.T) {
	if shortHash("abcdef1234567890") != "abcdef123456" {
		t.Error("wrong short hash")
	}
	if shortHash("abc") != "abc" {
		t.Error("short hash should not truncate short strings")
	}
}

func TestParseTime(t *testing.T) {
	now := "2026-04-25T12:00:00Z"
	parsed := parseTime(now)
	if parsed.IsZero() {
		t.Error("failed to parse valid time")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("/home/user/my.project")
	if cfg.ComposeStaging != "/home/user/my.project/docker-compose.staging.yml" {
		t.Error("wrong staging path")
	}
	if cfg.ComposeProd != "/home/user/my.project/docker-compose.yml" {
		t.Error("wrong prod path")
	}
	if cfg.ProjectName != "my-project" {
		t.Error("wrong project name")
	}
}

func TestResult_JSON(t *testing.T) {
	r := Result{
		Phase:    "staging",
		Target:   "staging",
		ImageTag: "test:latest",
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Result
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Phase != "staging" {
		t.Error("phase mismatch")
	}
}

func TestHealthCheckResult(t *testing.T) {
	h := HealthCheckResult{
		Passed:  true,
		Checks:  []string{"running"},
		Minutes: 5.0,
	}
	if !h.Passed {
		t.Error("should be passed")
	}
}

func TestSmokeTestResult(t *testing.T) {
	s := TestResult{
		Passed:   true,
		ExitCode: 0,
		Output:   "all tests passed",
	}
	if !s.Passed {
		t.Error("should be passed")
	}
}

func TestContainerInfo(t *testing.T) {
	c := ContainerInfo{
		Name:   "app",
		ID:     "abc123",
		Image:  "test:latest",
		Status: "running",
		Ports:  "80:80",
	}
	if c.Name != "app" {
		t.Error("name mismatch")
	}
}

func TestNilConfig(t *testing.T) {
	_, err := Staging(context.TODO(), nil, "hash")
	if err == nil {
		t.Error("expected error for nil config")
	}
	_, err = Production(context.TODO(), nil, "tag")
	if err == nil {
		t.Error("expected error for nil config")
	}
	_, err = Rollback(context.TODO(), nil, "tag")
	if err == nil {
		t.Error("expected error for nil config")
	}
}

// --- Gate tests ---

func TestDefaultGatesConfig(t *testing.T) {
	g := DefaultGatesConfig()
	if !g.SecretScan || !g.Evidence || !g.SmokeTest {
		t.Error("default gates should have all hard gates enabled")
	}
	if g.Staged {
		t.Error("staged rollout should be off by default")
	}
	if g.CanaryPct != 10 {
		t.Errorf("CanaryPct = %d, want 10", g.CanaryPct)
	}
	if g.CanaryService != "app" {
		t.Errorf("CanaryService = %q, want %q", g.CanaryService, "app")
	}
}

func TestCheckGates_AllPass(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)

	// Create a valid evidence.json.
	evDir := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ev := map[string]any{"status": "converged", "run_id": "test"}
	data, _ := json.Marshal(ev)
	if err := os.WriteFile(filepath.Join(evDir, "evidence.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := CheckGates(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("CheckGates: %v", err)
	}
	for _, r := range results {
		if !r.Passed {
			t.Errorf("gate %s should pass: %s", r.Gate, r.Message)
		}
	}
}

func TestCheckGates_SecretScanDetectsSecret(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)

	// Create a file with an AWS secret key.
	secretFile := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(secretFile, []byte("aws_secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create valid evidence.json.
	evDir := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ev := map[string]any{"status": "ok"}
	data, _ := json.Marshal(ev)
	if err := os.WriteFile(filepath.Join(evDir, "evidence.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	gates := DefaultGatesConfig()
	results, err := CheckGates(context.Background(), cfg, gates)
	if err == nil {
		t.Fatal("expected error from secret scan gate")
	}

	// Find the secret_scan gate result.
	var found bool
	for _, r := range results {
		if r.Gate == "secret_scan" && !r.Passed {
			found = true
		}
	}
	if !found {
		t.Error("secret_scan gate should have failed")
	}
}

func TestCheckGates_NoEvidenceFile(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)

	gates := DefaultGatesConfig()
	results, err := CheckGates(context.Background(), cfg, gates)
	if err == nil {
		t.Fatal("expected error from evidence gate")
	}

	var found bool
	for _, r := range results {
		if r.Gate == "evidence" && !r.Passed {
			found = true
		}
	}
	if !found {
		t.Error("evidence gate should have failed")
	}
}

func TestCheckGates_InvalidEvidenceFile(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)

	evDir := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evDir, "evidence.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	gates := DefaultGatesConfig()
	results, err := CheckGates(context.Background(), cfg, gates)
	if err == nil {
		t.Fatal("expected error from evidence gate")
	}

	var found bool
	for _, r := range results {
		if r.Gate == "evidence" && !r.Passed {
			found = true
		}
	}
	if !found {
		t.Error("evidence gate should have failed for invalid JSON")
	}
}

func TestCheckGates_SkipGates(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)

	// No evidence file, no nothing — should fail all gates normally.
	// But if we disable all gates, it should pass.
	gates := &GatesConfig{
		SecretScan: false,
		Evidence:   false,
		SmokeTest:  false,
	}

	results, err := CheckGates(context.Background(), cfg, gates)
	if err != nil {
		t.Fatalf("CheckGates with all gates disabled: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results with all gates disabled, got %d", len(results))
	}
}

func TestCheckGates_SmokeTestFail(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)

	// Create valid evidence.json.
	evDir := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ev := map[string]any{"status": "ok"}
	data, _ := json.Marshal(ev)
	if err := os.WriteFile(filepath.Join(evDir, "evidence.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a failing smoke test script.
	smokeScript := `#!/bin/bash
echo "FAIL: something broke"
exit 1
`
	if err := os.WriteFile(filepath.Join(dir, "smoke-test.sh"), []byte(smokeScript), 0o755); err != nil {
		t.Fatal(err)
	}

	gates := DefaultGatesConfig()
	results, err := CheckGates(context.Background(), cfg, gates)
	if err == nil {
		t.Fatal("expected error from smoke test gate")
	}

	var found bool
	for _, r := range results {
		if r.Gate == "smoke_test" && !r.Passed {
			found = true
		}
	}
	if !found {
		t.Error("smoke_test gate should have failed")
	}
}

func TestGateResult_JSON(t *testing.T) {
	gr := GateResult{
		Gate:    "secret_scan",
		Passed:  true,
		Message: "clean (42 files scanned)",
	}
	data, err := json.Marshal(gr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded GateResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Gate != "secret_scan" {
		t.Error("gate mismatch")
	}
}

func TestTruncate(t *testing.T) {
	if truncate("short", 10) != "short" {
		t.Error("should not truncate short strings")
	}
	long := strings.Repeat("x", 200)
	result := truncate(long, 50)
	if len(result) != 53 { // 50 + "..."
		t.Errorf("truncated length = %d, want 53", len(result))
	}
}

func TestWriteRollbackEvidence(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)

	writeRollbackEvidence(cfg, "canary unhealthy", "test:canary")

	path := filepath.Join(dir, ".sdp", "deploy.rollback.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rollback evidence: %v", err)
	}

	var ev map[string]any
	if err := json.Unmarshal(data, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev["action"] != "rollback" {
		t.Error("wrong action")
	}
	if ev["reason"] != "canary unhealthy" {
		t.Error("wrong reason")
	}
	if ev["tag"] != "test:canary" {
		t.Error("wrong tag")
	}
}

func TestStagedRollout_InvalidCanaryPct(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)
	gates := &GatesConfig{CanaryPct: 0, CanaryService: "app"}

	_, err := StagedRollout(context.Background(), cfg, gates, "test:latest")
	if err == nil {
		t.Error("expected error for invalid canary percentage")
	}

	gates.CanaryPct = 100
	_, err = StagedRollout(context.Background(), cfg, gates, "test:latest")
	if err == nil {
		t.Error("expected error for 100% canary")
	}
}

func TestStagedRollout_EmptyCanaryService(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)
	gates := &GatesConfig{CanaryPct: 10, CanaryService: ""}

	_, err := StagedRollout(context.Background(), cfg, gates, "test:latest")
	if err == nil {
		t.Error("expected error for empty canary service")
	}
}

func TestCheckEvidenceGate_PerRunEvidence(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)

	// Create per-run evidence under .sdp/evidence/<run_id>/evidence.json.
	runDir := filepath.Join(dir, ".sdp", "evidence", "run-123")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ev := map[string]any{"status": "converged", "run_id": "run-123"}
	data, _ := json.Marshal(ev)
	if err := os.WriteFile(filepath.Join(runDir, "evidence.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	gates := &GatesConfig{Evidence: true}
	results, err := CheckGates(context.Background(), cfg, gates)
	if err != nil {
		t.Fatalf("CheckGates: %v", err)
	}

	var found bool
	for _, r := range results {
		if r.Gate == "evidence" && r.Passed {
			found = true
		}
	}
	if !found {
		t.Error("evidence gate should pass with per-run evidence file")
	}
}

func TestCheckEvidenceGate_PerFeatureEvidence(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)

	// Create per-feature evidence under .sdp/evidence/F135.json.
	evDir := filepath.Join(dir, ".sdp", "evidence")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ev := map[string]any{"status": "ok", "feature": "F135"}
	data, _ := json.Marshal(ev)
	if err := os.WriteFile(filepath.Join(evDir, "F135.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	gates := &GatesConfig{Evidence: true}
	results, err := CheckGates(context.Background(), cfg, gates)
	if err != nil {
		t.Fatalf("CheckGates: %v", err)
	}

	var found bool
	for _, r := range results {
		if r.Gate == "evidence" && r.Passed {
			found = true
		}
	}
	if !found {
		t.Error("evidence gate should pass with per-feature evidence file")
	}
}

func TestGatesConfig_SecretScanOnly(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig(dir)

	// Create evidence so that gate passes.
	evDir := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ev := map[string]any{"status": "ok"}
	data, _ := json.Marshal(ev)
	if err := os.WriteFile(filepath.Join(evDir, "evidence.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Only secret scan enabled.
	gates := &GatesConfig{
		SecretScan: true,
		Evidence:   false,
		SmokeTest:  false,
	}

	results, err := CheckGates(context.Background(), cfg, gates)
	if err != nil {
		t.Fatalf("CheckGates: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].Gate != "secret_scan" {
		t.Errorf("expected secret_scan gate, got %s", results[0].Gate)
	}
}
