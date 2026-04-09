package omoclient

import (
	"context"
	"errors"
	"net"
	"strings"
	"syscall"
)

// FailureKind represents the classification of a failure
type FailureKind int

const (
	// FailureTransport indicates network/HTTP layer failures
	FailureTransport FailureKind = iota
	// FailureProtocol indicates protocol/schema violations
	FailureProtocol
	// FailureRuntime indicates runtime/execution errors
	FailureRuntime
	// FailureGovernance indicates governance/policy violations
	FailureGovernance
	// FailureValidation indicates input validation failures
	FailureValidation
	// FailurePersistence indicates storage/persistence failures
	FailurePersistence
)

// String returns the string representation of FailureKind
func (k FailureKind) String() string {
	switch k {
	case FailureTransport:
		return "transport"
	case FailureProtocol:
		return "protocol"
	case FailureRuntime:
		return "runtime"
	case FailureGovernance:
		return "governance"
	case FailureValidation:
		return "validation"
	case FailurePersistence:
		return "persistence"
	default:
		return "unknown"
	}
}

// Failure represents a classified failure
type Failure struct {
	Kind      FailureKind `json:"kind"`
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Retryable bool        `json:"retryable"`
	Temporary bool        `json:"temporary"`
	Cause     error       `json:"-"`
}

// Error returns the failure message
func (f *Failure) Error() string {
	if f.Cause != nil {
		return f.Message + ": " + f.Cause.Error()
	}
	return f.Message
}

// Unwrap returns the underlying cause
func (f *Failure) Unwrap() error {
	return f.Cause
}

// ClassifyError classifies an error into a Failure
func ClassifyError(err error) *Failure {
	if err == nil {
		return nil
	}

	// Check for wrapped Failure
	var f *Failure
	if errors.As(err, &f) {
		return f
	}

	// Check for context errors
	if errors.Is(err, context.Canceled) {
		return &Failure{
			Kind:      FailureRuntime,
			Code:      "ctx_canceled",
			Message:   "operation canceled",
			Retryable: false,
			Temporary: false,
			Cause:     err,
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return &Failure{
			Kind:      FailureTransport,
			Code:      "ctx_timeout",
			Message:   "operation timed out",
			Retryable: true,
			Temporary: true,
			Cause:     err,
		}
	}

	// Check for network/transport errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		retryable := netErr.Timeout()
		return &Failure{
			Kind:      FailureTransport,
			Code:      "net_error",
			Message:   "network error",
			Retryable: retryable,
			Temporary: retryable,
			Cause:     err,
		}
	}

	// Check for specific system errors
	if errors.Is(err, syscall.ECONNREFUSED) {
		return &Failure{
			Kind:      FailureTransport,
			Code:      "conn_refused",
			Message:   "connection refused",
			Retryable: true,
			Temporary: true,
			Cause:     err,
		}
	}

	if errors.Is(err, syscall.ECONNRESET) {
		return &Failure{
			Kind:      FailureTransport,
			Code:      "conn_reset",
			Message:   "connection reset",
			Retryable: true,
			Temporary: true,
			Cause:     err,
		}
	}

	// Check for timeout errors
	if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
		return &Failure{
			Kind:      FailureTransport,
			Code:      "timeout",
			Message:   "operation timed out",
			Retryable: true,
			Temporary: true,
			Cause:     err,
		}
	}

	// Check for validation/malformed errors
	if strings.Contains(err.Error(), "invalid") ||
		strings.Contains(err.Error(), "malformed") ||
		strings.Contains(err.Error(), "unmarshal") ||
		strings.Contains(err.Error(), "marshal") ||
		strings.Contains(err.Error(), "json") ||
		strings.Contains(err.Error(), "schema") {
		return &Failure{
			Kind:      FailureValidation,
			Code:      "validation_failed",
			Message:   "validation or schema error",
			Retryable: false,
			Temporary: false,
			Cause:     err,
		}
	}

	// Check for persistence errors
	if strings.Contains(err.Error(), "database") ||
		strings.Contains(err.Error(), "storage") ||
		strings.Contains(err.Error(), "disk") ||
		strings.Contains(err.Error(), "file not found") ||
		strings.Contains(err.Error(), "permission denied") {
		return &Failure{
			Kind:      FailurePersistence,
			Code:      "persistence_error",
			Message:   "storage/persistence error",
			Retryable: false,
			Temporary: false,
			Cause:     err,
		}
	}

	// Default to runtime error for unknown failures
	return &Failure{
		Kind:      FailureRuntime,
		Code:      "runtime_error",
		Message:   "unclassified runtime error",
		Retryable: false,
		Temporary: false,
		Cause:     err,
	}
}

// IsStrike determines if a failure counts as a strike against the system
func IsStrike(f *Failure) bool {
	if f == nil {
		return false
	}

	// Transport transient errors (retryable or temporary) are not strikes
	// Non-transient transport errors are strikes
	if f.Kind == FailureTransport {
		return !f.Retryable && !f.Temporary
	}

	// Malformed and validation errors are strikes
	if f.Kind == FailureValidation {
		return true
	}

	// Protocol errors are strikes
	if f.Kind == FailureProtocol {
		return true
	}

	// Governance violations are hard strikes
	if f.Kind == FailureGovernance {
		return true
	}

	// Runtime errors are not strikes if temporary/retryable
	if f.Kind == FailureRuntime {
		return !f.Temporary
	}

	// Persistence errors are strikes
	if f.Kind == FailurePersistence {
		return true
	}

	return true
}
