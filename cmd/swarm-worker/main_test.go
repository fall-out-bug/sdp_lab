package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sdp_dev/internal/llm"
	"sdp_dev/internal/observability"
)

func TestProjectFromIssueID(t *testing.T) {
	if got := projectFromIssueID("sdp_dev-986"); got != "sdp_dev" {
		t.Errorf("projectFromIssueID(sdp_dev-986) = %q, want sdp_dev", got)
	}
	if got := projectFromIssueID("abc-123"); got != "abc" {
		t.Errorf("projectFromIssueID(abc-123) = %q, want abc", got)
	}
	if got := projectFromIssueID("nodash"); got != "" {
		t.Errorf("projectFromIssueID(nodash) = %q, want empty", got)
	}
	if got := projectFromIssueID(""); got != "" {
		t.Errorf("projectFromIssueID(\"\") = %q, want empty", got)
	}
}

func TestExtractLinkage(t *testing.T) {
	dir := t.TempDir()
	evDir := filepath.Join(dir, ".sdp", "evidence")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Missing file
	path, prURL := extractLinkage("missing-issue")
	if path != ".sdp/evidence/missing-issue.json" {
		t.Errorf("extractLinkage missing: path=%q", path)
	}
	if prURL != "" {
		t.Errorf("extractLinkage missing: prURL=%q", prURL)
	}

	// File with trace and pr_url
	evPath := filepath.Join(evDir, "issue-with-pr.json")
	payload := map[string]any{
		"trace": map[string]any{
			"evidence_context_link": "/path/to/evidence",
			"pr_url":                 "https://github.com/org/repo/pull/1",
		},
	}
	b, _ := json.Marshal(payload)
	if err := os.WriteFile(evPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	path, prURL = extractLinkage("issue-with-pr")
	if path != "/path/to/evidence" {
		t.Errorf("extractLinkage with trace: path=%q", path)
	}
	if prURL != "https://github.com/org/repo/pull/1" {
		t.Errorf("extractLinkage with trace: prURL=%q", prURL)
	}

	// File with trace but no pr_url
	evPath2 := filepath.Join(evDir, "issue-no-pr.json")
	payload2 := map[string]any{"trace": map[string]any{"evidence_context_link": ""}}
	b2, _ := json.Marshal(payload2)
	if err := os.WriteFile(evPath2, b2, 0o644); err != nil {
		t.Fatal(err)
	}
	path, prURL = extractLinkage("issue-no-pr")
	if path != ".sdp/evidence/issue-no-pr.json" {
		t.Errorf("extractLinkage no pr: path=%q", path)
	}
	if prURL != "" {
		t.Errorf("extractLinkage no pr: prURL=%q", prURL)
	}
}

func TestEnsureOneShotManifestFiles(t *testing.T) {
	dir := t.TempDir()
	if err := ensureOneShotManifestFiles(dir); err != nil {
		t.Fatalf("ensureOneShotManifestFiles: %v", err)
	}
	corePath := filepath.Join(dir, "internal", "oneshot", "manifest.go")
	testPath := filepath.Join(dir, "internal", "oneshot", "manifest_test.go")
	if _, err := os.Stat(corePath); err != nil {
		t.Fatalf("manifest.go not created: %v", err)
	}
	if _, err := os.Stat(testPath); err != nil {
		t.Fatalf("manifest_test.go not created: %v", err)
	}
}

func TestEnsurePlannerEnvelopeFiles(t *testing.T) {
	dir := t.TempDir()
	if err := ensurePlannerEnvelopeFiles(dir); err != nil {
		t.Fatalf("ensurePlannerEnvelopeFiles: %v", err)
	}
	corePath := filepath.Join(dir, "internal", "planner", "envelope.go")
	testPath := filepath.Join(dir, "internal", "planner", "envelope_test.go")
	if _, err := os.Stat(corePath); err != nil {
		t.Fatalf("envelope.go not created: %v", err)
	}
	if _, err := os.Stat(testPath); err != nil {
		t.Fatalf("envelope_test.go not created: %v", err)
	}
}

func TestEnsureTelegramIntakeFiles(t *testing.T) {
	dir := t.TempDir()
	if err := ensureTelegramIntakeFiles(dir); err != nil {
		t.Fatalf("ensureTelegramIntakeFiles: %v", err)
	}
	corePath := filepath.Join(dir, "internal", "intake", "telegram.go")
	testPath := filepath.Join(dir, "internal", "intake", "telegram_test.go")
	if _, err := os.Stat(corePath); err != nil {
		t.Fatalf("telegram.go not created: %v", err)
	}
	if _, err := os.Stat(testPath); err != nil {
		t.Fatalf("telegram_test.go not created: %v", err)
	}
}

func TestApplyWorkstreamFlowOneshot(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	got := applyWorkstreamFlow("oneshot-swarm-orchestrator", "issue-1", issueDetail{ID: "issue-1", Title: "T"}, "glm-5")
	if len(got) != 2 {
		t.Fatalf("applyWorkstreamFlow oneshot: %v", got)
	}
	if !strings.Contains(got[0], "manifest.go") || !strings.Contains(got[1], "manifest_test.go") {
		t.Fatalf("applyWorkstreamFlow oneshot unexpected files: %v", got)
	}
}

func TestApplyWorkstreamFlowPlanner(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	got := applyWorkstreamFlow("planner-boundary-decomposition", "issue-1", issueDetail{ID: "issue-1", Title: "T"}, "glm-5")
	if len(got) != 2 {
		t.Fatalf("applyWorkstreamFlow planner: %v", got)
	}
	if !strings.Contains(got[0], "envelope.go") || !strings.Contains(got[1], "envelope_test.go") {
		t.Fatalf("applyWorkstreamFlow planner unexpected files: %v", got)
	}
}

func TestApplyWorkstreamFlowTelegram(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	got := applyWorkstreamFlow("telegram-ingress-intake", "issue-1", issueDetail{ID: "issue-1", Title: "T"}, "glm-5")
	if len(got) != 2 {
		t.Fatalf("applyWorkstreamFlow telegram: %v", got)
	}
	if !strings.Contains(got[0], "telegram.go") || !strings.Contains(got[1], "telegram_test.go") {
		t.Fatalf("applyWorkstreamFlow telegram unexpected files: %v", got)
	}
}

func TestApplyWorkstreamFlowPolicySlugifyTrim(t *testing.T) {
	dir := t.TempDir()
	policyDir := filepath.Join(dir, "internal", "policy")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Old block that patchSlugifyForTrim looks for (exact match)
	decisionGo := `package policy
import ("regexp";"strings")
var nonAlnum = regexp.MustCompile(` + "`[^a-z0-9]+`" + `)
func slugify(text string) string {
	t := strings.ToLower(text)
	t = nonAlnum.ReplaceAllString(t, "-")
	t = strings.Trim(t, "-")
	if t == "" { return "task" }
	if len(t) > 48 {
		return t[:48]
	}
	return t
}
`
	decisionTest := `package policy
import "testing"
func TestDecideCriticalEscalates(t *testing.T) {}
`
	if err := os.WriteFile(filepath.Join(policyDir, "decision.go"), []byte(decisionGo), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "decision_test.go"), []byte(decisionTest), 0o644); err != nil {
		t.Fatal(err)
	}
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	got := applyWorkstreamFlow("policy-slugify-trim", "issue-1", issueDetail{ID: "issue-1", Title: "T"}, "glm-5")
	if len(got) != 2 {
		t.Fatalf("applyWorkstreamFlow policy-slugify-trim: %v", got)
	}
	// Verify patch applied
	b, _ := os.ReadFile("internal/policy/decision.go")
	if !strings.Contains(string(b), "strings.Trim(t, \"-\")") {
		t.Fatalf("patch not applied: %s", string(b))
	}
}

func TestEmitWorkerObservability(t *testing.T) {
	// Smoke test: ensure it doesn't panic when evidenceContextLink is empty
	emitWorkerObservability("issue-1", "plan", "running", "glm-5", time.Now(), 0, false, false, "", "")
}

func TestApplyWorkstreamFlowModelChain(t *testing.T) {
	dir := t.TempDir()
	policyDir := filepath.Join(dir, "internal", "policy")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Old block before patch (no AllowedModel check)
	modelChain := `package policy
var FallbackChain = []string{"glm-5","glm-4.7"}
func AllowedModel(m string) bool { return m=="glm-5"||m=="glm-4.7" }
func DefaultModel() string { return "glm-5" }
func ResolveFallbackSequence(start string) []string {
	sequence := []string{start}
	current := start
	for { break }
	return append(sequence, "escalated")
}
`
	modelChainTest := `package policy
import "testing"
func TestResolveFallbackSequence(t *testing.T) {}
`
	if err := os.WriteFile(filepath.Join(policyDir, "model_chain.go"), []byte(modelChain), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "model_chain_test.go"), []byte(modelChainTest), 0o644); err != nil {
		t.Fatal(err)
	}
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	got := applyWorkstreamFlow("model-chain-default-fallback", "issue-1", issueDetail{ID: "issue-1", Title: "T"}, "glm-5")
	if len(got) != 2 {
		t.Fatalf("applyWorkstreamFlow model-chain: %v", got)
	}
	b, _ := os.ReadFile("internal/policy/model_chain.go")
	if !strings.Contains(string(b), "!AllowedModel(start)") {
		t.Fatalf("patch not applied: %s", string(b))
	}
}

func TestApplyWorkstreamFlowPolicyK8sRiskHigh(t *testing.T) {
	dir := t.TempDir()
	policyDir := filepath.Join(dir, "internal", "policy")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// highPatterns with git but not k8s (patch adds k8s)
	decisionGo := `package policy
import "regexp"
var highPatterns = []*regexp.Regexp{
	regexp.MustCompile(` + "`git`" + `),
}
`
	decisionTest := `package policy
import "testing"
func TestDecideCriticalEscalates(t *testing.T) {}
`
	if err := os.WriteFile(filepath.Join(policyDir, "decision.go"), []byte(decisionGo), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "decision_test.go"), []byte(decisionTest), 0o644); err != nil {
		t.Fatal(err)
	}
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	got := applyWorkstreamFlow("policy-k8s-risk-high", "issue-1", issueDetail{ID: "issue-1", Title: "T"}, "glm-5")
	if len(got) != 2 {
		t.Fatalf("applyWorkstreamFlow policy-k8s-risk-high: %v", got)
	}
	b, _ := os.ReadFile("internal/policy/decision.go")
	if !strings.Contains(string(b), "regexp.MustCompile(`k8s`)") {
		t.Fatalf("patch not applied: %s", string(b))
	}
}

func TestHasStagedChanges(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	// Init git repo
	if out, err := exec.Command("git", "init").CombinedOutput(); err != nil {
		t.Skipf("git init failed: %v %s", err, out)
	}
	// No staged changes
	staged, err := hasStagedChanges()
	if err != nil {
		t.Fatalf("hasStagedChanges: %v", err)
	}
	if staged {
		t.Fatal("expected no staged changes in fresh repo")
	}
	// Create and stage a file
	if err := os.WriteFile("foo.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "add", "foo.txt").CombinedOutput(); err != nil {
		t.Skipf("git add failed: %v %s", err, out)
	}
	staged, err = hasStagedChanges()
	if err != nil {
		t.Fatalf("hasStagedChanges after add: %v", err)
	}
	if !staged {
		t.Fatal("expected staged changes after git add")
	}
}

func TestDiscardBeadsSyncNoise(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init").CombinedOutput(); err != nil {
		t.Skipf("git init: %v %s", err, out)
	}
	// No diff - should return without error
	discardBeadsSyncNoise()
	// Create only beads files - should run reset (no panic)
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "metadata.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "issues.jsonl"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "add", ".beads/metadata.json", ".beads/issues.jsonl").CombinedOutput(); err != nil {
		t.Skipf("git add: %v %s", err, out)
	}
	discardBeadsSyncNoise()
}

func TestCommitBodyForWorkstreamModelChainAndPolicyK8s(t *testing.T) {
	if got := commitBodyForWorkstream("model-chain-default-fallback"); got != "Make unknown model fallback deterministic and add regression coverage." {
		t.Errorf("commitBodyForWorkstream model-chain: %q", got)
	}
	if got := commitBodyForWorkstream("policy-k8s-risk-high"); got != "Implement workstream changes with regression coverage." {
		t.Errorf("commitBodyForWorkstream policy-k8s: %q", got)
	}
}

func TestResolveWorkstream(t *testing.T) {
	tests := []struct {
		labels []string
		want   string
	}{
		{[]string{"workstream:policy-slugify-trim"}, "policy-slugify-trim"},
		{[]string{"workstream:generic"}, "generic"},
		{[]string{"workstream:builder"}, "builder"},
		{[]string{"workstream:oneshot-swarm-orchestrator"}, "oneshot-swarm-orchestrator"},
		{[]string{"workstream:handoff-validation"}, "handoff-validation"},
		{[]string{"workstream:self-improvement"}, "self-improvement"},
		{[]string{"workstream:evaluator-recommendation"}, "evaluator-recommendation"},
		{[]string{"workstream:telegram-ingress-intake"}, "telegram-ingress-intake"},
		{[]string{"workstream:planner-boundary-decomposition"}, "planner-boundary-decomposition"},
		{[]string{}, ""},
		{[]string{"autonomy"}, ""},
	}
	for _, tt := range tests {
		got := resolveWorkstream(tt.labels)
		if got != tt.want {
			t.Errorf("resolveWorkstream(%v) = %q, want %q", tt.labels, got, tt.want)
		}
	}
}

func TestCommitBodyForWorkstream(t *testing.T) {
	tests := []struct {
		workstream string
		want       string
	}{
		{"policy-slugify-trim", "Fix slugify truncation and add regression coverage."},
		{"handoff-validation", "Add handoff validation timestamp for adapter checklist run."},
		{"generic", "Builder workstream: LLM-backed implementation via opencode run."},
		{"builder", "Builder workstream: LLM-backed implementation via opencode run."},
		{"unknown", "Implement workstream changes with regression coverage."},
	}
	for _, tt := range tests {
		got := commitBodyForWorkstream(tt.workstream)
		if got != tt.want {
			t.Errorf("commitBodyForWorkstream(%q) = %q, want %q", tt.workstream, got, tt.want)
		}
	}
}

func TestHasLabel(t *testing.T) {
	if !hasLabel([]string{"autonomy", "strict-evidence"}, "autonomy") {
		t.Error("hasLabel(autonomy) should be true")
	}
	if hasLabel([]string{"autonomy"}, "strict-evidence") {
		t.Error("hasLabel(strict-evidence) should be false")
	}
}

func TestParseClaim(t *testing.T) {
	valid := []byte(`{"issue_id":"sdp_dev-4pg","title":"x","model":"glm-5","branch":"feat/sdp_dev-4pg"}`)
	claim, err := parseClaim(valid)
	if err != nil {
		t.Fatalf("parseClaim: %v", err)
	}
	if claim.IssueID != "sdp_dev-4pg" || claim.Branch != "feat/sdp_dev-4pg" {
		t.Fatalf("unexpected claim: %+v", claim)
	}

	noise := []byte(`some output\n` + string(valid))
	claim2, err := parseClaim(noise)
	if err != nil {
		t.Fatalf("parseClaim with noise: %v", err)
	}
	if claim2.IssueID != "sdp_dev-4pg" {
		t.Fatalf("unexpected claim: %+v", claim2)
	}

	invalid := []byte(`{"issue_id":"","branch":"x"}`)
	_, err = parseClaim(invalid)
	if err == nil {
		t.Fatal("expected error for missing issue_id")
	}

	badJSON := []byte(`not json`)
	_, err = parseClaim(badJSON)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestToStringSlice(t *testing.T) {
	if got := toStringSlice([]any{"a", "b"}); len(got) != 2 || got[0] != "a" {
		t.Fatalf("toStringSlice([]any): %v", got)
	}
	if got := toStringSlice([]string{"x"}); len(got) != 1 || got[0] != "x" {
		t.Fatalf("toStringSlice([]string): %v", got)
	}
	if got := toStringSlice(123); got != nil {
		t.Fatalf("toStringSlice(123): %v", got)
	}
}

func TestHasPrefixAny(t *testing.T) {
	if !hasPrefixAny("internal/policy/foo.go", []string{"internal/", "cmd/"}) {
		t.Error("hasPrefixAny should match internal/")
	}
	if hasPrefixAny("docs/foo.md", []string{"internal/", "cmd/"}) {
		t.Error("hasPrefixAny should not match")
	}
}

func TestApplyBuilderWorkstream(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `workstreams:
  - label: workstream:builder
    path_prefixes:
      - internal/
      - cmd/
`
	if err := os.WriteFile(filepath.Join(specsDir, "workstream-config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := applyBuilderWorkstream(dir, "test-1", issueDetail{ID: "test-1", Title: "T", SpecID: "spec", Description: "d", AcceptanceCriteria: "ac"}, "glm-4.7")
	// Outcome depends on opencode availability; we only verify no panic
	if err != nil {
		t.Logf("applyBuilderWorkstream (opencode may be unavailable): %v", err)
	}
}

func TestEvaluateOneShotVerificationPassesWithTests(t *testing.T) {
	result, err := evaluateOneShotVerification([]string{"internal/oneshot/manifest.go", "internal/oneshot/manifest_test.go"}, true)
	if err != nil {
		t.Fatalf("evaluate oneshot verification: %v", err)
	}
	if !result.Report.OK {
		t.Fatalf("expected report OK, got %+v", result.Report)
	}
	if len(result.FailedTaskIDs) != 0 {
		t.Fatalf("expected no failed tasks, got %#v", result.FailedTaskIDs)
	}
	if result.RecoveryPlan != nil {
		t.Fatalf("expected no recovery plan, got %+v", *result.RecoveryPlan)
	}
}

func TestEvaluateOneShotVerificationBuildFailureCreatesRecovery(t *testing.T) {
	result, err := evaluateOneShotVerification([]string{"internal/oneshot/manifest.go", "internal/oneshot/manifest_test.go"}, false)
	if err != nil {
		t.Fatalf("evaluate oneshot verification: %v", err)
	}
	if result.Report.OK {
		t.Fatalf("expected report failure on failed tests, got %+v", result.Report)
	}
	if len(result.FailedTaskIDs) == 0 {
		t.Fatal("expected failed task ids")
	}
	if result.RecoveryPlan == nil {
		t.Fatal("expected recovery plan")
	}
	if len(result.RecoveryPlan.RequeueTaskIDs) == 0 {
		t.Fatalf("expected non-empty requeue tasks, got %+v", result.RecoveryPlan)
	}
}

func TestApplyOneShotVerificationWritesMachineReadableSections(t *testing.T) {
	payload := map[string]any{"verification": map[string]any{}}
	runPacket := map[string]any{}
	note, err := applyOneShotVerification(payload, runPacket, []string{"internal/oneshot/manifest.go"}, true)
	if err != nil {
		t.Fatalf("apply oneshot verification: %v", err)
	}
	if note == "" {
		t.Fatal("expected non-empty machine-readable note")
	}

	verification, ok := payload["verification"].(map[string]any)
	if !ok {
		t.Fatal("missing verification section")
	}
	ones, ok := verification["oneshot"].(map[string]any)
	if !ok {
		t.Fatal("missing verification.oneshot section")
	}
	if _, ok := ones["report"]; !ok {
		t.Fatal("missing oneshot report")
	}
	if _, ok := runPacket["oneshot_verification"]; !ok {
		t.Fatal("missing run packet oneshot_verification section")
	}
}

func TestLoadEvidencePayload(t *testing.T) {
	dir := t.TempDir()
	evPath := filepath.Join(dir, "evidence.json")
	validPayload := `{"intent":{"issue_id":"x"},"execution":{}}`
	if err := os.WriteFile(evPath, []byte(validPayload), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := loadEvidencePayload(evPath)
	if err != nil {
		t.Fatalf("loadEvidencePayload: %v", err)
	}
	if _, ok := got["intent"].(map[string]any); !ok {
		t.Fatalf("expected intent map, got %T", got["intent"])
	}

	_, err = loadEvidencePayload(filepath.Join(dir, "nonexistent.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}

	badPath := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(badPath, []byte("not json"), 0o644)
	_, err = loadEvidencePayload(badPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadRunPacket(t *testing.T) {
	dir := t.TempDir()
	runPath := filepath.Join(dir, "run.json")
	validPayload := `{"issue_id":"x","status":"in_progress"}`
	if err := os.WriteFile(runPath, []byte(validPayload), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := loadRunPacket(runPath)
	if err != nil {
		t.Fatalf("loadRunPacket: %v", err)
	}
	if got["issue_id"] != "x" {
		t.Fatalf("unexpected run packet: %+v", got)
	}

	_, err = loadRunPacket(filepath.Join(dir, "nonexistent.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestWriteEvidencePayloadAndWriteRunPacket(t *testing.T) {
	dir := t.TempDir()
	evPath := filepath.Join(dir, "evidence.json")
	payload := map[string]any{"intent": map[string]any{"issue_id": "test"}}
	if err := writeEvidencePayload(evPath, payload); err != nil {
		t.Fatalf("writeEvidencePayload: %v", err)
	}
	b, _ := os.ReadFile(evPath)
	var decoded map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("written file not valid JSON: %v", err)
	}
	if decoded["intent"].(map[string]any)["issue_id"] != "test" {
		t.Fatalf("unexpected written payload: %+v", decoded)
	}

	runPath := filepath.Join(dir, "run.json")
	runPacket := map[string]any{"status": "in_progress"}
	if err := writeRunPacket(runPath, runPacket); err != nil {
		t.Fatalf("writeRunPacket: %v", err)
	}
	if _, err := os.Stat(runPath); err != nil {
		t.Fatalf("run packet file not created: %v", err)
	}
}

func TestGetOrCreateMap(t *testing.T) {
	parent := map[string]any{}
	m := getOrCreateMap(parent, "foo")
	if m == nil {
		t.Fatal("getOrCreateMap returned nil")
	}
	m["k"] = "v"
	if parent["foo"].(map[string]any)["k"] != "v" {
		t.Fatalf("getOrCreateMap did not store in parent: %+v", parent)
	}
	m2 := getOrCreateMap(parent, "foo")
	if m2["k"] != "v" {
		t.Fatalf("getOrCreateMap did not return existing map: %+v", m2)
	}
}

func TestMergeEvidenceIntent(t *testing.T) {
	payload := map[string]any{"intent": map[string]any{}}
	mergeEvidenceIntent(payload)
	if len(payload["intent"].(map[string]any)) != 0 {
		t.Fatalf("mergeEvidenceIntent should not add when lastBuilderResult is nil")
	}
	lastBuilderResult = &llm.ExecuteResult{Prompt: "test prompt", ModelUsed: "glm-4.7"}
	defer func() { lastBuilderResult = nil }()
	mergeEvidenceIntent(payload)
	intent := payload["intent"].(map[string]any)
	if intent["llm_prompt"] != "test prompt" {
		t.Fatalf("mergeEvidenceIntent: got %v", intent)
	}
}

func TestMergeEvidenceExecution(t *testing.T) {
	payload := map[string]any{}
	mergeEvidenceExecution(payload, "issue-1", "feat/issue-1", []string{"internal/foo.go"})
	exec := payload["execution"].(map[string]any)
	if exec["branch"] != "feat/issue-1" || exec["claimed_issue_ids"].([]string)[0] != "issue-1" {
		t.Fatalf("mergeEvidenceExecution: %+v", exec)
	}
	if changed := exec["changed_files"].([]string); len(changed) != 1 || changed[0] != "internal/foo.go" {
		t.Fatalf("mergeEvidenceExecution changed_files: %v", changed)
	}
}

func TestMergeEvidenceTrace(t *testing.T) {
	payload := map[string]any{}
	mergeEvidenceTrace(payload, "issue-1", "feat/issue-1")
	trace := payload["trace"].(map[string]any)
	if trace["branch"] != "feat/issue-1" || trace["beads_ids"].([]string)[0] != "issue-1" {
		t.Fatalf("mergeEvidenceTrace: %+v", trace)
	}
}

func TestMergeEvidenceVerification(t *testing.T) {
	payload := map[string]any{}
	mergeEvidenceVerification(payload, true)
	verif := payload["verification"].(map[string]any)
	if verif["go_test_passed"] != true {
		t.Fatalf("mergeEvidenceVerification: %+v", verif)
	}
	mergeEvidenceVerification(payload, false)
	if verif["go_test_passed"] != false {
		t.Fatalf("mergeEvidenceVerification false: %+v", verif)
	}
}

func TestComputeOutOfBoundaryPaths(t *testing.T) {
	payload := map[string]any{
		"boundary": map[string]any{
			"declared": map[string]any{
				"allowed_path_prefixes": []string{"internal/", "cmd/"},
				"control_path_prefixes":  []string{".sdp/"},
				"forbidden_path_prefixes": []string{".git/"},
			},
		},
	}
	changed := []string{"internal/foo.go", ".git/config", "docs/readme.md"}
	got := computeOutOfBoundaryPaths(payload, changed)
	if len(got) != 2 {
		t.Fatalf("expected 2 out of boundary, got %v", got)
	}
	if got[0] != ".git/config" || got[1] != "docs/readme.md" {
		t.Fatalf("computeOutOfBoundaryPaths: %v", got)
	}

	got2 := computeOutOfBoundaryPaths(payload, []string{"internal/foo.go", ".sdp/evidence/x.json"})
	if len(got2) != 0 {
		t.Fatalf("control paths should be skipped: %v", got2)
	}
}

func TestMergeEvidenceBoundary(t *testing.T) {
	payload := map[string]any{}
	mergeEvidenceBoundary(payload, []string{"a.go"}, []string{"b.go"})
	boundary := payload["boundary"].(map[string]any)
	observed := boundary["observed"].(map[string]any)
	if observed["touched_paths"].([]string)[0] != "a.go" {
		t.Fatalf("mergeEvidenceBoundary touched: %v", observed["touched_paths"])
	}
	if observed["out_of_boundary_paths"].([]string)[0] != "b.go" {
		t.Fatalf("mergeEvidenceBoundary out: %v", observed["out_of_boundary_paths"])
	}
	compliance := boundary["compliance"].(map[string]any)
	if compliance["ok"] != false || compliance["reason"] != "changed paths exceed declared boundary" {
		t.Fatalf("mergeEvidenceBoundary compliance: %+v", compliance)
	}

	payload2 := map[string]any{}
	mergeEvidenceBoundary(payload2, []string{"internal/x.go"}, nil)
	comp2 := payload2["boundary"].(map[string]any)["compliance"].(map[string]any)
	if comp2["ok"] != true {
		t.Fatalf("mergeEvidenceBoundary ok when no out of boundary: %+v", comp2)
	}
}

func TestMergeEvidenceProvenance(t *testing.T) {
	payload := map[string]any{}
	mergeEvidenceProvenance(payload, "issue-1", "generic", true)
	prov := payload["provenance"].(map[string]any)
	if prov["orchestrator"] != "swarm-worker" || prov["phase"] != "verify" || prov["source_issue_id"] != "issue-1" {
		t.Fatalf("mergeEvidenceProvenance: %+v", prov)
	}
}

func TestSetProvenanceDefaultString(t *testing.T) {
	prov := map[string]any{}
	setProvenanceDefaultString(prov, "artifact_id", "default")
	if prov["artifact_id"] != "default" {
		t.Fatalf("setProvenanceDefaultString: %v", prov)
	}
	setProvenanceDefaultString(prov, "artifact_id", "other")
	if prov["artifact_id"] != "default" {
		t.Fatalf("setProvenanceDefaultString should not overwrite: %v", prov)
	}
}

func TestApplyEvaluatorRecommendationWorkstream(t *testing.T) {
	dir := t.TempDir()
	got := applyEvaluatorRecommendationWorkstream(dir, "issue-1", issueDetail{ID: "issue-1", Title: "Test Title"})
	if len(got) != 1 {
		t.Fatalf("expected 1 changed file, got %v", got)
	}
	path := filepath.Join(dir, "docs", "EVALUATOR_RECOMMENDATIONS_LOG.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	content := string(b)
	if content != "# Evaluator Recommendations Log\n\n## issue-1\n- Test Title\n" {
		t.Fatalf("unexpected content: %q", content)
	}
	got2 := applyEvaluatorRecommendationWorkstream(dir, "issue-2", issueDetail{ID: "issue-2", Title: "Second"})
	if len(got2) != 1 {
		t.Fatalf("expected 1 on append, got %v", got2)
	}
	b2, _ := os.ReadFile(path)
	if !strings.Contains(string(b2), "## issue-2") {
		t.Fatalf("append did not add second entry: %q", string(b2))
	}
}

func TestApplySelfImprovementWorkstream(t *testing.T) {
	dir := t.TempDir()
	got := applySelfImprovementWorkstream(dir, "issue-1", issueDetail{ID: "issue-1", Title: "T", SpecID: "spec-1"})
	if len(got) != 1 {
		t.Fatalf("expected 1 changed file, got %v", got)
	}
	path := filepath.Join(dir, "docs", "SELF_IMPROVEMENT_LOG.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	content := string(b)
	if content != "# Self-Improvement Log\n\n## issue-1\n- T\n- spec: spec-1\n" {
		t.Fatalf("unexpected content: %q", content)
	}
}

func TestApplyWorkstreamFlowHandoffValidation(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "AGENT_HANDOFF.md"), []byte("# Agent Handoff\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	got := applyWorkstreamFlow("handoff-validation", "issue-1", issueDetail{ID: "issue-1", Title: "T"}, "glm-5")
	if len(got) != 1 || got[0] != "docs/AGENT_HANDOFF.md" {
		t.Fatalf("applyWorkstreamFlow handoff-validation: %v", got)
	}
	b, _ := os.ReadFile("docs/AGENT_HANDOFF.md")
	if !strings.Contains(string(b), "## Validation Run") {
		t.Fatalf("handoff timestamp not appended: %q", string(b))
	}
}

func TestApplyWorkstreamFlowEvaluatorRecommendation(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	got := applyWorkstreamFlow("evaluator-recommendation", "issue-1", issueDetail{ID: "issue-1", Title: "Eval"}, "glm-5")
	if len(got) != 1 || !strings.HasSuffix(got[0], "EVALUATOR_RECOMMENDATIONS_LOG.md") {
		t.Fatalf("applyWorkstreamFlow evaluator-recommendation: %v", got)
	}
}

func TestWritePRBody(t *testing.T) {
	dir := t.TempDir()
	sdpDir := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(sdpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	path := writePRBody("issue-1", "handoff-validation")
	if path != ".sdp/pr-body-issue-1.md" {
		t.Fatalf("writePRBody path: %s", path)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "issue-1") || !strings.Contains(string(b), "handoff-validation") {
		t.Fatalf("writePRBody content: %q", string(b))
	}
}

func TestApplyWorkstreamFlowSelfImprovement(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	got := applyWorkstreamFlow("self-improvement", "issue-1", issueDetail{ID: "issue-1", Title: "SI", SpecID: "spec-1"}, "glm-5")
	if len(got) != 1 || !strings.HasSuffix(got[0], "SELF_IMPROVEMENT_LOG.md") {
		t.Fatalf("applyWorkstreamFlow self-improvement: %v", got)
	}
}

func TestAppendHandoffValidationTimestamp(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	handoffPath := filepath.Join(docsDir, "AGENT_HANDOFF.md")
	initial := "# Agent Handoff\n\nUpdated: 2026-02-21\n"
	if err := os.WriteFile(handoffPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := appendHandoffValidationTimestamp(dir); err != nil {
		t.Fatalf("appendHandoffValidationTimestamp: %v", err)
	}
	b, _ := os.ReadFile(handoffPath)
	content := string(b)
	if content == initial {
		t.Fatal("appendHandoffValidationTimestamp did not append")
	}
	if !strings.Contains(content, "## Validation Run") || !strings.Contains(content, "workstream:handoff-validation") {
		t.Fatalf("unexpected appended content: %q", content)
	}
}

func TestUpdateEvidence(t *testing.T) {
	dir := t.TempDir()
	sdpDir := filepath.Join(dir, ".sdp")
	evDir := filepath.Join(sdpDir, "evidence")
	runDir := filepath.Join(sdpDir, "runs")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	evPath := filepath.Join(evDir, "issue-1.json")
	initialPayload := map[string]any{
		"intent": map[string]any{"issue_id": "issue-1"},
		"boundary": map[string]any{
			"declared": map[string]any{
				"allowed_path_prefixes": []string{"internal/", "cmd/"},
				"control_path_prefixes":  []string{".sdp/"},
				"forbidden_path_prefixes": []string{".git/"},
			},
		},
	}
	b, _ := json.MarshalIndent(initialPayload, "", "  ")
	if err := os.WriteFile(evPath, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	note, err := updateEvidence("issue-1", "feat/issue-1", "generic", []string{"internal/foo.go"}, true)
	if err != nil {
		t.Fatalf("updateEvidence: %v", err)
	}
	if note != "" {
		t.Fatalf("updateEvidence note for generic workstream should be empty: %q", note)
	}

	reloaded, _ := loadEvidencePayload(".sdp/evidence/issue-1.json")
	exec := reloaded["execution"].(map[string]any)
	if exec["branch"] != "feat/issue-1" {
		t.Fatalf("updateEvidence did not merge execution: %+v", exec)
	}
	verif := reloaded["verification"].(map[string]any)
	if verif["go_test_passed"] != true {
		t.Fatalf("updateEvidence verification: %+v", verif)
	}
}

func TestUpdateEvidenceWithLastBuilderResult(t *testing.T) {
	dir := t.TempDir()
	sdpDir := filepath.Join(dir, ".sdp")
	evDir := filepath.Join(sdpDir, "evidence")
	runDir := filepath.Join(sdpDir, "runs")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	evPath := filepath.Join(evDir, "issue-2.json")
	initialPayload := map[string]any{
		"intent": map[string]any{"issue_id": "issue-2"},
		"boundary": map[string]any{
			"declared": map[string]any{
				"allowed_path_prefixes": []string{"internal/"},
				"control_path_prefixes":  []string{},
				"forbidden_path_prefixes": []string{},
			},
		},
	}
	b, _ := json.MarshalIndent(initialPayload, "", "  ")
	if err := os.WriteFile(evPath, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	lastBuilderResult = &llm.ExecuteResult{
		Prompt:     "test prompt",
		ModelUsed:  "glm-4.7",
		Duration:   100 * time.Millisecond,
		SessionID:  "sess-123",
	}
	defer func() { lastBuilderResult = nil }()

	_, err := updateEvidence("issue-2", "feat/issue-2", "generic", []string{"internal/bar.go"}, false)
	if err != nil {
		t.Fatalf("updateEvidence with lastBuilderResult: %v", err)
	}

	reloaded, _ := loadEvidencePayload(".sdp/evidence/issue-2.json")
	exec := reloaded["execution"].(map[string]any)
	if exec["model"] != "glm-4.7" {
		t.Fatalf("updateEvidence execution model: %v", exec["model"])
	}
	if exec["opencode_session_id"] != "sess-123" {
		t.Fatalf("updateEvidence execution session: %v", exec["opencode_session_id"])
	}
	prov := reloaded["provenance"].(map[string]any)
	if prov["model"] != "glm-4.7" {
		t.Fatalf("updateEvidence provenance model: %v", prov["model"])
	}
}

func TestUpdateEvidenceMissingFile(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	_, err := updateEvidence("nonexistent", "feat/x", "generic", nil, true)
	if err == nil {
		t.Fatal("expected error when evidence file missing")
	}
}

func TestUpdateEvidenceOneshotWorkstream(t *testing.T) {
	dir := t.TempDir()
	sdpDir := filepath.Join(dir, ".sdp")
	evDir := filepath.Join(sdpDir, "evidence")
	runDir := filepath.Join(sdpDir, "runs")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	evPath := filepath.Join(evDir, "issue-oneshot.json")
	initialPayload := map[string]any{
		"intent": map[string]any{"issue_id": "issue-oneshot"},
		"boundary": map[string]any{
			"declared": map[string]any{
				"allowed_path_prefixes": []string{"internal/", "cmd/"},
				"control_path_prefixes":  []string{".sdp/"},
				"forbidden_path_prefixes": []string{".git/"},
			},
		},
	}
	b, _ := json.MarshalIndent(initialPayload, "", "  ")
	if err := os.WriteFile(evPath, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// oneshot workstream with test change and tests passed - produces note
	note, err := updateEvidence("issue-oneshot", "feat/issue-oneshot", "oneshot-swarm-orchestrator", []string{"internal/oneshot/manifest.go", "internal/oneshot/manifest_test.go"}, true)
	if err != nil {
		t.Fatalf("updateEvidence oneshot: %v", err)
	}
	if note == "" {
		t.Fatal("expected non-empty note for oneshot workstream")
	}

	reloaded, _ := loadEvidencePayload(".sdp/evidence/issue-oneshot.json")
	verif := reloaded["verification"].(map[string]any)
	if verif["oneshot"] == nil {
		t.Fatalf("expected oneshot verification: %+v", verif)
	}

	// With run packet - exercises runPacket path in applyOneShotVerification
	runPath := filepath.Join(runDir, "issue-oneshot.json")
	runPayload := map[string]any{"run_id": "run-1", "phase": "verify"}
	runB, _ := json.MarshalIndent(runPayload, "", "  ")
	if err := os.WriteFile(runPath, append(runB, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	// Reload payload for second call (first call already wrote)
	payload2, _ := loadEvidencePayload(".sdp/evidence/issue-oneshot.json")
	runPkt, _ := loadRunPacket(".sdp/runs/issue-oneshot.json")
	_, err = applyOneShotVerification(payload2, runPkt, []string{"internal/oneshot/manifest_test.go"}, true)
	if err != nil {
		t.Fatalf("applyOneShotVerification with runPacket: %v", err)
	}
	if runPkt["oneshot_verification"] == nil {
		t.Fatalf("expected runPacket oneshot_verification: %+v", runPkt)
	}
}

func TestEvaluateOneShotVerificationNeedsChanges(t *testing.T) {
	// testsPassed=false or no test change -> buildStatus=needs_changes
	result, err := evaluateOneShotVerification([]string{"internal/foo.go"}, false)
	if err != nil {
		t.Fatalf("evaluateOneShotVerification: %v", err)
	}
	if result.Report.OK {
		t.Fatal("expected report not OK when tests failed")
	}
	if result.RecoveryPlan == nil {
		t.Fatal("expected recovery plan when verification fails")
	}
}

func TestUpdateEvidenceWithRunPacket(t *testing.T) {
	dir := t.TempDir()
	sdpDir := filepath.Join(dir, ".sdp")
	evDir := filepath.Join(sdpDir, "evidence")
	runDir := filepath.Join(sdpDir, "runs")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	evPath := filepath.Join(evDir, "issue-3.json")
	runPath := filepath.Join(runDir, "issue-3.json")
	initialPayload := map[string]any{
		"intent": map[string]any{"issue_id": "issue-3"},
		"boundary": map[string]any{
			"declared": map[string]any{
				"allowed_path_prefixes": []string{"internal/"},
				"control_path_prefixes":  []string{},
				"forbidden_path_prefixes": []string{},
			},
		},
	}
	b, _ := json.MarshalIndent(initialPayload, "", "  ")
	if err := os.WriteFile(evPath, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	runPayload := map[string]any{"run_id": "run-1", "phase": "verify"}
	runB, _ := json.MarshalIndent(runPayload, "", "  ")
	if err := os.WriteFile(runPath, append(runB, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	_, err := updateEvidence("issue-3", "feat/issue-3", "generic", []string{"internal/baz.go"}, true)
	if err != nil {
		t.Fatalf("updateEvidence with runPacket: %v", err)
	}

	runReloaded, _ := loadRunPacket(".sdp/runs/issue-3.json")
	if runReloaded == nil || runReloaded["run_id"] != "run-1" {
		t.Fatalf("updateEvidence should preserve run packet: %+v", runReloaded)
	}
}

func TestLoadIssue(t *testing.T) {
	orig := runFunc
	defer func() { runFunc = orig }()
	runFunc = func(name string, args ...string) ([]byte, error) {
		if name == "bd" && args[0] == "show" {
			return []byte(`{"id":"issue-1","title":"T","spec_id":"spec-1"}`), nil
		}
		return nil, nil
	}
	it, err := loadIssue("issue-1")
	if err != nil {
		t.Fatalf("loadIssue: %v", err)
	}
	if it.ID != "issue-1" || it.SpecID != "spec-1" {
		t.Fatalf("loadIssue: %+v", it)
	}
}

func TestLoadIssueAsList(t *testing.T) {
	orig := runFunc
	defer func() { runFunc = orig }()
	runFunc = func(name string, args ...string) ([]byte, error) {
		return []byte(`[{"id":"issue-2","title":"U","spec_id":"spec-2"}]`), nil
	}
	it, err := loadIssue("issue-2")
	if err != nil {
		t.Fatalf("loadIssue: %v", err)
	}
	if it.ID != "issue-2" {
		t.Fatalf("loadIssue list: %+v", it)
	}
}

func TestExtractJSON(t *testing.T) {
	noise := []byte("some output\n{\"x\":1}")
	got := extractJSON(noise)
	if string(got) != `{"x":1}` {
		t.Fatalf("extractJSON: %q", got)
	}
	arr := []byte("[1,2,3]")
	got2 := extractJSON(arr)
	if string(got2) != "[1,2,3]" {
		t.Fatalf("extractJSON array: %q", got2)
	}
	noJson := []byte("no json")
	got3 := extractJSON(noJson)
	// extractJSON returns from first [ or { to end; if none, returns full slice
	if string(got3) != "no json" {
		t.Fatalf("extractJSON no json: %q", got3)
	}
}

func TestRunComponent(t *testing.T) {
	orig := runFunc
	defer func() { runFunc = orig }()
	runFunc = func(name string, args ...string) ([]byte, error) {
		if name == "true" {
			return []byte("ok"), nil
		}
		return nil, nil
	}
	// "true" exists on Unix; runComponent uses it when in PATH
	out, err := runComponent("true", "fmt")
	if err != nil {
		t.Fatalf("runComponent: %v", err)
	}
	if string(out) != "ok" {
		t.Fatalf("runComponent output: %q", out)
	}
}

func TestRunComponentWithFallbackUsesFallbackWhenBinaryMissing(t *testing.T) {
	orig := runFunc
	defer func() { runFunc = orig }()
	runFunc = func(name string, args ...string) ([]byte, error) {
		if name == "go" {
			return []byte("mock output"), nil
		}
		return nil, nil
	}
	out, usedFallback, err := runComponentWithFallback("nonexistentbinaryxyz123", "fmt", "arg")
	if err != nil {
		t.Fatalf("runComponentWithFallback: %v", err)
	}
	if !usedFallback {
		t.Fatal("expected fallback to be used when binary missing")
	}
	if string(out) != "mock output" {
		t.Fatalf("runComponentWithFallback output: %q", out)
	}
}

func TestBuildWorkerObservabilityRecordsValidatorCompatible(t *testing.T) {
	records := buildWorkerObservabilityRecords(
		"sdp_dev-2aq.20.2",
		"verify",
		"fallback",
		"glm-4.7",
		1,
		true,
		false,
		".sdp/evidence/sdp_dev-2aq.20.2.json",
		"https://example.invalid/org/repo/pull/50",
		230*time.Millisecond,
	)
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	event, ok := records[0]["event"].(map[string]any)
	if !ok {
		t.Fatalf("missing event payload: %#v", records[0])
	}
	if errs := observability.ValidateUnifiedMetricsTraceEvent(event); len(errs) != 0 {
		t.Fatalf("event failed schema validation: %v", errs)
	}
	resilience, _ := event["resilience"].(map[string]any)
	if fallback, _ := resilience["fallback_used"].(bool); !fallback {
		t.Fatal("expected fallback_used=true")
	}
}
