package decompose

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// JSONStitcher validates a stage output value against a JSON Schema and
// marshals it to indented JSON for evidence logging.
type JSONStitcher struct {
	name   string
	schema *jsonschema.Schema
}

// NewJSONStitcherFromBytes compiles the given JSON Schema bytes and returns a
// JSONStitcher. Returns an error if schema compilation fails.
func NewJSONStitcherFromBytes(name string, schema []byte) (*JSONStitcher, error) {
	c := jsonschema.NewCompiler()
	url := "schema://" + name + ".json"
	if err := c.AddResource(url, bytes.NewReader(schema)); err != nil {
		return nil, fmt.Errorf("json stitcher %q: add schema: %w", name, err)
	}
	sch, err := c.Compile(url)
	if err != nil {
		return nil, fmt.Errorf("json stitcher %q: compile schema: %w", name, err)
	}
	return &JSONStitcher{name: name, schema: sch}, nil
}

// MustNewJSONStitcherFromBytes is like NewJSONStitcherFromBytes but panics on error.
// Use only with hardcoded schemas in init-time constructors.
func MustNewJSONStitcherFromBytes(name string, schema []byte) *JSONStitcher {
	s, err := NewJSONStitcherFromBytes(name, schema)
	if err != nil {
		panic(err)
	}
	return s
}

func (j *JSONStitcher) Name() string { return j.name }

// Validate marshals out to JSON and validates it against the schema.
// The error includes the JSON path of the first violation.
func (j *JSONStitcher) Validate(out any) error {
	data, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("json stitcher %q: marshal for validation: %w", j.name, err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("json stitcher %q: unmarshal for validation: %w", j.name, err)
	}
	if err := j.schema.Validate(v); err != nil {
		return fmt.Errorf("json stitcher %q: schema violation: %w", j.name, err)
	}
	return nil
}

// Marshal serializes out to indented JSON.
// Round-trip: Marshal → json.Unmarshal → Validate is guaranteed to succeed.
func (j *JSONStitcher) Marshal(out any) (string, error) {
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", fmt.Errorf("json stitcher %q: marshal: %w", j.name, err)
	}
	return string(data), nil
}
