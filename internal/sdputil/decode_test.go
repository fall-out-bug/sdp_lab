package sdputil

import (
	"strings"
	"testing"
)

func TestUnmarshalJSON_Basic(t *testing.T) {
	type sample struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	data := []byte(`{"name":"hello","count":7}`)
	var got sample
	if err := UnmarshalJSON(data, &got); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if got.Name != "hello" || got.Count != 7 {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestUnmarshalJSON_OversizeRejected(t *testing.T) {
	// Build a JSON payload larger than MaxJSONDecodeBytes.
	// MaxJSONDecodeBytes is 10MB; we generate ~11MB of data.
	bigValue := strings.Repeat("x", MaxJSONDecodeBytes+1)
	data := []byte(`{"v":"` + bigValue + `"}`)

	var out map[string]string
	err := UnmarshalJSON(data, &out)
	if err == nil {
		t.Fatal("expected error for oversize input, got nil")
	}
}
