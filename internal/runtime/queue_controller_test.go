package runtime

import "testing"

func TestEvaluateAdaptiveThrottling(t *testing.T) {
	controller := NewQueueController(DefaultConfig())
	base := Snapshot{DesiredReplicas: 2, AvailableReplicas: 2, LatencyP95Ms: 100, FailureRate: 0.001}

	tests := []struct {
		name             string
		depth            int64
		wantAllowed      int
		wantDispatch     bool
		wantBackpressure bool
	}{
		{name: "below yellow", depth: 5, wantAllowed: 3, wantDispatch: true, wantBackpressure: false},
		{name: "at yellow", depth: 10, wantAllowed: 3, wantDispatch: true, wantBackpressure: false},
		{name: "mid range", depth: 30, wantAllowed: 2, wantDispatch: true, wantBackpressure: true},
		{name: "at red", depth: 50, wantAllowed: 0, wantDispatch: false, wantBackpressure: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base.QueueDepth = tt.depth
			decision := controller.Evaluate(base)
			if decision.AllowedConcurrent != tt.wantAllowed {
				t.Fatalf("AllowedConcurrent=%d want=%d", decision.AllowedConcurrent, tt.wantAllowed)
			}
			if decision.ShouldDispatch != tt.wantDispatch {
				t.Fatalf("ShouldDispatch=%v want=%v", decision.ShouldDispatch, tt.wantDispatch)
			}
			backpressure := decision.ThrottleRatio < 1
			if backpressure != tt.wantBackpressure {
				t.Fatalf("backpressure=%v want=%v", backpressure, tt.wantBackpressure)
			}
		})
	}
}

func TestEvaluateNPlusOneFailover(t *testing.T) {
	controller := NewQueueController(DefaultConfig())

	tests := []struct {
		name         string
		snapshot     Snapshot
		wantFailover bool
		wantReplicas int
	}{
		{
			name:         "healthy replicas",
			snapshot:     Snapshot{QueueDepth: 0, DesiredReplicas: 3, AvailableReplicas: 3},
			wantFailover: false,
			wantReplicas: 3,
		},
		{
			name:         "failover when one replica lost",
			snapshot:     Snapshot{QueueDepth: 15, DesiredReplicas: 3, AvailableReplicas: 2},
			wantFailover: true,
			wantReplicas: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := controller.Evaluate(tt.snapshot)
			if decision.FailoverEnabled != tt.wantFailover {
				t.Fatalf("FailoverEnabled=%v want=%v", decision.FailoverEnabled, tt.wantFailover)
			}
			if decision.TargetReplicas != tt.wantReplicas {
				t.Fatalf("TargetReplicas=%d want=%d", decision.TargetReplicas, tt.wantReplicas)
			}
		})
	}
}

func TestEvaluateSLOSignals(t *testing.T) {
	controller := NewQueueController(DefaultConfig())

	decision := controller.Evaluate(Snapshot{
		QueueDepth:        20,
		DesiredReplicas:   2,
		AvailableReplicas: 2,
		LatencyP95Ms:      2600,
		FailureRate:       0.04,
		EscalationRate:    0.05,
	})

	if !decision.LatencyWarn || !decision.LatencyPage {
		t.Fatalf("expected latency warn+page true, got warn=%v page=%v", decision.LatencyWarn, decision.LatencyPage)
	}
	if !decision.FailureRateWarn || !decision.FailureRatePage {
		t.Fatalf("expected failure warn+page true, got warn=%v page=%v", decision.FailureRateWarn, decision.FailureRatePage)
	}
	if !decision.EscalationRatePage {
		t.Fatal("expected escalation page signal true")
	}
}

func TestLoadSimulationStableDegradation(t *testing.T) {
	controller := NewQueueController(DefaultConfig())
	decisions := controller.LoadSimulation(
		[]int64{0, 5, 10, 20, 30, 40, 49, 50, 60},
		Snapshot{DesiredReplicas: 2, AvailableReplicas: 2},
	)

	if len(decisions) != 9 {
		t.Fatalf("len(decisions)=%d want=9", len(decisions))
	}

	prevAllowed := decisions[0].AllowedConcurrent
	for i := 1; i < len(decisions); i++ {
		if decisions[i].AllowedConcurrent > prevAllowed {
			t.Fatalf("allowed concurrent increased at step %d: prev=%d current=%d", i, prevAllowed, decisions[i].AllowedConcurrent)
		}
		prevAllowed = decisions[i].AllowedConcurrent
	}

	if decisions[len(decisions)-1].AllowedConcurrent != 0 {
		t.Fatalf("expected final allowed concurrent=0, got=%d", decisions[len(decisions)-1].AllowedConcurrent)
	}
	if decisions[len(decisions)-1].ShouldDispatch {
		t.Fatal("expected dispatch blocked at final overload step")
	}
}

func TestSLOMetricsExportIncludesLatencyAndFailure(t *testing.T) {
	controller := NewQueueController(DefaultConfig())
	s := Snapshot{QueueDepth: 12, DesiredReplicas: 2, AvailableReplicas: 2, LatencyP95Ms: 1200, FailureRate: 0.02, EscalationRate: 0.01}
	d := controller.Evaluate(s)
	metrics := controller.SLOMetrics(s, d)

	foundLatency := false
	foundFailure := false
	for _, m := range metrics {
		if m.Name == "sdp_runtime_latency_p95_ms" {
			foundLatency = true
		}
		if m.Name == "sdp_runtime_failure_rate" {
			foundFailure = true
		}
	}

	if !foundLatency || !foundFailure {
		t.Fatalf("expected latency and failure metrics, got latency=%v failure=%v", foundLatency, foundFailure)
	}
}
