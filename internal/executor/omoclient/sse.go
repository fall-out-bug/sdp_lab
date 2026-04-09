package omoclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

// ReadSSEStream reads SSE events from body and sends them to the returned channel
func ReadSSEStream(ctx context.Context, body io.ReadCloser) <-chan OmOEvent {
	ch := make(chan OmOEvent, 10)

	go func() {
		defer close(ch)
		defer body.Close()

		scanner := bufio.NewScanner(body)
		var currentEvent OmOEvent

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := scanner.Text()

			if line == "" {
				if currentEvent.Class != "" {
					ch <- currentEvent
					currentEvent = OmOEvent{}
				}
				continue
			}

			if strings.HasPrefix(line, "event:") {
				currentEvent.Class = normalizeEventClass(strings.TrimSpace(strings.TrimPrefix(line, "event:")))
			} else if strings.HasPrefix(line, "data:") {
				currentEvent.Data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			} else if strings.HasPrefix(line, "prefix:") {
				currentEvent.Prefix = strings.TrimSpace(strings.TrimPrefix(line, "prefix:"))
			} else if strings.HasPrefix(line, "timestamp:") {
				ts := strings.TrimSpace(strings.TrimPrefix(line, "timestamp:"))
				if t, err := time.Parse(time.RFC3339, ts); err == nil {
					currentEvent.Timestamp = t
				}
			}
		}

		if currentEvent.Class != "" {
			ch <- currentEvent
		}

		if err := scanner.Err(); err != nil {
			slog.Warn("SSE scanner error", "error", err)
		}
	}()

	return ch
}

// normalizeEventClass normalizes event class strings to constants
func normalizeEventClass(class string) EventClass {
	switch class {
	case "tool.started":
		return EventToolStarted
	case "tool.completed":
		return EventToolCompleted
	case "completion.succeeded":
		return EventCompletionSucceeded
	case "completion.failed":
		return EventCompletionFailed
	case "warning":
		return EventWarning
	default:
		return EventUnknown
	}
}

// ParseToolCall parses tool call data from SSE event
func ParseToolCall(data string) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("parse tool call: %w", err)
	}
	return result, nil
}
