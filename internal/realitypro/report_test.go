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

func TestEmitReports_WritesAndValidatesProArtifacts(t *testing.T) {
	projectRoot := t.TempDir()
	seedRealityRepo(t, projectRoot, true)
	submoduleRoot := filepath.Join(projectRoot, "sdp")
	seedProtocolRepo(t, submoduleRoot)
	writeFile(t, filepath.Join(projectRoot, "docs", "shared", "architecture.md"), "# Architecture\nProtocol rollout is staged across repos.\n")
	writeFile(t, filepath.Join(projectRoot, ".github", "CODEOWNERS"), "* @platform\n/internal/billing/ @payments\n")
	writeFile(t, filepath.Join(projectRoot, ".github", "teams.json"), `{
  "teams": [
    {
      "team_id": "team:platform",
      "name": "Platform",
      "aliases": ["platform"],
      "slack": "#platform",
      "escalation_target": "@platform-oncall",
      "owns": ["*"]
    },
    {
      "team_id": "team:payments",
      "name": "Payments",
      "aliases": ["payments"],
      "email": "payments@example.com",
      "escalation_target": "@payments-oncall",
      "owns": ["/internal/billing/"]
    }
  ]
}`)
	for _, schemaName := range []string{
		"conflicts-report.schema.json",
		"intent-gap-report.schema.json",
		"c4-system-context.schema.json",
		"c4-container.schema.json",
		"c4-component.schema.json",
		"bootstrap-backlog.schema.json",
		"agent-readiness-plan.schema.json",
	} {
		copySchemaFile(t, projectRoot, schemaName)
	}

	if _, err := Ingest(Options{
		ProjectRoot: projectRoot,
		Repos:       []string{projectRoot, submoduleRoot},
		WithDocs:    true,
		Now: func() time.Time {
			return time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
		},
	}); err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}
	if _, err := Review(ReviewOptions{
		ProjectRoot: projectRoot,
		Now: func() time.Time {
			return time.Date(2026, 3, 12, 10, 1, 0, 0, time.UTC)
		},
	}); err != nil {
		t.Fatalf("Review failed: %v", err)
	}

	result, err := EmitReports(ReportOptions{
		ProjectRoot: projectRoot,
		Now: func() time.Time {
			return time.Date(2026, 3, 12, 10, 2, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("EmitReports failed: %v", err)
	}
	if len(result.WrittenPaths) != 10 {
		t.Fatalf("expected 10 written artifacts, got %d", len(result.WrittenPaths))
	}
	if result.BacklogCount == 0 {
		t.Fatal("expected bootstrap backlog entries")
	}
	if result.PhaseCount == 0 {
		t.Fatal("expected readiness phases")
	}
	if result.CurrentVerdict == "" || result.TargetVerdict == "" {
		t.Fatalf("expected readiness verdicts, got %+v", result)
	}

	systemContext := readJSONArtifact[C4SystemContext](t, filepath.Join(projectRoot, ".sdp", "reality", "c4-system-context.json"))
	if len(systemContext.Systems) == 0 || len(systemContext.Relationships) == 0 {
		t.Fatalf("expected systems and relationships, got %#v", systemContext)
	}
	if !containsSourceKind(systemContext.Sources, "doc") {
		t.Fatalf("expected documentation evidence in system context sources, got %#v", systemContext.Sources)
	}
	if !containsPerson(systemContext.People, "Platform") {
		t.Fatalf("expected ownership people in system context, got %#v", systemContext.People)
	}
	validateArtifactSchema(t, projectRoot, "c4-system-context.schema.json", systemContext)

	containerView := readJSONArtifact[C4ContainerView](t, filepath.Join(projectRoot, ".sdp", "reality", "c4-container.json"))
	if len(containerView.Containers) < 2 {
		t.Fatalf("expected repo containers, got %#v", containerView.Containers)
	}
	validateArtifactSchema(t, projectRoot, "c4-container.schema.json", containerView)

	componentView := readJSONArtifact[C4ComponentView](t, filepath.Join(projectRoot, ".sdp", "reality", "c4-component.json"))
	if len(componentView.Components) == 0 {
		t.Fatal("expected component inventory")
	}
	if len(componentView.Relationships) == 0 {
		t.Fatal("expected component relationships")
	}
	validateArtifactSchema(t, projectRoot, "c4-component.schema.json", componentView)

	backlog := readJSONArtifact[BootstrapBacklog](t, filepath.Join(projectRoot, ".sdp", "reality", "bootstrap-backlog.json"))
	if len(backlog.Workstreams) == 0 {
		t.Fatal("expected backlog workstreams")
	}
	if backlog.Workstreams[0].BacklogID == "" {
		t.Fatalf("expected stable backlog IDs, got %#v", backlog.Workstreams[0])
	}
	validateArtifactSchema(t, projectRoot, "bootstrap-backlog.schema.json", backlog)

	readiness := readJSONArtifact[AgentReadinessPlan](t, filepath.Join(projectRoot, ".sdp", "reality", "agent-readiness-plan.json"))
	if readiness.CurrentVerdict == "ready" {
		t.Fatalf("expected constrained verdict for seeded hotspots/gaps, got %#v", readiness)
	}
	if len(readiness.Phases) == 0 {
		t.Fatalf("expected phased plan, got %#v", readiness)
	}
	validateArtifactSchema(t, projectRoot, "agent-readiness-plan.schema.json", readiness)

	for _, rel := range []string{
		"docs/reality/c4-system-context.md",
		"docs/reality/c4-containers.md",
		"docs/reality/c4-components.md",
		"docs/reality/intent-gap.md",
		"docs/reality/multi-repo-map.md",
	} {
		data, err := os.ReadFile(filepath.Join(projectRoot, rel))
		if err != nil {
			t.Fatalf("read markdown artifact %s: %v", rel, err)
		}
		if !strings.Contains(string(data), "# Reality") {
			t.Fatalf("expected reality heading in %s, got:\n%s", rel, data)
		}
	}
	mapData, err := os.ReadFile(filepath.Join(projectRoot, "docs", "reality", "multi-repo-map.md"))
	if err != nil {
		t.Fatalf("read multi-repo map: %v", err)
	}
	if !strings.Contains(string(mapData), "## Ownership Zones") || !strings.Contains(string(mapData), "Payments") {
		t.Fatalf("expected ownership rendering in multi-repo map, got:\n%s", mapData)
	}
}

func TestEmitReports_RequiresReviewArtifacts(t *testing.T) {
	projectRoot := t.TempDir()
	seedRealityRepo(t, projectRoot, false)
	if err := writeJSON(filepath.Join(projectRoot, ".sdp", "reality", "repo-memory.json"), RepoMemory{
		SpecVersion: specVersion,
		GeneratedAt: "2026-03-12T09:59:00Z",
		Repos: []RepoRecord{
			{
				RepoID:   "repo:" + sanitizeID(filepath.Base(projectRoot)),
				Name:     filepath.Base(projectRoot),
				RootPath: filepath.ToSlash(projectRoot),
				Role:     "service",
			},
		},
		ModuleSummaries: []ModuleSummary{
			{
				ModuleID: "module:repo:" + sanitizeID(filepath.Base(projectRoot)) + ":internal",
				RepoID:   "repo:" + sanitizeID(filepath.Base(projectRoot)),
				Summary:  "internal runtime",
				Paths:    []string{"internal/app/service.go"},
			},
		},
	}); err != nil {
		t.Fatalf("seed repo-memory: %v", err)
	}

	if _, err := EmitReports(ReportOptions{ProjectRoot: projectRoot}); err == nil {
		t.Fatal("expected missing review artifacts to fail")
	}
}

func readJSONArtifact[T any](t *testing.T, path string) T {
	t.Helper()
	var payload T
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return payload
}

func validateArtifactSchema(t *testing.T, projectRoot, schemaName string, payload any) {
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
	if err := schema.Validate(normalizeJSONValue(t, payload)); err != nil {
		t.Fatalf("validate %s: %v", schemaName, err)
	}
}

func containsPerson(values []C4Person, name string) bool {
	for _, value := range values {
		if value.Name == name {
			return true
		}
	}
	return false
}
