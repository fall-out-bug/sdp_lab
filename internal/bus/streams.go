package bus

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
)

// Stream names for SDP JetStream.
const (
	StreamIntake    = "SDP_INTAKE"
	StreamArtifacts = "SDP_ARTIFACTS"
	StreamReviews   = "SDP_REVIEWS"
	StreamLifecycle = "SDP_LIFECYCLE"
	StreamRetro     = "SDP_RETRO"
)

// StreamConfig holds JetStream stream configuration.
type StreamConfig struct {
	Name     string
	Subjects []string
	MaxBytes int64
}

// DefaultStreams returns the default SDP stream configurations.
func DefaultStreams() []StreamConfig {
	return []StreamConfig{
		{Name: StreamIntake, Subjects: []string{"sdp.intake.>"}, MaxBytes: 1024 * 1024 * 100},
		{Name: StreamArtifacts, Subjects: []string{"sdp.artifact.>"}, MaxBytes: 1024 * 1024 * 500},
		{Name: StreamReviews, Subjects: []string{"sdp.review.>"}, MaxBytes: 1024 * 1024 * 100},
		{Name: StreamLifecycle, Subjects: []string{"sdp.lifecycle.>"}, MaxBytes: 1024 * 1024 * 100},
		{Name: StreamRetro, Subjects: []string{"sdp.retro.>"}, MaxBytes: 1024 * 1024 * 50},
	}
}

// ProvisionStreams creates or updates JetStream streams for SDP.
func ProvisionStreams(ctx context.Context, client *Client, configs []StreamConfig) error {
	js := client.JetStream()

	for _, cfg := range configs {
		_, err := js.AddStream(&nats.StreamConfig{
			Name:      cfg.Name,
			Subjects:  cfg.Subjects,
			Storage:   nats.FileStorage,
			MaxBytes:  cfg.MaxBytes,
			Retention: nats.LimitsPolicy,
		})
		if err != nil {
			// Stream may already exist; try update
			_, err2 := js.UpdateStream(&nats.StreamConfig{
				Name:      cfg.Name,
				Subjects:  cfg.Subjects,
				Storage:   nats.FileStorage,
				MaxBytes:  cfg.MaxBytes,
				Retention: nats.LimitsPolicy,
			})
			if err2 != nil {
				return fmt.Errorf("provision stream %s: %w", cfg.Name, err)
			}
		}
	}
	return nil
}

// ProvisionStreamsDefault provisions the default SDP streams.
func ProvisionStreamsDefault(ctx context.Context, client *Client) error {
	return ProvisionStreams(ctx, client, DefaultStreams())
}
