package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockOllamaServer returns a test server that responds to POST /api/embeddings
// with a deterministic embedding based on the prompt text length.
func mockOllamaServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			http.NotFound(w, r)
			return
		}
		var req struct {
			Model  string `json:"model"`
			Prompt string `json:"prompt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		// Generate a deterministic 4-dim embedding from prompt.
		vec := deterministicEmbed(req.Prompt)
		resp := map[string]interface{}{
			"embedding": vec,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
}

// deterministicEmbed produces a 4-dim unit-ish vector from text.
// It uses character codes so similar texts get similar vectors.
func deterministicEmbed(text string) []float64 {
	vec := make([]float64, 4)
	for i, ch := range text {
		vec[i%4] += float64(ch) / 1000.0
	}
	// Normalize to unit length.
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		norm = 1.0 / (norm * norm) // rough normalization
		for i := range vec {
			vec[i] *= norm
		}
	}
	return vec
}

// writeCorpus writes a minimal issues.jsonl corpus to a temp dir.
func writeCorpus(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "issues.jsonl")

	issues := []map[string]interface{}{
		{"_type": "issue", "id": "sdplab-001", "title": "fix nil pointer in dispatcher", "description": "crash on startup", "status": "closed", "priority": "P1", "issue_type": "bug", "created_at": "2025-01-01T00:00:00Z"},
		{"_type": "issue", "id": "sdplab-002", "title": "add support for custom models", "description": "feature request", "status": "closed", "priority": "P2", "issue_type": "feature", "created_at": "2025-01-02T00:00:00Z"},
		{"_type": "issue", "id": "sdplab-003", "title": "refactor cascade layer", "description": "clean up code", "status": "closed", "priority": "P3", "issue_type": "task", "created_at": "2025-01-03T00:00:00Z"},
		{"_type": "issue", "id": "sdplab-004", "title": "critical production outage", "description": "all services down", "status": "closed", "priority": "P0", "issue_type": "bug", "created_at": "2025-01-04T00:00:00Z"},
		{"_type": "issue", "id": "sdplab-005", "title": "update documentation", "description": "improve docs", "status": "closed", "priority": "P3", "issue_type": "task", "created_at": "2025-01-05T00:00:00Z"},
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create corpus: %v", err)
	}
	defer f.Close()

	for _, issue := range issues {
		data, err := json.Marshal(issue)
		if err != nil {
			t.Fatalf("marshal issue: %v", err)
		}
		fmt.Fprintf(f, "%s\n", data)
	}
	return path
}

// TestRun_MissingTitle tests that run() returns exit code 1 when --title is missing.
func TestRun_MissingTitle(t *testing.T) {
	var stdout, stderr strings.Builder
	code := run([]string{}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--title is required") {
		t.Errorf("expected error message about --title, got: %s", stderr.String())
	}
}

// TestRun_ValidArgs tests that run() succeeds with valid arguments.
func TestRun_ValidArgs(t *testing.T) {
	srv := mockOllamaServer(t)
	defer srv.Close()
	corpusPath := writeCorpus(t)

	var stdout, stderr strings.Builder
	code := run([]string{
		"--title=fix nil pointer",
		"--format=json",
		"--ollama-url=" + srv.URL,
		"--corpus-path=" + corpusPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}
	var result jsonOutput
	if err := json.Unmarshal([]byte(stdout.String()), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, stdout.String())
	}
}

// TestRun_BadFlag tests that run() returns exit code 1 for unknown flags.
func TestRun_BadFlag(t *testing.T) {
	var stdout, stderr strings.Builder
	code := run([]string{"--unknown-flag=xyz"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1 for unknown flag, got %d", code)
	}
}

// TestRunClassify_JSONOutput tests that runClassify produces valid JSON.
func TestRunClassify_JSONOutput(t *testing.T) {
	srv := mockOllamaServer(t)
	defer srv.Close()
	corpusPath := writeCorpus(t)

	cfg := config{
		title:       "fix nil pointer in dispatcher",
		description: "crash on startup",
		format:      "json",
		ollamaURL:   srv.URL,
		corpusPath:  corpusPath,
	}

	var sb strings.Builder
	err := runClassify(context.Background(), cfg, &sb)
	if err != nil {
		t.Fatalf("runClassify: %v", err)
	}

	out := sb.String()
	var result jsonOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, out)
	}
	if result.Title != cfg.title {
		t.Errorf("title mismatch: got %q, want %q", result.Title, cfg.title)
	}
	if result.Type.Value == "" {
		t.Error("type.value should not be empty")
	}
	if result.Priority.Value == "" {
		t.Error("priority.value should not be empty")
	}
	if result.Type.Status == "" {
		t.Error("type.status should not be empty")
	}
}

// TestRunClassify_HumanOutput tests human-readable format.
func TestRunClassify_HumanOutput(t *testing.T) {
	srv := mockOllamaServer(t)
	defer srv.Close()
	corpusPath := writeCorpus(t)

	cfg := config{
		title:       "add custom model support",
		description: "feature request for new models",
		format:      "human",
		ollamaURL:   srv.URL,
		corpusPath:  corpusPath,
	}

	var sb strings.Builder
	err := runClassify(context.Background(), cfg, &sb)
	if err != nil {
		t.Fatalf("runClassify: %v", err)
	}

	out := sb.String()
	if !strings.Contains(out, "Title:") {
		t.Errorf("human output missing 'Title:' section\noutput: %s", out)
	}
	if !strings.Contains(out, "Type:") {
		t.Errorf("human output missing 'Type:' section\noutput: %s", out)
	}
	if !strings.Contains(out, "Priority:") {
		t.Errorf("human output missing 'Priority:' section\noutput: %s", out)
	}
	if !strings.Contains(out, "confidence:") {
		t.Errorf("human output missing confidence\noutput: %s", out)
	}
}

// TestRunClassify_TitleOnly tests classification without description.
func TestRunClassify_TitleOnly(t *testing.T) {
	srv := mockOllamaServer(t)
	defer srv.Close()
	corpusPath := writeCorpus(t)

	cfg := config{
		title:      "critical production outage",
		format:     "json",
		ollamaURL:  srv.URL,
		corpusPath: corpusPath,
	}

	var sb strings.Builder
	err := runClassify(context.Background(), cfg, &sb)
	if err != nil {
		t.Fatalf("runClassify: %v", err)
	}

	var result jsonOutput
	if err := json.Unmarshal([]byte(sb.String()), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// Should have priority and type.
	if result.Priority.Value == "" {
		t.Error("priority.value should not be empty for title-only input")
	}
}

// TestRunClassify_MissingCorpus tests graceful error on missing corpus.
func TestRunClassify_MissingCorpus(t *testing.T) {
	srv := mockOllamaServer(t)
	defer srv.Close()

	cfg := config{
		title:      "some issue",
		format:     "json",
		ollamaURL:  srv.URL,
		corpusPath: "/nonexistent/path/issues.jsonl",
	}

	var sb strings.Builder
	err := runClassify(context.Background(), cfg, &sb)
	if err == nil {
		t.Fatal("expected error for missing corpus, got nil")
	}
}

// TestRunClassify_NeighborsInHumanOutput tests that top neighbors appear in human output.
func TestRunClassify_NeighborsInHumanOutput(t *testing.T) {
	srv := mockOllamaServer(t)
	defer srv.Close()
	corpusPath := writeCorpus(t)

	cfg := config{
		title:      "refactor dispatch layer",
		format:     "human",
		ollamaURL:  srv.URL,
		corpusPath: corpusPath,
	}

	var sb strings.Builder
	err := runClassify(context.Background(), cfg, &sb)
	if err != nil {
		t.Fatalf("runClassify: %v", err)
	}

	out := sb.String()
	if !strings.Contains(out, "Top neighbors") {
		t.Errorf("human output missing 'Top neighbors' section\noutput: %s", out)
	}
}

// TestRunClassify_OllamaDown tests that embed failure produces an error.
func TestRunClassify_OllamaDown(t *testing.T) {
	corpusPath := writeCorpus(t)

	cfg := config{
		title:      "fix crash",
		format:     "json",
		ollamaURL:  "http://127.0.0.1:1", // nothing listening here
		corpusPath: corpusPath,
	}

	var sb strings.Builder
	err := runClassify(context.Background(), cfg, &sb)
	if err == nil {
		t.Fatal("expected error when Ollama is unreachable, got nil")
	}
}

// TestRunClassify_DefaultFormatIsJSON tests that an unknown format falls back to JSON.
func TestRunClassify_DefaultFormatIsJSON(t *testing.T) {
	srv := mockOllamaServer(t)
	defer srv.Close()
	corpusPath := writeCorpus(t)

	cfg := config{
		title:      "some issue",
		format:     "unknown",
		ollamaURL:  srv.URL,
		corpusPath: corpusPath,
	}

	var sb strings.Builder
	err := runClassify(context.Background(), cfg, &sb)
	if err != nil {
		t.Fatalf("runClassify: %v", err)
	}

	var result jsonOutput
	if err := json.Unmarshal([]byte(sb.String()), &result); err != nil {
		t.Fatalf("unknown format should fall back to JSON, but got invalid JSON: %v\noutput: %s", err, sb.String())
	}
}

// TestJSONSchema_AllFields verifies the JSON output has all required schema fields.
func TestJSONSchema_AllFields(t *testing.T) {
	srv := mockOllamaServer(t)
	defer srv.Close()
	corpusPath := writeCorpus(t)

	cfg := config{
		title:       "update docs for new API",
		description: "write migration guide",
		format:      "json",
		ollamaURL:   srv.URL,
		corpusPath:  corpusPath,
	}

	var sb strings.Builder
	if err := runClassify(context.Background(), cfg, &sb); err != nil {
		t.Fatalf("runClassify: %v", err)
	}

	// Unmarshal to raw map to check presence of all keys.
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(sb.String()), &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	requiredTop := []string{"title", "type", "priority"}
	for _, key := range requiredTop {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing top-level key %q in JSON output", key)
		}
	}

	typeObj, _ := raw["type"].(map[string]interface{})
	if typeObj == nil {
		t.Fatal("'type' field is not an object")
	}
	requiredTypeKeys := []string{"value", "confidence", "status"}
	for _, key := range requiredTypeKeys {
		if _, ok := typeObj[key]; !ok {
			t.Errorf("missing key %q in 'type' object", key)
		}
	}

	priObj, _ := raw["priority"].(map[string]interface{})
	if priObj == nil {
		t.Fatal("'priority' field is not an object")
	}
	requiredPriKeys := []string{"value", "confidence", "status"}
	for _, key := range requiredPriKeys {
		if _, ok := priObj[key]; !ok {
			t.Errorf("missing key %q in 'priority' object", key)
		}
	}
}
