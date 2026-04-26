package decompose

import "fmt"

// EnumStitcher validates that a stage output string belongs to a fixed
// closed-set of allowed values (case-sensitive). Cheapest stitcher: no
// schema compilation, no JSON overhead — the marshalled form is the raw
// string value itself.
type EnumStitcher struct {
	name    string
	allowed map[string]struct{}
}

// NewEnumStitcher creates a stitcher that accepts only the given allowed values.
// Comparison is case-sensitive.
func NewEnumStitcher(name string, allowed []string) *EnumStitcher {
	m := make(map[string]struct{}, len(allowed))
	for _, v := range allowed {
		m[v] = struct{}{}
	}
	return &EnumStitcher{name: name, allowed: m}
}

func (e *EnumStitcher) Name() string { return e.name }

// Validate returns an error if out is not a string in the allowed set.
func (e *EnumStitcher) Validate(out any) error {
	s, ok := out.(string)
	if !ok {
		return fmt.Errorf("enum stitcher %q: expected string, got %T", e.name, out)
	}
	if _, ok := e.allowed[s]; !ok {
		allowed := make([]string, 0, len(e.allowed))
		for v := range e.allowed {
			allowed = append(allowed, v)
		}
		return fmt.Errorf("enum stitcher %q: value %q not in allowed set %v", e.name, s, allowed)
	}
	return nil
}

// Marshal returns the value as a plain string (no quoting or wrapping).
func (e *EnumStitcher) Marshal(out any) (string, error) {
	if err := e.Validate(out); err != nil {
		return "", err
	}
	return out.(string), nil
}
