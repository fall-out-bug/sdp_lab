# F150 Product Layering And Release Readiness Design

Status: active design
Owner: Andrei
Beads epic: `sdplab-nyr0`
Created: 2026-04-27

## Cold Start Answers

1. This is platform work, not "use SDP in my project" onboarding.
2. The owner is `F150` / `sdplab-nyr0`.
3. Canonical context: `docs/reference/project-map.md`, `docs/reference/product-surface.md`, `docs/reference/maturity-matrix.md`, `docs/MULTI-REPO-WORKFLOW.md`, and the 2026-04-26 ChangePassport / Enterprise Perimeter research documents.
4. This document is Discovery output. The linked workstreams are Delivery units.
5. Protocol artifact publishing is not automatic. Publish to the public `sdp` repo only if a later workstream changes `prompts/`, `schema/`, `templates/`, hooks, or harness entrypoints that external users need.

## Problem

The request "global review and refactor the repo" is too broad to execute safely.

It mixes at least five different jobs:

1. product-layering and market framing;
2. public install and Homebrew packaging;
3. Go module, build, dependency, and duplicate-code cleanup;
4. telemetry and consent;
5. explicit debt capture.

Treating those as one refactor would create a large diff with weak acceptance criteria. F150 turns the request into a product architecture decision plus executable readiness lanes.

## AI Fluency 4D Reframe

### Delegation

Delegate the work to narrow lanes:

- product taxonomy and SKU boundaries;
- release surface inventory;
- module path migration;
- experimental build isolation;
- dependency and duplicate-code audit;
- coverage policy rebaseline;
- telemetry consent and OTEL export;
- Homebrew formula dry run;
- product-facing documentation;
- final debt ledger.

### Description

The executable description is not "make the repo cleaner." It is:

> Prepare the selected SDP product surfaces for public installation and evaluation by defining product layers, release boundaries, build scope, coverage policy, telemetry consent, and tracked debt.

### Discernment

Quality is decided through evidence:

- release surface inventory exists before code changes;
- `go test`, `go vet`, and repo quality gates pass for the selected surface;
- dependency removals are backed by tests or no-op diffs;
- coverage thresholds are maturity-specific;
- telemetry export requires explicit consent;
- Homebrew formula install/test is exercised locally;
- deferred work has Beads ownership.

### Diligence

The product cannot hide lab instability. Anything not release-ready must be marked as one of:

- stable product surface;
- operator tooling;
- lab-only experiment;
- future product candidate;
- retired or archived.

## Product Layering Decision

The answer is not a pure matryoshka and not a set of unrelated products.

The correct model is:

> parallel product surfaces over shared technical substrates, with strict dependency rules.

Product buyers see separate offers. Engineers maintain shared core packages where reuse is real and versioned.

## Layers

### 1. SDP Lab

Role: research and platform workspace.

Owns:

- experimental Go binaries;
- inference research;
- agentloop experiments;
- internal operator tooling;
- protocol proposals;
- workstreams, Beads, evals, and dogfood artifacts.

Rule: `sdp_lab` is never a customer runtime dependency for ChangePassport or the Enterprise Perimeter product. It can feed them as an evidence provider or reference implementation.

### 2. SDP Toolkit

Role: installable developer/toolkit surface.

Primary distribution target for the first Homebrew formula.

Owns:

- `sdp` CLI;
- multi-harness install and adapter generation;
- `scout`, `metrics`, `index`, `spec`, `bootstrap`;
- manifest validation and adapter doctor commands;
- low-risk read-only adoption path.

Rule: this layer must be useful without ChangePassport, agentloop, enterprise deployment, or local model routing.

### 3. SDP Operator Mode

Role: queue-backed delivery workflow for teams that want Beads, workstreams, evidence, and QA/UAT.

Owns:

- `sdp-orchestrate`;
- `sdp-ready`;
- `sdp-ci-loop`;
- workstream and evidence gates;
- findings loop.

Rule: operator mode may remain advanced tooling. It should not block Toolkit install or first-run UX.

### 4. ChangePassport

Role: merge-readiness product.

Owns:

- Passport Schema v1;
- Evidence Provider API v1;
- Decision Record v1;
- override protocol;
- Markdown and JSON passport renderers;
- GitHub PR Gate Loop v1.

Rule: ChangePassport may consume SDP evidence, but it must run without `sdp_lab`. Its paid object is the governed readiness decision, not a report.

### 5. Enterprise Perimeter Delivery Control Plane

Role: enterprise on-prem / private-cloud control layer for AI-assisted software delivery.

Owns:

- model gateway and provider allowlist;
- local/sovereign model routing;
- context compiler;
- evidence provider mesh;
- audit policy;
- OTEL and local observability;
- deployment blueprints.

Rule: this is not "local Copilot." It packages governed delivery inside an enterprise perimeter. First wedge should be MR readiness plus ChangePassport for GitLab Self-Managed, not autonomous feature delivery.

### 6. Shared Substrates

These are technical assets, not products by themselves:

- evidence and policy primitives;
- telemetry schema and local trace storage;
- model gateway and inference cascade primitives;
- manifest and adapter registry;
- common CLI helpers;
- eval harness.

Rule: shared substrates must not leak lab instability into product surfaces. A substrate becomes product-facing only after it has docs, tests, owner, maturity label, and release contract.

## Dependency Rules

| Product surface | May depend on | Must not depend on |
|---|---|---|
| SDP Toolkit | stable CLI helpers, manifest/adapters, read-only analysis packages | operator Beads workflow, ChangePassport, enterprise runtime |
| SDP Operator Mode | Toolkit primitives, Beads, evidence, workstreams | ChangePassport product schema as a hard dependency |
| ChangePassport | evidence provider contracts, Git provider adapters, renderer, decision store | `sdp_lab` workstreams, Beads, lab-only binaries |
| Enterprise Perimeter Control Plane | ChangePassport, model gateway, telemetry, deployment blueprints | unversioned lab experiments |
| SDP Lab | everything experimental | customer-facing stability claims |

## Release Surface Policy

F150 treats `sdp` as the first formula target.

The formula should not install every `cmd/sdp-*` binary by default. The release inventory must decide which helpers are:

- bundled and linked;
- bundled but hidden;
- source-only;
- lab-only;
- excluded by build tags;
- retired.

The current module path `sdp_dev` must be replaced. The default migration target is:

```text
github.com/fall-out-bug/sdp_lab
```

Reason: Go source currently lives in `sdp_lab`. Using `github.com/fall-out-bug/sdp` would point Go users at the distilled protocol mirror, not the source module.

## Experimental Isolation

Experimental code should be isolated, not deleted.

Allowed mechanisms:

- release allowlist in GoReleaser;
- explicit build tags such as `sdp_experimental`;
- package-level maturity annotations;
- docs marking lab-only commands;
- separate formula or cask later if needed.

Not allowed:

- hiding compile failures by excluding tests silently;
- deleting integration tests to make gates green;
- making unstable functionality part of the stable formula by accident.

## Coverage Policy

Accepted policy:

- happy paths: `>=80%`;
- GA components: `>=60%`;
- Beta components: `>=50%`;
- Experimental components: measured only when useful, not gate-blocking.

This supersedes stronger blanket claims in older docs. The denominator must be explicit: package, command, product surface, or happy-path scenario.

## Telemetry Policy

Telemetry export must be opt-in.

Default:

- local trace storage may exist;
- no external export;
- no collector endpoint called.

Export requires:

- explicit consent;
- explicit OTEL collector endpoint;
- visible command/config path;
- schema allowlist;
- no content export unless consent level permits it;
- documented disable path.

## Execution Plan

| WS | Purpose | First useful output |
|---|---|---|
| 00-150-01 | product layering and SKU boundary | this design + registered workstreams |
| 00-150-02 | release surface inventory | keep/exclude matrix for CLI/packages |
| 00-150-03 | module path migration | `sdp_dev` removed from active Go imports |
| 00-150-04 | experimental isolation | release build excludes lab-only code |
| 00-150-05 | dependency and duplicate audit | concrete removals or filed debt |
| 00-150-06 | coverage policy | maturity-aligned checks/docs |
| 00-150-07 | telemetry consent/OTEL | opt-in export contract |
| 00-150-08 | Homebrew formula dry run | local formula install/test evidence |
| 00-150-09 | product docs alignment | README/product-surface match layers |
| 00-150-10 | readiness report | shipped/blocked/deferred ledger |

## Non-goals

- Release a tagged version in this workstream.
- Publish to the public `sdp` repo before relevant PRs merge.
- Build a full ChangePassport implementation before Schema/API v1 are locked.
- Build the Enterprise Perimeter Control Plane before the product boundary is stable.
- Delete experimental code as a shortcut.

## Open Decisions

1. Whether ChangePassport lives in this repo initially or starts as a separate repo when Schema/API v1 begins.
2. Whether Homebrew installs only `sdp` or also selected helper binaries.
3. Whether `sdp_lab` remains the long-term Go module path or only a transitional module path before a product repo split.
4. Whether release builds should use build tags or GoReleaser allowlists as the primary isolation mechanism.

## Success Criteria

- Product layers are understandable without reading historical plans.
- A user can install the first formula without receiving lab-only commands as a product promise.
- Active Go imports no longer use `sdp_dev`.
- Experimental code is isolated from stable release builds.
- Telemetry export is impossible without explicit consent.
- Remaining debt is in Beads with owners and not hidden in prose.
