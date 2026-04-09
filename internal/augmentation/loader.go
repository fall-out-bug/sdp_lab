package augmentation

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	"sdp_dev/internal/kernel"
)

var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

type Loader interface {
	Load(ctx context.Context, packID string) (kernel.WorkflowPack, error)
}

type StaticLoader struct {
	packs map[string]kernel.WorkflowPack
}

func NewStaticLoader(packs []kernel.WorkflowPack) *StaticLoader {
	index := make(map[string]kernel.WorkflowPack, len(packs))
	for _, pack := range packs {
		index[pack.ID] = pack
	}
	return &StaticLoader{packs: index}
}

func (l *StaticLoader) Load(_ context.Context, packID string) (kernel.WorkflowPack, error) {
	if l == nil {
		return kernel.WorkflowPack{}, fmt.Errorf("nil workflow pack loader")
	}
	pack, ok := l.packs[packID]
	if !ok {
		return kernel.WorkflowPack{}, fmt.Errorf("workflow pack %q not found", packID)
	}
	if err := ValidatePack(pack); err != nil {
		return kernel.WorkflowPack{}, err
	}
	return pack, nil
}

type ResolvedPacks struct {
	Order []string
	Packs map[string]kernel.WorkflowPack
}

type Resolver struct {
	loader Loader
}

func NewResolver(loader Loader) *Resolver {
	return &Resolver{loader: loader}
}

func (r *Resolver) Resolve(ctx context.Context, roots []string) (*ResolvedPacks, error) {
	if r == nil || r.loader == nil {
		return nil, fmt.Errorf("resolver requires a loader")
	}

	resolved := &ResolvedPacks{
		Order: make([]string, 0, len(roots)),
		Packs: make(map[string]kernel.WorkflowPack),
	}
	visiting := make(map[string]bool)
	visited := make(map[string]bool)

	var visit func(packID string) error
	visit = func(packID string) error {
		if visited[packID] {
			return nil
		}
		if visiting[packID] {
			return fmt.Errorf("workflow pack dependency cycle at %q", packID)
		}

		visiting[packID] = true
		pack, err := r.loader.Load(ctx, packID)
		if err != nil {
			return fmt.Errorf("load workflow pack %s: %w", packID, err)
		}
		for _, dep := range pack.Dependencies {
			if err := visit(dep); err != nil {
				return fmt.Errorf("visit dependency %s of %s: %w", dep, packID, err)
			}
		}

		visiting[packID] = false
		visited[packID] = true
		resolved.Packs[pack.ID] = pack
		resolved.Order = append(resolved.Order, pack.ID)
		return nil
	}

	for _, root := range roots {
		if err := visit(root); err != nil {
			return nil, fmt.Errorf("resolve workflow pack %s: %w", root, err)
		}
	}
	return resolved, nil
}

func ValidatePack(pack kernel.WorkflowPack) error {
	if pack.ID == "" {
		return fmt.Errorf("workflow pack id is required")
	}
	if !semverPattern.MatchString(pack.Version) {
		return fmt.Errorf("workflow pack %q version %q must match x.y.z", pack.ID, pack.Version)
	}

	fragmentIDs := make(map[string]struct{}, len(pack.PromptFragments))
	for _, fragment := range pack.PromptFragments {
		if fragment.ID == "" {
			return fmt.Errorf("workflow pack %q has prompt fragment without id", pack.ID)
		}
		fragmentIDs[fragment.ID] = struct{}{}
	}

	roleIDs := make(map[string]struct{}, len(pack.Roles))
	for _, role := range pack.Roles {
		if role.ID == "" {
			return fmt.Errorf("workflow pack %q has role without id", pack.ID)
		}
		if role.Phase == "" {
			return fmt.Errorf("workflow pack %q role %q has empty phase", pack.ID, role.ID)
		}
		roleIDs[role.ID] = struct{}{}
		for _, fragmentID := range role.PromptFragmentIDs {
			if _, ok := fragmentIDs[fragmentID]; !ok {
				return fmt.Errorf("workflow pack %q role %q references unknown prompt fragment %q", pack.ID, role.ID, fragmentID)
			}
		}
	}

	hookIDs := make(map[string]struct{}, len(pack.Hooks))
	for _, hook := range pack.Hooks {
		if hook.ID == "" {
			return fmt.Errorf("workflow pack %q has hook without id", pack.ID)
		}
		switch hook.Kind {
		case kernel.HookKindApproval, kernel.HookKindToolPolicy, kernel.HookKindMemoryCandidate, kernel.HookKindTraceEnrichment:
		default:
			return fmt.Errorf("workflow pack %q hook %q has unsupported kind %q", pack.ID, hook.ID, hook.Kind)
		}
		hookIDs[hook.ID] = struct{}{}
	}

	_ = roleIDs
	_ = hookIDs
	return nil
}

func (r *ResolvedPacks) List() []kernel.WorkflowPack {
	if r == nil {
		return nil
	}
	out := make([]kernel.WorkflowPack, 0, len(r.Order))
	for _, id := range r.Order {
		out = append(out, r.Packs[id])
	}
	return out
}

func (r *ResolvedPacks) SortedIDs() []string {
	if r == nil {
		return nil
	}
	out := append([]string(nil), r.Order...)
	sort.Strings(out)
	return out
}
