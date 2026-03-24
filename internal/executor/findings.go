package executor

import (
	"fmt"
	"strings"

	"sdp_dev/internal/control"
)

// RouteFindingsToCard inspects an executor result for failure patterns
// and creates follow-up findings on the source card.
// If execution failed, it creates blocking_reasons and optionally creates a new finding card.
func RouteFindingsToCard(store *control.Store, projectID, cardID string, result *control.ExecutorResultPacket) error {
	if store == nil {
		return fmt.Errorf("nil control store")
	}
	if result == nil {
		return fmt.Errorf("nil executor result packet")
	}

	card, err := store.LoadCard(projectID, cardID)
	if err != nil {
		return fmt.Errorf("load card: %w", err)
	}

	changed := false
	if result.Status == control.ResultStatusFailed {
		card.BlockingReasons = appendUnique(card.BlockingReasons, "execution failed")
		card.ExecutionAttemptCount++
		if card.ExecutionAttemptCount >= 3 {
			card.BlockingReasons = appendUnique(card.BlockingReasons, "max execution attempts reached")
			card.Status = "blocked"
		} else {
			card.RecommendedNextAction = "retry_dispatch"
		}
		changed = true
	}

	if len(result.Findings) > 0 {
		card.AdminActionRequired = appendUnique(card.AdminActionRequired, result.Findings...)
		changed = true
	}

	if !changed {
		return nil
	}
	if err := store.SaveCard(card); err != nil {
		return fmt.Errorf("save card: %w", err)
	}
	return nil
}

func appendUnique(dst []string, values ...string) []string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		found := false
		for _, existing := range dst {
			if existing == value {
				found = true
				break
			}
		}
		if !found {
			dst = append(dst, value)
		}
	}
	if len(dst) == 0 {
		return nil
	}
	return dst
}
