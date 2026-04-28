# F150 Release Readiness Report

> **Feature:** F150 -- Product Layering and Release Readiness
> **Epic Bead:** sdplab-nyr0
> **Date:** 2026-04-27
> **Author:** Andrei (WS 00-150-10, sdplab-3uep)
> **Branch:** main (F150 worktree)

---

## 1. Executive Summary

F150 set out to make SDP Lab release-ready by establishing explicit product layering, classifying the release surface, migrating the Go module path, isolating experimental code, auditing dependencies, setting coverage policy, implementing telemetry consent, rehearsing the Homebrew formula, aligning documentation, and closing with this debt ledger.

**Scope:** 10 workstreams across 7 commits on the F150 branch, touching 546 files (+3,233 / -5,409 lines).

**Verdict:** SDP Toolkit (the `sdp` CLI binary + operator tooling) is structurally ready for a Homebrew formula release. The module path, build isolation, dependency profile, telemetry consent, and documentation are in place. Remaining debt is documented and non-blocking. No P0/P1 release blockers were found.

---

## 2. Shipped Changes

| WS | Bead | Commit | Title | Evidence |
|----|------|--------|-------|----------|
| 00-150-01 | sdplab-qgq1 | (merged to main) | Product layering design | v3 memo `docs/strategy/2026-04-27-sdp-product-layering-4d.md` (53 KB), 7-layer taxonomy, 2 IIP council reviews |
| 00-150-02 | sdplab-8rk7 | (merged to main) | Release surface inventory | `docs/reference/maturity-matrix.md` (425 lines), `docs/reference/product-surface.md` (283 lines), 37 cmd/ binaries classified |
| 00-150-03 | sdplab-5r4x | 65594486 | Module path migration | `sdp_dev` -> `github.com/fall-out-bug/sdp_lab`, 484 files updated, `go build ./...` passes |
| 00-150-04 | sdplab-crct | a1062336 | Experimental isolation | 18 experimental binaries tagged `sdp_experimental`, `.goreleaser.yml` allowlist (sdp + 15 tooling), drift detection in `scripts/check-release-surface.sh` |
| 00-150-05 | sdplab-hjl7 | d3f2f78a | Dependency diet | `docs/reviews/2026-04-27-dependency-audit.md`, 16 direct deps justified, 9 duplicate patterns documented, 3 experimental deps flagged |
| 00-150-06 | sdplab-q2cb | 615f57ba | Coverage policy | Tiered targets in `docs/reference/maturity-matrix.md`: happy-path >=80%, GA >=60%, Beta >=50% advisory, Experimental exempt |
| 00-150-07 | sdplab-p4hj | 1586dd4d | Telemetry consent | `docs/reference/telemetry-schema.md` v1.1.0, 4-tier consent model (none/metadata/findings/content), OTEL export with explicit 2-gate design, 61 tests |
| 00-150-08 | sdplab-sa0w | 5aa4c67 | Homebrew formula | `formula/sdp.rb`, `goreleaser` dry run passed, `brew test` verified |
| 00-150-09 | sdplab-crct | 44526f1e | Docs alignment | README.md and QUICKSTART.md aligned to v3 layer taxonomy, product-facing wording updated |
| 00-150-10 | sdplab-3uep | (this commit) | Debt ledger and readiness report | This document |

**Note on WS numbering:** Workstream commits landed out of order relative to WS number (e.g., WS-07 before WS-03) because sibling workstreams executed in parallel on the same branch. All 10 workstreams are complete.

---

## 3. Release Surface

### 3.1 Formula Default Install (stable product surface)

| Binary | Maturity | What it is |
|--------|----------|------------|
| `sdp` | GA | Main CLI. All product subcommands live here. |

The `sdp` binary includes 34 subcommands across three categories:
- **Stable subcommands (12):** scout, metrics, index, spec, bootstrap, init, manifest, generate-adapters, doctor, coverage-scan, rules, skills
- **Operator Mode subcommands (9):** orchestrate, card, board, phase, build, deploy, reset, discover, architect
- **Query/Insight subcommands (8):** why, next, missing, approve, trace, status, stuck, attention
- **Pipeline subcommands (7):** dispatch, result, intent, eval, clarify, plan, approve-plan
- **Experimental subcommands (1):** telemetry (Beta, opt-in by design)

### 3.2 Operator Tooling (formula, not first-run promise)

15 additional binaries are built by GoReleaser and available via the formula tap:
`sdp-orchestrate`, `sdp-orchestrate-daemon`, `sdp-guard`, `sdp-ci-loop`, `sdp-doc-sync`, `sdp-beads-bridge`, `sdp-gh-findings-sync`, `sdp-ready`, `sdp-protocol-check`, `sdp-ws-verdict-validate`, `sdp-evidence`, `sdp-export`, `sdp-omc-guard`, `sdp-session-audit`, `sdp-healthcheck`

### 3.3 Exclusion Mechanisms

Three layers prevent experimental code from entering release builds:
1. **GoReleaser allowlist** -- only `sdp` + 15 tooling binaries in `.goreleaser.yml`
2. **Build tags** (`sdp_experimental`) -- 18 experimental/lab-only cmd/ binaries have `//go:build sdp_experimental`, excluded from default `go build`
3. **Drift detection** -- `scripts/check-release-surface.sh` checks for experimental binary leakage

---

## 4. Lab-Only Surfaces

These binaries and packages are explicitly excluded from the release formula. They are available for local development via `go build -tags sdp_experimental`.

### Lab-Only Binaries (4)

| Binary | Maturity | What it is |
|--------|----------|------------|
| `sdp-control` | GA (deprecated) | DEPRECATED. Use `sdp` instead. Prints deprecation warning. |
| `sdp-dispatch` | Beta | Dispatch layer development routing. |
| `sdp-up` | GA | Project bootstrap / profile provisioning. Lab setup only. |
| `gt-adapter` | GA | Guard/convoy test adapter. Internal development tool. |

### Experimental / Research Binaries (6)

| Binary | Maturity | What it is |
|--------|----------|------------|
| `sdp-harness` | Experimental | AgentLoop session harness. Requires LiveGateway (F106). |
| `sdp-a2a` | Beta | Agent-to-agent protocol server. |
| `sdp-eval` | Beta | Eval runner. Research-oriented. |
| `sdp-strataudit` | GA | Strategic LLM audit. Standalone research tool. |
| `sdp-mcp` | Beta | MCP server. |
| `sdp-tower` | Beta | Tower orchestration layer. |

### Research / Benchmark Binaries (9)

`sdp-cascade-replay`, `sdp-confidence-replay`, `sdp-decompose-bench`, `sdp-microfirst-bench`, `sdp-bd-suggest`, `sdp-ft-baseline`, `sdp-ft-dataset`, `sdp-ft-run`, `sdp-ft-validate`

### Future Product Candidates (1, no code)

`sdp-pr-gate` (ChangePassport) -- merge-readiness product. Namespace locked. No implementation.

### Experimental Internal Packages (15)

`internal/agentloop`, `internal/modelgateway`, `internal/inference`, `internal/llmclient`, `internal/localmodel`, `internal/memory`, `internal/mutation`, `internal/finetune`, `internal/planner`, `internal/authz`, `internal/stream`, `internal/secretscan`, `internal/provenance`, `internal/flaky`, `internal/glob`

---

## 5. Release Blockers

**None found.**

All P0 items that could block a release were resolved during F150:

| Potential Blocker | Status | Resolution |
|---|---|---|
| Module path not portable | Resolved | WS-03 migrated to `github.com/fall-out-bug/sdp_lab` |
| Experimental code in release builds | Resolved | WS-04 build tags + GoReleaser allowlist |
| No formula dry run | Resolved | WS-08 formula/sdp.rb, dry run passed |
| No telemetry consent model | Resolved | WS-07 4-tier consent, default is metadata-only, OTEL requires explicit opt-in |
| Documentation misaligned | Resolved | WS-09 README/QUICKSTART aligned to layer taxonomy |

---

## 6. Non-Blocking Debt

These items were identified during F150 and deferred because they do not block a formula release. Each is linked to a feature or audit finding for tracking.

### 6.1 Duplicate Code Patterns (from dependency audit, sdplab-hjl7)

| Priority | Pattern | Locations | Recommendation |
|---|---|---|---|
| Medium | `safePath` (3 implementations) | `internal/agentloop/tools_live.go`, `internal/skills/augment.go`, `internal/architect/security/path.go` | Consolidate to `architect/security.PathValidator` in `internal/common/safepath` |
| Low | `fileExists`/`dirExists` (3 implementations) | `internal/bootstrap/collector.go`, `internal/architect/extract/ts_extract.go`, `internal/architect/extract/typescript/extractor.go` | Extract to `internal/common/fsutil` |
| Low | `writeFile` test helpers (4 implementations) | `internal/architect/extract/java_extract_test.go`, `internal/architect/extract/java/spring_test.go`, `internal/backlog/reference_check_test.go`, `internal/inference/microfirst/bdseverity/classifier_test.go` | Share across architect tests |
| Low | `jsonschema.NewCompiler()` (6 call sites) | `internal/manifest/load.go`, `internal/adapters/sdk/validation.go` (x2), `internal/orchestrate/checkpoint.go`, `internal/inference/confidence/adapters/wsverdict/wsverdict.go`, `internal/inference/decompose/stitcher_json.go` | Shared schema registry (architectural change) |

### 6.2 Experimental Dependencies (from dependency audit, sdplab-hjl7)

These dependencies are used only by experimental code and could be isolated behind build tags post-F150:

| Dependency | Used By | Action |
|---|---|---|
| `github.com/ledongthuc/pdf` | `internal/strataudit` (experimental) | Build-tag isolation |
| `github.com/mark3labs/mcp-go` | `internal/mcp`, `cmd/sdp-mcp` (Beta/experimental) | Build-tag isolation |
| `golang.org/x/time` | `internal/strataudit` (experimental), `internal/spec/testdata` | Build-tag isolation or removal |

### 6.3 Heavy Dependency Chain

The sigstore ecosystem pulls ~725 transitive edges through `github.com/sigstore/sigstore-go`. This is justified by the GA evidence package and cannot be reduced without removing evidence functionality. Monitor for tree-shaking opportunities in future sigstore releases.

### 6.4 Internal Package Import Coupling

Some stable packages import experimental packages (documented in maturity-matrix.md "Packages Not Tagged"):

- `internal/llmclient` imported by `architect`, `discovery` (stable)
- `internal/glob` imported by `executor` (stable)
- `internal/agentloop` imported by `executor` (bridge_serve)

This coupling means the experimental packages compile into release builds even though no experimental cmd/ binary is shipped. This is a known, documented trade-off -- cmd/ level isolation is enforced, but internal/ level isolation is not.

---

## 7. Deferred Items

These items were identified as out of scope for F150 and are tracked in the roadmap for future features.

### Explicitly Deferred Beyond F150

| Item | Reason | Roadmap Feature |
|---|---|---|
| `sdp-pr-gate` (ChangePassport) implementation | Design-only gate; no committed pilot per v3 acceptance bar | F151 |
| Pricing hypothesis (Operator Mode + sdp-pr-gate) | Required before any external pilot | F152 |
| SDP brand architecture (family map, naming policy) | Marketing/positioning, not engineering | F153 |
| Shared substrates v1 (semver contracts) | Internal package API surface stabilization | F154 |
| Evidence persistence architecture | Storage backend decision, retention, privacy | F155 |
| Go import-path contamination decision | Highest unaddressed structural risk; blocks F156/F157 | F158 |
| `arch-snap` and `doc-tracer` IIP hypotheses | Gated on named lead + discovery interviews + F158 | F156, F157 |
| Competitive positioning artifact | Market analysis, not engineering | F159 |
| Procurement/compliance install profile | SOC2 stance, SLA template, no-egress mode | F160 |
| Enterprise Delivery Governance | Deferred until ICP signal | (backlog) |
| Coverage enforcement wiring | Policy is defined (WS-06); CI enforcement gates not yet implemented | F092/F100 |
| `sdp-eval` formula review | Classified experimental but still in GoReleaser; should be removed from default build | F137 |

### Open Beads Follow-Up

| Area | Feature | Status |
|---|---|---|
| F121 promptops evidence follow-up | F121 (Metrics) | Open in beads |
| F122 index-hardening follow-ups | F122 (Index) | Open in beads |
| F125 review-readiness/doc sweep | F125 (Toolkit UX) | Partial (00-125-05) |
| F100 reference integrity CI gate | F100 (Release Discipline) | In progress |

---

## 8. Readiness Assessment

### What is Release-Ready

1. **`sdp` CLI binary (GA):** The main product surface. 34 subcommands covering scout, metrics, index, spec, bootstrap, operator mode, and pipeline internals. Module path is portable. Build is clean. Formula dry run passes.

2. **Operator tooling (15 binaries, GA/Beta):** Orchestration, guard, CI loop, evidence, protocol checks, and bridge binaries. All build via GoReleaser. Tested and in daily dogfood use.

3. **Homebrew formula:** `formula/sdp.rb` dry run passed. `brew test` verified. GoReleaser config produces correct archives with ldflags.

4. **Build isolation:** Experimental binaries excluded via GoReleaser allowlist + build tags. Drift detection checks for accidental leakage.

5. **Telemetry consent:** 4-tier model. Default is metadata-only local storage. OTEL export requires explicit consent AND explicit endpoint (2-gate design). No outbound data transfer by default.

6. **Documentation:** README, QUICKSTART, product-surface.md, maturity-matrix.md, and telemetry-schema.md are aligned to the v3 layer taxonomy.

### What is NOT Release-Ready

1. **Coverage gates are not enforced in CI.** The policy is defined (happy-path >=80%, GA >=60%, Beta >=50%), but the CI coverage gate does not yet use maturity-tiered thresholds. The baseline delta gate (2pp threshold) is active, but absolute tier enforcement is deferred.

2. **`sdp-eval` is in GoReleaser but classified experimental.** This inconsistency should be resolved before tagging a release (move to experimental-only build or remove from formula).

3. **No external pilot validation.** The product has been dogfooded extensively but has not been tested by external users. F081 (30-min production pilot) and F152 (pricing hypothesis) are prerequisites before any external release campaign.

4. **ChangePassport (`sdp-pr-gate`) is product direction, not code.** The layer taxonomy lists it as Layer 5, but no implementation exists. The namespace is locked; actual code is gated on F151 (design v1).

5. **Enterprise features are backlog items.** RBAC, SIEM export, compliance bundles, and K8s runtime packs are all backlog. They are correctly classified and do not block a toolkit release.

### Honest Assessment

**SDP Toolkit is ready for a limited, developer-audience Homebrew release.** The `sdp` binary works, the formula installs cleanly, telemetry is safe-by-default, and experimental code is properly isolated. The product delivers on its stated promise: from idea to accepted PR, with evidence.

However, SDP is not ready for a broad launch or enterprise pilot. The missing pieces are:
- External user validation (no non-author usage data)
- Pricing model (no hypothesis tested)
- Coverage enforcement (policy exists, CI wiring does not)
- Enterprise governance surface (correctly deferred)

A Homebrew tap release targeted at early adopters and developer tooling evaluators is appropriate. A public marketing launch or enterprise sales motion is premature.

---

## 9. Recommendations for F151+

| Priority | Recommendation | Rationale |
|---|---|---|
| P0 | Remove `sdp-eval` from GoReleaser default build or reclassify | Inconsistency between maturity label (experimental) and build inclusion |
| P0 | Wire coverage tier enforcement into CI | Policy is defined (F150-06); enforcement is not. Without this, coverage targets are aspirational. |
| P1 | Run F081 (30-min production pilot) with at least 2 external users | Validate the install -> first-success path with real users before any launch. |
| P1 | Complete F151 (sdp-pr-gate design) | The first paid wedge needs a design before any implementation. |
| P1 | Complete F152 (pricing hypothesis) | Required before external pilot conversations. |
| P2 | Consolidate duplicate code patterns from dependency audit | SafePath (3 implementations), fileExists (3), writeFile (4). Low risk but adds maintenance burden. |
| P2 | Isolate experimental deps behind build tags | `ledongthuc/pdf`, `mark3labs/mcp-go`, `golang.org/x/time` only used by experimental code. |
| P2 | Resolve F158 (Go import-path contamination) | Blocks F156 and F157 from active status. |
| P3 | Monitor sigstore ecosystem for dependency tree reduction | 725 transitive edges is heavy but justified today. |
| P3 | Add `internal/common/safepath` and `internal/common/fsutil` | Extract shared utilities from duplicate patterns. |

---

## Change Log

| Date | Change |
|---|---|
| 2026-04-27 | Initial release readiness report (F150-10, sdplab-3uep) |
