package stream

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// EventType represents the type of event
type EventType string

const (
	EventTypeOrchestration EventType = "orchestration"
	EventTypeRuntime       EventType = "runtime"
	EventTypePolicy        EventType = "policy"
	EventTypeEvidence      EventType = "evidence"
)

// Severity represents event severity
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
	SeverityDebug   Severity = "debug"
)

// Event represents a normalized SDP event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Severity  Severity               `json:"severity,omitempty"`
	RunID     string                 `json:"run_id,omitempty"`
	FeatureID string                 `json:"feature_id,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// Filter defines event filtering criteria
type Filter struct {
	RunID     string     `json:"run_id,omitempty"`
	FeatureID string     `json:"feature_id,omitempty"`
	Severity  Severity   `json:"severity,omitempty"`
	Types     []EventType `json:"types,omitempty"`
	Sources   []string   `json:"sources,omitempty"`
}

// Matches checks if an event matches the filter
func (f *Filter) Matches(event Event) bool {
	if f.RunID != "" && event.RunID != f.RunID {
		return false
	}
	if f.FeatureID != "" && event.FeatureID != f.FeatureID {
		return false
	}
	if f.Severity != "" && event.Severity != f.Severity {
		return false
	}
	if len(f.Types) > 0 && !containsEventType(f.Types, event.Type) {
		return false
	}
	if len(f.Sources) > 0 && !containsString(f.Sources, event.Source) {
		return false
	}
	return true
}

func containsEventType(types []EventType, eventType EventType) bool {
	for _, t := range types {
		if t == eventType {
			return true
		}
	}
	return false
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// EventBus handles event streaming and subscription
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[chan Event]*Filter
	buffer      []Event
	bufferSize  int
}

// NewEventBus creates a new event bus
func NewEventBus(bufferSize int) *EventBus {
	if bufferSize <= 0 {
		bufferSize = 1000
	}
	return &EventBus{
		subscribers: make(map[chan Event]*Filter),
		buffer:      make([]Event, 0, bufferSize),
		bufferSize:  bufferSize,
	}
}

// Publish publishes an event to all matching subscribers
func (eb *EventBus) Publish(event Event) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Add to buffer
	eb.buffer = append(eb.buffer, event)
	if len(eb.buffer) > eb.bufferSize {
		// Remove oldest events
		eb.buffer = eb.buffer[1:]
	}

	// Send to matching subscribers
	for ch, filter := range eb.subscribers {
		if filter == nil || filter.Matches(event) {
			select {
			case ch <- event:
			default:
				// Channel is full, skip this subscriber
			}
		}
	}
}

// Subscribe subscribes to events matching the filter
func (eb *EventBus) Subscribe(filter *Filter) chan Event {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan Event, 100)
	eb.subscribers[ch] = filter
	return ch
}

// Unsubscribe unsubscribes a channel
func (eb *EventBus) Unsubscribe(ch chan Event) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	delete(eb.subscribers, ch)
	close(ch)
}

// Replay returns buffered events matching the filter
func (eb *EventBus) Replay(filter *Filter) []Event {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	var result []Event
	for _, event := range eb.buffer {
		if filter == nil || filter.Matches(event) {
			result = append(result, event)
		}
	}
	return result
}

// Close closes all subscriber channels
func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for ch := range eb.subscribers {
		close(ch)
		delete(eb.subscribers, ch)
	}
}

// Stream reads events from a channel and writes them to a writer
func Stream(ctx context.Context, events <-chan Event, writer EventWriter, filter *Filter) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if filter == nil || filter.Matches(event) {
				if err := writer.Write(event); err != nil {
					return err
				}
			}
		}
	}
}

// EventWriter writes events
type EventWriter interface {
	Write(event Event) error
}

// JSONEventWriter writes events as JSON lines
type JSONEventWriter struct {
	w interface{ Write([]byte) (int, error) }
}

// NewJSONEventWriter creates a new JSON event writer
func NewJSONEventWriter(w interface{ Write([]byte) (int, error) }) *JSONEventWriter {
	return &JSONEventWriter{w: w}
}

// Write writes an event as JSON
func (w *JSONEventWriter) Write(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = w.w.Write(append(data, '\n'))
	return err
}
