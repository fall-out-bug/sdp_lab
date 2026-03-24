package omoclient

import "sync"

// StrikePolicy defines thresholds for strike escalation
type StrikePolicy struct {
	MaxTransportRetries int
	MaxQualityStrikes   int
	MaxPolicyStrikes    int
}

// DefaultStrikePolicy returns sensible default thresholds
func DefaultStrikePolicy() *StrikePolicy {
	return &StrikePolicy{
		MaxTransportRetries: 5,
		MaxQualityStrikes:   3,
		MaxPolicyStrikes:    1,
	}
}

// StrikeTracker tracks failures by kind and determines when to block
type StrikeTracker struct {
	policy           *StrikePolicy
	transportRetries int
	qualityStrikes   int
	policyStrikes    int
	mu               sync.RWMutex
}

// NewStrikeTracker creates a new strike tracker with the given policy
func NewStrikeTracker(policy *StrikePolicy) *StrikeTracker {
	if policy == nil {
		policy = DefaultStrikePolicy()
	}
	return &StrikeTracker{
		policy: policy,
	}
}

// Record records a failure and increments the appropriate counter
func (st *StrikeTracker) Record(f *Failure) {
	if f == nil {
		return
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	switch f.Kind {
	case FailureTransport:
		// Transport failures are retries, not strikes
		if f.Retryable || f.Temporary {
			st.transportRetries++
		}
	case FailureValidation, FailureProtocol, FailurePersistence:
		// Malformed/validation/protocol/persistence errors are quality strikes
		st.qualityStrikes++
	case FailureGovernance:
		// Governance violations are policy strikes (hard strikes)
		st.policyStrikes++
	case FailureRuntime:
		// Runtime errors are quality strikes if not temporary
		if !f.Temporary {
			st.qualityStrikes++
		}
	}
}

// ShouldBlock returns true if the system should block execution based on strike counts
func (st *StrikeTracker) ShouldBlock() bool {
	st.mu.RLock()
	defer st.mu.RUnlock()

	// Block if transport retries exceeded
	if st.transportRetries >= st.policy.MaxTransportRetries {
		return true
	}

	// Block if quality strikes exceeded
	if st.qualityStrikes >= st.policy.MaxQualityStrikes {
		return true
	}

	// Block if any policy strikes (hard strike)
	if st.policyStrikes >= st.policy.MaxPolicyStrikes {
		return true
	}

	return false
}

// Reset clears all strike counters
func (st *StrikeTracker) Reset() {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.transportRetries = 0
	st.qualityStrikes = 0
	st.policyStrikes = 0
}

// GetCounts returns the current strike counts
func (st *StrikeTracker) GetCounts() (transportRetries, qualityStrikes, policyStrikes int) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return st.transportRetries, st.qualityStrikes, st.policyStrikes
}

// SetPolicy updates the strike policy
func (st *StrikeTracker) SetPolicy(policy *StrikePolicy) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.policy = policy
}
