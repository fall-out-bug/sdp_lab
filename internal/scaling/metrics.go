package scaling

// Metrics holds queue depth and agent counts for scaling decisions.
type Metrics struct {
	QueueDepth    int
	ActiveAgents  int
	CompletionRate float64
}

// Collector gathers metrics from NATS/bus.
type Collector struct{}

// Collect returns current metrics (placeholder).
func (c *Collector) Collect() Metrics {
	return Metrics{}
}
