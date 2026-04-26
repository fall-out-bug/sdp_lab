package memory

import (
	"errors"
	"time"
)

var (
	ErrInvalidLimit = errors.New("limit must be positive")
)

// FilterOpts provides filtering options for querying memory entries.
type FilterOpts struct {
	Actor     string
	SessionID string
	FeatureID string
	EntryType string
	Since     *time.Time
	Until     *time.Time
	Limit     int
}

// Query retrieves entries from the store that match the given filters.
func Query(store *Store, opts FilterOpts) ([]MemoryEntry, error) {
	if opts.Limit < 0 {
		return nil, ErrInvalidLimit
	}

	entries, err := store.ReadAll()
	if err != nil {
		return nil, err
	}

	var filtered []MemoryEntry
	count := 0

	for _, entry := range entries {
		// Check limit
		if opts.Limit > 0 && count >= opts.Limit {
			break
		}

		// Filter by actor
		if opts.Actor != "" && entry.Actor != opts.Actor {
			continue
		}

		// Filter by session ID
		if opts.SessionID != "" && entry.SessionID != opts.SessionID {
			continue
		}

		// Filter by feature ID
		if opts.FeatureID != "" && entry.FeatureID != opts.FeatureID {
			continue
		}

		// Filter by entry type
		if opts.EntryType != "" && entry.EntryType != opts.EntryType {
			continue
		}

		// Filter by time range
		if opts.Since != nil && entry.Timestamp.Before(*opts.Since) {
			continue
		}

		if opts.Until != nil && entry.Timestamp.After(*opts.Until) {
			continue
		}

		filtered = append(filtered, entry)
		count++
	}

	return filtered, nil
}

// Recent retrieves the n most recent entries from the store.
func Recent(store *Store, n int) ([]MemoryEntry, error) {
	if n <= 0 {
		return nil, ErrInvalidLimit
	}

	entries, err := store.ReadAll()
	if err != nil {
		return nil, err
	}

	// Get last n entries
	if len(entries) <= n {
		return entries, nil
	}

	return entries[len(entries)-n:], nil
}
