package parallel

import (
	"sort"
	"strings"
)

type LockDomain string

const (
	DomainRepoTree        LockDomain = "repo-tree"
	DomainBranchRef       LockDomain = "branch-ref"
	DomainBeadsState      LockDomain = "beads-state"
	DomainEvidenceStore   LockDomain = "evidence-store"
	DomainK8sControlPlane LockDomain = "k8s-control-plane"
)

type Hazard struct {
	Key         string       `json:"key"`
	Description string       `json:"description"`
	Domains     []LockDomain `json:"domains"`
}

type LockRequest struct {
	Domain LockDomain `json:"domain"`
	Scope  string     `json:"scope"`
	Reason string     `json:"reason"`
}

func IdentifyHazards(paths []string) []Hazard {
	hazards := map[string]Hazard{}
	add := func(key string, hazard Hazard) {
		hazards[key] = hazard
	}

	for _, raw := range paths {
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}

		if strings.HasPrefix(path, ".beads/") {
			add("beads-state-race", Hazard{
				Key:         "beads-state-race",
				Description: "Concurrent writes to Beads state can clobber task transitions and dependency links.",
				Domains:     []LockDomain{DomainBeadsState},
			})
		}

		if strings.HasPrefix(path, ".sdp/evidence/") {
			add("evidence-trace-interleave", Hazard{
				Key:         "evidence-trace-interleave",
				Description: "Parallel evidence writes can interleave traces and break strict-evidence verification.",
				Domains:     []LockDomain{DomainEvidenceStore},
			})
		}

		if strings.HasPrefix(path, "internal/") || strings.HasPrefix(path, "cmd/") || strings.HasPrefix(path, "docs/") || strings.HasPrefix(path, "specs/") || strings.HasPrefix(path, "scripts/") || strings.HasPrefix(path, "deploy/") {
			add("repo-tree-conflict", Hazard{
				Key:         "repo-tree-conflict",
				Description: "Overlapping edits in the same repository tree cause merge conflicts and semantic drift.",
				Domains:     []LockDomain{DomainRepoTree, DomainBranchRef},
			})
		}

		if strings.HasPrefix(path, "deploy/") || strings.HasPrefix(path, "scripts/apply_") || strings.HasPrefix(path, "scripts/orchestrate_k8s_") {
			add("cluster-rollout-collision", Hazard{
				Key:         "cluster-rollout-collision",
				Description: "Concurrent rollout operations can race deployments and leave cluster state inconsistent.",
				Domains:     []LockDomain{DomainK8sControlPlane},
			})
		}
	}

	keys := make([]string, 0, len(hazards))
	for key := range hazards {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]Hazard, 0, len(keys))
	for _, key := range keys {
		out = append(out, hazards[key])
	}
	return out
}

func BuildLockRequests(paths []string, branch string) []LockRequest {
	hazards := IdentifyHazards(paths)
	requests := map[LockDomain]LockRequest{}
	for _, hazard := range hazards {
		for _, domain := range hazard.Domains {
			if _, exists := requests[domain]; exists {
				continue
			}
			requests[domain] = LockRequest{Domain: domain, Scope: "global", Reason: hazard.Key}
		}
	}

	branchName := strings.TrimSpace(branch)
	if branchName != "" {
		requests[DomainBranchRef] = LockRequest{Domain: DomainBranchRef, Scope: branchName, Reason: "branch-update-serialization"}
	}

	domains := make([]string, 0, len(requests))
	for domain := range requests {
		domains = append(domains, string(domain))
	}
	sort.Strings(domains)

	out := make([]LockRequest, 0, len(domains))
	for _, domain := range domains {
		out = append(out, requests[LockDomain(domain)])
	}
	return out
}
