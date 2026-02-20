package policy

// FallbackChain defines the only allowed runtime model order.
var FallbackChain = []string{"glm-5", "glm-4.7"}

// NextFallback returns the next model in fallback order.
// If no further fallback exists, it returns "" and false.
func NextFallback(current string) (string, bool) {
	for idx, model := range FallbackChain {
		if model != current {
			continue
		}
		next := idx + 1
		if next >= len(FallbackChain) {
			return "", false
		}
		return FallbackChain[next], true
	}
	return "", false
}

// ResolveFallbackSequence returns the runtime sequence ending with "escalated".
func ResolveFallbackSequence(start string) []string {
	if start == "" || !AllowedModel(start) {
		start = DefaultModel()
	}
	sequence := []string{start}
	current := start
	for {
		next, ok := NextFallback(current)
		if !ok {
			break
		}
		sequence = append(sequence, next)
		current = next
	}
	sequence = append(sequence, "escalated")
	return sequence
}
