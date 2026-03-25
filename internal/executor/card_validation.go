package executor

import (
	"errors"
	"strings"

	"sdp_dev/internal/control"
)

// ErrMissingRequiredField is returned when a required field is missing or empty
var ErrMissingRequiredField = errors.New("missing required field")

// ValidateFeatureCard checks that a FeatureCard has all required fields populated.
// Required fields: ID, Title, NormalizedIntent
// Returns nil if all required fields are present and non-empty.
func ValidateFeatureCard(card *control.FeatureCard) error {
	if card == nil {
		return errors.New("card is nil")
	}

	missing := []string{}
	if strings.TrimSpace(card.ID) == "" {
		missing = append(missing, "ID")
	}
	if strings.TrimSpace(card.Title) == "" {
		missing = append(missing, "Title")
	}
	if strings.TrimSpace(card.NormalizedIntent) == "" {
		missing = append(missing, "NormalizedIntent")
	}

	if len(missing) > 0 {
		return &ValidationError{
			MissingFields: missing,
			Err:           ErrMissingRequiredField,
		}
	}

	return nil
}

// ValidationError describes which fields are missing from a FeatureCard
type ValidationError struct {
	MissingFields []string
	Err           error
}

func (e *ValidationError) Error() string {
	return e.Err.Error() + ": " + strings.Join(e.MissingFields, ", ")
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}
