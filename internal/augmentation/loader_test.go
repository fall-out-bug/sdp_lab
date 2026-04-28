package augmentation

import (
	"context"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/kernel"
)

func TestResolvePromptContext_LazyDependencyOrder(t *testing.T) {
	loader := NewStaticLoader(DefaultPacks())
	segments, err := ResolvePromptContext(context.Background(), loader, []string{"planner.pack"})
	if err != nil {
		t.Fatalf("ResolvePromptContext: %v", err)
	}
	if len(segments) != 2 {
		t.Fatalf("segments = %d, want 2", len(segments))
	}
	if segments[0].Source != "core.pack" || segments[1].Source != "planner.pack" {
		t.Fatalf("unexpected source order: %+v", segments)
	}
}

func TestValidatePack_InvalidPromptRef(t *testing.T) {
	pack := kernel.WorkflowPack{
		ID:      "broken.pack",
		Version: "1.0.0",
		Roles: []kernel.RoleDefinition{
			{ID: "planner", Phase: "plan", PromptFragmentIDs: []string{"missing"}},
		},
	}
	if err := ValidatePack(pack); err == nil {
		t.Fatal("expected invalid prompt fragment reference to fail validation")
	}
}

func TestRoleRegistryResolveDefaultRole(t *testing.T) {
	role, ok := ResolveDefaultRole("review")
	if !ok {
		t.Fatal("expected review role to resolve")
	}
	if role.Agent != "momus" {
		t.Fatalf("agent = %q, want momus", role.Agent)
	}
}
