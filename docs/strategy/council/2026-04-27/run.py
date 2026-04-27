#!/usr/bin/env python3
"""
SDP Product Layering — LLM Council orchestration (2026-04-27)

Two rounds:
  R1 (Socratic) — each role attacks the strategy memo claims.
  R2 (Council)  — each role re-deliberates with all R1 outputs visible.

Models: 5 non-Anthropic, non-OpenAI top models on OpenRouter as of April 2026.
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
TIMEOUT_S = 180

ROLES = [
    {"role": "Architect",
     "primary": "google/gemini-3.1-pro-preview",
     "fallback": "google/gemini-3.0-pro",
     "focus": "challenge layer boundaries, dependency rules, repo topology, cascade AGENTS.md feasibility"},
    {"role": "Critic",
     "primary": "x-ai/grok-4.20",
     "fallback": "x-ai/grok-4",
     "focus": "attack unsubstantiated claims, broad framing, process theater risk, weak evidence"},
    {"role": "Technician",
     "primary": "moonshotai/kimi-k2.6",
     "fallback": "moonshotai/kimi-k2.5",
     "focus": "feasibility, implementation cost, tooling, CI, semver discipline, extraction lifecycle"},
    {"role": "Philosopher",
     "primary": "deepseek/deepseek-v4-pro",
     "fallback": "deepseek/deepseek-v3.2",
     "focus": "naming, identity, product category coherence, market positioning, brand strategy"},
    {"role": "Pragmatist",
     "primary": "qwen/qwen3.6-plus",
     "fallback": "qwen/qwen3.5-plus",
     "focus": "commercial viability, ICP, wedge ordering, paid object, pricing, time-to-revenue"},
]

CLAIMS = """
C1: Operator Mode is NOT a separate product layer; it is an advanced Toolkit feature surfaced when a team enables Beads + workstreams + evidence collection.
C2: Standalone Tools is a first-class new product category — single-purpose extractable utilities (doc-tracer, arch-snap, tok-economy, local-model-router, doc-analyzer) that may eventually live in their own repos.
C3: The architectural meta-rule is the cascade AGENTS.md ≤60 lines model: every separable surface MUST be cold-startable from root AGENTS.md (≤60) + module AGENTS.md (≤60). Current 606-line root is the migration target.
C4: Two parallel commercial wedges: Wedge A = free dev adoption via Toolkit + selected Standalone Tools on Homebrew; Wedge B = first paid via ChangePassport GitHub PR Gate Loop v1 after Schema v1 lock.
C5: Enterprise Perimeter Control Plane is out of F150 scope; only a reserved slot in the layer model.
C6: Russian sovereign model adapters (GigaChat, YandexGPT, MWS, vLLM/NIM/Ollama) are a separate F-track of multiple epics, not part of F150.
C7: ChangePassport is a working name; rename criteria are: domain available, no trademark collision, ICP recognizes the name, council/buyer language test passes. Until then keep `ChangePassport`.
C8: Shared Substrates must be explicitly versioned packages (e.g., sdp-evidence-core, sdp-policy-core, sdp-modelgw-core) with semver contracts and deprecation policy — not vague "technical assets".
C9: ChangePassport repo split is a downstream event triggered by Schema v1 + Evidence Provider API v1 + Decision Record v1 freeze AND first external pilot landing — not an upfront F150 commitment.
C10: F150 keeps 10 workstreams (00-150-01..10) without renumbering; two optional additions (Standalone Tools registry, cascade AGENTS.md migration plan) are defer-able and can fold into 00-150-09.
C11: Discernment metrics per surface (pilot-stage targets, not GA SLOs): Toolkit install ≤30 min; ChangePassport useful decision rate ≥70%, hallucination <5%, install ≤30 min, passport ≤60 sec post-checks, reviewer time -20% in 4-week pilot, false-block <5%; Enterprise Perimeter install ≤2-4 weeks, useful suggestion ≥30-40%.
C12: Operator Mode is treated as a Toolkit Happy Path (matching product-surface.md §"Run Operator Mode"). If signals emerge that buyers want Operator Mode in isolation, re-evaluate as a separate SKU; default for now: not a SKU.
"""

CONTEXT_SUMMARY = """
Context: SDP (Software Delivery Productivity) lab at /Users/fall_out_bug/projects/vibe_coding/sdp_lab.
F150 is a release-readiness program turning a broad refactor request into 10 workstreams (taxonomy, release surface, module path migration, experimental isolation, dependency audit, coverage policy, telemetry consent, Homebrew dry run, product docs alignment, debt ledger).

Prior 2026-04-26 council outputs (different document, ChangePassport manifesto):
- Accepted: ChangePassport = merge-readiness system for AI-assisted PRs with stakeholder handoff.
- Paid object = governed readiness decision (merge/hold/rework/escalate/override) + evidence + override trail + reviewer-readable passport.
- v1 = GitHub PR Gate Loop. Required artifacts before implementation: Passport Schema v1, Evidence Provider API v1, Decision Record v1, override protocol, GitHub App v1 flow, pilot measurement plan.

Prior 2026-04-26 enterprise perimeter research:
- "Inside enterprise perimeter" alone is NOT unique. Factory, Tabnine, GitLab Duo Self-Hosted, OpenHands Enterprise already cover this.
- Real wedge = neutral governed delivery protocol across multiple agents/models.
- First wedge for enterprise: on-prem MR readiness + ChangePassport for GitLab Self-Managed with local/sovereign model routing.

Prior 2026-04-26 SDP strategy through AI Fluency 4D:
- SDP should NOT be framed as "AI development team" or "full PDLC platform"; that market is taken.
- SDP's defensible layer = the contract around execution: scope, gates, evidence, trace, findings loop.
- Replacement risk: 70-85% of SDP can be replaced today by GitHub Projects + Spec Kit/Kiro + Codex/Claude + CodeRabbit/Sonar.
- Strongest near-term product: agent-neutral delivery governance + AI PR governance pack.

Repo reality today (docs/reference/product-surface.md):
- Toolkit (Scout/Metrics/Index/Spec/Bootstrap) is GA inside sdp_lab.
- Multi-harness install Beta (Claude Code, OpenCode, Codex, Cursor).
- Operator Mode loop is GA inside sdp_lab (Beads + workstreams + draft PR + findings loop + QA/UAT).
- ChangePassport is direction, not implementation.
- Enterprise Perimeter is hypothesis.

Author asserts the 12 claims above as the revised product layering taxonomy. Council job: challenge them and propose corrections.
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

The author asks: challenge our claims. We may be wrong. We are willing to change our minds.

Your role focus: {focus}

For each claim C1..C12, provide:
- VERDICT: STRONG | OK | WEAK | OPPOSE | VETO
- EVIDENCE: one paragraph (3-6 sentences) explaining your reasoning. Cite specific assumptions or risks.
- PROPOSAL: if WEAK/OPPOSE/VETO, write a one-sentence revised claim.

After all claims:
- MINORITY REPORT: any claim where you predict you will disagree with majority, with rationale.
- DOMAIN VETOES: any claim that is unacceptable from your role perspective, marked [DOMAIN VETO: <reason>].
- THREE BIGGEST RISKS the document fails to address.

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
1. For each claim C1..C12, give your FINAL verdict: ACCEPT | ACCEPT WITH REVISION | REJECT.
2. If "ACCEPT WITH REVISION", write the revised claim in one sentence.
3. Mark which claims you changed your mind on between R1 and R2 and WHY (specifically referencing which other role's argument moved you).
4. Identify the 3 most important RISKS the document still fails to address.
5. Identify the 3 most important CORRECTIONS the author should make before shipping the memo.
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
        "max_tokens": 4000,
    }
    req = urllib.request.Request(
        OPENROUTER_URL,
        data=json.dumps(body).encode("utf-8"),
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
            "HTTP-Referer": "https://github.com/fall-out-bug/sdp_lab",
            "X-Title": "SDP Product Layering Council 2026-04-27",
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
    if primary["ok"] and primary.get("data", {}).get("choices"):
        primary["role"] = role_def["role"]
        primary["used_fallback"] = False
        return primary
    fallback = call_openrouter(api_key, role_def["fallback"], prompt)
    fallback["role"] = role_def["role"]
    fallback["used_fallback"] = True
    fallback["primary_error"] = primary.get("error") or "no choices in response"
    return fallback


def extract_text(resp: dict) -> str:
    if not resp.get("ok"):
        return f"[ERROR: {resp.get('error')}]\n{resp.get('body', '')}"
    choices = resp.get("data", {}).get("choices") or []
    if not choices:
        return "[ERROR: no choices in response]"
    msg = choices[0].get("message", {})
    return msg.get("content") or "[ERROR: empty content]"


def run_round(api_key: str, prompts_by_role: dict[str, str], label: str) -> dict[str, dict]:
    results: dict[str, dict] = {}
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


def write_outputs(results: dict[str, dict], round_label: str) -> None:
    raw_path = ROOT / f"{round_label}-raw.json"
    raw_path.write_text(json.dumps(results, indent=2, ensure_ascii=False), encoding="utf-8")
    for role, resp in results.items():
        md_path = ROOT / f"{round_label}-{role.lower()}.md"
        text = extract_text(resp)
        header = (
            f"# {round_label.upper()} — {role}\n\n"
            f"Model: `{resp.get('model', '?')}`  \n"
            f"Fallback used: `{resp.get('used_fallback', False)}`  \n"
            f"Elapsed: {resp.get('elapsed_s', 0):.1f}s  \n"
            f"OK: `{resp.get('ok')}`\n\n---\n\n"
        )
        md_path.write_text(header + text, encoding="utf-8")


def build_r2_blob(r1_results: dict[str, dict]) -> str:
    parts = []
    for rd in ROLES:
        role = rd["role"]
        text = extract_text(r1_results.get(role, {}))
        parts.append(f"<{role.upper()}>\n{text}\n</{role.upper()}>")
    return "\n\n".join(parts)


def main() -> int:
    api_key = load_api_key()
    print(f"[init] {len(ROLES)} roles, output dir: {ROOT}", flush=True)

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
