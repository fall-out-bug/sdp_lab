package bus

import (
	"encoding/json"
	"fmt"
	"time"
)

// Request sends a request envelope and waits for a reply envelope.
func Request(client *Client, subject string, envelope Envelope, timeout time.Duration) (Envelope, error) {
	data, err := json.Marshal(envelope)
	if err != nil {
		return Envelope{}, fmt.Errorf("marshal request: %w", err)
	}

	nc := client.Conn()
	msg, err := nc.Request(subject, data, timeout)
	if err != nil {
		return Envelope{}, fmt.Errorf("nats request: %w", err)
	}

	var reply Envelope
	if err := json.Unmarshal(msg.Data, &reply); err != nil {
		return Envelope{}, fmt.Errorf("unmarshal reply: %w", err)
	}
	return reply, nil
}

