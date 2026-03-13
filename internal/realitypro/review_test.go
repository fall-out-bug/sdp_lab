package realitypro

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestReview_WritesConflictAndIntentGapArtifacts(t *testing.T) {
	projectRoot := t.TempDir()
	seedRealityRepo(t, projectRoot, true)
	submoduleRoot := filepath.Join(projectRoot, "sdp")
	seedProtocolRepo(t, submoduleRoot)
	writeFile(t, filepath.Join(projectRoot, "adr", "ADR-0001-boundary.md"), "# ADR\nCross-repo rollout uses staged contract promotion.\n")
	writeFile(t, filepath.Join(projectRoot, ".github", "CODEOWNERS"), "* @platform\n/internal/billing/ @payments\n")
	copySchemaFile(t, projectRoot, "conflicts-report.schema.json")
	copySchemaFile(t, projectRoot, "intent-gap-report.schema.json")

	if _, err := Ingest(Options{
		ProjectRoot: projectRoot,
		Repos:       []string{projectRoot, submoduleRoot},
		WithDocs:    true,
		Now: func() time.Time {
			return time.Date(2026, 3, 12, 9, 30, 0, 0, time.UTC)
		},
	}); err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}

	result, err := Review(ReviewOptions{
		ProjectRoot: projectRoot,
		Now: func() time.Time {
			return time.Date(2026, 3, 12, 9, 31, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("Review failed: %v", err)
	}
	if result.GapCount == 0 {
		t.Fatal("expected at least one intent gap")
	}
	if len(result.Specialists) == 0 {
		t.Fatal("expected selected specialists")
	}
	if !containsString(result.Specialists, "ownership-analyst") {
		t.Fatalf("expected ownership specialist, got %#v", result.Specialists)
	}

	conflicts := readConflicts(t, filepath.Join(projectRoot, ".sdp", "reality", "conflicts-report.json"))
	if len(conflicts.Conflicts) == 0 {
		t.Fatal("expected explicit conflict artifacts")
	}
	if !containsConflictNote(conflicts, "Synthesis reviewer kept the finding") {
		t.Fatalf("expected synthesis review note, got %#v", conflicts.Conflicts)
	}
	validateReviewSchema(t, projectRoot, "conflicts-report.schema.json", conflicts)

	intent := readIntentGaps(t, filepath.Join(projectRoot, ".sdp", "reality", "intent-gap-report.json"))
	if len(intent.Gaps) == 0 {
		t.Fatal("expected intent gap artifacts")
	}
	if !containsGapWithActions(intent) {
		t.Fatalf("expected recommended actions in intent gaps, got %#v", intent.Gaps)
	}
	if !containsGapID(intent, "gap:ownership-") {
		t.Fatalf("expected ownership-related intent gap, got %#v", intent.Gaps)
	}
	if !containsSourceKind(intent.Sources, "adr") {
		t.Fatalf("expected ingested ADR evidence in review sources, got %#v", intent.Sources)
	}
	validateReviewSchema(t, projectRoot, "intent-gap-report.schema.json", intent)

	memoryData, err := os.ReadFile(filepath.Join(projectRoot, ".sdp", "reality", "repo-memory.json"))
	if err != nil {
		t.Fatalf("read updated repo-memory: %v", err)
	}
	var memory RepoMemory
	if err := json.Unmarshal(memoryData, &memory); err != nil {
		t.Fatalf("parse updated repo-memory: %v", err)
	}
	if len(memory.PreviousValidatedClaimIDs) == 0 {
		t.Fatal("expected arbitrated claims to be written back into repo memory")
	}
	if !containsString(memory.PreviousValidatedClaimIDs, "finding:contract-boundary:final") {
		t.Fatalf("expected validated lineage to include arbitrated claim, got %#v", memory.PreviousValidatedClaimIDs)
	}
}

func TestReview_RequiresRepoMemory(t *testing.T) {
	projectRoot := t.TempDir()
	if _, err := Review(ReviewOptions{ProjectRoot: projectRoot}); err == nil {
		t.Fatal("expected review without repo memory to fail")
	}
}

func readConflicts(t *testing.T, path string) ConflictReport {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read conflicts: %v", err)
	}
	var report ConflictReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("parse conflicts: %v", err)
	}
	return report
}

func readIntentGaps(t *testing.T, path string) IntentGapReport {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read intent gaps: %v", err)
	}
	var report IntentGapReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("parse intent gaps: %v", err)
	}
	return report
}

func validateReviewSchema(t *testing.T, projectRoot, schemaName string, payload any) {
	t.Helper()
	compiler := jsonschema.NewCompiler()

	claimData, err := os.ReadFile(filepath.Join(projectRoot, "schema", "reality", "claim.schema.json"))
	if err != nil {
		t.Fatalf("read claim schema: %v", err)
	}
	sourceData, err := os.ReadFile(filepath.Join(projectRoot, "schema", "reality", "source.schema.json"))
	if err != nil {
		t.Fatalf("read source schema: %v", err)
	}
	baseName := strings.TrimSuffix(schemaName, ".schema.json")
	for _, alias := range []string{"claim.schema.json", "https://sdp.dev/reality/" + baseName + "/claim.schema.json"} {
		if err := compiler.AddResource(alias, bytes.NewReader(claimData)); err != nil {
			t.Fatalf("add claim alias %s: %v", alias, err)
		}
	}
	for _, alias := range []string{"source.schema.json", "https://sdp.dev/reality/" + baseName + "/source.schema.json"} {
		if err := compiler.AddResource(alias, bytes.NewReader(sourceData)); err != nil {
			t.Fatalf("add source alias %s: %v", alias, err)
		}
	}

	schemaPath := filepath.Join(projectRoot, "schema", "reality", schemaName)
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile %s: %v", schemaName, err)
	}
	normalized := normalizeJSONValue(t, payload)
	if err := schema.Validate(normalized); err != nil {
		t.Fatalf("validate %s: %v", schemaName, err)
	}
}

func containsConflictNote(report ConflictReport, fragment string) bool {
	for _, item := range report.Conflicts {
		if strings.Contains(item.ResolutionNotes, fragment) {
			return true
		}
	}
	return false
}

func containsGapWithActions(report IntentGapReport) bool {
	for _, item := range report.Gaps {
		if len(item.RecommendedActions) > 0 {
			return true
		}
	}
	return false
}

func containsGapID(report IntentGapReport, prefix string) bool {
	for _, item := range report.Gaps {
		if strings.HasPrefix(item.GapID, prefix) {
			return true
		}
	}
	return false
}
