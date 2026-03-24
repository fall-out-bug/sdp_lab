package omoclient

import (
	"context"
	"io"
	"log"
	"strings"
	"testing"
	"time"
)

func TestReadSSEStream(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []OmOEvent
	}{
		{
			name:  "single event",
			input: "event: tool.started\ndata: tool_call_data\n\n",
			expected: []OmOEvent{
				{Class: EventToolStarted, Data: "tool_call_data"},
			},
		},
		{
			name:  "multiple events",
			input: "event: tool.started\ndata: call1\n\nevent: tool.completed\ndata: call2\n\n",
			expected: []OmOEvent{
				{Class: EventToolStarted, Data: "call1"},
				{Class: EventToolCompleted, Data: "call2"},
			},
		},
		{
			name:  "event with prefix",
			input: "prefix: >\nevent: warning\ndata: warning_msg\n\n",
			expected: []OmOEvent{
				{Class: EventWarning, Prefix: ">", Data: "warning_msg"},
			},
		},
		{
			name:  "completion event",
			input: "event: completion.succeeded\n\n",
			expected: []OmOEvent{
				{Class: EventCompletionSucceeded},
			},
		},
		{
			name:  "unknown event class",
			input: "event: custom.event\ndata: custom_data\n\n",
			expected: []OmOEvent{
				{Class: EventUnknown, Data: "custom_data"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := log.New(io.Discard, "", 0)

			body := io.NopCloser(strings.NewReader(tt.input))
			ch := ReadSSEStream(ctx, body, logger)

			var events []OmOEvent
			for event := range ch {
				events = append(events, event)
			}

			if len(events) != len(tt.expected) {
				t.Errorf("Expected %d events, got %d", len(tt.expected), len(events))
			}

			for i, expected := range tt.expected {
				if i >= len(events) {
					t.Errorf("Missing expected event at index %d: %+v", i, expected)
					continue
				}
				actual := events[i]
				if actual.Class != expected.Class {
					t.Errorf("Event %d: expected class %s, got %s", i, expected.Class, actual.Class)
				}
				if actual.Data != expected.Data {
					t.Errorf("Event %d: expected data %q, got %q", i, expected.Data, actual.Data)
				}
				if actual.Prefix != expected.Prefix {
					t.Errorf("Event %d: expected prefix %q, got %q", i, expected.Prefix, actual.Prefix)
				}
			}
		})
	}
}

func TestReadSSEStreamCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	logger := log.New(io.Discard, "", 0)
	body := io.NopCloser(strings.NewReader("event: tool.started\ndata: test\n"))
	ch := ReadSSEStream(ctx, body, logger)

	cancel()

	eventsReceived := 0
	for range ch {
		eventsReceived++
	}

	if eventsReceived > 1 {
		t.Errorf("Expected at most 1 event after cancellation, got %d", eventsReceived)
	}
}

func TestNormalizeEventClass(t *testing.T) {
	tests := []struct {
		input    string
		expected EventClass
	}{
		{"tool.started", EventToolStarted},
		{"tool.completed", EventToolCompleted},
		{"completion.succeeded", EventCompletionSucceeded},
		{"completion.failed", EventCompletionFailed},
		{"warning", EventWarning},
		{"unknown.event", EventUnknown},
		{"custom.event", EventUnknown},
	}

	for _, tt := range tests {
		result := normalizeEventClass(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeEventClass(%q): expected %s, got %s", tt.input, tt.expected, result)
		}
	}
}

func TestParseToolCall(t *testing.T) {
	data := `{"name":"test_tool","args":{"arg1":"value1"}}`

	result, err := ParseToolCall(data)
	if err != nil {
		t.Fatalf("ParseToolCall failed: %v", err)
	}

	if result["name"] != "test_tool" {
		t.Errorf("Expected name 'test_tool', got %v", result["name"])
	}

	if result["args"] == nil {
		t.Error("Expected args to be present")
	}
}

func TestParseToolCallError(t *testing.T) {
	data := `{invalid json}`

	_, err := ParseToolCall(data)
	if err == nil {
		t.Fatal("Expected error from ParseToolCall, got nil")
	}
}

func TestTimestampParsing(t *testing.T) {
	ctx := context.Background()
	logger := log.New(io.Discard, "", 0)

	input := `timestamp: ` + time.Now().UTC().Format(time.RFC3339) + `
event: tool.started
data: test

` + "\n"

	body := io.NopCloser(strings.NewReader(input))
	ch := ReadSSEStream(ctx, body, logger)

	events := []OmOEvent{}
	for event := range ch {
		events = append(events, event)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	if events[0].Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}
