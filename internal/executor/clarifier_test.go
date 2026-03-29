package executor

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sdp_dev/internal/control"
)

func TestClarifyIntent_AlreadyClarified(t *testing.T) {
	card := &control.FeatureCard{
		ID:               "card-1",
		NormalizedIntent: "Implement clarifier",
		ScopeIn:          []string{"internal/executor/**"},
		ScopeOut:         []string{"docs/**"},
	}

	result, err := ClarifyIntent(context.Background(), t.TempDir(), "raw intent", card)
	if err != nil {
		t.Fatalf("ClarifyIntent error: %v", err)
	}
	if result.Status != "ready" {
		t.Fatalf("status = %s, want ready", result.Status)
	}
	if result.Card != card {
		t.Fatalf("expected original card pointer to be returned")
	}
}

func TestClarifyIntent_NeedsClarification(t *testing.T) {
	server := newClarifyTestServer(t, `{`+
		`"normalized_intent":"Fix the bug",`+
		`"scope_in":[],`+
		`"scope_out":[],`+
		`"phase":"fix",`+
		`"risk_level":"medium",`+
		`"clarification_needed":true,`+
		`"questions":["Which bug?","Which package should be changed?"],`+
		`"estimated_complexity":"medium"}`)
	defer server.Close()

	t.Setenv("OMO_SERVE_URL", server.URL)
	card := &control.FeatureCard{ID: "card-2", RawRequest: "fix it"}
	result, err := ClarifyIntent(context.Background(), t.TempDir(), card.RawRequest, card)
	if err != nil {
		t.Fatalf("ClarifyIntent error: %v", err)
	}
	if result.Status != "needs_clarification" {
		t.Fatalf("status = %s, want needs_clarification", result.Status)
	}
	if len(result.Questions) != 2 {
		t.Fatalf("questions = %v, want 2 questions", result.Questions)
	}
	if result.Card == nil || !hasBlockingReason(result.Card, clarificationBlockingReason) {
		t.Fatalf("expected card blocking reason to include %q: %+v", clarificationBlockingReason, result.Card)
	}
}

func TestClarifyIntent_CardPopulation(t *testing.T) {
	server := newClarifyTestServer(t, `{`+
		`"normalized_intent":"Add clarifier before build dispatch",`+
		`"scope_in":["internal/executor/**","cmd/sdp/main.go"],`+
		`"scope_out":["internal/deploy/**"],`+
		`"phase":"feature",`+
		`"risk_level":"low",`+
		`"clarification_needed":false,`+
		`"questions":[],`+
		`"estimated_complexity":"high"}`)
	defer server.Close()

	t.Setenv("OMO_SERVE_URL", server.URL)
	card := &control.FeatureCard{ID: "card-3", RawRequest: "add clarifier"}
	result, err := ClarifyIntent(context.Background(), t.TempDir(), card.RawRequest, card)
	if err != nil {
		t.Fatalf("ClarifyIntent error: %v", err)
	}
	if result.Status != "ready" {
		t.Fatalf("status = %s, want ready", result.Status)
	}
	if result.Card == nil {
		t.Fatal("expected populated card")
	}
	if result.Card.NormalizedIntent != "Add clarifier before build dispatch" {
		t.Fatalf("normalized intent = %q", result.Card.NormalizedIntent)
	}
	if result.Card.TaskType != "feature" {
		t.Fatalf("task type = %q, want feature", result.Card.TaskType)
	}
	if result.Card.RiskLevel != "low" {
		t.Fatalf("risk level = %q, want low", result.Card.RiskLevel)
	}
	if got := strings.Join(result.Card.ScopeIn, ","); !strings.Contains(got, "internal/executor/**") {
		t.Fatalf("scope_in = %v", result.Card.ScopeIn)
	}
}

func TestClarifyIntent_OmOUnavailable(t *testing.T) {
	t.Setenv("OMO_SERVE_URL", "http://127.0.0.1:1")

	card := &control.FeatureCard{ID: "card-4", RawRequest: "clarify this intent"}
	result, err := ClarifyIntent(context.Background(), t.TempDir(), card.RawRequest, card)
	if err != nil {
		t.Fatalf("ClarifyIntent error: %v", err)
	}
	if result.Status != "error" {
		t.Fatalf("status = %s, want error", result.Status)
	}
}

func newClarifyTestServer(t *testing.T, payload string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/session":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/session":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"default","project":"test","session":"default"}`))
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/session/"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/session/default/messages":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "event: unknown\ndata: %s\n\n", payload)
			_, _ = fmt.Fprint(w, "event: completion.succeeded\n\n")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}
