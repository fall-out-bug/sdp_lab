package scaling

// MetricsProvider returns current metrics for scaling decisions.
type MetricsProvider interface {
	Collect() Metrics
}

// Planner recommends scaling based on queue depth and historical rate.
type Planner struct {
	Collector MetricsProvider
}

// Recommend returns suggested replica count (0-10).
func (p *Planner) Recommend(role string) int {
	if p.Collector == nil {
		return 0
	}
	m := p.Collector.Collect()
	if m.QueueDepth == 0 {
		return 0
	}
	if m.QueueDepth <= 2 {
		return 1
	}
	if m.QueueDepth <= 5 {
		return 2
	}
	if m.QueueDepth <= 10 {
		return 3
	}
	return 5
}
