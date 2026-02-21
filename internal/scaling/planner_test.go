package scaling

import (
	"testing"
)

func TestPlanner_Recommend(t *testing.T) {
	p := &Planner{Collector: &Collector{}}
	// Default Collector returns Metrics{QueueDepth:0}, so Recommend returns 0
	got := p.Recommend("coder")
	if got != 0 {
		t.Errorf("Recommend(QueueDepth=0) = %d, want 0", got)
	}
}
