package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_WithSingleRepo(t *testing.T) {
	root := t.TempDir()
	seedRepo(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"--project-root", root, "--repo", root}, &stdout, &stderr); err != nil {
		t.Fatalf("run failed: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "indexed 1 repo(s)") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".sdp", "reality", "repo-memory.json")); err != nil {
		t.Fatalf("expected repo-memory output: %v", err)
	}
}

func TestRun_RejectsConflictingInputs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"--repo", ".", "--reposet", ".,./sdp"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected conflicting inputs to fail")
	}
}

func seedRepo(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.26\n")
	writeFile(t, filepath.Join(root, "cmd", "app", "main.go"), "package main\n\nfunc main() {}\n")
	writeFile(t, filepath.Join(root, "internal", "billing", "service.go"), "package billing\n\nfunc Enabled() bool { return true }\n")
	writeFile(t, filepath.Join(root, "schema", "reality", "claim.schema.json"), "{\n  \"$schema\": \"https://json-schema.org/draft/2020-12/schema\",\n  \"$id\": \"https://sdp.dev/reality/claim/v1\",\n  \"type\": \"object\",\n  \"required\": [\"claim_id\", \"title\", \"statement\", \"status\", \"confidence\", \"source_ids\", \"review_state\"],\n  \"properties\": {\"claim_id\": {\"type\": \"string\"}, \"title\": {\"type\": \"string\"}, \"statement\": {\"type\": \"string\"}, \"status\": {\"type\": \"string\"}, \"confidence\": {\"type\": \"number\"}, \"source_ids\": {\"type\": \"array\"}, \"review_state\": {\"type\": \"string\"}},\n  \"additionalProperties\": true\n}\n")
	writeFile(t, filepath.Join(root, "schema", "reality", "source.schema.json"), "{\n  \"$schema\": \"https://json-schema.org/draft/2020-12/schema\",\n  \"$id\": \"https://sdp.dev/reality/source/v1\",\n  \"type\": \"object\",\n  \"required\": [\"source_id\", \"kind\", \"locator\", \"revision\"],\n  \"properties\": {\"source_id\": {\"type\": \"string\"}, \"kind\": {\"type\": \"string\"}, \"locator\": {\"type\": \"string\"}, \"revision\": {\"type\": \"string\"}},\n  \"additionalProperties\": true\n}\n")
	writeSchemaFixture(t, root, "repo-memory.schema.json")
}

func writeSchemaFixture(t *testing.T, projectRoot, name string) {
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
