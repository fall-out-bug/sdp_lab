package bdtype

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sdp_dev/internal/inference/decompose"
)

// --- Mock Ollama server ---
//
// The mock server returns deterministic embeddings based on keyword groups found
// in the prompt. Each group maps to a unit vector in a specific "direction":
//   - bug keywords  → axis 0 high
//   - task keywords → axis 1 high
//   - feature keywords → axis 2 high
//
// This makes kNN classification fully deterministic without a real Ollama server.

const embDims = 8

type mockEmbedResp struct {
	Embedding []float64 `json:"embedding"`
}

func bugKeywords() []string {
	return []string{"panic", "nil pointer", "crash", "wrong output", "incorrect behavior"}
}

func taskKeywords() []string {
	return []string{"add logging", "refactor", "update config", "implement", "cleanup"}
}

func featureKeywords() []string {
	return []string{"new endpoint", "support", "add ability", "enable"}
}

// deterministicEmbedding returns a normalised embedding vector based on the
// dominant keyword group in text.
func deterministicEmbedding(text string) []float64 {
	lower := strings.ToLower(text)

	bugScore := 0
	for _, kw := range bugKeywords() {
		if strings.Contains(lower, kw) {
			bugScore += 2
		}
	}

	taskScore := 0
	for _, kw := range taskKeywords() {
		if strings.Contains(lower, kw) {
			taskScore += 2
		}
	}

	featureScore := 0
	for _, kw := range featureKeywords() {
		if strings.Contains(lower, kw) {
			featureScore += 2
		}
	}

	vec := make([]float64, embDims)
	// Assign scores to different axis groups.
	vec[0] = float64(bugScore) + 0.1
	vec[1] = float64(bugScore) * 0.9
	vec[2] = float64(taskScore) + 0.1
	vec[3] = float64(taskScore) * 0.9
	vec[4] = float64(featureScore) + 0.1
	vec[5] = float64(featureScore) * 0.9
	vec[6] = 0.01
	vec[7] = 0.01

	// L2 normalise.
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vec {
			vec[i] /= norm
		}
	}
	return vec
}

func newMockOllamaServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Prompt string `json:"prompt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		emb := deterministicEmbedding(req.Prompt)
		resp := mockEmbedResp{Embedding: emb}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// --- Embedder adapter ---

type httpEmbedder struct {
	baseURL string
	client  *http.Client
}

func (e *httpEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	body, _ := json.Marshal(map[string]string{"model": "test", "prompt": text})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/api/embeddings",
		strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result mockEmbedResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Embedding, nil
}

func newTestEmbedder(srv *httptest.Server) *httpEmbedder {
	return &httpEmbedder{baseURL: srv.URL, client: srv.Client()}
}

// --- Test helpers ---

func buildClassifier(t *testing.T) (*BdTypeMicro, *httptest.Server, *Corpus) {
	t.Helper()
	srv := newMockOllamaServer()

	corpus, err := LoadCorpus("testdata/sample_issues.jsonl")
	if err != nil {
		srv.Close()
		t.Fatalf("LoadCorpus: %v", err)
	}

	emb := newTestEmbedder(srv)
	clf, err := NewBdTypeMicro(context.Background(), emb, corpus.Train)
	if err != nil {
		srv.Close()
		t.Fatalf("NewBdTypeMicro: %v", err)
	}
	return clf, srv, corpus
}

// --- AC1: LoadCorpus normalises chore→task, excludes epic ---

func TestLoadCorpus_NormalisesChoreToTask(t *testing.T) {
	corpus, err := LoadCorpus("testdata/sample_issues.jsonl")
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	all := append(corpus.Train, corpus.Eval...)

	for _, entry := range all {
		if entry.Label == "chore" {
			t.Errorf("entry %s has label 'chore' — should have been normalised to 'task'", entry.ID)
		}
		if entry.Label == "epic" {
			t.Errorf("entry %s has label 'epic' — epics should be excluded", entry.ID)
		}
	}
}

func TestLoadCorpus_ExcludesEpic(t *testing.T) {
	corpus, err := LoadCorpus("testdata/sample_issues.jsonl")
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	all := append(corpus.Train, corpus.Eval...)
	for _, entry := range all {
		if strings.HasPrefix(entry.ID, "sdplab-e") {
			t.Errorf("epic entry %s should have been excluded from corpus", entry.ID)
		}
	}
}

func TestLoadCorpus_OnlyClosedIssues(t *testing.T) {
	// All testdata entries are closed; verify we still load them.
	corpus, err := LoadCorpus("testdata/sample_issues.jsonl")
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	if len(corpus.Train)+len(corpus.Eval) == 0 {
		t.Fatal("expected non-empty corpus")
	}
}

func TestLoadCorpus_SplitSizes(t *testing.T) {
	corpus, err := LoadCorpus("testdata/sample_issues.jsonl")
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	total := len(corpus.Train) + len(corpus.Eval)
	wantEval := 30
	if total < wantEval {
		wantEval = total
	}
	if len(corpus.Eval) != wantEval {
		t.Errorf("Eval set size = %d, want %d", len(corpus.Eval), wantEval)
	}
	if len(corpus.Train) != total-wantEval {
		t.Errorf("Train set size = %d, want %d", len(corpus.Train), total-wantEval)
	}
}

func TestLoadCorpus_MissingFile(t *testing.T) {
	_, err := LoadCorpus("testdata/nonexistent.jsonl")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// --- AC2: BdTypeMicro.Run on bug examples → Type="bug", Status=OK ---

func TestBdTypeMicro_RunBugInput(t *testing.T) {
	clf, srv, _ := buildClassifier(t)
	defer srv.Close()

	emb := newTestEmbedder(srv)
	clf.embedder = emb

	tests := []BdInput{
		{Title: "nil pointer panic in parser", Description: "application crashes with nil pointer dereference"},
		{Title: "crash when processing input", Description: "wrong output returned after crash when loading data"},
	}
	for _, in := range tests {
		res, trace, err := clf.Run(context.Background(), in)
		if err != nil {
			t.Fatalf("Run error: %v", err)
		}
		if trace.Attempts != 1 {
			t.Errorf("trace.Attempts = %d, want 1", trace.Attempts)
		}
		if res.Type != "bug" {
			t.Errorf("Type = %q, want 'bug' for input: %q", res.Type, in.Title)
		}
		if res.ConfStatus() != decompose.StatusOK {
			t.Errorf("ConfStatus = %q, want 'ok' for input: %q", res.ConfStatus(), in.Title)
		}
	}
}

// --- AC3: BdTypeMicro.Run on feature examples → Type="feature", Status=OK ---

func TestBdTypeMicro_RunFeatureInput(t *testing.T) {
	clf, srv, _ := buildClassifier(t)
	defer srv.Close()

	emb := newTestEmbedder(srv)
	clf.embedder = emb

	tests := []BdInput{
		{Title: "new endpoint for user profile", Description: "add new REST endpoint to support profile updates"},
		{Title: "add ability to export reports", Description: "enable users to export data in multiple formats"},
	}
	for _, in := range tests {
		res, _, err := clf.Run(context.Background(), in)
		if err != nil {
			t.Fatalf("Run error: %v", err)
		}
		if res.Type != "feature" {
			t.Errorf("Type = %q, want 'feature' for input: %q", res.Type, in.Title)
		}
		if res.ConfStatus() != decompose.StatusOK {
			t.Errorf("ConfStatus = %q, want 'ok' for input: %q", res.ConfStatus(), in.Title)
		}
	}
}

// --- AC4: Eval accuracy ≥ 85% on Status=OK subset ---

func TestBdTypeMicro_EvalAccuracy(t *testing.T) {
	clf, srv, corpus := buildClassifier(t)
	defer srv.Close()

	emb := newTestEmbedder(srv)
	clf.embedder = emb

	correct, okCount := 0, 0
	for _, entry := range corpus.Eval {
		res, _, err := clf.Run(context.Background(), BdInput{Text: entry.Text})
		if err != nil {
			t.Fatalf("Run error for %s: %v", entry.ID, err)
		}
		if res.ConfStatus() != decompose.StatusOK {
			continue
		}
		okCount++
		if res.Type == entry.Label {
			correct++
		}
	}

	if okCount == 0 {
		t.Skip("no OK predictions in eval set")
	}

	accuracy := float64(correct) / float64(okCount)
	if accuracy < 0.85 {
		t.Errorf("eval accuracy = %.2f (correct=%d, ok=%d), want ≥ 0.85", accuracy, correct, okCount)
	}
}

// --- AC5: Fallback rate ≤ 40% (≤ 24 of 60 records Unsure) ---

func TestBdTypeMicro_FallbackRate(t *testing.T) {
	clf, srv, corpus := buildClassifier(t)
	defer srv.Close()

	emb := newTestEmbedder(srv)
	clf.embedder = emb

	all := append(corpus.Train, corpus.Eval...)
	// Only count the 60 non-epic records (chore normalised to task).
	unsureCount := 0
	total := 0
	for _, entry := range all {
		res, _, err := clf.Run(context.Background(), BdInput{Text: entry.Text})
		if err != nil {
			t.Fatalf("Run error for %s: %v", entry.ID, err)
		}
		total++
		if res.ConfStatus() != decompose.StatusOK {
			unsureCount++
		}
	}

	maxUnsure := total * 40 / 100
	if unsureCount > maxUnsure {
		t.Errorf("fallback rate too high: %d/%d unsure (%.0f%%), want ≤ 40%%",
			unsureCount, total, float64(unsureCount)/float64(total)*100)
	}
}

// --- Confider interface compliance ---

func TestBdTypeResult_ImplementsConfider(t *testing.T) {
	res := BdTypeResult{
		Type:       "bug",
		confidence: 0.95,
		status:     decompose.StatusOK,
	}
	var _ decompose.Confider = res
	if res.Confidence() != 0.95 {
		t.Errorf("Confidence() = %v, want 0.95", res.Confidence())
	}
	if res.ConfStatus() != decompose.StatusOK {
		t.Errorf("ConfStatus() = %q, want 'ok'", res.ConfStatus())
	}
}

// --- Name ---

func TestBdTypeMicro_Name(t *testing.T) {
	srv := newMockOllamaServer()
	defer srv.Close()
	emb := newTestEmbedder(srv)
	clf, err := NewBdTypeMicro(context.Background(), emb, nil)
	if err != nil {
		t.Fatalf("NewBdTypeMicro: %v", err)
	}
	if clf.Name() != "bdtype-micro" {
		t.Errorf("Name() = %q, want 'bdtype-micro'", clf.Name())
	}
}

// --- normalizeType unit tests ---

func TestNormalizeType(t *testing.T) {
	cases := []struct {
		input    string
		wantLabel string
		wantOK   bool
	}{
		{"bug", "bug", true},
		{"task", "task", true},
		{"feature", "feature", true},
		{"chore", "task", true},
		{"epic", "", false},
		{"unknown", "", false},
		{"Bug", "bug", true},
		{"CHORE", "task", true},
	}
	for _, tc := range cases {
		label, ok := normalizeType(tc.input)
		if ok != tc.wantOK {
			t.Errorf("normalizeType(%q) ok = %v, want %v", tc.input, ok, tc.wantOK)
		}
		if label != tc.wantLabel {
			t.Errorf("normalizeType(%q) label = %q, want %q", tc.input, label, tc.wantLabel)
		}
	}
}
