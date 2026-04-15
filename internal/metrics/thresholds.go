package metrics

// TrafficLight represents a health rating.
type TrafficLight string

const (
	Green  TrafficLight = "green"
	Yellow TrafficLight = "yellow"
	Red    TrafficLight = "red"
)

// RateTicketLinkedRatio rates ticket linkage ratio (>0.7 green, 0.4-0.7 yellow, <0.4 red).
func RateTicketLinkedRatio(r float64) TrafficLight {
	if r > 0.7 {
		return Green
	}
	if r >= 0.4 {
		return Yellow
	}
	return Red
}

// RateFixToFeature rates fix-to-feature ratio (<0.3 green, 0.3-0.5 yellow, >0.5 red).
func RateFixToFeature(r float64) TrafficLight {
	if r < 0.3 {
		return Green
	}
	if r <= 0.5 {
		return Yellow
	}
	return Red
}

// RateChurnRatio rates churn ratio (<0.15 green, 0.15-0.25 yellow, >0.25 red).
func RateChurnRatio(r float64) TrafficLight {
	if r < 0.15 {
		return Green
	}
	if r <= 0.25 {
		return Yellow
	}
	return Red
}

// RateRevertRate rates revert rate (<0.01 green, 0.01-0.03 yellow, >0.03 red).
func RateRevertRate(r float64) TrafficLight {
	if r < 0.01 {
		return Green
	}
	if r <= 0.03 {
		return Yellow
	}
	return Red
}

// RateBusFactor rates minimum module bus factor (>=3 green, 2 yellow, 1 red).
func RateBusFactor(n int) TrafficLight {
	if n >= 3 {
		return Green
	}
	if n == 2 {
		return Yellow
	}
	return Red
}

// RateShotgunRatio rates shotgun surgery ratio (<0.02 green, 0.02-0.05 yellow, >0.05 red).
func RateShotgunRatio(r float64) TrafficLight {
	if r < 0.02 {
		return Green
	}
	if r <= 0.05 {
		return Yellow
	}
	return Red
}
