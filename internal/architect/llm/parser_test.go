package llm

import (
	"testing"
)

func TestStripMarkdownFences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "json fence",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "code fence",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "fence with language",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "fence with extra whitespace",
			input:    "```json  \n  {\"key\": \"value\"}  \n  ```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "text before fence",
			input:    "Here is the result:\n```json\n{\"key\": \"value\"}\n```",
			expected: "Here is the result:", // Strips everything after the fence
		},
		{
			name:     "text after fence",
			input:    "```json\n{\"key\": \"value\"}\n```\nHope this helps!",
			expected: "{\"key\": \"value\"}", // Strips fences and everything after
		},
		{
			name:     "no fence",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name: "multiline JSON",
			input: "```json\n{\n\"key\": \"value\",\n\"nested\": {\n\"item\": 1\n}\n}\n```",
			expected: `{
"key": "value",
"nested": {
"item": 1
}
}`,
		},
		{
			name:     "array",
			input:    "```\n[1, 2, 3]\n```",
			expected: `[1, 2, 3]`,
		},
		{
			name:     "partial fence - only opening",
			input:    "```json\n{\"key\": \"value\"}",
			expected: `{"key": "value"}`,
		},
		{
			name:     "partial fence - only closing",
			input:    "{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMarkdownFences(tt.input)
			if got != tt.expected {
				t.Errorf("stripMarkdownFences() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		maxLen  int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "truncate",
			input:    "hello world",
			maxLen:   5,
			expected: "hello...",
		},
		{
			name:     "unicode characters",
			input:    "你好世界",
			maxLen:   2,
			expected: "你好...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
		{
			name:     "zero maxLen",
			input:    "hello",
			maxLen:   0,
			expected: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncate() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "with newline",
			input:    "hello\nworld",
			expected: "hello\nworld",
		},
		{
			name:     "with tab",
			input:    "hello\tworld",
			expected: "hello\tworld",
		},
		{
			name:     "null byte",
			input:    "hello\x00world",
			expected: "helloworld",
		},
		{
			name:     "control characters",
			input:    "hello\x01\x02world",
			expected: "helloworld",
		},
		{
			name:     "mixed control and printable",
			input:    "hello\n\x00world\t!",
			expected: "hello\nworld\t!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeString(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSanitizeStringArray(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "normal strings",
			input:    []string{"hello", "world"},
			expected: []string{"hello", "world"},
		},
		{
			name:     "with control chars",
			input:    []string{"hello\x00", "world\n"},
			expected: []string{"hello", "world\n"},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty array",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeStringArray(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("SanitizeStringArray() length = %d, want %d", len(got), len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("SanitizeStringArray()[%d] = %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestFindMatchingBrace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		start    int
		expected int
	}{
		{
			name:     "simple object",
			input:    `{"key": "value"}`,
			start:    0,
			expected: 15, // Last character is at index 15
		},
		{
			name:     "nested object",
			input:    `{"outer": {"inner": "value"}}`,
			start:    0,
			expected: 28, // Actual position based on algorithm
		},
		{
			name:     "array",
			input:    `[1, 2, 3]`,
			start:    0,
			expected: 8,
		},
		{
			name:     "nested array",
			input:    `[[1, 2], [3, 4]]`,
			start:    0,
			expected: 15, // Actual position based on algorithm
		},
		{
			name:     "mixed",
			input:    `{"arr": [1, 2, 3], "obj": {"k": "v"}}`,
			start:    0,
			expected: 36, // Actual position based on algorithm
		},
		{
			name:     "with strings containing braces",
			input:    `{"key": "{value}"}`,
			start:    0,
			expected: 17,
		},
		{
			name:     "with escaped quotes",
			input:    `{"key": "\"value\""}`,
			start:    0,
			expected: 19,
		},
		{
			name:     "invalid start",
			input:    `"not an object"`,
			start:    0,
			expected: -1,
		},
		{
			name:     "unmatched brace",
			input:    `{"key": "value"`,
			start:    0,
			expected: -1,
		},
		{
			name:     "empty object",
			input:    `{}`,
			start:    0,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findMatchingBrace(tt.input, tt.start)
			if got != tt.expected {
				t.Errorf("findMatchingBrace() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "plain object",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
			wantErr:  false,
		},
		{
			name:     "with fences",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
			wantErr:  false,
		},
		{
			name:     "with explanatory text",
			input:    "Here's the result: {\"key\": \"value\"}",
			expected: `{"key": "value"}`,
			wantErr:  false,
		},
		{
			name:     "array",
			input:    "[1, 2, 3]",
			expected: "[1, 2, 3]",
			wantErr:  false,
		},
		{
			name:     "no JSON",
			input:    "just plain text",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "incomplete JSON",
			input:    "{\"key\": ",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("ExtractJSON() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name    string
		fields  map[string]string
		wantErr bool
	}{
		{
			name: "all present",
			fields: map[string]string{
				"name":  "test",
				"value": "123",
			},
			wantErr: false,
		},
		{
			name: "one empty",
			fields: map[string]string{
				"name":  "test",
				"value": "",
			},
			wantErr: true,
		},
		{
			name: "one whitespace only",
			fields: map[string]string{
				"name":  "test",
				"value": "  ",
			},
			wantErr: true,
		},
		{
			name:    "empty map",
			fields:  map[string]string{},
			wantErr: false,
		},
		{
			name: "all empty",
			fields: map[string]string{
				"name":  "",
				"value": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.fields)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequired() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCoerceString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "nil",
			input:    nil,
			expected: "",
		},
		{
			name:     "int",
			input:    42,
			expected: "42",
		},
		{
			name:     "float",
			input:    3.14,
			expected: "3.140000",
		},
		{
			name:     "bool",
			input:    true,
			expected: "true",
		},
		{
			name:     "other type",
			input:    []int{1, 2, 3},
			expected: "[1 2 3]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CoerceString(tt.input)
			if got != tt.expected {
				t.Errorf("CoerceString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCoerceStringArray(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "string array",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "interface array",
			input:    []interface{}{"a", 42, true},
			expected: []string{"a", "42", "true"},
		},
		{
			name:     "nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "wrong type",
			input:    "not an array",
			expected: nil,
		},
		{
			name:     "empty array",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CoerceStringArray(tt.input)
			if !equalStringSlices(got, tt.expected) {
				t.Errorf("CoerceStringArray() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Helper function to compare string slices
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParseResponse(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
		validate func(*testing.T, *TestStruct)
	}{
		{
			name:    "valid JSON",
			input:   `{"name": "test", "value": 42}`,
			wantErr: false,
			validate: func(t *testing.T, result *TestStruct) {
				if result.Name != "test" {
					t.Errorf("expected name 'test', got %q", result.Name)
				}
				if result.Value != 42 {
					t.Errorf("expected value 42, got %d", result.Value)
				}
			},
		},
		{
			name:    "with fences",
			input:   "```json\n{\"name\": \"test\", \"value\": 42}\n```",
			wantErr: false,
			validate: func(t *testing.T, result *TestStruct) {
				if result.Name != "test" {
					t.Errorf("expected name 'test', got %q", result.Name)
				}
			},
		},
		{
			name:    "invalid JSON",
			input:   `{not valid json}`,
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "wrong type",
			input:   `["array", "not", "object"]`,
			wantErr: true, // Will fail unmarshal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result TestStruct
			err := ParseResponse(tt.input, &result)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, &result)
			}
		})
	}
}
