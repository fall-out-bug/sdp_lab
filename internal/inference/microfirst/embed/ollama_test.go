package embed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockEmbedHandler returns a handler that always yields the given vector.
func mockEmbedHandler(vec []float64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
			return
		}
		resp := embedResponse{Embedding: vec}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func TestOllamaEmbedder_Embed_Success(t *testing.T) {
	want := []float64{0.1, 0.2, 0.3}
	srv := httptest.NewServer(mockEmbedHandler(want))
	defer srv.Close()

	e := NewOllamaEmbedder(srv.URL)
	got, err := e.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %f want %f", i, got[i], want[i])
		}
	}
}

func TestOllamaEmbedder_Embed_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	e := NewOllamaEmbedder(srv.URL)
	_, err := e.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for HTTP 500, got nil")
	}
}

func TestOllamaEmbedder_Embed_HTTP4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	e := NewOllamaEmbedder(srv.URL)
	_, err := e.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for HTTP 400, got nil")
	}
}

func TestCachedEmbedder_CacheHit(t *testing.T) {
	callCount := 0
	vec := []float64{1.0, 0.0}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := embedResponse{Embedding: vec}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	inner := NewOllamaEmbedder(srv.URL)
	cached := NewCachedEmbedder(inner, 10)

	ctx := context.Background()

	// First call: miss
	got1, err := cached.Embed(ctx, "test text")
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Second call: should be cache hit, no HTTP call
	got2, err := cached.Embed(ctx, "test text")
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", callCount)
	}

	for i := range got1 {
		if got1[i] != got2[i] {
			t.Errorf("index %d: first=%f second=%f", i, got1[i], got2[i])
		}
	}

	hits, misses := cached.Stats()
	if hits != 1 {
		t.Errorf("expected 1 hit, got %d", hits)
	}
	if misses != 1 {
		t.Errorf("expected 1 miss, got %d", misses)
	}
}

func TestCachedEmbedder_Eviction(t *testing.T) {
	vec := []float64{0.5, 0.5}
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := embedResponse{Embedding: vec}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	inner := NewOllamaEmbedder(srv.URL)
	cached := NewCachedEmbedder(inner, 2) // capacity 2

	ctx := context.Background()

	// Fill cache with 2 entries
	_, _ = cached.Embed(ctx, "a")
	_, _ = cached.Embed(ctx, "b")
	// Overflow: "a" should be evicted
	_, _ = cached.Embed(ctx, "c")
	// Fetching "a" again should cause another HTTP call (evicted)
	_, _ = cached.Embed(ctx, "a")

	// Expected: "a", "b", "c", "a" → 4 HTTP calls (a evicted before last fetch)
	if callCount != 4 {
		t.Errorf("expected 4 HTTP calls after eviction, got %d", callCount)
	}
}
