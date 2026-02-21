package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sdp_dev/internal/policy"
)

func TestWriteJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "data.json")
	payload := map[string]any{"key": "value"}
	if err := writeJSON(path, payload); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if decoded["key"] != "value" {
		t.Fatalf("writeJSON: %+v", decoded)
	}
}

func TestLoadEvidenceTemplate(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tmpl := map[string]any{
		"intent":    map[string]any{"issue_id": ""},
		"execution": map[string]any{},
		"boundary":  map[string]any{},
		"provenance": map[string]any{},
		"trace":     map[string]any{},
	}
	b, _ := json.Marshal(tmpl)
	if err := os.WriteFile(filepath.Join(specsDir, "strict-evidence-template.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := loadEvidenceTemplate(dir)
	if err != nil {
		t.Fatalf("loadEvidenceTemplate: %v", err)
	}
	if _, ok := got["intent"].(map[string]any); !ok {
		t.Fatalf("loadEvidenceTemplate: %+v", got)
	}

	_, err = loadEvidenceTemplate(filepath.Join(dir, "nonexistent"))
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestPopulateEvidence(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	evDir := filepath.Join(dir, ".sdp", "evidence")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tmpl := map[string]any{
		"intent":     map[string]any{"issue_id": ""},
		"execution":  map[string]any{},
		"boundary":   map[string]any{"declared": map[string]any{}, "observed": map[string]any{}, "compliance": map[string]any{}},
		"provenance": map[string]any{},
		"trace":      map[string]any{},
	}
	b, _ := json.Marshal(tmpl)
	if err := os.WriteFile(filepath.Join(specsDir, "strict-evidence-template.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}

	picked := &issue{ID: "issue-1", Title: "T", Labels: []string{"workstream:generic"}}
	decision := policy.DecisionResponse{
		RiskClass:     "low",
		SelectedModel: "glm-5",
		Lane:          "commit",
	}
	path, err := populateEvidence(dir, picked, "feat/issue-1", decision)
	if err != nil {
		t.Fatalf("populateEvidence: %v", err)
	}
	if path != filepath.Join(dir, ".sdp", "evidence", "issue-1.json") {
		t.Fatalf("populateEvidence path: %s", path)
	}
	data, _ := os.ReadFile(path)
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	intent := doc["intent"].(map[string]any)
	if intent["issue_id"] != "issue-1" || intent["risk_class"] != "low" {
		t.Fatalf("populateEvidence intent: %+v", intent)
	}
	exec := doc["execution"].(map[string]any)
	if exec["branch"] != "feat/issue-1" {
		t.Fatalf("populateEvidence execution: %+v", exec)
	}
	prov := doc["provenance"].(map[string]any)
	if prov["orchestrator"] != "autonomy-worker" || prov["model"] != "glm-5" {
		t.Fatalf("populateEvidence provenance: %+v", prov)
	}
}

func TestHasLabel(t *testing.T) {
	tests := []struct {
		labels []string
		name   string
		want   bool
	}{
		{[]string{"autonomy", "strict-evidence"}, "autonomy", true},
		{[]string{"autonomy", "strict-evidence"}, "strict-evidence", true},
		{[]string{"autonomy"}, "strict-evidence", false},
		{[]string{}, "autonomy", false},
		{[]string{"workstream:generic"}, "workstream:generic", true},
	}
	for _, tt := range tests {
		got := hasLabel(tt.labels, tt.name)
		if got != tt.want {
			t.Errorf("hasLabel(%v, %q) = %v, want %v", tt.labels, tt.name, got, tt.want)
		}
	}
}

func TestHasWorkstreamLabel(t *testing.T) {
	// supportedWorkstreams default includes workstream:generic
	tests := []struct {
		labels []string
		want   bool
	}{
		{[]string{"workstream:generic"}, true},
		{[]string{"workstream:policy-slugify-trim"}, true},
		{[]string{"autonomy"}, false},
		{[]string{}, false},
	}
	for _, tt := range tests {
		got := hasWorkstreamLabel(tt.labels)
		if got != tt.want {
			t.Errorf("hasWorkstreamLabel(%v) = %v, want %v", tt.labels, got, tt.want)
		}
	}
}

func TestLaneFromLabels(t *testing.T) {
	tests := []struct {
		labels []string
		want   string
	}{
		{[]string{"lane:commit"}, "commit"},
		{[]string{"lane:explore"}, "explore"},
		{[]string{"lane:other"}, "commit"}, // invalid fallback
		{[]string{}, "commit"},
		{[]string{"autonomy", "lane:explore"}, "explore"},
	}
	for _, tt := range tests {
		got := laneFromLabels(tt.labels)
		if got != tt.want {
			t.Errorf("laneFromLabels(%v) = %q, want %q", tt.labels, got, tt.want)
		}
	}
}

func TestAllowedPrefixesFromLabels(t *testing.T) {
	restricted := []string{"internal/policy/", "internal/evidence/", "cmd/", "docs/", "specs/", "scripts/"}
	forbidden := []string{"internal/", "cmd/", "docs/", "specs/", "scripts/", "deploy/"}
	tests := []struct {
		labels []string
		want   []string
	}{
		{[]string{"workstream:policy-slugify-trim"}, restricted},
		{[]string{"workstream:generic"}, forbidden},
		{[]string{"workstream:self-improvement"}, forbidden},
		{[]string{}, forbidden},
	}
	for _, tt := range tests {
		got := allowedPrefixesFromLabels(tt.labels)
		if len(got) != len(tt.want) {
			t.Errorf("allowedPrefixesFromLabels(%v) = %v, want %v", tt.labels, got, tt.want)
		}
	}
}

func TestDepsSatisfied(t *testing.T) {
	closed := issue{ID: "dep-1", Status: "closed"}
	open := issue{ID: "dep-2", Status: "open"}
	byID := map[string]issue{"dep-1": closed, "dep-2": open}

	tests := []struct {
		name string
		it   issue
		want bool
	}{
		{"no deps", issue{ID: "x", Dependencies: []dep{}}, true},
		{"parent-child skip", issue{ID: "x", Dependencies: []dep{{Type: "parent-child"}}}, true},
		{"dep closed", issue{ID: "x", Dependencies: []dep{{DependsOnID: "dep-1"}}}, true},
		{"dep open", issue{ID: "x", Dependencies: []dep{{DependsOnID: "dep-2"}}}, false},
		{"dep missing", issue{ID: "x", Dependencies: []dep{{DependsOnID: "missing"}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := depsSatisfied(tt.it, byID)
			if got != tt.want {
				t.Errorf("depsSatisfied() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelFromLabels(t *testing.T) {
	tests := []struct {
		labels []string
		want   string
		wantErr bool
	}{
		{[]string{}, "glm-5", false}, // default
		{[]string{"model:glm-5"}, "glm-5", false},
		{[]string{"model:glm-4.7"}, "glm-4.7", false},
	}
	for _, tt := range tests {
		got, err := modelFromLabels(tt.labels)
		if (err != nil) != tt.wantErr {
			t.Errorf("modelFromLabels(%v) err = %v, wantErr %v", tt.labels, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("modelFromLabels(%v) = %q, want %q", tt.labels, got, tt.want)
		}
	}
}

func TestEmitAutonomyObservabilityNoPanic(t *testing.T) {
	// Smoke test: emit should not panic
	emitAutonomyObservability("sdp_dev-4pg", "claim", "success", "glm-5", time.Now().Add(-time.Second))
	emitAutonomyObservability("", "plan", "blocked", "unknown", time.Now())
}

func TestListIssues(t *testing.T) {
	orig := bdRunner
	defer func() { bdRunner = orig }()
	bdRunner = func(args ...string) ([]byte, error) {
		if args[0] == "list" {
			return []byte(`[{"id":"issue-1","title":"T","status":"open"},{"id":"issue-2","title":"U","status":"closed"}]`), nil
		}
		return nil, nil
	}
	byID, err := listIssues()
	if err != nil {
		t.Fatalf("listIssues: %v", err)
	}
	if len(byID) != 2 || byID["issue-1"].Title != "T" {
		t.Fatalf("listIssues: %+v", byID)
	}
}

func TestListIssuesWithNoise(t *testing.T) {
	orig := bdRunner
	defer func() { bdRunner = orig }()
	bdRunner = func(args ...string) ([]byte, error) {
		return []byte("some stderr\n[{\"id\":\"x\",\"title\":\"X\",\"status\":\"open\"}]"), nil
	}
	byID, err := listIssues()
	if err != nil {
		t.Fatalf("listIssues: %v", err)
	}
	if len(byID) != 1 || byID["x"].Title != "X" {
		t.Fatalf("listIssues with noise: %+v", byID)
	}
}

func TestLoadIssueDetail(t *testing.T) {
	orig := bdRunner
	defer func() { bdRunner = orig }()
	bdRunner = func(args ...string) ([]byte, error) {
		if args[0] == "show" {
			return []byte(`{"id":"issue-1","title":"Test","spec_id":"spec","labels":["autonomy","workstream:generic"]}`), nil
		}
		return nil, nil
	}
	it, err := loadIssueDetail("issue-1")
	if err != nil {
		t.Fatalf("loadIssueDetail: %v", err)
	}
	if it.ID != "issue-1" || it.Title != "Test" {
		t.Fatalf("loadIssueDetail: %+v", it)
	}
}

func TestLoadIssueDetailAsList(t *testing.T) {
	orig := bdRunner
	defer func() { bdRunner = orig }()
	bdRunner = func(args ...string) ([]byte, error) {
		return []byte(`[{"id":"issue-2","title":"FromList"}]`), nil
	}
	it, err := loadIssueDetail("issue-2")
	if err != nil {
		t.Fatalf("loadIssueDetail: %v", err)
	}
	if it.ID != "issue-2" || it.Title != "FromList" {
		t.Fatalf("loadIssueDetail list format: %+v", it)
	}
}

func TestPickCandidate(t *testing.T) {
	orig := bdRunner
	defer func() { bdRunner = orig }()
	callCount := 0
	bdRunner = func(args ...string) ([]byte, error) {
		callCount++
		if args[0] == "list" {
			return []byte(`[{"id":"task-1","title":"T","status":"open","issue_type":"task","labels":["autonomy","strict-evidence","workstream:generic"],"priority":1,"created_at":"2026-01-01"}]`), nil
		}
		if args[0] == "show" {
			return []byte(`{"id":"task-1","title":"T","status":"open","issue_type":"task","labels":["autonomy","strict-evidence","workstream:generic"],"dependencies":[],"priority":1,"created_at":"2026-01-01"}`), nil
		}
		return nil, nil
	}
	byID, _ := listIssues()
	picked, err := pickCandidate(byID, false)
	if err != nil {
		t.Fatalf("pickCandidate: %v", err)
	}
	if picked == nil || picked.ID != "task-1" {
		t.Fatalf("pickCandidate: %+v", picked)
	}
	if callCount < 2 {
		t.Fatalf("pickCandidate should call bd show: %d", callCount)
	}
}

func TestAppendNote(t *testing.T) {
	orig := bdRunner
	defer func() { bdRunner = orig }()
	called := false
	bdRunner = func(args ...string) ([]byte, error) {
		called = true
		if args[0] != "update" || args[1] != "issue-1" {
			return nil, nil
		}
		return []byte("ok"), nil
	}
	if err := appendNote("issue-1", "test note"); err != nil {
		t.Fatalf("appendNote: %v", err)
	}
	if !called {
		t.Fatal("appendNote did not call bdRunner")
	}
}

func TestPickCandidateReturnsNilWhenNoEligible(t *testing.T) {
	orig := bdRunner
	defer func() { bdRunner = orig }()
	bdRunner = func(args ...string) ([]byte, error) {
		if args[0] == "list" {
			return []byte(`[{"id":"x","status":"closed","issue_type":"task","labels":["autonomy","strict-evidence","workstream:generic"]}]`), nil
		}
		return nil, nil
	}
	byID, _ := listIssues()
	picked, _ := pickCandidate(byID, false)
	if picked != nil {
		t.Fatalf("pickCandidate should return nil when no open: %+v", picked)
	}
}

func TestPickCandidateSkipsNonTaskAndMissingLabels(t *testing.T) {
	orig := bdRunner
	defer func() { bdRunner = orig }()
	bdRunner = func(args ...string) ([]byte, error) {
		if args[0] == "list" {
			return []byte(`[
				{"id":"epic-1","status":"open","issue_type":"epic","labels":["autonomy","workstream:generic"]},
				{"id":"task-2","status":"open","issue_type":"task","labels":["workstream:generic"],"priority":1,"created_at":"2026-01-02"},
				{"id":"task-3","status":"open","issue_type":"task","labels":["autonomy","strict-evidence","workstream:generic"],"dependencies":[],"priority":2,"created_at":"2026-01-01"}
			]`), nil
		}
		if args[0] == "show" {
			return []byte(`{"id":"task-3","status":"open","issue_type":"task","labels":["autonomy","strict-evidence","workstream:generic"],"dependencies":[],"priority":2,"created_at":"2026-01-01"}`), nil
		}
		return nil, nil
	}
	byID, _ := listIssues()
	picked, _ := pickCandidate(byID, true)
	if picked == nil || picked.ID != "task-3" {
		t.Fatalf("pickCandidate should pick task-3 (only eligible): %+v", picked)
	}
}

func TestPickCandidateWithDepsNotSatisfied(t *testing.T) {
	orig := bdRunner
	defer func() { bdRunner = orig }()
	bdRunner = func(args ...string) ([]byte, error) {
		if args[0] == "list" {
			return []byte(`[{"id":"task-1","status":"open","issue_type":"task","labels":["autonomy","strict-evidence","workstream:generic"],"dependencies":[{"depends_on_id":"dep-1"}],"priority":1,"created_at":"2026-01-01"}]`), nil
		}
		if args[0] == "show" {
			return []byte(`{"id":"task-1","status":"open","issue_type":"task","labels":["autonomy","strict-evidence","workstream:generic"],"dependencies":[{"depends_on_id":"dep-1"}],"priority":1,"created_at":"2026-01-01"}`), nil
		}
		return nil, nil
	}
	byID, _ := listIssues()
	byID["dep-1"] = issue{ID: "dep-1", Status: "open"}
	picked, _ := pickCandidate(byID, false)
	if picked != nil {
		t.Fatalf("pickCandidate should return nil when dep not satisfied: %+v", picked)
	}
}

func TestLoadWorkstreamConfig(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `workstreams:
  - label: workstream:custom
    path_prefixes: [internal/]
`
	if err := os.WriteFile(filepath.Join(specsDir, "workstream-config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := supportedWorkstreams
	defer func() { supportedWorkstreams = orig }()
	loadWorkstreamConfig(dir)
	if len(supportedWorkstreams) != 1 || supportedWorkstreams[0] != "workstream:custom" {
		t.Fatalf("loadWorkstreamConfig: %v", supportedWorkstreams)
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"pure json", []byte(`{"id":"x"}`), `{"id":"x"}`},
		{"leading noise", []byte(`some output\n{"id":"x"}`), `{"id":"x"}`},
		{"array", []byte(`[{"id":"x"}]`), `[{"id":"x"}]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(extractJSON(tt.in))
			if got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}
