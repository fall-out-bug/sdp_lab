package intake

import (
	"errors"
	"unicode"
)

// ValidateProjectID ensures ProjectID is safe for NATS subject usage.
// Allows ASCII [a-zA-Z0-9_-] only; rejects control chars, '.', '>', '*', spaces, and non-ASCII.
func ValidateProjectID(id string) error {
	if id == "" {
		return nil // Normalize sets default
	}
	if len(id) > 128 {
		return errors.New("project_id too long")
	}
	for _, r := range id {
		if r > unicode.MaxASCII {
			return errors.New("project_id contains invalid characters")
		}
		if r == '.' || r == '>' || r == '*' || r == ' ' || unicode.IsControl(r) {
			return errors.New("project_id contains invalid characters")
		}
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			return errors.New("project_id contains invalid characters")
		}
	}
	return nil
}
