package replay

import "testing"

func TestPercentileNearestRank(t *testing.T) {
	xs := []int64{10, 20, 30, 40}

	if got := percentile(xs, 0.50); got != 20 {
		t.Fatalf("p50 = %d, want 20", got)
	}
	if got := percentile(xs, 0.95); got != 40 {
		t.Fatalf("p95 = %d, want 40", got)
	}
}
