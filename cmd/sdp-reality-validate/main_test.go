package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const testGeneratedAt = "2026-03-11T00:00:00Z"

func TestValidateRealityArtifacts_Valid(t *testing.T) {
	root := t.TempDir()
	copyRealitySchemas(t, root)
	writeValidRealityArtifacts(t, root)

	issues, err := validateRealityArtifacts(root)
	if err != nil {
		t.Fatalf("expected valid artifacts, got error: %v\nissues:\n%s", err, strings.Join(issues, "\n"))
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got: %v", issues)
	}
}

func TestValidateRealityArtifacts_Invalid(t *testing.T) {
	root := t.TempDir()
	copyRealitySchemas(t, root)
	writeValidRealityArtifacts(t, root)

	invalidReadiness := map[string]any{
		"spec_version": "v1.0",
		"generated_at": testGeneratedAt,
		"verdict":      "broken",
		"dimensions": map[string]any{
			"boundary_clarity":          map[string]any{"score": 0.5},
			"verification_coverage":     map[string]any{"score": 0.5},
			"hotspot_concentration":     map[string]any{"score": 0.5},
			"integration_fragility":     map[string]any{"score": 0.5},
			"documentation_trust_level": map[string]any{"score": 0.5},
		},
		"justification_claim_ids": []string{"claim:source-footprint"},
	}
	writeJSON(t, filepath.Join(root, ".sdp", "reality", "readiness-report.json"), invalidReadiness)

	issues, err := validateRealityArtifacts(root)
	if err == nil {
		t.Fatal("expected validation error for invalid readiness verdict")
	}
	if len(issues) == 0 {
		t.Fatal("expected non-empty validation issues")
	}

	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "readiness-report.json") {
		t.Fatalf("expected readiness report in issues, got: %s", joined)
	}
}

func TestRealitySchemasCompileAndValidateProSamples(t *testing.T) {
	root := t.TempDir()
	copyRealitySchemas(t, root)

	schemaFiles := []string{
		"schema/reality/repo-memory.schema.json",
		"schema/reality/conflicts-report.schema.json",
		"schema/reality/intent-gap-report.schema.json",
		"schema/reality/bootstrap-backlog.schema.json",
		"schema/reality/agent-readiness-plan.schema.json",
		"schema/reality/c4-system-context.schema.json",
		"schema/reality/c4-container.schema.json",
		"schema/reality/c4-component.schema.json",
	}
	for _, schemaRel := range schemaFiles {
		schema, err := compileSchema(root, schemaRel)
		if err != nil {
			t.Fatalf("compile %s: %v", schemaRel, err)
		}
		payload, ok := validProSamples()[filepath.Base(schemaRel)]
		if !ok {
			t.Fatalf("missing sample payload for %s", schemaRel)
		}
		payload = normalizeJSONValue(t, payload)
		if err := schema.Validate(payload); err != nil {
			t.Fatalf("validate sample for %s: %v", schemaRel, err)
		}
	}

	entries, err := os.ReadDir(filepath.Join(root, "schema", "reality"))
	if err != nil {
		t.Fatalf("read schema dir: %v", err)
	}
	compiled := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if _, err := compileSchema(root, filepath.Join("schema", "reality", entry.Name())); err != nil {
			t.Fatalf("compile %s: %v", entry.Name(), err)
		}
		compiled = append(compiled, entry.Name())
	}
	if !slices.Contains(compiled, "agent-readiness-plan.schema.json") {
		t.Fatalf("expected new pro schemas to be copied: %v", compiled)
	}
}

func copyRealitySchemas(t *testing.T, dstRoot string) {
	t.Helper()

	srcRoot := findModuleRoot(t)
	srcDir := filepath.Join(srcRoot, "schema", "reality")
	dstDir := filepath.Join(dstRoot, "schema", "reality")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dstDir, err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatalf("read schema dir %s: %v", srcDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		srcPath := filepath.Join(srcDir, entry.Name())
		data, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("read schema %s: %v", srcPath, err)
		}
		dstPath := filepath.Join(dstDir, entry.Name())
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			t.Fatalf("write schema %s: %v", dstPath, err)
		}
	}
}

func writeValidRealityArtifacts(t *testing.T, root string) {
	t.Helper()

	writeJSON(t, filepath.Join(root, ".sdp", "reality", "reality-summary.json"), map[string]any{
		"spec_version":          "v1.0",
		"run_id":                "reality-oss-test",
		"generated_at":          testGeneratedAt,
		"scope":                 map[string]any{"repos": []string{"test-repo"}, "mode": "emit-oss"},
		"readiness_verdict":     "ready",
		"top_finding_claim_ids": []string{"claim:source-footprint"},
		"artifacts":             []string{".sdp/reality/readiness-report.json"},
	})

	writeJSON(t, filepath.Join(root, ".sdp", "reality", "feature-inventory.json"), map[string]any{
		"spec_version": "v1.0",
		"generated_at": testGeneratedAt,
		"features": []map[string]any{
			{
				"feature_id":         "feature:test",
				"title":              "Test feature",
				"summary":            "summary",
				"status":             "implemented",
				"evidence_claim_ids": []string{"claim:source-footprint"},
				"confidence":         0.9,
				"mapped_components":  []string{"cmd"},
			},
		},
	})

	writeJSON(t, filepath.Join(root, ".sdp", "reality", "architecture-map.json"), map[string]any{
		"spec_version": "v1.0",
		"generated_at": testGeneratedAt,
		"nodes": []map[string]any{
			{"node_id": "module:cmd", "name": "cmd", "kind": "module"},
		},
		"edges": []map[string]any{},
	})

	writeJSON(t, filepath.Join(root, ".sdp", "reality", "integration-map.json"), map[string]any{
		"spec_version": "v1.0",
		"generated_at": testGeneratedAt,
		"integrations": []map[string]any{},
	})

	writeJSON(t, filepath.Join(root, ".sdp", "reality", "quality-report.json"), map[string]any{
		"spec_version": "v1.0",
		"generated_at": testGeneratedAt,
		"findings": []map[string]any{
			{
				"finding_id": "finding:test",
				"title":      "Test finding",
				"severity":   "low",
				"claim_ids":  []string{"claim:source-footprint"},
			},
		},
	})

	writeJSON(t, filepath.Join(root, ".sdp", "reality", "drift-report.json"), map[string]any{
		"spec_version":         "v1.0",
		"generated_at":         testGeneratedAt,
		"contradictions":       []map[string]any{},
		"unresolved_questions": []string{},
	})

	writeJSON(t, filepath.Join(root, ".sdp", "reality", "readiness-report.json"), map[string]any{
		"spec_version": "v1.0",
		"generated_at": testGeneratedAt,
		"verdict":      "ready",
		"dimensions": map[string]any{
			"boundary_clarity":          map[string]any{"score": 0.5},
			"verification_coverage":     map[string]any{"score": 0.5},
			"hotspot_concentration":     map[string]any{"score": 0.5},
			"integration_fragility":     map[string]any{"score": 0.5},
			"documentation_trust_level": map[string]any{"score": 0.5},
		},
		"justification_claim_ids": []string{"claim:source-footprint"},
	})
}

func writeJSON(t *testing.T, path string, payload any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal json for %s: %v", path, err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write json %s: %v", path, err)
	}
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for dir := wd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
	}
	t.Fatal("could not locate module root")
	return ""
}

func normalizeJSONValue(t *testing.T, payload any) any {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal sample payload: %v", err)
	}
	var normalized any
	if err := json.Unmarshal(data, &normalized); err != nil {
		t.Fatalf("unmarshal sample payload: %v", err)
	}
	return normalized
}

func validProSamples() map[string]any {
	return map[string]any{
		"repo-memory.schema.json": map[string]any{
			"spec_version": "v1.0",
			"generated_at": testGeneratedAt,
			"repos": []map[string]any{
				{
					"repo_id":         "repo:app",
					"name":            "app",
					"root_path":       "/repos/app",
					"role":            "service",
					"summary":         "Main application",
					"last_indexed_at": testGeneratedAt,
				},
			},
			"module_summaries": []map[string]any{
				{
					"module_id": "module:app:internal/api",
					"repo_id":   "repo:app",
					"summary":   "API layer",
					"paths":     []string{"internal/api"},
				},
			},
			"unresolved_questions": []string{"Who owns the external billing contract?"},
		},
		"conflicts-report.schema.json": map[string]any{
			"spec_version": "v1.0",
			"generated_at": testGeneratedAt,
			"conflicts": []map[string]any{
				{
					"conflict_id":         "conflict:1",
					"summary":             "Docs say async, code is sync.",
					"competing_claim_ids": []string{"claim:docs-async", "claim:code-sync"},
					"severity":            "high",
					"status":              "arbitrated",
					"arbitrated_claim_id": "claim:code-sync",
				},
			},
		},
		"intent-gap-report.schema.json": map[string]any{
			"spec_version": "v1.0",
			"generated_at": testGeneratedAt,
			"gaps": []map[string]any{
				{
					"gap_id":               "gap:1",
					"title":                "Missing retry policy",
					"expected_state":       "Retries are mandated by ADR",
					"observed_state":       "Handler performs a single attempt",
					"gap_type":             "missing",
					"severity":             "high",
					"status":               "open",
					"supporting_claim_ids": []string{"claim:adr-retry", "claim:handler-no-retry"},
				},
			},
		},
		"bootstrap-backlog.schema.json": map[string]any{
			"spec_version": "v1.0",
			"generated_at": testGeneratedAt,
			"workstreams": []map[string]any{
				{
					"backlog_id":         "bootstrap:1",
					"title":              "Fence billing boundary",
					"goal":               "Make billing integration testable",
					"priority":           "P1",
					"status":             "sequenced",
					"repositories":       []string{"repo:app"},
					"evidence_claim_ids": []string{"claim:integration-surface"},
				},
			},
		},
		"agent-readiness-plan.schema.json": map[string]any{
			"spec_version":    "v1.0",
			"generated_at":    testGeneratedAt,
			"current_verdict": "ready_with_constraints",
			"target_verdict":  "ready",
			"phases": []map[string]any{
				{
					"phase_id":          "phase:1",
					"title":             "Stabilize boundaries",
					"objective":         "Reduce hidden integration risk",
					"allowed_scope":     []string{"internal/billing"},
					"blocked_zones":     []string{"deploy/prod"},
					"required_evidence": []string{"integration contracts"},
					"exit_criteria":     []string{"Contract tests exist"},
				},
			},
		},
		"c4-system-context.schema.json": map[string]any{
			"spec_version": "v1.0",
			"generated_at": testGeneratedAt,
			"scope": map[string]any{
				"system_name": "Billing Platform",
				"repos":       []string{"repo:app"},
			},
			"systems": []map[string]any{
				{
					"system_id": "system:billing",
					"name":      "Billing Platform",
					"boundary":  "internal",
				},
				{
					"system_id": "system:stripe",
					"name":      "Stripe",
					"boundary":  "external",
				},
			},
			"relationships": []map[string]any{
				{
					"relationship_id": "rel:1",
					"from":            "system:billing",
					"to":              "system:stripe",
					"description":     "Charges cards",
				},
			},
		},
		"c4-container.schema.json": map[string]any{
			"spec_version": "v1.0",
			"generated_at": testGeneratedAt,
			"system_name":  "Billing Platform",
			"containers": []map[string]any{
				{
					"container_id": "container:api",
					"name":         "Billing API",
					"technology":   "Go",
				},
				{
					"container_id": "container:db",
					"name":         "Billing DB",
					"technology":   "Postgres",
				},
			},
			"relationships": []map[string]any{
				{
					"relationship_id": "rel:api-db",
					"from":            "container:api",
					"to":              "container:db",
					"description":     "Reads and writes billing state",
				},
			},
		},
		"c4-component.schema.json": map[string]any{
			"spec_version": "v1.0",
			"generated_at": testGeneratedAt,
			"container_id": "container:api",
			"components": []map[string]any{
				{
					"component_id": "component:invoice-service",
					"name":         "Invoice Service",
					"paths":        []string{"internal/invoice/service.go"},
				},
				{
					"component_id": "component:gateway",
					"name":         "Billing Gateway",
					"paths":        []string{"internal/billing/gateway.go"},
				},
			},
			"relationships": []map[string]any{
				{
					"relationship_id": "rel:invoice-gateway",
					"from":            "component:invoice-service",
					"to":              "component:gateway",
					"description":     "Sends charge request",
				},
			},
		},
	}
}
