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

func TestIngest_ExplicitReposetWritesMemoryAndMap(t *testing.T) {
	projectRoot := t.TempDir()
	seedRealityRepo(t, projectRoot, true)
	submoduleRoot := filepath.Join(projectRoot, "sdp")
	seedProtocolRepo(t, submoduleRoot)

	result, err := Ingest(Options{
		ProjectRoot: projectRoot,
		Repos:       []string{projectRoot, submoduleRoot},
		Now: func() time.Time {
			return time.Date(2026, 3, 12, 8, 45, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}
	if result.RepoCount != 2 {
		t.Fatalf("expected 2 repos, got %d", result.RepoCount)
	}

	memoryPath := filepath.Join(projectRoot, ".sdp", "reality", "repo-memory.json")
	data, err := os.ReadFile(memoryPath)
	if err != nil {
		t.Fatalf("read repo-memory: %v", err)
	}
	var memory RepoMemory
	if err := json.Unmarshal(data, &memory); err != nil {
		t.Fatalf("parse repo-memory: %v", err)
	}
	if len(memory.Repos) != 2 {
		t.Fatalf("expected 2 repos in memory, got %d", len(memory.Repos))
	}
	if got := memory.Repos[0].Role + "," + memory.Repos[1].Role; !strings.Contains(got, "protocol") || !strings.Contains(got, "service") {
		t.Fatalf("expected protocol and service roles, got %q", got)
	}
	if len(memory.Hotspots) == 0 {
		t.Fatal("expected hotspot history to be stored")
	}
	if len(memory.UnresolvedQuestions) == 0 {
		t.Fatal("expected unresolved questions to persist")
	}

	validateRepoMemorySchema(t, projectRoot, memory)

	mapData, err := os.ReadFile(filepath.Join(projectRoot, "docs", "reality", "multi-repo-map.md"))
	if err != nil {
		t.Fatalf("read multi-repo map: %v", err)
	}
	text := string(mapData)
	if !strings.Contains(text, "consumes contracts from") {
		t.Fatalf("expected protocol/service boundary in map, got:\n%s", text)
	}
	if !strings.Contains(text, "contains") {
		t.Fatalf("expected nested repo boundary in map, got:\n%s", text)
	}
}

func TestIngest_IncrementalRefreshPreservesLineage(t *testing.T) {
	projectRoot := t.TempDir()
	seedRealityRepo(t, projectRoot, false)

	existing := RepoMemory{
		SpecVersion:               specVersion,
		GeneratedAt:               "2026-03-11T00:00:00Z",
		Repos:                     []RepoRecord{{RepoID: "repo:" + sanitizeID(filepath.Base(projectRoot)), Name: filepath.Base(projectRoot), RootPath: filepath.ToSlash(projectRoot)}},
		ModuleSummaries:           []ModuleSummary{{ModuleID: "module:repo:" + sanitizeID(filepath.Base(projectRoot)) + ":internal", RepoID: "repo:" + sanitizeID(filepath.Base(projectRoot)), Summary: "existing"}},
		PreviousValidatedClaimIDs: []string{"claim:validated-1"},
		UnresolvedQuestions:       []string{"carry this question forward"},
		Hotspots: []HotspotRecord{
			{
				HotspotID: hotspotID("repo:"+sanitizeID(filepath.Base(projectRoot)), "internal/billing/service.go"),
				RepoID:    "repo:" + sanitizeID(filepath.Base(projectRoot)),
				Path:      "internal/billing/service.go",
				Reason:    "line_count=900",
				Severity:  "high",
			},
		},
	}
	if err := writeJSON(filepath.Join(projectRoot, ".sdp", "reality", "repo-memory.json"), existing); err != nil {
		t.Fatalf("seed existing memory: %v", err)
	}

	if _, err := Ingest(Options{
		ProjectRoot: projectRoot,
		Repos:       []string{projectRoot},
		Now: func() time.Time {
			return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
		},
	}); err != nil {
		t.Fatalf("Ingest refresh failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectRoot, ".sdp", "reality", "repo-memory.json"))
	if err != nil {
		t.Fatalf("read refreshed memory: %v", err)
	}
	var refreshed RepoMemory
	if err := json.Unmarshal(data, &refreshed); err != nil {
		t.Fatalf("parse refreshed memory: %v", err)
	}

	if len(refreshed.PreviousValidatedClaimIDs) != 1 || refreshed.PreviousValidatedClaimIDs[0] != "claim:validated-1" {
		t.Fatalf("expected validated claim lineage to persist, got %#v", refreshed.PreviousValidatedClaimIDs)
	}
	if !containsString(refreshed.UnresolvedQuestions, "carry this question forward") {
		t.Fatalf("expected unresolved question lineage to persist, got %#v", refreshed.UnresolvedQuestions)
	}
	if !containsHotspot(refreshed.Hotspots, hotspotID("repo:"+sanitizeID(filepath.Base(projectRoot)), "internal/billing/service.go")) {
		t.Fatalf("expected hotspot lineage to persist, got %#v", refreshed.Hotspots)
	}
}

func TestIngest_WithDocsStoresEvidenceSourcesAndMapCoverage(t *testing.T) {
	projectRoot := t.TempDir()
	seedRealityRepo(t, projectRoot, false)
	writeFile(t, filepath.Join(projectRoot, "adr", "ADR-0001-contract-rollout.md"), "# ADR\nProtocol rollout is staged.\n")
	externalDocs := filepath.Join(projectRoot, "shared-docs")
	writeFile(t, filepath.Join(externalDocs, "runbooks", "oncall.md"), "# Runbook\nEscalate protocol drift before rollout.\n")

	result, err := Ingest(Options{
		ProjectRoot: projectRoot,
		Repos:       []string{projectRoot},
		WithDocs:    true,
		DocRoots:    []string{externalDocs},
		Now: func() time.Time {
			return time.Date(2026, 3, 12, 11, 15, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("Ingest with docs failed: %v", err)
	}
	if result.SourceCount < 2 {
		t.Fatalf("expected evidence sources to be ingested, got %d", result.SourceCount)
	}

	data, err := os.ReadFile(filepath.Join(projectRoot, ".sdp", "reality", "repo-memory.json"))
	if err != nil {
		t.Fatalf("read repo-memory: %v", err)
	}
	var memory RepoMemory
	if err := json.Unmarshal(data, &memory); err != nil {
		t.Fatalf("parse repo-memory: %v", err)
	}
	if !containsSourceKind(memory.Sources, "adr") {
		t.Fatalf("expected ADR source, got %#v", memory.Sources)
	}
	if !containsSourceKind(memory.Sources, "runbook") {
		t.Fatalf("expected runbook source, got %#v", memory.Sources)
	}
	validateRepoMemorySchema(t, projectRoot, memory)

	mapData, err := os.ReadFile(filepath.Join(projectRoot, "docs", "reality", "multi-repo-map.md"))
	if err != nil {
		t.Fatalf("read multi-repo map: %v", err)
	}
	text := string(mapData)
	if !strings.Contains(text, "## Evidence Sources") {
		t.Fatalf("expected evidence section in map, got:\n%s", text)
	}
	if !strings.Contains(text, "ADR-0001-contract-rollout.md") || !strings.Contains(text, "oncall.md") {
		t.Fatalf("expected ingested evidence paths in map, got:\n%s", text)
	}
}

func validateRepoMemorySchema(t *testing.T, projectRoot string, payload any) {
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
	if err := compiler.AddResource("claim.schema.json", bytes.NewReader(claimData)); err != nil {
		t.Fatalf("add claim schema: %v", err)
	}
	if err := compiler.AddResource("https://sdp.dev/reality/repo-memory/claim.schema.json", bytes.NewReader(claimData)); err != nil {
		t.Fatalf("add claim schema alias: %v", err)
	}
	if err := compiler.AddResource("source.schema.json", bytes.NewReader(sourceData)); err != nil {
		t.Fatalf("add source schema: %v", err)
	}
	if err := compiler.AddResource("https://sdp.dev/reality/repo-memory/source.schema.json", bytes.NewReader(sourceData)); err != nil {
		t.Fatalf("add source schema alias: %v", err)
	}

	schemaPath := filepath.Join(projectRoot, "schema", "reality", "repo-memory.schema.json")
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile repo-memory schema: %v", err)
	}
	normalized := normalizeJSONValue(t, payload)
	if err := schema.Validate(normalized); err != nil {
		t.Fatalf("validate repo-memory payload: %v", err)
	}
}

func normalizeJSONValue(t *testing.T, payload any) any {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var normalized any
	if err := json.Unmarshal(data, &normalized); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return normalized
}

func seedRealityRepo(t *testing.T, root string, withHotspot bool) {
	t.Helper()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.26\n")
	writeFile(t, filepath.Join(root, "cmd", "app", "main.go"), "package main\n\nfunc main() {}\n")
	body := "package billing\n\nfunc Enabled() bool { return true }\n"
	if withHotspot {
		var b strings.Builder
		b.WriteString("package billing\n\n")
		for i := 0; i < 850; i++ {
			b.WriteString("func Value")
			b.WriteString(strings.Repeat("X", 0))
			b.WriteString("() int { return 1 }\n")
		}
		body = b.String()
	}
	writeFile(t, filepath.Join(root, "internal", "billing", "service.go"), body)
	writeFile(t, filepath.Join(root, "internal", "billing", "service_test.go"), "package billing\n\nimport \"testing\"\n\nfunc TestEnabled(t *testing.T) {}\n")
	writeFile(t, filepath.Join(root, "schema", "reality", "claim.schema.json"), "{\n  \"$schema\": \"https://json-schema.org/draft/2020-12/schema\",\n  \"$id\": \"https://sdp.dev/reality/claim/v1\",\n  \"type\": \"object\",\n  \"required\": [\"claim_id\", \"title\", \"statement\", \"status\", \"confidence\", \"source_ids\", \"review_state\"],\n  \"properties\": {\"claim_id\": {\"type\": \"string\"}, \"title\": {\"type\": \"string\"}, \"statement\": {\"type\": \"string\"}, \"status\": {\"type\": \"string\"}, \"confidence\": {\"type\": \"number\"}, \"source_ids\": {\"type\": \"array\"}, \"review_state\": {\"type\": \"string\"}},\n  \"additionalProperties\": true\n}\n")
	writeFile(t, filepath.Join(root, "schema", "reality", "source.schema.json"), "{\n  \"$schema\": \"https://json-schema.org/draft/2020-12/schema\",\n  \"$id\": \"https://sdp.dev/reality/source/v1\",\n  \"type\": \"object\",\n  \"required\": [\"source_id\", \"kind\", \"locator\", \"revision\"],\n  \"properties\": {\"source_id\": {\"type\": \"string\"}, \"kind\": {\"type\": \"string\"}, \"locator\": {\"type\": \"string\"}, \"revision\": {\"type\": \"string\"}},\n  \"additionalProperties\": true\n}\n")
	copySchemaFile(t, root, "repo-memory.schema.json")
}

func seedProtocolRepo(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/protocol\n\ngo 1.26\n")
	writeFile(t, filepath.Join(root, "prompts", "skills", "reality", "SKILL.md"), "# skill\n")
	writeFile(t, filepath.Join(root, "schema", "reality", "placeholder.json"), "{}\n")
}

func copySchemaFile(t *testing.T, projectRoot, name string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	srcPath := filepath.Join(filepath.Dir(filepath.Dir(wd)), "schema", "reality", name)
	data, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read schema fixture %s: %v", srcPath, err)
	}
	writeFile(t, filepath.Join(projectRoot, "schema", "reality", name), string(data))
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func containsHotspot(values []HotspotRecord, hotspotID string) bool {
	for _, value := range values {
		if value.HotspotID == hotspotID {
			return true
		}
	}
	return false
}

func containsSourceKind(values []ReviewSource, expected string) bool {
	for _, value := range values {
		if value.Kind == expected {
			return true
		}
	}
	return false
}
