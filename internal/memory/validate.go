package memory

import (
	"errors"
	"strings"
)

var (
	ErrInvalidEntry       = errors.New("invalid memory entry")
	ErrMissingID          = errors.New("missing required field: id")
	ErrMissingActor       = errors.New("missing required field: actor")
	ErrMissingSessionID   = errors.New("missing required field: session_id")
	ErrMissingEntryType   = errors.New("missing required field: entry_type")
	ErrMissingContent     = errors.New("missing required field: content")
	ErrInvalidEntryType   = errors.New("invalid entry type")
	ErrEmptyActor         = errors.New("actor cannot be empty")
	ErrEmptySessionID     = errors.New("session_id cannot be empty")
	ErrEmptyContent       = errors.New("content cannot be empty")
	ErrEmptyID            = errors.New("id cannot be empty")
)

// Valid entry types
var validEntryTypes = map[string]bool{
	"decision":         true,
	"context":          true,
	"phase_transition": true,
	"observation":      true,
}

// ValidateEntry checks if a MemoryEntry has all required fields and valid values.
func ValidateEntry(e *MemoryEntry) error {
	if e == nil {
		return ErrInvalidEntry
	}

	// Check for zero-value entry (equivalent to nil for practical purposes)
	if e.ID == "" && e.Actor == "" && e.SessionID == "" && e.EntryType == "" && e.Content == "" {
		return ErrInvalidEntry
	}

	if e.ID == "" {
		return ErrMissingID
	}
	if strings.TrimSpace(e.ID) == "" {
		return ErrEmptyID
	}

	if e.Actor == "" {
		return ErrMissingActor
	}
	if strings.TrimSpace(e.Actor) == "" {
		return ErrEmptyActor
	}

	if e.SessionID == "" {
		return ErrMissingSessionID
	}
	if strings.TrimSpace(e.SessionID) == "" {
		return ErrEmptySessionID
	}

	if e.EntryType == "" {
		return ErrMissingEntryType
	}
	if !ValidateEntryType(e.EntryType) {
		return ErrInvalidEntryType
	}

	if e.Content == "" {
		return ErrMissingContent
	}
	if strings.TrimSpace(e.Content) == "" {
		return ErrEmptyContent
	}

	return nil
}

// ValidateEntryType checks if the given entry type is valid.
func ValidateEntryType(t string) bool {
	return validEntryTypes[t]
}
