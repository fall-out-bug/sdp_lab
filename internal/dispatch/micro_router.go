package dispatch

import "context"

// MicroRouter can suggest a capability hint from task title and description.
// It is implemented by routing.RoutingColdStartMicro.
// nil = disabled (backward-compat).
type MicroRouter interface {
	// SuggestCapability returns a capability hint (e.g. "go-backend") and whether
	// the classifier is confident. When confident=false, the caller should fall
	// through to the configured ColdStartStrategy.
	SuggestCapability(ctx context.Context, title, description string) (hint string, confident bool)
}
