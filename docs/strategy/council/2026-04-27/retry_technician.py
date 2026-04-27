#!/usr/bin/env python3
"""Retry Technician for both rounds with bumped max_tokens / fallback model."""

from __future__ import annotations

import json
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).parent))
import run as base  # type: ignore

ROOT = pathlib.Path(__file__).parent.resolve()


def call(api_key: str, model: str, prompt: str, max_tokens: int) -> dict:
    import urllib.request, time
    body = {
        "model": model,
        "messages": [{"role": "user", "content": prompt}],
        "temperature": 0.3,
        "max_tokens": max_tokens,
    }
    req = urllib.request.Request(
        base.OPENROUTER_URL,
        data=json.dumps(body).encode("utf-8"),
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
            "HTTP-Referer": "https://github.com/fall-out-bug/sdp_lab",
            "X-Title": "SDP Council Technician retry 2026-04-27",
        },
        method="POST",
    )
    t0 = time.time()
    with urllib.request.urlopen(req, timeout=base.TIMEOUT_S) as resp:
        data = json.loads(resp.read().decode("utf-8"))
    return {"ok": True, "model": model, "elapsed_s": time.time() - t0, "data": data}


def get_text(resp: dict) -> str:
    ch = resp["data"]["choices"][0]
    return (ch.get("message") or {}).get("content") or ""


def main() -> int:
    api_key = base.load_api_key()
    role_def = next(r for r in base.ROLES if r["role"] == "Technician")
    role = role_def["role"]
    focus = role_def["focus"]

    candidates = [
        ("moonshotai/kimi-k2.6", 12000),
        ("moonshotai/kimi-k2.5", 12000),
        ("qwen/qwen3.6-coder-plus", 8000),
    ]

    # R1 retry
    p1 = base.round1_prompt(role, focus)
    r1_resp = None
    for model, mt in candidates:
        try:
            print(f"[r1 retry] {model} max_tokens={mt}", flush=True)
            resp = call(api_key, model, p1, mt)
            txt = get_text(resp)
            print(f"  -> finish={resp['data']['choices'][0].get('finish_reason')} content={len(txt)}", flush=True)
            if len(txt) > 500:
                resp["used_fallback"] = (model != role_def["primary"])
                resp["role"] = role
                r1_resp = resp
                break
        except Exception as e:
            print(f"  -> ERROR: {type(e).__name__}: {e}", flush=True)
    if not r1_resp:
        print("R1 retry failed; aborting.")
        return 1

    # save R1
    raw_path = ROOT / "r1-raw.json"
    raw = json.loads(raw_path.read_text(encoding="utf-8"))
    raw[role] = r1_resp
    raw_path.write_text(json.dumps(raw, indent=2, ensure_ascii=False), encoding="utf-8")
    md = (
        f"# R1 — {role}\n\n"
        f"Model: `{r1_resp['model']}`  \nFallback used: `{r1_resp.get('used_fallback')}`  \n"
        f"Elapsed: {r1_resp['elapsed_s']:.1f}s  \nOK: `True`\n\n---\n\n{get_text(r1_resp)}"
    )
    (ROOT / f"r1-{role.lower()}.md").write_text(md, encoding="utf-8")

    # Build R2 blob from updated R1 set
    r1_blob = base.build_r2_blob(raw)
    p2 = base.round2_prompt(role, focus, r1_blob)
    r2_resp = None
    for model, mt in candidates:
        try:
            print(f"[r2 retry] {model} max_tokens={mt}", flush=True)
            resp = call(api_key, model, p2, mt)
            txt = get_text(resp)
            print(f"  -> finish={resp['data']['choices'][0].get('finish_reason')} content={len(txt)}", flush=True)
            if len(txt) > 500:
                resp["used_fallback"] = (model != role_def["primary"])
                resp["role"] = role
                r2_resp = resp
                break
        except Exception as e:
            print(f"  -> ERROR: {type(e).__name__}: {e}", flush=True)
    if not r2_resp:
        print("R2 retry failed; aborting.")
        return 1

    raw_path2 = ROOT / "r2-raw.json"
    raw2 = json.loads(raw_path2.read_text(encoding="utf-8"))
    raw2[role] = r2_resp
    raw_path2.write_text(json.dumps(raw2, indent=2, ensure_ascii=False), encoding="utf-8")
    md2 = (
        f"# R2 — {role}\n\n"
        f"Model: `{r2_resp['model']}`  \nFallback used: `{r2_resp.get('used_fallback')}`  \n"
        f"Elapsed: {r2_resp['elapsed_s']:.1f}s  \nOK: `True`\n\n---\n\n{get_text(r2_resp)}"
    )
    (ROOT / f"r2-{role.lower()}.md").write_text(md2, encoding="utf-8")
    print("done.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
