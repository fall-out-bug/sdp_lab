# 2026-03-06: Backlog Directions (0/1/2/3)

Goal: classify current SDP improvement stream into four execution directions and map each direction to concrete backlog artifacts.

## Direction 0 — Do Now (repo-state polish)

Status: in progress via OSS PRs and immediate fixes.

- Scope: remove trust-breaking drift in metadata, links, and toolchain declarations.
- Current execution reference: `sdp` PR `#107` (repo-state polish alignment).
- Exit condition:
  - no stale org/repo links in public docs/manifests
  - plugin/changelog/docs version line aligned
  - CI workflow toolchain aligned with `.go-version`

## Direction 1 — High-Priority Backlog (features + workstreams)

### F078: Trust Surface Consistency

- 00-078-01 — Version source of truth and drift check
- 00-078-02 — Metadata and links drift CI gate
- 00-078-03 — Release surface manifest alignment

### F079: Enterprise Trust Pack

- 00-079-01 — Public maturity matrix
- 00-079-02 — Canonical guarantees and non-guarantees
- 00-079-03 — CI gates map with local reproduce commands

### F080: Contract Governance Policy

- 00-080-01 — Schema semver compatibility policy
- 00-080-02 — Schema compatibility enforcement in CI
- 00-080-03 — Canonical examples and conformance tests

### F081: 30-Min Production Pilot

- 00-081-01 — CI-gate-only enterprise pilot quickstart
- 00-081-02 — Contracted runtime pilot package
- 00-081-03 — Rollback and disable playbook

## Direction 2 — Backlog (feature-level, medium priority)

- F082 — Compliance Control Mapping (`sdplab-cyha`)
- F083 — Policy Engine Enforcement Pack (`sdplab-ogwa`)
- F084 — Enterprise Runtime Hardening (`sdplab-78ys`)
- F085 — Platform Productization Kit (`sdplab-813o`)

Note: workstreams are intentionally deferred until Direction 1 reaches stable execution.

## Direction 3 — Parked Ideas (long horizon)

- F086 — Cross-Project Evidence Federation
- F087 — Adversarial Reviewer Quorum
- F088 — Autonomous Backlog Synthesis
- F089 — Adaptive Gate Tuning

Promotion rule: idea can move to Direction 2 only after a scoped RFC and one measurable adoption hypothesis.
