package llmguard

import (
	"fmt"
	"sort"
)

// Reducer merges chunk-level classifier results into a single request verdict.
type Reducer struct {
	threshold float64
	strict    bool
}

// NewReducer creates a reducer from config.
func NewReducer(cfg ClassifierConfig) *Reducer {
	return &Reducer{
		threshold: cfg.BlockConfidenceThreshold,
		strict:    cfg.StrictMode,
	}
}

// ReduceVerdict merges chunk results and returns the request-level outcome.
// deterministicBlocked is true when the deterministic scanner already found
// high-severity secrets that should not be weakened.
func (r *Reducer) ReduceVerdict(chunkResults map[string]*ClassifierResult, failedChunks []string, deterministicBlocked bool) (VerdictState, []SuggestedSpan, string) {
	if len(failedChunks) > 0 {
		if r.strict {
			return VerdictClassifierIncomplete, nil, ""
		}
		if deterministicBlocked {
			return VerdictInputBlocked, nil, ""
		}
		return VerdictClassifierAdvisoryAllowed, nil, ""
	}

	// Escalation priority: block > redact > needs_review > allow.
	var hasBlock, hasRedact, hasNeedsReview bool
	var allSpans []SuggestedSpan
	var categories []ClassifierCategory

	for _, result := range chunkResults {
		action := r.mapAction(result)
		switch action {
		case ActionBlock:
			hasBlock = true
		case ActionRedact:
			hasRedact = true
		case ActionNeedsReview:
			hasNeedsReview = true
		}
		for _, span := range result.SuggestedSpans {
			allSpans = append(allSpans, span)
		}
		for _, cat := range result.Categories {
			categories = append(categories, cat)
		}
	}

	// Deduplicate categories.
	categories = dedupeCategories(categories)

	// If deterministic scanner already blocked, classifier cannot weaken.
	if deterministicBlocked {
		return VerdictInputBlocked, nil, ""
	}

	if hasBlock {
		return VerdictInputBlocked, mergeSpans(allSpans), classifierReason(categories, "block")
	}
	if hasRedact {
		return VerdictRedactedAllowed, mergeSpans(allSpans), classifierReason(categories, "redact")
	}
	if hasNeedsReview {
		if r.strict {
			return VerdictNeedsReview, nil, classifierReason(categories, "needs_review")
		}
		return VerdictClassifierAdvisoryAllowed, nil, classifierReason(categories, "advisory")
	}
	return VerdictCleanAllowed, nil, ""
}

// mapAction applies confidence threshold to block actions.
func (r *Reducer) mapAction(result *ClassifierResult) ClassifierAction {
	if result.Action == ActionBlock && result.Confidence < r.threshold {
		return ActionNeedsReview
	}
	return result.Action
}

func mergeSpans(spans []SuggestedSpan) []SuggestedSpan {
	if len(spans) == 0 {
		return nil
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].Start == spans[j].Start {
			return spans[i].End > spans[j].End
		}
		return spans[i].Start < spans[j].Start
	})
	merged := []SuggestedSpan{spans[0]}
	for i := 1; i < len(spans); i++ {
		last := &merged[len(merged)-1]
		if spans[i].Start <= last.End {
			if spans[i].End > last.End {
				last.End = spans[i].End
			}
			// Keep stronger category if they differ.
			if categoryRank(spans[i].Type) > categoryRank(last.Type) {
				last.Type = spans[i].Type
			}
		} else {
			merged = append(merged, spans[i])
		}
	}
	return merged
}

func categoryRank(c ClassifierCategory) int {
	switch c {
	case CategorySecret, CategoryCredentialExfil:
		return 4
	case CategoryPromptInjection, CategoryPolicyBypass:
		return 3
	case CategoryUnsafeToolRequest:
		return 2
	case CategoryPII:
		return 1
	}
	return 0
}

func dedupeCategories(cats []ClassifierCategory) []ClassifierCategory {
	seen := make(map[ClassifierCategory]bool)
	var out []ClassifierCategory
	for _, c := range cats {
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}

func classifierReason(cats []ClassifierCategory, action string) string {
	if len(cats) == 0 {
		return fmt.Sprintf("classifier %s", action)
	}
	var names []string
	for _, c := range cats {
		names = append(names, string(c))
	}
	return fmt.Sprintf("classifier %s: %v", action, names)
}
