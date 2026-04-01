package augmentation

import (
	"context"
	"strings"

	"sdp_dev/internal/kernel"
)

type RoleRegistry struct {
	resolved *ResolvedPacks
}

func NewRoleRegistry(resolved *ResolvedPacks) *RoleRegistry {
	return &RoleRegistry{resolved: resolved}
}

func (r *RoleRegistry) Resolve(phase string) (kernel.RoleDefinition, bool) {
	if r == nil || r.resolved == nil {
		return kernel.RoleDefinition{}, false
	}
	normalized := strings.ToLower(strings.TrimSpace(phase))
	for _, packID := range r.resolved.Order {
		pack := r.resolved.Packs[packID]
		for _, role := range pack.Roles {
			if strings.EqualFold(role.Phase, normalized) {
				return role, true
			}
		}
	}
	return kernel.RoleDefinition{}, false
}

func ResolveDefaultRole(phase string) (kernel.RoleDefinition, bool) {
	resolved, err := NewResolver(DefaultLoader()).Resolve(context.Background(), []string{
		"planner.pack",
		"implementer.pack",
		"reviewer.pack",
	})
	if err != nil {
		return kernel.RoleDefinition{}, false
	}
	return NewRoleRegistry(resolved).Resolve(phase)
}
