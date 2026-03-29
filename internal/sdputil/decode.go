package sdputil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// UnmarshalJSON decodes JSON from data into v with a size limit
// to prevent OOM from untrusted input.
func UnmarshalJSON(data []byte, v any) error {
	r := io.LimitReader(bytes.NewReader(data), MaxJSONDecodeBytes)
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	return nil
}
