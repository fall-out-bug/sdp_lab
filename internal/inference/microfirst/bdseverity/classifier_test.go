package bdseverity

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"sdp_dev/internal/inference/decompose"
	"sdp_dev/internal/inference/microfirst/embed"
)

// writeFile is a test helper to write string content to a file.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}

// --- Mock Ollama server ---

// mockEmbedResponse is the shape returned by the mock server.
type mockEmbedResponse struct {
	Embedding []float64 `json:"embedding"`
}

// keywordVector returns a deterministic unit-normalised embedding based on
// priority-associated keywords found in the prompt. Each label maps to a
// distinct dimension in a 4-D vector.
//
//	dim 0 → P0 (outage, critical, breach, "production down", "data loss")
//	dim 1 → P1 (regression, blocked, failing, "CI", "major feature")
//	dim 2 → P2 (performance, degradation, minor, documentation, outdated)
//	dim 3 → P3 (typo, cosmetic, cleanup, stale, "nice to have")
func keywordVector(prompt string) []float64 {
	lower := strings.ToLower(prompt)
	scores := [4]float64{}

	p0Keywords := []string{"outage", "critical", "breach", "production down", "data loss", "security", "crash", "emergency", "zero-day", "corruption", "total failure", "unresponsive", "system down", "unavailable"}
	p1Keywords := []string{"regression", "blocked", "failing", "ci pipeline", "ci fail", "major feature", "broken feature", "blocked user", "failing test", "major broken", "critical regression", "launch blocked"}
	p2Keywords := []string{"performance", "degradation", "minor bug", "documentation", "outdated", "docs", "slower", "small regression", "minor regression", "minor ui"}
	p3Keywords := []string{"typo", "cosmetic", "cleanup", "stale", "nice to have", "unused", "comment", "alignment", "spacing", "color inconsistency", "keyboard shortcut"}

	for _, kw := range p0Keywords {
		if strings.Contains(lower, kw) {
			scores[0] += 1.0
		}
	}
	for _, kw := range p1Keywords {
		if strings.Contains(lower, kw) {
			scores[1] += 1.0
		}
	}
	for _, kw := range p2Keywords {
		if strings.Contains(lower, kw) {
			scores[2] += 1.0
		}
	}
	for _, kw := range p3Keywords {
		if strings.Contains(lower, kw) {
			scores[3] += 1.0
		}
	}

	// Find dominant dimension; fall back to uniform if no keywords matched.
	maxIdx := 0
	for i := 1; i < 4; i++ {
		if scores[i] > scores[maxIdx] {
			maxIdx = i
		}
	}

	// Build a vector: dominant dim gets high weight (0.9), others get small noise.
	vec := make([]float64, 4)
	if scores[maxIdx] == 0 {
		// No keyword hit — return a balanced vector that won't pass the threshold.
		for i := range vec {
			vec[i] = 0.5
		}
	} else {
		for i := range vec {
			if i == maxIdx {
				vec[i] = 0.9
			} else {
				vec[i] = 0.05
			}
		}
	}

	return normalise(vec)
}

func normalise(vec []float64) []float64 {
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return vec
	}
	out := make([]float64, len(vec))
	for i, v := range vec {
		out[i] = v / norm
	}
	return out
}

// newMockOllamaServer returns an httptest.Server that mimics POST /api/embeddings.
func newMockOllamaServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" || r.Method != http.MethodPost {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var req struct {
			Prompt string `json:"prompt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		vec := keywordVector(req.Prompt)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockEmbedResponse{Embedding: vec})
	}))
}

// newCachedEmbedder creates a CachedEmbedder pointing at a mock server.
func newCachedEmbedder(serverURL string) *embed.CachedEmbedder {
	ollama := embed.NewOllamaEmbedder(serverURL)
	return embed.NewCachedEmbedder(ollama, 1000)
}

// --- Test: LoadCorpus ---

func TestLoadCorpus_FiltersOpenAndEpic(t *testing.T) {
	train, eval, err := LoadCorpus("testdata/sample_issues.jsonl")
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	all := append(train, eval...)
	for _, issue := range all {
		if issue.Status != "closed" {
			t.Errorf("expected all issues closed, got status=%q for %s", issue.Status, issue.ID)
		}
		if issue.Priority == "" {
			t.Errorf("expected non-empty priority for %s", issue.ID)
		}
		if issue.IssueType == "epic" {
			t.Errorf("expected no epic issues, got epic for %s", issue.ID)
		}
	}
}

func TestLoadCorpus_Split(t *testing.T) {
	train, eval, err := LoadCorpus("testdata/sample_issues.jsonl")
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}

	// Our testdata has 60 qualifying issues → train=30, eval=30.
	if len(eval) != 30 {
		t.Errorf("expected 30 eval items, got %d", len(eval))
	}
	if len(train) != 30 {
		t.Errorf("expected 30 train items, got %d", len(train))
	}

	// Train items should come before eval items chronologically.
	if len(train) > 0 && len(eval) > 0 {
		lastTrain := train[len(train)-1].CreatedAt
		firstEval := eval[0].CreatedAt
		if lastTrain >= firstEval {
			t.Errorf("train/eval ordering violated: lastTrain=%s firstEval=%s", lastTrain, firstEval)
		}
	}
}

func TestLoadCorpus_MissingFile(t *testing.T) {
	_, _, err := LoadCorpus("testdata/nonexistent.jsonl")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// --- Test: BdSeverityMicro ---

func buildClassifier(t *testing.T) (*BdSeverityMicro, *httptest.Server) {
	t.Helper()
	srv := newMockOllamaServer(t)
	e := newCachedEmbedder(srv.URL)

	train, _, err := LoadCorpus("testdata/sample_issues.jsonl")
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}

	ctx := context.Background()
	c, err := New(ctx, e, train, defaultThreshold)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c, srv
}

func TestBdSeverityMicro_Name(t *testing.T) {
	c, srv := buildClassifier(t)
	defer srv.Close()
	if c.Name() != "bd-severity-micro" {
		t.Errorf("Name() = %q, want %q", c.Name(), "bd-severity-micro")
	}
}

func TestBdSeverityMicro_RunP0(t *testing.T) {
	c, srv := buildClassifier(t)
	defer srv.Close()

	ctx := context.Background()
	result, trace, err := c.Run(ctx, BdInput{
		Title:       "production is down critical outage",
		Description: "The production system is completely down, all services unavailable.",
	})
	if err != nil {
		t.Fatalf("Run P0: %v", err)
	}
	if result.ConfStatus() != decompose.StatusOK {
		t.Errorf("expected StatusOK, got %v", result.ConfStatus())
	}
	if result.Priority != "P0" {
		t.Errorf("expected P0, got %q", result.Priority)
	}
	if trace.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", trace.Attempts)
	}
}

func TestBdSeverityMicro_RunP3(t *testing.T) {
	c, srv := buildClassifier(t)
	defer srv.Close()

	ctx := context.Background()
	result, _, err := c.Run(ctx, BdInput{
		Title:       "fix typo in comment",
		Description: "Cosmetic typo in a stale comment block.",
	})
	if err != nil {
		t.Fatalf("Run P3: %v", err)
	}
	if result.ConfStatus() != decompose.StatusOK {
		t.Errorf("expected StatusOK, got %v", result.ConfStatus())
	}
	if result.Priority != "P3" {
		t.Errorf("expected P3, got %q", result.Priority)
	}
}

func TestBdSeverityMicro_RunAmbiguous(t *testing.T) {
	c, srv := buildClassifier(t)
	defer srv.Close()

	ctx := context.Background()
	// Ambiguous prompt — no strong keyword signal
	result, _, err := c.Run(ctx, BdInput{
		Title:       "issue with deployment",
		Description: "There is an issue with the deployment process.",
	})
	if err != nil {
		t.Fatalf("Run ambiguous: %v", err)
	}
	// Any status is acceptable — just check it runs without error.
	_ = result.ConfStatus()
}

func TestBdSeverityMicro_ResultHasNeighbors(t *testing.T) {
	c, srv := buildClassifier(t)
	defer srv.Close()

	ctx := context.Background()
	result, _, err := c.Run(ctx, BdInput{
		Title:       "production is down critical outage",
		Description: "Complete production outage.",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Neighbors) == 0 {
		t.Error("expected non-empty Neighbors")
	}
	if len(result.Neighbors) > neighborCount {
		t.Errorf("expected at most %d neighbors, got %d", neighborCount, len(result.Neighbors))
	}
}

// TestBdSeverityResult_Confidence exercises the exported Confidence accessor.
func TestBdSeverityResult_Confidence(t *testing.T) {
	r := BdSeverityResult{
		Priority:   "P0",
		confidence: 0.92,
	}
	if r.Confidence() != 0.92 {
		t.Errorf("Confidence() = %f, want 0.92", r.Confidence())
	}
}

// TestIssueText_EmptyDescription exercises the title-only path.
func TestIssueText_EmptyDescription(t *testing.T) {
	got := issueText("just a title", "")
	if got != "just a title" {
		t.Errorf("issueText empty desc = %q, want %q", got, "just a title")
	}
}

// TestIssueText_WithDescription exercises the combined path.
func TestIssueText_WithDescription(t *testing.T) {
	got := issueText("title", "desc")
	want := "title. desc"
	if got != want {
		t.Errorf("issueText = %q, want %q", got, want)
	}
}

// TestLoadCorpus_SmallSet covers the path where total issues <= evalSize.
func TestLoadCorpus_SmallSet(t *testing.T) {
	// Temp file with fewer than 30 qualifying issues.
	content := `{"_type":"issue","id":"s-01","title":"prod down","description":"outage","status":"closed","priority":"P0","issue_type":"task","created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:00:00Z"}
{"_type":"issue","id":"s-02","title":"typo fix","description":"cosmetic stale","status":"closed","priority":"P3","issue_type":"task","created_at":"2025-01-02T00:00:00Z","updated_at":"2025-01-02T00:00:00Z"}
`
	path := t.TempDir() + "/small.jsonl"
	if err := writeFile(path, content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	train, eval, err := LoadCorpus(path)
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	if len(train) != 0 {
		t.Errorf("expected empty train for small set, got %d", len(train))
	}
	if len(eval) != 2 {
		t.Errorf("expected 2 eval items, got %d", len(eval))
	}
}

// TestLoadCorpus_SkipsEpicAndOpen covers filtering of epics and open issues.
func TestLoadCorpus_SkipsEpicAndOpen(t *testing.T) {
	content := `{"_type":"issue","id":"e-01","title":"epic","description":"big epic","status":"closed","priority":"P0","issue_type":"epic","created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:00:00Z"}
{"_type":"issue","id":"e-02","title":"open issue","description":"not done","status":"open","priority":"P1","issue_type":"task","created_at":"2025-01-02T00:00:00Z","updated_at":"2025-01-02T00:00:00Z"}
{"_type":"issue","id":"e-03","title":"no priority","description":"no pri","status":"closed","priority":"","issue_type":"task","created_at":"2025-01-03T00:00:00Z","updated_at":"2025-01-03T00:00:00Z"}
{"_type":"issue","id":"e-04","title":"prod down","description":"outage critical","status":"closed","priority":"P0","issue_type":"task","created_at":"2025-01-04T00:00:00Z","updated_at":"2025-01-04T00:00:00Z"}
`
	path := t.TempDir() + "/filter.jsonl"
	if err := writeFile(path, content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	train, eval, err := LoadCorpus(path)
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	all := append(train, eval...)
	if len(all) != 1 {
		t.Errorf("expected 1 qualifying issue, got %d", len(all))
	}
	if all[0].ID != "e-04" {
		t.Errorf("expected e-04, got %s", all[0].ID)
	}
}

// TestNew_EmbedError covers error propagation from the embedder.
func TestNew_EmbedError(t *testing.T) {
	// Server that always returns 500.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	e := newCachedEmbedder(srv.URL)
	corpus := []BdIssue{{ID: "x", Title: "prod down", Description: "critical outage", Status: "closed", Priority: "P0"}}
	_, err := New(context.Background(), e, corpus, defaultThreshold)
	if err == nil {
		t.Fatal("expected error from New with failing embedder")
	}
}

// TestRun_EmbedError covers error propagation in Run.
func TestRun_EmbedError(t *testing.T) {
	// Build classifier with a working server first.
	c, srv := buildClassifier(t)
	srv.Close() // close server so subsequent calls fail

	_, _, err := c.Run(context.Background(), BdInput{Title: "prod down", Description: "outage"})
	if err == nil {
		t.Fatal("expected error from Run with closed server")
	}
}

func TestBdSeverityMicro_EvalAccuracy(t *testing.T) {
	srv := newMockOllamaServer(t)
	defer srv.Close()
	e := newCachedEmbedder(srv.URL)

	train, eval, err := LoadCorpus("testdata/sample_issues.jsonl")
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}

	ctx := context.Background()
	c, err := New(ctx, e, train, defaultThreshold)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var total, correct, okCount int
	for _, issue := range eval {
		result, _, err := c.Run(ctx, BdInput{
			Title:       issue.Title,
			Description: issue.Description,
		})
		if err != nil {
			t.Fatalf("Run eval %s: %v", issue.ID, err)
		}
		if result.ConfStatus() != decompose.StatusOK {
			continue
		}
		okCount++
		total++
		if result.Priority == issue.Priority {
			correct++
		}
	}

	if total == 0 {
		t.Fatal("no StatusOK results in eval set — mock embedder may be misconfigured")
	}

	accuracy := float64(correct) / float64(total)
	t.Logf("eval accuracy: %d/%d = %.2f%% (okCount=%d/%d)", correct, total, accuracy*100, okCount, len(eval))

	const minAccuracy = 0.80
	if accuracy < minAccuracy {
		t.Errorf("eval accuracy %.2f%% below required %.2f%%", accuracy*100, minAccuracy*100)
	}
}
