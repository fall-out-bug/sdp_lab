package decompose

import "fmt"

// typeName returns a human-readable type descriptor for runtime error messages.
func typeName(v any) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%T", v)
}
