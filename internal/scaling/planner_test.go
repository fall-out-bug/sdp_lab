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

func TestPlanner_Recommend_byDepth(t *testing.T) {
	tests := []struct {
		depth int
		want  int
	}{
		{0, 0}, {1, 1}, {2, 1}, {3, 2}, {5, 2}, {6, 3}, {10, 3}, {11, 5},
	}
	for _, tt := range tests {
		p := &Planner{Collector: &mockCollector{QueueDepth: tt.depth}}
		got := p.Recommend("coder")
		if got != tt.want {
			t.Errorf("Recommend(QueueDepth=%d) = %d, want %d", tt.depth, got, tt.want)
		}
	}
}

func TestPlanner_Recommend_nilCollector(t *testing.T) {
	p := &Planner{}
	if got := p.Recommend("coder"); got != 0 {
		t.Errorf("nil Collector: got %d, want 0", got)
	}
}

type mockCollector struct {
	QueueDepth int
}

func (m *mockCollector) Collect() Metrics {
	return Metrics{QueueDepth: m.QueueDepth}
}
