package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/stream"
)

func TestEventBus_BasicOperations(t *testing.T) {
	bus := stream.NewEventBus(10)

	// Create a filter
	filter := &stream.Filter{
		Severity: stream.SeverityError,
	}

	// Subscribe to events
	ch := bus.Subscribe(filter)
	defer bus.Unsubscribe(ch)

	// Publish an error event
	event := stream.Event{
		ID:        "test-1",
		Type:      stream.EventTypeEvidence,
		Timestamp: time.Now(),
		Source:    "test",
		Severity:  stream.SeverityError,
		Message:   "Test error event",
	}

	bus.Publish(event)

	// Should receive the event
	select {
	case received := <-ch:
		if received.ID != event.ID {
			t.Errorf("expected ID %s, got %s", event.ID, received.ID)
		}
		if received.Severity != stream.SeverityError {
			t.Errorf("expected severity error, got %s", received.Severity)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("did not receive event")
	}
}

func TestEventBus_Filtering(t *testing.T) {
	bus := stream.NewEventBus(10)

	// Subscribe to error events only
	errorFilter := &stream.Filter{
		Severity: stream.SeverityError,
	}
	errorCh := bus.Subscribe(errorFilter)
	defer bus.Unsubscribe(errorCh)

	// Subscribe to all events
	allFilter := &stream.Filter{}
	allCh := bus.Subscribe(allFilter)
	defer bus.Unsubscribe(allCh)

	// Publish an info event
	infoEvent := stream.Event{
		ID:        "info-1",
		Type:      stream.EventTypeEvidence,
		Timestamp: time.Now(),
		Source:    "test",
		Severity:  stream.SeverityInfo,
	}
	bus.Publish(infoEvent)

	// Publish an error event
	errorEvent := stream.Event{
		ID:        "error-1",
		Type:      stream.EventTypeEvidence,
		Timestamp: time.Now(),
		Source:    "test",
		Severity:  stream.SeverityError,
	}
	bus.Publish(errorEvent)

	// Error channel should receive only error event
	select {
	case received := <-errorCh:
		if received.ID != errorEvent.ID {
			t.Errorf("error channel: expected ID %s, got %s", errorEvent.ID, received.ID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("error channel: did not receive event")
	}

	// All channel should receive both events
	expectedIDs := map[string]bool{infoEvent.ID: false, errorEvent.ID: false}
	for i := 0; i < 2; i++ {
		select {
		case received := <-allCh:
			expectedIDs[received.ID] = true
		case <-time.After(100 * time.Millisecond):
			t.Fatal("all channel: did not receive event")
		}
	}

	for id, found := range expectedIDs {
		if !found {
			t.Errorf("all channel: did not receive event %s", id)
		}
	}
}

func TestEventBus_Replay(t *testing.T) {
	bus := stream.NewEventBus(10)

	// Publish some events
	for i := 0; i < 5; i++ {
		event := stream.Event{
			ID:        fmt.Sprintf("event-%d", i),
			Type:      stream.EventTypeEvidence,
			Timestamp: time.Now(),
			Source:    "test",
			Severity:  stream.SeverityInfo,
		}
		bus.Publish(event)
	}

	// Replay all events
	filter := &stream.Filter{}
	events := bus.Replay(filter)

	if len(events) != 5 {
		t.Errorf("expected 5 events, got %d", len(events))
	}

	// Replay with filter
	errorFilter := &stream.Filter{
		Severity: stream.SeverityError,
	}
	errorEvents := bus.Replay(errorFilter)

	if len(errorEvents) != 0 {
		t.Errorf("expected 0 error events, got %d", len(errorEvents))
	}
}

func TestFilter_Matches(t *testing.T) {
	tests := []struct {
		name     string
		filter   *stream.Filter
		event    stream.Event
		expected bool
	}{
		{
			name:     "no filter matches all",
			filter:   &stream.Filter{},
			event:    stream.Event{ID: "test", Severity: stream.SeverityInfo},
			expected: true,
		},
		{
			name:     "severity filter matches",
			filter:   &stream.Filter{Severity: stream.SeverityError},
			event:    stream.Event{ID: "test", Severity: stream.SeverityError},
			expected: true,
		},
		{
			name:     "severity filter does not match",
			filter:   &stream.Filter{Severity: stream.SeverityError},
			event:    stream.Event{ID: "test", Severity: stream.SeverityInfo},
			expected: false,
		},
		{
			name:     "run ID filter matches",
			filter:   &stream.Filter{RunID: "run-123"},
			event:    stream.Event{ID: "test", RunID: "run-123"},
			expected: true,
		},
		{
			name:     "run ID filter does not match",
			filter:   &stream.Filter{RunID: "run-123"},
			event:    stream.Event{ID: "test", RunID: "run-456"},
			expected: false,
		},
		{
			name:     "type filter matches",
			filter:   &stream.Filter{Types: []stream.EventType{stream.EventTypeEvidence}},
			event:    stream.Event{ID: "test", Type: stream.EventTypeEvidence},
			expected: true,
		},
		{
			name:     "type filter does not match",
			filter:   &stream.Filter{Types: []stream.EventType{stream.EventTypePolicy}},
			event:    stream.Event{ID: "test", Type: stream.EventTypeEvidence},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Matches(tt.event)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEventWriter_JSON(t *testing.T) {
	var buf strings.Builder
	writer := stream.NewJSONEventWriter(&buf)

	event := stream.Event{
		ID:        "test-1",
		Type:      stream.EventTypeEvidence,
		Timestamp: time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
		Source:    "test",
		Severity:  stream.SeverityError,
		Message:   "Test event",
	}

	err := writer.Write(event)
	if err != nil {
		t.Fatalf("write event: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"id":"test-1"`) {
		t.Errorf("output missing id: %s", output)
	}
	if !strings.Contains(output, `"severity":"error"`) {
		t.Errorf("output missing severity: %s", output)
	}
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("output missing newline: %s", output)
	}
}

func TestStream_Context(t *testing.T) {
	bus := stream.NewEventBus(10)
	ch := bus.Subscribe(nil)
	defer bus.Unsubscribe(ch)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	err := stream.Stream(ctx, ch, stream.NewJSONEventWriter(io.Discard), nil)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
