package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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
