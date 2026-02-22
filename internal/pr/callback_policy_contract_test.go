package pr

import "testing"

func TestDefaultCallbackRoutingReliabilityContract(t *testing.T) {
	contract := DefaultCallbackRoutingReliabilityContract()
	if contract.ContractVersion != CallbackRoutingReliabilityContractVersion {
		t.Fatalf("unexpected contract version: got=%s want=%s", contract.ContractVersion, CallbackRoutingReliabilityContractVersion)
	}
	if contract.AckTimeoutS <= 0 {
		t.Fatalf("ack timeout must be positive: %d", contract.AckTimeoutS)
	}
	if contract.MaxRetryWindowS != 900 {
		t.Fatalf("unexpected max retry window: %d", contract.MaxRetryWindowS)
	}
	if !contract.DeadLetterEnabled {
		t.Fatal("dead-letter fallback must be enabled")
	}
	if contract.DeadLetterReason == "" {
		t.Fatal("dead-letter reason must be set")
	}

	if len(contract.RouteFallbackOrder) == 0 {
		t.Fatal("expected route fallback order")
	}
	if contract.RouteFallbackOrder[0] != "issue-owner" {
		t.Fatalf("first fallback route must be issue-owner, got=%s", contract.RouteFallbackOrder[0])
	}

	totalDelay := 0
	prevAttempt := 0
	for _, stage := range contract.RetryBudget {
		if stage.Attempt <= prevAttempt {
			t.Fatalf("retry attempts must be strictly increasing: prev=%d current=%d", prevAttempt, stage.Attempt)
		}
		if stage.DelayS <= 0 {
			t.Fatalf("retry delay must be positive at attempt %d", stage.Attempt)
		}
		if stage.MaxJitterS < 0 {
			t.Fatalf("retry jitter must be non-negative at attempt %d", stage.Attempt)
		}
		if stage.Trigger == "" {
			t.Fatalf("retry trigger required at attempt %d", stage.Attempt)
		}
		totalDelay += stage.DelayS
		prevAttempt = stage.Attempt
	}
	if totalDelay > contract.MaxRetryWindowS {
		t.Fatalf("retry delay budget exceeds max window: total=%d window=%d", totalDelay, contract.MaxRetryWindowS)
	}

	requiredControls := map[string]struct{}{
		"callback.route.mode":             {},
		"callback.retry.profile":          {},
		"callback.notify.watchers":        {},
		"callback.escalate.on.deadletter": {},
	}
	for _, control := range contract.PolicyControls {
		if control.Key == "" {
			t.Fatalf("empty control key: %+v", control)
		}
		if control.DefaultValue == "" {
			t.Fatalf("missing default value for control %s", control.Key)
		}
		if len(control.AllowedValues) == 0 {
			t.Fatalf("missing allowed values for control %s", control.Key)
		}
		delete(requiredControls, control.Key)
	}
	if len(requiredControls) != 0 {
		t.Fatalf("missing required controls: %v", requiredControls)
	}
}

func TestDefaultUserNotificationPolicyContract(t *testing.T) {
	contract := DefaultUserNotificationPolicyContract()
	if contract.ContractVersion != UserNotificationPolicyContractVersion {
		t.Fatalf("unexpected contract version: got=%s want=%s", contract.ContractVersion, UserNotificationPolicyContractVersion)
	}
	if contract.DefaultAudience != "issue-owner" {
		t.Fatalf("unexpected default audience: %s", contract.DefaultAudience)
	}
	if contract.EscalationMode != "dead-letter-notify" {
		t.Fatalf("unexpected escalation mode: %s", contract.EscalationMode)
	}
	if len(contract.RequiredAudiences) != 2 {
		t.Fatalf("unexpected required audience count: %d", len(contract.RequiredAudiences))
	}
	if len(contract.PolicyControls) == 0 {
		t.Fatal("expected notification policy controls")
	}
}

func TestIsRetryableCallbackStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{name: "timeout", statusCode: 408, want: true},
		{name: "too many requests", statusCode: 429, want: true},
		{name: "server error", statusCode: 503, want: true},
		{name: "ok", statusCode: 200, want: false},
		{name: "bad request", statusCode: 400, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsRetryableCallbackStatus(tc.statusCode)
			if got != tc.want {
				t.Fatalf("unexpected retryability for status=%d: got=%t want=%t", tc.statusCode, got, tc.want)
			}
		})
	}
}
