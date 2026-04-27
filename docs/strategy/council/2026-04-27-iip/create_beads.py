#!/usr/bin/env python3
"""
Create F151..F160 epics and their child workstreams as beads.

Run once. Outputs created IDs to stdout.
Idempotent: skips creation if an exact title match already exists.
"""

from __future__ import annotations

import json
import re
import subprocess
import sys


def bd(*args: str) -> tuple[int, str, str]:
    proc = subprocess.run(
        ["bd"] + list(args),
        capture_output=True,
        text=True,
    )
    return proc.returncode, proc.stdout, proc.stderr


def list_titles() -> set[str]:
    code, out, err = bd("list", "--status=open", "--json")
    if code != 0:
        # fallback: parse non-json
        code2, out2, err2 = bd("list", "--status=open")
        titles = set()
        for line in out2.splitlines():
            m = re.search(r"sdplab-[a-z0-9]+\s+.*?\]\s+(.*)", line)
            if m:
                titles.add(m.group(1).strip())
        return titles
    try:
        data = json.loads(out)
        items = data if isinstance(data, list) else data.get("issues", data.get("data", []))
        return {item.get("title", "") for item in items}
    except json.JSONDecodeError:
        return set()


def create_issue(title: str, description: str, type_: str, priority: int, parent: str | None = None) -> str:
    args = [
        "create",
        f"--title={title}",
        f"--description={description}",
        f"--type={type_}",
        f"--priority={priority}",
    ]
    if parent:
        args.append(f"--parent={parent}")
    code, out, err = bd(*args)
    if code != 0:
        print(f"FAILED to create {title!r}: {err}", file=sys.stderr)
        sys.exit(1)
    m = re.search(r"sdplab-[a-z0-9]+", out)
    if not m:
        print(f"NO ID in output for {title!r}: {out}", file=sys.stderr)
        sys.exit(1)
    issue_id = m.group(0)
    print(f"  + {issue_id}  {title}")
    return issue_id


# -- Epic + children specs ---------------------------------------------------

EPICS = [
    {
        "title": "F151: sdp-pr-gate Design v1 (Schema, API, Decision Record, Override, GitHub App design)",
        "priority": 2,
        "description": (
            "Freeze Schema v1 + Evidence Provider API v1 + Decision Record v1 + Override protocol "
            "+ GitHub App v1 flow + pilot measurement plan. Design only, no implementation. "
            "Source: ChangePassport manifesto v2 §Build Next. "
            "Internal namespace `sdp-pr-gate` (locked). Display name `ChangePassport`. "
            "Implementation track is separate and gated on a committed pilot. "
            "See docs/strategy/2026-04-27-sdp-product-layering-4d.md (memo v3) and "
            "docs/roadmap/2026-04-27-roadmap-v3-post-iip.md."
        ),
        "children": [
            ("F151-01: Passport Schema v1 (JSON Schema)",
             "Author Passport Schema v1 as a JSON Schema document. Cover intent, scope, actors, evidence, findings, risk, decision. "
             "Stable contract; manual annotations cannot overwrite observed facts. WS: 00-151-01."),
            ("F151-02: Evidence Provider API v1",
             "Define Evidence Provider API v1 contract (schema_version, source, external_ref, repository, pull_request, commit_sha, "
             "observed_at, collected_at, actor, event_type, status, summary, artifact_uri, artifact_hash, error_state). "
             "Idempotent ingestion. Provider states: accepted/rejected/degraded. WS: 00-151-02."),
            ("F151-03: Decision Record v1",
             "Decision Record schema: decision (merge|hold|rework|escalate|override), owner, reason, timestamp, linked evidence snapshot, "
             "append-only audit reference. WS: 00-151-03."),
            ("F151-04: Override protocol",
             "Override protocol design: who can override, comment-trigger format, required fields (owner+reason+timestamp+evidence snapshot), "
             "audit log entry. GitHub PR comment example. WS: 00-151-04."),
            ("F151-05: GitHub App v1 flow design (auth, webhook, check, comment-override)",
             "GitHub App v1 flow: install scopes (least privilege), webhook events (PR open/sync/comment, check_run), check status mapping "
             "(ready/hold/rework/escalate), comment-override parsing, decision-finalize path. No implementation. WS: 00-151-05."),
            ("F151-06: Pilot measurement plan",
             "Pilot measurement plan: install time target (≤30 min), passport gen time (≤60 sec post-checks), useful-decision rate (≥70%), "
             "evidence-mismatch rate (<5%), false-block rate (<5%), reviewer time delta (≥-20% in 4-week pilot), post-merge incident rate "
             "(not above baseline). Sample size, baseline window, stop/go rules. WS: 00-151-06."),
        ],
    },
    {
        "title": "F152: Pricing Hypothesis (Operator Mode + sdp-pr-gate)",
        "priority": 2,
        "description": (
            "Provisional pricing hypothesis required before any external pilot per IIP-council Pragmatist. "
            "Not a commitment; it is the measurement instrument that lets a pilot answer 'does the buyer value this enough to pay'. "
            "Includes ≥3 discovery interviews. "
            "Memo v3 §Operator Mode and §Wedge Ordering."
        ),
        "children": [
            ("F152-01: Operator Mode pricing hypothesis doc",
             "Per-active-repo or per-team monthly base; included evidence/workstream volume; expansion path Operator → ChangePassport → EDG; "
             "comparable references (Tabnine, GitLab Duo, CodeRabbit). WS: 00-152-01."),
            ("F152-02: sdp-pr-gate pricing hypothesis doc",
             "Per-active-repo per-month base; included monthly governed-decision volume; overage by decision; expansion path. "
             "Reference manifesto v2 §Packaging Hypothesis. WS: 00-152-02."),
            ("F152-03: ≥3 discovery interviews with target ICP",
             "Boutique consulting / agency / fractional CTO with 10-50 engineers and ≥8 AI-assisted PRs/week. "
             "Interview script: review-burden quantification, current AI-PR governance pain, willingness-to-pay ranges, "
             "competitive alternatives. Document each in commercial-discovery/<date>-<org>.md. WS: 00-152-03."),
        ],
    },
    {
        "title": "F153: SDP Brand Architecture",
        "priority": 2,
        "description": (
            "Resolve SDP brand family confusion (Lab / Toolbox / Toolkit / Operator / ChangePassport / EDG / IIPs). "
            "Required before first external launch (memo-council Philosopher; IIP-council all roles flagged Toolbox-as-funnel framing). "
            "Independent of F150 closure; can start now."
        ),
        "children": [
            ("F153-01: Brand family map artifact",
             "One-page map showing Lab / Toolbox (with IIP flag) / Toolkit / Operator Mode / ChangePassport (display) | sdp-pr-gate (internal) / "
             "Enterprise Delivery Governance / Shared Substrates. Names, target audience, paid status. "
             "Output: docs/strategy/sdp-brand-architecture.md. WS: 00-153-01."),
            ("F153-02: Naming policy",
             "When `sdp-` prefix is required vs forbidden. Working-name rename criteria. Display-vs-internal namespace decoupling. "
             "Naming rules for Toolbox tools and IIP candidates. WS: 00-153-02."),
            ("F153-03: Trademark and domain check on key working names",
             "Check `ChangePassport`, `arch-snap`, `doc-tracer`, `sdp-pr-gate`, `Enterprise Delivery Governance` for trademark collisions and "
             "domain availability (.com, .ai, .io, .dev). Capture results in docs/strategy/naming/<name>.md. WS: 00-153-03."),
        ],
    },
    {
        "title": "F154: Shared Substrates v1 (semver contracts + SDP-runtime assumption docs)",
        "priority": 2,
        "description": (
            "Lock semver v1 API contracts for `sdp-evidence-core`, `sdp-policy-core`, `sdp-modelgw-core`, `sdp-context-core`, `sdp-eval-core`. "
            "Each substrate AGENTS.md must document SDP-runtime assumptions (context objects, env vars, config) so IIP imports can be "
            "audited. Risk: substrate transitive coupling — substrate may carry implicit SDP-runtime assumptions even with clean import paths. "
            "Memo v3 §Shared Substrates and §IIP unaddressed risks #2."
        ),
        "children": [
            ("F154-01: sdp-evidence-core v1 (semver + assumption doc)",
             "API v1 of evidence primitives. Public types, function signatures, ingest/render contracts, idempotency rules. "
             "AGENTS.md ≤60 lines documenting SDP-runtime assumptions. WS: 00-154-01."),
            ("F154-02: sdp-policy-core v1",
             "API v1 of policy primitives. Rule types, evaluation contract, override hooks. ≤60-line AGENTS.md with assumptions. WS: 00-154-02."),
            ("F154-03: sdp-modelgw-core v1",
             "API v1 of model-gateway primitives. Provider interface, allowlist contract, cost/latency envelope, fallback rules. "
             "≤60-line AGENTS.md with assumptions. WS: 00-154-03."),
            ("F154-04: sdp-context-core v1",
             "API v1 of context-compiler primitives. Repo map, diff-aware retrieval, prompt-budget contract, cache hash. "
             "≤60-line AGENTS.md with assumptions. WS: 00-154-04."),
            ("F154-05: sdp-eval-core v1",
             "API v1 of eval-harness primitives. Task-class evals, baseline comparison, scoreboard, hallucination/evidence-mismatch metric. "
             "≤60-line AGENTS.md with assumptions. WS: 00-154-05."),
        ],
    },
    {
        "title": "F155: Evidence Persistence Architecture",
        "priority": 3,
        "description": (
            "Decision artifact: storage backend (git LFS vs object storage vs local SQLite vs MCP server), retention, backup, privacy policy. "
            "Required before Schema v1 freeze (F151) so decisions don't get unwound. Surfaced by IIP-council Technician as risk #7."
        ),
        "children": [
            ("F155-01: Evidence persistence decision artifact",
             "Compare storage backends across: durability, append-only log support, replay/rebuild, storage cost, privacy/redaction, backup, "
             "compliance, ops complexity. Recommend one per surface (Operator Mode, sdp-pr-gate). Output: docs/strategy/evidence-persistence-architecture.md. WS: 00-155-01."),
        ],
    },
    {
        "title": "F156: arch-snap IIP Hypothesis (gated)",
        "priority": 3,
        "description": (
            "HYPOTHESIS-ONLY tracker for arch-snap (architecture extraction from code). "
            "No code work allowed until promotion gate lands: named IIP lead + commercial_hypothesis.md + ≥3 discovery interviews + F158 import-path decision applied. "
            "ICPs to validate via discovery: due-diligence/M&A buyers, security architects, new CTOs onboarding to legacy codebases, tech writers building docs. "
            "Memo v3 §IIP hypotheses currently under evaluation."
        ),
        "children": [
            ("F156-01: Land arch-snap promotion gate (named lead + commercial_hypothesis.md + ≥3 interviews)",
             "When ALL four are true (named lead, commercial_hypothesis.md exists in arch-snap module, ≥3 documented discovery interviews, "
             "F158 decision applied): promote F156 to active IIP and carve child workstreams (skeleton repo, AGENTS.md ≤60, semver v0.1, etc.). "
             "Until then, this is the only child. WS: 00-156-01."),
        ],
    },
    {
        "title": "F157: doc-tracer IIP Hypothesis (gated)",
        "priority": 3,
        "description": (
            "HYPOTHESIS-ONLY tracker for doc-tracer (docs↔code traceability). "
            "Same gate as F156: named IIP lead + commercial_hypothesis.md + ≥3 discovery interviews + F158 import-path decision applied. "
            "ICPs to validate via discovery: docs-as-code shops, compliance teams (FDA, ISO 13485, GxP), regulated industries, audit functions. "
            "Memo v3 §IIP hypotheses currently under evaluation."
        ),
        "children": [
            ("F157-01: Land doc-tracer promotion gate (named lead + commercial_hypothesis.md + ≥3 interviews)",
             "When ALL four gate conditions are met: promote F157 to active IIP. Until then, this is the only child. WS: 00-157-01."),
        ],
    },
    {
        "title": "F158: Go Import-Path Contamination Decision",
        "priority": 2,
        "description": (
            "Highest unaddressed structural risk per IIP-council (all 5 roles flagged). "
            "Module path `github.com/<sdp-lab-org>/sdp_lab/arch-snap` permanently associates IIP with SDP in every Go import statement, "
            "potentially suppressing non-SDP adoption. Three options to analyze: "
            "(A) neutral GitHub org from inception, "
            "(B) accept contamination during incubation, change at extraction, "
            "(C) hybrid (monorepo at very early phase, neutral org at v0.1). "
            "Decision blocks F156 and F157 promotion to active IIP. "
            "Memo v3 §IIP unaddressed risks #1 and §Open Items 14."
        ),
        "children": [
            ("F158-01: Decision artifact for Go import-path contamination (options A/B/C analysis + recommendation)",
             "Compare A/B/C across: non-SDP adopter perception, dev-experience cost, infrastructure cost, governance complexity, extraction "
             "mechanics. Recommend one. Output: docs/strategy/iip-import-path-decision.md. WS: 00-158-01."),
        ],
    },
    {
        "title": "F159: Competitive Positioning Artifact",
        "priority": 3,
        "description": (
            "Differentiator analysis vs Copilot Workspace, CodeRabbit, GitLab Duo Self-Hosted, Tabnine Enterprise, Factory Droid, "
            "OpenHands Enterprise, Sourcegraph. Required because competitive moat erosion is a real risk (IIP-council Pragmatist). "
            "Output: positioning statement and battle card."
        ),
        "children": [
            ("F159-01: Competitive positioning + battle card",
             "Per-competitor: what they do, where they win, where SDP differentiates (agent-neutral cross-tool evidence + override trail + "
             "Operator Mode + Toolbox + IIP family). Position statement, top 3 buying objections, top 3 SDP rebuttals. "
             "Output: docs/strategy/competitive-positioning.md. WS: 00-159-01."),
        ],
    },
    {
        "title": "F160: Procurement / Compliance Install Profile",
        "priority": 3,
        "description": (
            "Install profile that survives basic security review for dev-led-to-manager-paid path. SOC2 stance, SLA template, "
            "indemnification template, no-egress-by-default, scoped GitHub App permissions, data-residency posture. "
            "Required before any regulated-industry pilot. Memo-council and IIP-council both flagged."
        ),
        "children": [
            ("F160-01: Procurement-ready install profile artifact",
             "Document: scoped permissions, no-egress mode, data residency, SOC2/SOC3 posture, SLA template, indemnification template, "
             "vendor-onboarding answers (DPA, BCP/DR, sub-processors). Output: docs/strategy/procurement-install-profile.md. WS: 00-160-01."),
        ],
    },
]


def main() -> int:
    existing = list_titles()
    if existing:
        print(f"[init] {len(existing)} existing open issues; will skip exact-title matches")

    summary: list[tuple[str, str]] = []
    for spec in EPICS:
        epic_title = spec["title"]
        if epic_title in existing:
            print(f"[skip] epic exists: {epic_title}")
            continue
        epic_id = create_issue(epic_title, spec["description"], "epic", spec["priority"])
        summary.append((epic_id, epic_title))
        for ch_title, ch_desc in spec["children"]:
            if ch_title in existing:
                print(f"  [skip] child exists: {ch_title}")
                continue
            child_id = create_issue(ch_title, ch_desc, "task", spec["priority"], parent=epic_id)
            summary.append((child_id, ch_title))

    print()
    print("=" * 60)
    print("Created issues:")
    for issue_id, title in summary:
        print(f"  {issue_id}  {title}")
    print(f"\nTotal: {len(summary)} issues")
    return 0


if __name__ == "__main__":
    sys.exit(main())
