package roles

import (
	"context"
	"fmt"
	"sync"
)

var (
	strategies   = make(map[string]RoleStrategy)
	strategiesMu sync.RWMutex
)

func init() {
	Register("analyst", &AnalystStrategy{})
	Register("coder", &CoderStrategy{})
	Register("reviewer", &ReviewerStrategy{PersonaID: "correctness"})
	Register("reviewer-security", &ReviewerStrategy{PersonaID: "security"})
	Register("reviewer-dx", &ReviewerStrategy{PersonaID: "dx"})
}

// Register registers a RoleStrategy for the given role name.
func Register(role string, s RoleStrategy) {
	strategiesMu.Lock()
	defer strategiesMu.Unlock()
	strategies[role] = s
}

// Get returns the RoleStrategy for the role, or nil.
func Get(role string) RoleStrategy {
	strategiesMu.RLock()
	defer strategiesMu.RUnlock()
	return strategies[role]
}

// Execute runs the strategy for the role.
func Execute(ctx context.Context, role string, input TaskInput) (TaskResult, error) {
	s := Get(role)
	if s == nil {
		return TaskResult{}, fmt.Errorf("unknown role: %s", role)
	}
	return s.Execute(ctx, input)
}
