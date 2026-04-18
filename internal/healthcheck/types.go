// Package healthcheck reports the health of the SDP development environment.
package healthcheck

import "context"

// Status is the result of a single check.
type Status string

const (
	StatusPass Status = "pass"
	StatusFail Status = "fail"
	StatusWarn Status = "warn"
)

// CheckResult holds the outcome of one health check.
type CheckResult struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Detail string `json:"detail"`
}

// Checker is implemented by anything that can perform a health check.
type Checker interface {
	Name() string
	Run(ctx context.Context) CheckResult
}

// Config carries configuration for the Runner.
type Config struct {
	ProjectRoot string
	// Only, when non-empty, restricts execution to this single check name.
	Only string
}
