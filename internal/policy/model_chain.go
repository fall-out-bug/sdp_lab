package policy

// FallbackChain defines the only allowed runtime model order (model IDs without provider prefix).
var FallbackChain = []string{"glm-5", "glm-4.7"}

// ResolveModelWithProvider returns model for routing. If input has provider prefix (e.g. openrouter/gpt-4o),
// it is preserved; otherwise the model ID is used with default provider.
func ResolveModelWithProvider(input string) string {
	provider, model := ParseProviderModel(input)
	if model == "" {
		return input
	}
	if provider != "" {
		return input // keep full provider/model
	}
	return model
}

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
