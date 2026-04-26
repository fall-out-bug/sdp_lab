package memory

import (
	"crypto/rand"
	"fmt"
)

// generateUUID creates a random UUID v4 using crypto/rand.
func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("fallback-%d", randInt())
	}

	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// randInt generates a random integer for fallback UUID generation.
func randInt() int {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return 0
	}

	// Use bytes to form an int
	return int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
}
