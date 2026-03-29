package dispatch

import "errors"

// StandaloneInfra implements Infrastructure for environments without Gastown.
type StandaloneInfra struct{}

// Name returns "standalone".
func (s *StandaloneInfra) Name() string { return "standalone" }

// CreateConvoy returns a synthetic convoy ID of the form "standalone-{name}".
// The issues parameter is accepted but ignored in standalone mode.
func (s *StandaloneInfra) CreateConvoy(name string, issues []string) (string, error) {
	return "standalone-" + name, nil
}

// ConvoyStatus always returns "unknown" in standalone mode.
func (s *StandaloneInfra) ConvoyStatus(id string) (string, error) {
	return "unknown", nil
}

// Sling is not supported in standalone mode and always returns an error.
func (s *StandaloneInfra) Sling(issue, rig, agent string) error {
	return errors.New("standalone mode: sling not supported")
}
