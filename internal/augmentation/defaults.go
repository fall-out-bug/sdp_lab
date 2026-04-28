package augmentation

import "github.com/fall-out-bug/sdp_lab/internal/kernel"

func DefaultPacks() []kernel.WorkflowPack {
	return []kernel.WorkflowPack{
		{
			ID:          "core.pack",
			Version:     "1.0.0",
			Description: "Shared augmentation primitives for SDP prompts and hooks.",
			PromptFragments: []kernel.PromptFragment{
				{ID: "core.boundary", Kind: "system", Content: "Prefer narrow, reviewable changes. Keep scope explicit and avoid unrelated edits."},
			},
			Hooks: []kernel.HookRegistration{
				{ID: "approval.default", Kind: kernel.HookKindApproval, Description: "Default approval surface"},
				{ID: "tool-policy.default", Kind: kernel.HookKindToolPolicy, Description: "Default tool policy interception surface"},
				{ID: "memory.default", Kind: kernel.HookKindMemoryCandidate, Description: "Default memory candidate surface"},
				{ID: "trace.default", Kind: kernel.HookKindTraceEnrichment, Description: "Default trace enrichment surface"},
			},
		},
		{
			ID:           "planner.pack",
			Version:      "1.0.0",
			Description:  "Planner and clarification role pack.",
			Dependencies: []string{"core.pack"},
			PromptFragments: []kernel.PromptFragment{
				{ID: "planner.brief", Kind: "system", Content: "Plan before implementation. Prefer file-specific steps, tests, named risks, and explicit leaf workstream boundaries."},
			},
			Roles: []kernel.RoleDefinition{
				{ID: "planner", Phase: "plan", Agent: "metis", Description: "Planning and design", PromptFragmentIDs: []string{"planner.brief"}},
				{ID: "clarifier", Phase: "explore", Agent: "metis", Description: "Intent clarification and exploration", PromptFragmentIDs: []string{"planner.brief"}},
			},
		},
		{
			ID:           "implementer.pack",
			Version:      "1.0.0",
			Description:  "Implementation role pack.",
			Dependencies: []string{"core.pack"},
			PromptFragments: []kernel.PromptFragment{
				{ID: "implementer.brief", Kind: "system", Content: "Implement one leaf workstream or one finding issue directly. Keep diffs coherent and verify behavior before handing off."},
			},
			Roles: []kernel.RoleDefinition{
				{ID: "implementer-build", Phase: "build", Agent: "hephaestus", Description: "Implementation", PromptFragmentIDs: []string{"implementer.brief"}},
				{ID: "implementer-fix", Phase: "fix", Agent: "hephaestus", Description: "Bug fix implementation", PromptFragmentIDs: []string{"implementer.brief"}},
				{ID: "implementer-refactor", Phase: "refactor", Agent: "hephaestus", Description: "Refactoring implementation", PromptFragmentIDs: []string{"implementer.brief"}},
				{ID: "implementer-feature", Phase: "feature", Agent: "hephaestus", Description: "Feature implementation", PromptFragmentIDs: []string{"implementer.brief"}},
			},
		},
		{
			ID:           "reviewer.pack",
			Version:      "1.0.0",
			Description:  "Review and QA role pack.",
			Dependencies: []string{"core.pack"},
			PromptFragments: []kernel.PromptFragment{
				{ID: "reviewer.brief", Kind: "system", Content: "Review the leaf-scoped contract against correctness, regressions, and missing tests before polish."},
				{ID: "qa.brief", Kind: "system", Content: "Validate behavior against user intent and call out ambiguous evidence explicitly."},
			},
			Roles: []kernel.RoleDefinition{
				{ID: "reviewer", Phase: "review", Agent: "momus", Description: "Code review", PromptFragmentIDs: []string{"reviewer.brief"}},
				{ID: "qa", Phase: "qa", Agent: "oracle", Description: "Quality assurance", PromptFragmentIDs: []string{"qa.brief"}},
			},
		},
	}
}

func DefaultLoader() Loader {
	return NewStaticLoader(DefaultPacks())
}
