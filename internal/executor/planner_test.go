package executor

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"sdp_dev/internal/control"
)

func TestGeneratePlan_AlreadyApproved(t *testing.T) {
	card := &control.FeatureCard{
		ID:                   "card-approved-1",
		NormalizedIntent:     "Implement planner",
		ExecutorRuntimeState: "plan-approved",
	}

	result, err := GeneratePlan(context.Background(), t.TempDir(), card, DefaultPlannerConfig())
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if result.Status != "approved" {
		t.Fatalf("status = %s, want approved", result.Status)
	}
}

func TestGeneratePlan_AlreadyApprovedByBlockingReason(t *testing.T) {
	card := &control.FeatureCard{
		ID:               "card-approved-2",
		NormalizedIntent: "Implement planner",
	}

	// Create plan.json to simulate existing plan
	tmpDir := t.TempDir()
	planPath := tmpDir + "/.sdp/artifacts/card-approved-2/plan.json"
	_ = os.MkdirAll(tmpDir+"/.sdp/artifacts/card-approved-2", 0o755)
	existingPlan := `{"card_id":"card-approved-2","status":"approved","approach":"test approach"}`
	_ = os.WriteFile(planPath, []byte(existingPlan), 0o644)

	// Card has no blocking reason for plan_pending_approval, so it's considered approved
	result, err := GeneratePlan(context.Background(), tmpDir, card, DefaultPlannerConfig())
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if result.Status != "approved" {
		t.Fatalf("status = %s, want approved", result.Status)
	}
}

func TestGeneratePlan_LowRiskAutoApprove(t *testing.T) {
	server := newPlanTestServer(t, `{"approach":"Add new files for planner","files_to_change":["internal/executor/planner.go","internal/executor/plan_prompt.go"],"tests_to_write":["internal/executor/planner_test.go"],"risk_assessment":"Low risk - new files only","estimated_steps":5}`)
	defer server.Close()

	t.Setenv("OMO_SERVE_URL", server.URL)
	card := &control.FeatureCard{
		ID:               "card-low-risk",
		RawRequest:       "implement planner",
		NormalizedIntent: "Implement planner before build",
		RiskLevel:        "low",
		BlockingReasons:  []string{planPendingBlockingReason},
	}

	result, err := GeneratePlan(context.Background(), t.TempDir(), card, DefaultPlannerConfig())
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if result.Status != "approved" {
		t.Fatalf("status = %s, want approved (auto-approved for low risk)", result.Status)
	}
	if result.Approach != "Add new files for planner" {
		t.Fatalf("approach = %q", result.Approach)
	}
	if len(result.FilesToChange) != 2 {
		t.Fatalf("files_to_change = %v, want 2 files", result.FilesToChange)
	}
	if result.EstimatedSteps != 5 {
		t.Fatalf("estimated_steps = %d, want 5", result.EstimatedSteps)
	}
}

func TestGeneratePlan_HighRiskPending(t *testing.T) {
	server := newPlanTestServer(t, `{"approach":"Modify critical auth flow","files_to_change":["internal/auth/handler.go","internal/auth/middleware.go"],"tests_to_write":["internal/auth/handler_test.go"],"risk_assessment":"High risk - authentication changes","estimated_steps":12}`)
	defer server.Close()

	t.Setenv("OMO_SERVE_URL", server.URL)
	card := &control.FeatureCard{
		ID:               "card-high-risk",
		RawRequest:       "modify auth",
		NormalizedIntent: "Modify authentication flow",
		RiskLevel:        "high",
		BlockingReasons:  []string{planPendingBlockingReason},
	}

	result, err := GeneratePlan(context.Background(), t.TempDir(), card, DefaultPlannerConfig())
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if result.Status != "pending_approval" {
		t.Fatalf("status = %s, want pending_approval (high risk requires approval)", result.Status)
	}
	if !result.ApprovalPending {
		t.Fatal("approval_pending should be true for high risk")
	}
	if result.Approach != "Modify critical auth flow" {
		t.Fatalf("approach = %q", result.Approach)
	}
}

func TestGeneratePlan_MediumRiskPending(t *testing.T) {
	server := newPlanTestServer(t, `{"approach":"Refactor database layer","files_to_change":["internal/db/queries.go"],"tests_to_write":["internal/db/queries_test.go"],"risk_assessment":"Medium risk - database changes","estimated_steps":8}`)
	defer server.Close()

	t.Setenv("OMO_SERVE_URL", server.URL)
	card := &control.FeatureCard{
		ID:               "card-medium-risk",
		RawRequest:       "refactor db",
		NormalizedIntent: "Refactor database layer",
		RiskLevel:        "medium",
		BlockingReasons:  []string{planPendingBlockingReason},
	}

	result, err := GeneratePlan(context.Background(), t.TempDir(), card, DefaultPlannerConfig())
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if result.Status != "pending_approval" {
		t.Fatalf("status = %s, want pending_approval (medium risk requires approval)", result.Status)
	}
}

func TestGeneratePlan_OmOUnavailable(t *testing.T) {
	t.Setenv("OMO_SERVE_URL", "http://127.0.0.1:1")

	card := &control.FeatureCard{
		ID:               "card-omo-unavailable",
		RawRequest:       "plan this",
		NormalizedIntent: "Plan implementation",
		RiskLevel:        "low",
		BlockingReasons:  []string{planPendingBlockingReason},
	}

	result, err := GeneratePlan(context.Background(), t.TempDir(), card, DefaultPlannerConfig())
	if err == nil {
		t.Fatal("expected error when OmO serve is unavailable")
	}
	if !strings.Contains(err.Error(), "OmO serve unavailable") {
		t.Fatalf("error should mention OmO serve unavailable: %v", err)
	}
	if result.Status != "error" {
		t.Fatalf("status = %s, want error", result.Status)
	}
}

func TestGeneratePlan_Disabled(t *testing.T) {
	card := &control.FeatureCard{
		ID:               "card-disabled",
		RawRequest:       "plan this",
		NormalizedIntent: "Plan implementation",
		BlockingReasons:  []string{planPendingBlockingReason},
	}

	cfg := PlannerConfig{Enabled: false}
	result, err := GeneratePlan(context.Background(), t.TempDir(), card, cfg)
	if err == nil {
		t.Fatal("expected error when planner is disabled")
	}
	if !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("error should mention disabled: %v", err)
	}
	if result.Status != "error" {
		t.Fatalf("status = %s, want error", result.Status)
	}
}

func TestGeneratePlan_NilCard(t *testing.T) {
	result, err := GeneratePlan(context.Background(), t.TempDir(), nil, DefaultPlannerConfig())
	if err == nil {
		t.Fatal("expected error for nil card")
	}
	if result.Status != "error" {
		t.Fatalf("status = %s, want error", result.Status)
	}
}

func TestIsAutoApproveRisk(t *testing.T) {
	tests := []struct {
		riskLevel   string
		autoApprove []string
		want        bool
	}{
		{"low", []string{"low"}, true},
		{"LOW", []string{"low"}, true},
		{"low", []string{"LOW"}, true},
		{"medium", []string{"low"}, false},
		{"high", []string{"low"}, false},
		{"low", []string{"low", "medium"}, true},
		{"medium", []string{"low", "medium"}, true},
		{"high", []string{"low", "medium"}, false},
		{"", []string{"low"}, false},
		{"low", []string{}, false},
		{"low", nil, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("risk=%s_auto=%v", tt.riskLevel, tt.autoApprove), func(t *testing.T) {
			got := isAutoApproveRisk(tt.riskLevel, tt.autoApprove)
			if got != tt.want {
				t.Errorf("isAutoApproveRisk(%q, %v) = %v, want %v", tt.riskLevel, tt.autoApprove, got, tt.want)
			}
		})
	}
}

func newPlanTestServer(t *testing.T, payload string) *httptest.Server {
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
