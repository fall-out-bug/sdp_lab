#!/usr/bin/env python3
"""
SDP Incubated Independent Products (IIP) — LLM Council orchestration (2026-04-27)

Question being challenged: tools that have value OUTSIDE SDP context
(e.g. arch-snap = architecture extraction; doc-tracer = docs<->code traceability).

memo v2 subordinates them as "SDP Toolbox" / freemium funnel. The author argues
this undersells tools whose buyers do not care about SDP at all (architects,
due-diligence, M&A, tech writers, compliance).

Proposal D: introduce a new taxonomy row "Incubated Independent Products" (IIP),
tools developed in sdp_lab monorepo BUT named without `sdp-` prefix from
inception, with strict isolation, own go.mod path, own semver, ready for
extraction when criteria met. SDP-team is maintainer, not brand owner.

Two rounds:
  R1 (Socratic) — each role attacks the 12 D-claims.
  R2 (Council)  — each role re-deliberates with all R1 outputs visible.

Models per user request 2026-04-27:
  - xiaomi/mimo-v2.5-pro
  - minimax/minimax-m2.7
  - moonshotai/kimi-k2.6
  - deepseek/deepseek-v4-pro
  - qwen/qwen3.6-plus
"""

from __future__ import annotations

import concurrent.futures as cf
import json
import os
import pathlib
import sys
import time
import urllib.error
import urllib.request

ROOT = pathlib.Path(__file__).parent.resolve()
REPO = pathlib.Path("/Users/fall_out_bug/projects/vibe_coding/sdp_lab")
ENV_FILE = REPO / ".env"

OPENROUTER_URL = "https://openrouter.ai/api/v1/chat/completions"
TIMEOUT_S = 600  # bumped for kimi-style long-reasoning models
MAX_TOKENS = 100000  # per user request "максимум 100 000"

ROLES = [
    {"role": "Architect",
     "primary": "xiaomi/mimo-v2.5-pro",
     "fallback": "xiaomi/mimo-v2.5",
     "focus": "challenge the new taxonomy row, dependency rules, repo topology, semver/extraction mechanics, isolation lint feasibility"},
    {"role": "Critic",
     "primary": "minimax/minimax-m2.7",
     "fallback": "minimax/minimax-m2.5",
     "focus": "attack unsubstantiated claims about independent value; demand evidence for ICP separateness; flag fragmentation and brand-dilution risks"},
    {"role": "Technician",
     "primary": "moonshotai/kimi-k2.6",
     "fallback": "moonshotai/kimi-k2.5",
     "focus": "feasibility of go.mod sub-paths, separate semver tag prefixes, separate Homebrew formulas, CI matrix complexity, isolation lint implementation"},
    {"role": "Philosopher",
     "primary": "deepseek/deepseek-v4-pro",
     "fallback": "deepseek/deepseek-v3.2",
     "focus": "naming, identity, brand strategy, product category coherence; whether IIPs can credibly stand alone without `sdp-` prefix; long-term ownership and licensing strategy"},
    {"role": "Pragmatist",
     "primary": "qwen/qwen3.6-plus",
     "fallback": "qwen/qwen3.5-plus",
     "focus": "commercial viability of standalone IIPs, ICP for arch-snap and doc-tracer, pricing independent of SDP, willingness-to-pay, time-to-revenue, fragmentation risk"},
]

CLAIMS = """
D1: Some tools (arch-snap = architecture extraction, doc-tracer = docs<->code traceability) have inherent value OUTSIDE SDP and should NOT be subordinated to the SDP-funnel framing from inception. They should be treated as future standalone products incubated in sdp_lab, not as freemium acquisition for SDP/Toolkit/ChangePassport.

D2: A new taxonomy row "Incubated Independent Products" (IIP) sits between SDP Lab (row 1) and SDP Toolbox (row 2) in memo v2 layering; tools here use NO `sdp-` prefix from inception (`arch-snap`, `doc-tracer`, not `sdp-arch-snap`, `sdp-doc-tracer`).

D3: Each IIP carries an `independent_value: yes` annotation in its module AGENTS.md; the AGENTS.md is written as if SDP did not exist (no `sdp-` references in 60-line cold-start text, no SDP-runtime assumptions).

D4: IIP tools MUST NOT import `internal/sdp-pr-gate/`, `internal/sdp-operator/`, or `internal/sdp-edg/`. Substrate imports (`sdp-evidence-core`, `sdp-policy-core`, `sdp-context-core`, etc.) are allowed only via pinned semver and only when the substrate has independently usable value. CI lint enforces this.

D5: Each IIP gets: own go.mod sub-path; own semver tag prefix (e.g., `arch-snap/v0.1.0`, `doc-tracer/v0.1.0`); own Homebrew formula or installer separate from the main `sdp` formula; own README and landing copy that does not reference SDP unless contextually necessary.

D6: IIP extraction criteria are stricter than SDP Toolbox promotion criteria. To extract to a separate repo, ALL of: (a) >= 2 external consumers using it weekly; (b) >= 50% of those consumers do NOT use SDP at all; (c) a distinct revenue or adoption signal independent of SDP funnel; (d) finalized brand decision (name, domain, license).

D7: The existing SDP Toolbox row in memo v2 narrows to ONLY tools whose value is fundamentally tied to SDP onboarding/adoption: `sdp-scout`, `sdp-metrics`, `sdp-index`, `sdp-spec`, `sdp-bootstrap`. Anything else with independent value migrates to the new IIP row.

D8: Two flagship IIP candidates today: `arch-snap` (architecture extraction from code; ICPs: due-diligence/M&A buyers, security architects, new CTOs onboarding to legacy codebases, tech writers building docs); `doc-tracer` (docs<->code traceability; ICPs: docs-as-code shops, compliance teams in regulated industries like FDA/ISO 13485, technical writers, audit functions).

D9: Each IIP gets its OWN beads epic (`F-ARCH-SNAP`, `F-DOC-TRACER`, etc.) tracked outside F150. F150 only ensures the architectural rules (cascade AGENTS.md, package-level isolation lint, `extractable: yes` annotation, `independent_value: yes` annotation) are in place; F150 does NOT ship any IIP.

D10: IIP brand strategy: the SDP team is the maintainer, not the brand owner. An IIP may eventually be cross-licensed, sponsored by other organizations, or spun out to a new entity. SDP gets no preferential treatment in the tool's marketing positioning. License default: permissive (Apache-2.0 or MIT) unless a specific commercial track requires otherwise.

D11: Pricing for IIPs is independent of SDP pricing. Each IIP gets its own willingness-to-pay hypothesis when commercial extraction is considered. Some IIPs may stay free/open-source forever; others may become commercial standalone products with their own pricing model. There is no requirement that IIP pricing align with SDP pricing.

D12: Cap of 3 active IIPs incubating simultaneously to prevent organizational fragmentation; remaining ideas stay in Lab. Promoting a Lab experiment to IIP status requires explicit founder/owner approval (not a default path). Existing IIP slot exits when a tool is extracted, archived, or downgraded back to Lab.
"""

CONTEXT_SUMMARY = """
Context: SDP product layering memo v2 (2026-04-27, post-council) at `/Users/fall_out_bug/projects/vibe_coding/sdp_lab/docs/strategy/2026-04-27-sdp-product-layering-4d.md` defined the following taxonomy:

memo v2 layers:
  1. SDP Lab — research workspace
  2. SDP Toolbox — subordinate freemium acquisition lever for SDP family (NOT a parallel product category); examples include doc-tracer, arch-snap, tok-economy, local-model-router, doc-analyzer
  3. SDP Toolkit — `sdp` CLI, multi-harness install
  4. Operator Mode — default Toolkit Happy Path embodying governed delivery; stateful orchestration
  5. ChangePassport (display) / `sdp-pr-gate` (internal namespace, locked) — first paid wedge
  6. Enterprise Delivery Governance — hypothesis, out of F150 scope
  7. Shared Substrates — semver packages

memo v2 council (5-model llm-council) consensus changes (just landed):
  - "Standalone Tools" -> "SDP Toolbox" (subordinate, not parallel category)
  - Operator Mode = default Happy Path embodying governed delivery, not "advanced feature"
  - "Enterprise Perimeter Control Plane" -> "Enterprise Delivery Governance" (drop "Perimeter")
  - hallucination rate -> evidence-mismatch rate
  - internal namespace `sdp-pr-gate` locked NOW, decoupled from `ChangePassport` display name
  - cascade AGENTS.md <=60 with executable migration plan
  - Wedge B (ChangePassport) gated on committed pilot
  - package-level isolation lint enforced now

User pushback on memo v2 (2026-04-27):
  > "А что делать с задачами, которые отлично живут ВНЕ SDP, и их могут хотеть использовать отдельно? Архитектор, Трассировка"
  (What about tools that live great OUTSIDE SDP and people might want to use separately? Architect, Tracing.)

User point: subordinating arch-snap and doc-tracer as "SDP Toolbox / freemium funnel" undersells their independent value and undermines their adoption among non-SDP buyers (architects, due-diligence, M&A, tech writers, compliance). The `sdp-` prefix actively repels these buyers when searching for "architecture extraction" or "docs traceability".

Author proposal (Option D — incubate-then-spin-out, recommended):
  - Tool starts in `sdp_lab` monorepo for development convenience
  - Named WITHOUT `sdp-` prefix from inception (`arch-snap`, `doc-tracer`)
  - Strict isolation from SDP-runtime modules (allowed: substrates only)
  - Own go.mod sub-path, own semver, own README, own AGENTS.md (60 lines, no SDP refs)
  - Own beads epic outside F150
  - Extraction to separate repo when independent demand signal arrives

Alternative options the author considered and rejected:
  A. Promote standalone NOW (separate repo, separate brand) — too early, no commercial validation
  B. Keep under SDP brand with independent landing pages — `sdp-` prefix repels non-SDP buyers
  C. Spin out completely now to separate repo — premature infrastructure overhead

The author's twelve claims (D1..D12) above codify Option D. Council job: challenge them aggressively. Each council role brings a specific angle (Architect = architecture/feasibility; Critic = unsubstantiated claims and risks; Technician = build/CI/semver mechanics; Philosopher = naming/identity/brand; Pragmatist = commercial/ICP/pricing).

Repo reality:
  - Today arch-snap and doc-tracer DO NOT YET EXIST in code. They are research / lab-only hypotheses in memo v2.
  - The cascade AGENTS.md <=60 rule is a target, not current state; root AGENTS.md is 606 lines.
  - The package-level isolation lint is planned for WS 00-150-04 in F150; not yet implemented.
  - The `extractable: yes` annotation is planned for WS 00-150-02; not yet implemented.

Decisions earlier in this session can and SHOULD be challenged. We may be wrong. We are willing to change our minds.
""".strip()


def load_api_key() -> str:
    if not ENV_FILE.exists():
        raise RuntimeError(f".env not found: {ENV_FILE}")
    for line in ENV_FILE.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if line.startswith("OPENROUTER_API_KEY="):
            return line.split("=", 1)[1].strip().strip('"').strip("'")
    raise RuntimeError("OPENROUTER_API_KEY not in .env")


def round1_prompt(role: str, focus: str) -> str:
    return f"""You are {role} in an llm-council protocol reviewing a product strategy decision.

The author asks: challenge the proposal aggressively. Earlier decisions can and SHOULD be revised. We may be wrong. We are willing to change our minds.

Your role focus: {focus}

For each claim D1..D12, provide:
- VERDICT: STRONG | OK | WEAK | OPPOSE | VETO
- EVIDENCE: 3-6 sentences explaining your reasoning. Cite specific assumptions, risks, or precedent (e.g., open-source projects that successfully spun out from a parent).
- PROPOSAL: if WEAK/OPPOSE/VETO, write a one-sentence revised claim.

After all claims:
- MINORITY REPORT: any claim where you predict you will disagree with majority, with rationale.
- DOMAIN VETOES: claims unacceptable from your role perspective, marked [DOMAIN VETO: <reason>].
- THREE BIGGEST RISKS the proposal fails to address.
- PRECEDENT REFERENCES: name 1-3 real open-source or commercial projects that succeeded or failed at incubate-then-spin-out (etcd from CoreOS, containerd from Docker, Ginkgo from Cloud Foundry, Tailwind from Laravel ecosystem, etc.) and what they teach about this strategy.

Be terse, specific, evidence-based. Avoid hedging. Russian or English — your choice.

CLAIMS:
{CLAIMS}

CONTEXT:
{CONTEXT_SUMMARY}
"""


def round2_prompt(role: str, focus: str, r1_blob: str) -> str:
    return f"""You are {role} in ROUND 2 of the llm-council protocol.

Below are ALL round-1 outputs from all roles (Architect, Critic, Technician, Philosopher, Pragmatist).

Your job:
1. For each claim D1..D12, give your FINAL verdict: ACCEPT | ACCEPT WITH REVISION | REJECT.
2. If "ACCEPT WITH REVISION", write the revised claim in one sentence.
3. Mark which claims you changed your mind on between R1 and R2 and WHY (specifically referencing which other role's argument moved you).
4. Identify the 3 most important RISKS the proposal still fails to address.
5. Identify the 3 most important CORRECTIONS the author should make before shipping the memo update.
6. Final overall recommendation: ACCEPT | ACCEPT WITH CHANGES (list) | REJECT.

Be terse, specific, decision-grade. Russian or English — your choice.

YOUR ROLE FOCUS (still active): {focus}

ROUND 1 OUTPUTS:
{r1_blob}

ORIGINAL CLAIMS (for reference):
{CLAIMS}

CONTEXT (for reference):
{CONTEXT_SUMMARY}
"""


def call_openrouter(api_key: str, model: str, prompt: str) -> dict:
    body = {
        "model": model,
        "messages": [{"role": "user", "content": prompt}],
        "temperature": 0.3,
        "max_tokens": MAX_TOKENS,
    }
    req = urllib.request.Request(
        OPENROUTER_URL,
        data=json.dumps(body).encode("utf-8"),
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
            "HTTP-Referer": "https://github.com/fall-out-bug/sdp_lab",
            "X-Title": "SDP IIP Council 2026-04-27",
        },
        method="POST",
    )
    t0 = time.time()
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT_S) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            elapsed = time.time() - t0
            return {"ok": True, "model": model, "elapsed_s": elapsed, "data": data}
    except urllib.error.HTTPError as e:
        body_text = e.read().decode("utf-8", errors="replace")
        return {"ok": False, "model": model, "elapsed_s": time.time() - t0,
                "error": f"HTTP {e.code}", "body": body_text}
    except Exception as e:
        return {"ok": False, "model": model, "elapsed_s": time.time() - t0,
                "error": f"{type(e).__name__}: {e}"}


def call_with_fallback(api_key: str, role_def: dict, prompt: str) -> dict:
    primary = call_openrouter(api_key, role_def["primary"], prompt)
    if primary["ok"]:
        choices = primary.get("data", {}).get("choices") or []
        if choices and (choices[0].get("message") or {}).get("content"):
            primary["role"] = role_def["role"]
            primary["used_fallback"] = False
            return primary
    fallback = call_openrouter(api_key, role_def["fallback"], prompt)
    fallback["role"] = role_def["role"]
    fallback["used_fallback"] = True
    fallback["primary_error"] = primary.get("error") or "no/empty content in primary response"
    return fallback


def extract_text(resp: dict) -> str:
    if not resp.get("ok"):
        return f"[ERROR: {resp.get('error')}]\n{resp.get('body', '')}"
    choices = resp.get("data", {}).get("choices") or []
    if not choices:
        return "[ERROR: no choices in response]"
    msg = choices[0].get("message", {})
    return msg.get("content") or "[ERROR: empty content]"


def run_round(api_key: str, prompts_by_role: dict, label: str) -> dict:
    results: dict = {}
    with cf.ThreadPoolExecutor(max_workers=len(ROLES)) as ex:
        futures = {
            ex.submit(call_with_fallback, api_key, rd, prompts_by_role[rd["role"]]): rd
            for rd in ROLES
        }
        for fut in cf.as_completed(futures):
            rd = futures[fut]
            try:
                resp = fut.result()
            except Exception as e:
                resp = {"ok": False, "role": rd["role"], "error": f"{type(e).__name__}: {e}"}
            results[rd["role"]] = resp
            print(f"[{label}] {rd['role']}: {'OK' if resp.get('ok') else 'FAIL'} "
                  f"({resp.get('model', '?')}, {resp.get('elapsed_s', 0):.1f}s, "
                  f"fallback={resp.get('used_fallback', False)})", flush=True)
    return results


def write_outputs(results: dict, round_label: str) -> None:
    raw_path = ROOT / f"{round_label}-raw.json"
    raw_path.write_text(json.dumps(results, indent=2, ensure_ascii=False), encoding="utf-8")
    for role, resp in results.items():
        md_path = ROOT / f"{round_label}-{role.lower()}.md"
        text = extract_text(resp)
        usage = resp.get("data", {}).get("usage", {}) if resp.get("ok") else {}
        finish = ((resp.get("data", {}).get("choices") or [{}])[0] or {}).get("finish_reason") if resp.get("ok") else "n/a"
        header = (
            f"# {round_label.upper()} — {role}\n\n"
            f"Model: `{resp.get('model', '?')}`  \n"
            f"Fallback used: `{resp.get('used_fallback', False)}`  \n"
            f"Elapsed: {resp.get('elapsed_s', 0):.1f}s  \n"
            f"OK: `{resp.get('ok')}`  \n"
            f"Finish: `{finish}`  \n"
            f"Usage: `{usage}`\n\n---\n\n"
        )
        md_path.write_text(header + text, encoding="utf-8")


def build_r2_blob(r1_results: dict) -> str:
    parts = []
    for rd in ROLES:
        role = rd["role"]
        text = extract_text(r1_results.get(role, {}))
        parts.append(f"<{role.upper()}>\n{text}\n</{role.upper()}>")
    return "\n\n".join(parts)


def main() -> int:
    api_key = load_api_key()
    print(f"[init] {len(ROLES)} roles, MAX_TOKENS={MAX_TOKENS}, output dir: {ROOT}", flush=True)

    r1_prompts = {rd["role"]: round1_prompt(rd["role"], rd["focus"]) for rd in ROLES}
    print("[r1] dispatching round 1 (Socratic)...", flush=True)
    r1 = run_round(api_key, r1_prompts, "r1")
    write_outputs(r1, "r1")

    r1_blob = build_r2_blob(r1)
    r2_prompts = {rd["role"]: round2_prompt(rd["role"], rd["focus"], r1_blob) for rd in ROLES}
    print("[r2] dispatching round 2 (Council)...", flush=True)
    r2 = run_round(api_key, r2_prompts, "r2")
    write_outputs(r2, "r2")

    print(f"[done] outputs in {ROOT}", flush=True)
    return 0


if __name__ == "__main__":
    sys.exit(main())
