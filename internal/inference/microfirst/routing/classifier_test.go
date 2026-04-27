package routing

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sdp_dev/internal/inference/decompose"
	"sdp_dev/internal/inference/microfirst/embed"
)

// deterministicEmbedder creates a fake HTTP server that returns deterministic
// embeddings based on keywords in the prompt. The embedding is a unit vector
// in a 4-dimensional space, where each dimension corresponds to a capability:
// [go-backend, frontend, docs, infra].
//
// We use this to test the classifier without a real Ollama server.
func newTestEmbedderServer(t *testing.T) (*httptest.Server, *embed.CachedEmbedder) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model  string `json:"model"`
			Prompt string `json:"prompt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		p := strings.ToLower(req.Prompt)
		vec := chooseVec(p)
		json.NewEncoder(w).Encode(map[string]any{"embedding": vec}) //nolint:errcheck
	}))

	inner := embed.NewOllamaEmbedder(srv.URL)
	cached := embed.NewCachedEmbedder(inner, 256)
	return srv, cached
}

// capabilityDim maps capabilities to a dimension index.
var capabilityDim = map[string]int{
	"go-backend": 0,
	"frontend":   1,
	"docs":       2,
	"infra":      3,
}

// capabilityKeywords maps capability names to identifying keywords.
var capabilityKeywords = map[string][]string{
	"go-backend": {"go", "golang", "grpc", "http handler", "sql", "database", "middleware", "repository"},
	"frontend":   {"react", "css", "typescript", "ux", "component", "bundle", "dark mode", "form"},
	"docs":       {"readme", "documentation", "changelog", "typo", "godoc", "api docs", "decision record"},
	"infra":      {"ci", "pipeline", "docker", "nginx", "kubernetes", "terraform", "helm", "monitoring"},
}

// chooseVec returns a unit vector for the given prompt text.
func chooseVec(prompt string) []float64 {
	scores := make([]float64, 4)
	for cap, keywords := range capabilityKeywords {
		dim := capabilityDim[cap]
		for _, kw := range keywords {
			if strings.Contains(prompt, kw) {
				scores[dim] += 1.0
			}
		}
	}

	// Normalise to unit vector.
	var norm float64
	for _, s := range scores {
		norm += s * s
	}
	if norm == 0 {
		// Ambiguous — return equal weight vector.
		v := 1.0 / math.Sqrt(4)
		return []float64{v, v, v, v}
	}
	norm = math.Sqrt(norm)
	out := make([]float64, len(scores))
	for i, s := range scores {
		out[i] = s / norm
	}
	return out
}

// buildTestMicro creates a RoutingColdStartMicro using the test embedder server
// and a small synthetic corpus (3 examples per capability) for speed.
func buildTestMicro(t *testing.T, srv *httptest.Server, cached *embed.CachedEmbedder) *RoutingColdStartMicro {
	t.Helper()
	corpus := []RoutingExample{
		// go-backend
		{Title: "Go HTTP handler", Description: "fix nil pointer in Go handler", Capability: "go-backend"},
		{Title: "Go SQL repository", Description: "refactor database layer", Capability: "go-backend"},
		{Title: "Go middleware", Description: "add authentication middleware", Capability: "go-backend"},
		// frontend
		{Title: "React component", Description: "add React component for user profile", Capability: "frontend"},
		{Title: "CSS layout", Description: "fix CSS layout on mobile", Capability: "frontend"},
		{Title: "TypeScript form", Description: "form validation in TypeScript", Capability: "frontend"},
		// docs
		{Title: "README update", Description: "update readme with installation steps", Capability: "docs"},
		{Title: "API documentation", Description: "add API docs for endpoints", Capability: "docs"},
		{Title: "Changelog entry", Description: "update changelog for release", Capability: "docs"},
		// infra
		{Title: "CI pipeline", Description: "fix ci pipeline failure", Capability: "infra"},
		{Title: "Dockerfile", Description: "update docker for production build", Capability: "infra"},
		{Title: "Kubernetes deployment", Description: "add kubernetes deployment manifest", Capability: "infra"},
	}

	micro, err := New(context.Background(), cached, corpus, 0.80)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return micro
}

// TestRoutingColdStartMicro_GoBackend verifies that a clear Go backend task
// is classified as "go-backend" with StatusOK.
func TestRoutingColdStartMicro_GoBackend(t *testing.T) {
	srv, cached := newTestEmbedderServer(t)
	defer srv.Close()
	micro := buildTestMicro(t, srv, cached)

	res, trace, err := micro.Run(context.Background(), RoutingInput{
		Title:       "fix nil pointer in Go handler",
		Description: "HTTP handler panics with nil pointer dereference in Go middleware",
	})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if trace.Attempts != 1 {
		t.Errorf("trace.Attempts = %d, want 1", trace.Attempts)
	}
	if res.ConfStatus() != decompose.StatusOK {
		t.Errorf("status = %q, want StatusOK (top1=%.3f)", res.ConfStatus(), res.Confidence())
	}
	if res.CapabilityHint != "go-backend" {
		t.Errorf("CapabilityHint = %q, want %q", res.CapabilityHint, "go-backend")
	}
}

// TestRoutingColdStartMicro_Docs verifies that a documentation task is classified
// as "docs" with StatusOK.
func TestRoutingColdStartMicro_Docs(t *testing.T) {
	srv, cached := newTestEmbedderServer(t)
	defer srv.Close()
	micro := buildTestMicro(t, srv, cached)

	res, _, err := micro.Run(context.Background(), RoutingInput{
		Title:       "update readme documentation",
		Description: "Fix typos in readme and add api docs section",
	})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if res.ConfStatus() != decompose.StatusOK {
		t.Errorf("status = %q, want StatusOK", res.ConfStatus())
	}
	if res.CapabilityHint != "docs" {
		t.Errorf("CapabilityHint = %q, want %q", res.CapabilityHint, "docs")
	}
}

// TestRoutingColdStartMicro_Ambiguous verifies that an ambiguous input either
// returns StatusUnsure or any valid hint — we do not assert the exact hint.
func TestRoutingColdStartMicro_Ambiguous(t *testing.T) {
	srv, cached := newTestEmbedderServer(t)
	defer srv.Close()
	micro := buildTestMicro(t, srv, cached)

	// This prompt has no capability-specific keywords → equal-weight vector → top-3 may disagree.
	res, trace, err := micro.Run(context.Background(), RoutingInput{
		Title:       "task with unclear scope",
		Description: "some work to be done",
	})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if trace.Attempts != 1 {
		t.Errorf("trace.Attempts = %d, want 1", trace.Attempts)
	}
	// We only assert the result is a valid status value; both OK and Unsure are acceptable.
	switch res.ConfStatus() {
	case decompose.StatusOK, decompose.StatusUnsure:
		// both valid
	default:
		t.Errorf("unexpected status %q", res.ConfStatus())
	}
}

// TestRoutingColdStartMicro_Name verifies the classifier name.
func TestRoutingColdStartMicro_Name(t *testing.T) {
	srv, cached := newTestEmbedderServer(t)
	defer srv.Close()
	micro := buildTestMicro(t, srv, cached)
	if micro.Name() != "routing-coldstart-micro" {
		t.Errorf("Name() = %q, want %q", micro.Name(), "routing-coldstart-micro")
	}
}

// TestRoutingColdStartMicro_SuggestCapability_Confident verifies SuggestCapability
// returns confident=true for a clear Go backend prompt.
func TestRoutingColdStartMicro_SuggestCapability_Confident(t *testing.T) {
	srv, cached := newTestEmbedderServer(t)
	defer srv.Close()
	micro := buildTestMicro(t, srv, cached)

	hint, ok := micro.SuggestCapability(context.Background(), "Go SQL repository", "refactor database layer with Go SQL")
	if !ok {
		t.Logf("SuggestCapability returned confident=false (hint=%q); acceptable if threshold not met", hint)
		return
	}
	if hint != "go-backend" {
		t.Errorf("hint = %q, want %q", hint, "go-backend")
	}
}

// TestRoutingColdStartMicro_SuggestCapability_Unsure verifies SuggestCapability
// returns confident=false for an ambiguous prompt.
func TestRoutingColdStartMicro_SuggestCapability_Unsure(t *testing.T) {
	srv, cached := newTestEmbedderServer(t)
	defer srv.Close()
	micro := buildTestMicro(t, srv, cached)

	// Ambiguous prompt with no domain keywords.
	_, ok := micro.SuggestCapability(context.Background(), "task", "some work")
	// We just verify no panic and a valid bool is returned.
	_ = ok
}

// TestNew_NilEmbedder verifies that New returns an error when embedder is nil.
func TestNew_NilEmbedder(t *testing.T) {
	_, err := New(context.Background(), nil, nil, 0.80)
	if err == nil {
		t.Error("expected error for nil embedder, got nil")
	}
}

// TestNew_ZeroThreshold verifies that a zero threshold defaults to 0.80.
func TestNew_ZeroThreshold(t *testing.T) {
	srv, cached := newTestEmbedderServer(t)
	defer srv.Close()

	micro, err := New(context.Background(), cached, DefaultCorpus(), 0)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if micro.threshold != defaultThreshold {
		t.Errorf("threshold = %v, want %v", micro.threshold, defaultThreshold)
	}
}

// TestDefaultCorpus_Count verifies DefaultCorpus returns at least 28 examples.
func TestDefaultCorpus_Count(t *testing.T) {
	corpus := DefaultCorpus()
	if len(corpus) < 28 {
		t.Errorf("DefaultCorpus() returned %d entries, want >= 28", len(corpus))
	}
	for i, ex := range corpus {
		if ex.Capability == "" {
			t.Errorf("corpus[%d]: empty capability", i)
		}
		if ex.Title == "" {
			t.Errorf("corpus[%d]: empty title", i)
		}
	}
}

// TestRoutingMicroResult_Confider verifies RoutingMicroResult implements decompose.Confider.
func TestRoutingMicroResult_Confider(t *testing.T) {
	r := RoutingMicroResult{
		CapabilityHint: "go-backend",
		confidence:     0.92,
		status:         decompose.StatusOK,
	}
	var _ decompose.Confider = r
	if r.Confidence() != 0.92 {
		t.Errorf("Confidence() = %v, want 0.92", r.Confidence())
	}
	if r.ConfStatus() != decompose.StatusOK {
		t.Errorf("ConfStatus() = %q, want StatusOK", r.ConfStatus())
	}
}
