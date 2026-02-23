# FR-011: Cross-Project PR Integration

Priority: P0
Effort: 5d
Dependencies: FR-003 (CRD types exist)

## Problem

The agent swarm works only with one repo (`sdp_dev`). For full operation, cross-cutting integration with 5 projects is needed:

| Project | Role | PR Target |
|---------|------|-----------|
| **sdp** | SDP protocol (skills, commands, hooks) | upstream features |
| **opencode** | AI coding runtime | runtime patches, capabilities |
| **kubeopencode operator** | K8s CRD controller | UP-001, UP-002, UP-003 |
| **openclaw** + plugins | Alternative runtime | adapter parity, plugins |
| **beads** | Issue tracker | upstream features, sync API |

## Design

### Multi-Repo Project Registry

Extend `internal/registry/` and `project-registry.yaml`:

```yaml
projects:
  - id: sdp_dev
    repo_url: git@github.com:org/sdp_dev.git
    repo_branch: dev
    beads_prefix: sdp_dev
    language: go
    workstreams: [workstream:agentrun-operator, workstream:builder]
  - id: sdp_protocol
    repo_url: git@github.com:org/sdp.git
    repo_branch: main
    beads_prefix: sdp
    language: markdown
    workstreams: [workstream:skills, workstream:commands]
  - id: opencode
    repo_url: git@github.com:org/opencode.git
    repo_branch: main
    beads_prefix: oc
    language: go
    workstreams: [workstream:runtime]
  - id: kubeopencode_operator
    repo_url: git@github.com:org/kubeopencode.git
    repo_branch: sdp-upstream
    beads_prefix: koc
    language: go
    workstreams: [workstream:operator]
    fork: true
    upstream_remote: upstream
  - id: openclaw
    repo_url: git@github.com:org/openclaw.git
    repo_branch: main
    beads_prefix: ocl
    language: typescript
    workstreams: [workstream:adapter, workstream:plugins]
  - id: beads
    repo_url: git@github.com:org/beads.git
    repo_branch: main
    beads_prefix: bd
    language: go
    workstreams: [workstream:core, workstream:api]
```

### Federation Bridge Enhancement

`internal/federation/bridge.go` already supports per-project bridge. It is necessary to:

1. Extend `WorkspaceManager` for fork-based projects (dual remote: origin + upstream)
2. Add `pr-publish` cross-project support (PR to the correct remote)
3. Extend `IntentTranslator` for mapping issue → target project
4. Add cross-project dependency tracking in Beads

### PR Routing

```
AgentRun.spec.repo → WorkspaceManager.EnsureWorkspace() → git clone/pull
                   → pr-publish --repo {repo_url} --base {branch}
                   → Beads.Update(issueID, prUrl)
```

## Acceptance Criteria

- [ ] project-registry.yaml contains all 5 projects
- [ ] WorkspaceManager clones/updates all 5 repos
- [ ] Fork-based projects support upstream remote
- [ ] pr-publish creates PR in the correct repo/branch
- [ ] Federation bridge works for all registered projects
- [ ] Cross-project Beads dependencies are tracked
- [ ] E2E: intake issue for sdp_protocol → agent → PR in sdp repo
