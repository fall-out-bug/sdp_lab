package runtime

import "math"

type Thresholds struct {
	Yellow int64
	Red    int64
}

type SLO struct {
	LatencyP95WarnMs   float64
	LatencyP95PageMs   float64
	FailureRateWarn    float64
	FailureRatePage    float64
	EscalationRatePage float64
}

type Config struct {
	MaxConcurrent int
	Thresholds    Thresholds
	SLO           SLO
}

type Snapshot struct {
	QueueDepth        int64
	DesiredReplicas   int
	AvailableReplicas int
	LatencyP95Ms      float64
	FailureRate       float64
	EscalationRate    float64
}

type Decision struct {
	ThrottleRatio      float64
	AllowedConcurrent  int
	ShouldDispatch     bool
	TargetReplicas     int
	FailoverEnabled    bool
	LatencyWarn        bool
	LatencyPage        bool
	FailureRateWarn    bool
	FailureRatePage    bool
	EscalationRatePage bool
	StableDegradation  bool
}

type MetricSample struct {
	Name   string
	Value  float64
	Labels map[string]string
}

type QueueController struct {
	config        Config
	previousDepth int64
}

func DefaultConfig() Config {
	return Config{
		MaxConcurrent: 3,
		Thresholds: Thresholds{
			Yellow: 10,
			Red:    50,
		},
		SLO: SLO{
			LatencyP95WarnMs:   1500,
			LatencyP95PageMs:   2500,
			FailureRateWarn:    0.01,
			FailureRatePage:    0.03,
			EscalationRatePage: 0.03,
		},
	}
}

func NewQueueController(config Config) *QueueController {
	if config.MaxConcurrent < 1 {
		config.MaxConcurrent = 1
	}
	if config.Thresholds.Yellow < 1 {
		config.Thresholds.Yellow = 10
	}
	if config.Thresholds.Red <= config.Thresholds.Yellow {
		config.Thresholds.Red = config.Thresholds.Yellow + 1
	}
	if config.SLO.LatencyP95WarnMs <= 0 {
		config.SLO.LatencyP95WarnMs = 1500
	}
	if config.SLO.LatencyP95PageMs <= config.SLO.LatencyP95WarnMs {
		config.SLO.LatencyP95PageMs = config.SLO.LatencyP95WarnMs + 1000
	}
	if config.SLO.FailureRateWarn <= 0 {
		config.SLO.FailureRateWarn = 0.01
	}
	if config.SLO.FailureRatePage <= config.SLO.FailureRateWarn {
		config.SLO.FailureRatePage = config.SLO.FailureRateWarn * 3
	}
	if config.SLO.EscalationRatePage <= 0 {
		config.SLO.EscalationRatePage = 0.03
	}
	return &QueueController{config: config}
}

func (c *QueueController) Evaluate(snapshot Snapshot) Decision {
	throttle := c.throttleRatio(snapshot.QueueDepth)
	allowed := int(math.Round(float64(c.config.MaxConcurrent) * throttle))
	if snapshot.QueueDepth < c.config.Thresholds.Red && allowed < 1 {
		allowed = 1
	}
	if snapshot.QueueDepth >= c.config.Thresholds.Red {
		allowed = 0
	}
	if allowed > c.config.MaxConcurrent {
		allowed = c.config.MaxConcurrent
	}
	if allowed < 0 {
		allowed = 0
	}

	desired := snapshot.DesiredReplicas
	if desired < 1 {
		desired = 1
	}
	available := snapshot.AvailableReplicas
	if available < 0 {
		available = 0
	}

	failoverEnabled := available < desired
	targetReplicas := desired
	if failoverEnabled {
		targetReplicas = desired + 1
	}
	if snapshot.QueueDepth >= c.config.Thresholds.Yellow && targetReplicas < desired+1 {
		targetReplicas = desired + 1
	}

	decision := Decision{
		ThrottleRatio:      throttle,
		AllowedConcurrent:  allowed,
		ShouldDispatch:     allowed > 0,
		TargetReplicas:     targetReplicas,
		FailoverEnabled:    failoverEnabled,
		LatencyWarn:        snapshot.LatencyP95Ms > c.config.SLO.LatencyP95WarnMs,
		LatencyPage:        snapshot.LatencyP95Ms > c.config.SLO.LatencyP95PageMs,
		FailureRateWarn:    snapshot.FailureRate >= c.config.SLO.FailureRateWarn,
		FailureRatePage:    snapshot.FailureRate >= c.config.SLO.FailureRatePage,
		EscalationRatePage: snapshot.EscalationRate >= c.config.SLO.EscalationRatePage,
		StableDegradation:  snapshot.QueueDepth >= c.previousDepth,
	}
	c.previousDepth = snapshot.QueueDepth
	return decision
}

func (c *QueueController) LoadSimulation(depths []int64, base Snapshot) []Decision {
	out := make([]Decision, 0, len(depths))
	for _, depth := range depths {
		base.QueueDepth = depth
		out = append(out, c.Evaluate(base))
	}
	return out
}

func (c *QueueController) SLOMetrics(snapshot Snapshot, decision Decision) []MetricSample {
	return []MetricSample{
		{Name: "sdp_runtime_queue_depth", Value: float64(snapshot.QueueDepth), Labels: map[string]string{}},
		{Name: "sdp_runtime_queue_backpressure_ratio", Value: 1 - decision.ThrottleRatio, Labels: map[string]string{}},
		{Name: "sdp_runtime_queue_allowed_concurrent", Value: float64(decision.AllowedConcurrent), Labels: map[string]string{}},
		{Name: "sdp_runtime_target_replicas", Value: float64(decision.TargetReplicas), Labels: map[string]string{}},
		{Name: "sdp_runtime_latency_p95_ms", Value: snapshot.LatencyP95Ms, Labels: map[string]string{}},
		{Name: "sdp_runtime_failure_rate", Value: snapshot.FailureRate, Labels: map[string]string{}},
		{Name: "sdp_runtime_escalation_rate", Value: snapshot.EscalationRate, Labels: map[string]string{}},
	}
}

func (c *QueueController) throttleRatio(depth int64) float64 {
	if depth < c.config.Thresholds.Yellow {
		return 1
	}
	if depth >= c.config.Thresholds.Red {
		return 0
	}
	rangeSize := float64(c.config.Thresholds.Red - c.config.Thresholds.Yellow)
	position := float64(depth - c.config.Thresholds.Yellow)
	ratio := 1 - (position / rangeSize)
	if ratio < 0.2 {
		return 0.2
	}
	return ratio
}
