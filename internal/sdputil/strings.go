package sdputil

import "strings"

// NonEmptyStrings filters out empty strings from the input.
func NonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

// FirstNonEmpty returns the first non-empty string from the input.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

// FirstOrEmpty returns the first element of a string slice, or empty string if slice is empty.
func FirstOrEmpty(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

// SplitLines splits a string by newlines and returns non-empty lines.
func SplitLines(s string) []string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	result := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			result = append(result, l)
		}
	}
	return result
}

// MinLen returns the minimum of two integers.
func MinLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}
