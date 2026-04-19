#!/usr/bin/env python3
"""
session_audit.py — Claude Code session analytics for sdp_lab.

Usage:
    python3 scripts/session_audit.py                    # all sessions, summary
    python3 scripts/session_audit.py --top 5            # top 5 by size
    python3 scripts/session_audit.py --session <id>     # single session detail
    python3 scripts/session_audit.py --json             # machine-readable output
    python3 scripts/session_audit.py --since 7d         # sessions from last 7 days
"""

import json
import os
import sys
import re
import argparse
from collections import Counter, defaultdict
from datetime import datetime, timedelta
from pathlib import Path

PROJECT_DIR = Path.home() / ".claude/projects/-Users-fall-out-bug-projects-vibe-coding-sdp-lab"

NUDGE_PATTERNS = [
    r"^(да|ок|окей|yes|ok|continue|продолжай|погнали|давай|go|next|готово|хорошо|верно|гут|кайф|разумно)$",
    r"^continue from where",
    r"^продолжай$",
]
NUDGE_RE = [re.compile(p, re.I) for p in NUDGE_PATTERNS]

CONTEXT_LOSS_RE = re.compile(r"continue from where you left off", re.I)

REVIEW_SKILL_RE = re.compile(r"(review|reality.?check|verify.?workstream)", re.I)


def is_nudge(text: str) -> bool:
    t = text.strip()
    if len(t) > 80:
        return False
    return any(r.search(t) for r in NUDGE_RE)


def parse_session(path: Path) -> dict:
    stats = {
        "session_id": path.stem,
        "size_mb": round(path.stat().st_size / 1024 / 1024, 1),
        "user_msgs": 0,
        "assistant_msgs": 0,
        "nudges": [],
        "context_loss_count": 0,
        "skill_calls": Counter(),
        "tool_calls": Counter(),
        "review_iterations": 0,
        "issues_closed": [],
        "first_ts": None,
        "last_ts": None,
        "errors": 0,
    }

    try:
        with open(path, encoding="utf-8", errors="replace") as f:
            for raw in f:
                raw = raw.strip()
                if not raw:
                    continue
                try:
                    rec = json.loads(raw)
                except json.JSONDecodeError:
                    stats["errors"] += 1
                    continue

                msg = rec.get("message", {})
                role = msg.get("role", "")
                content = msg.get("content", "")
                ts = rec.get("timestamp") or msg.get("timestamp")
                if ts:
                    if stats["first_ts"] is None:
                        stats["first_ts"] = ts
                    stats["last_ts"] = ts

                if role == "user":
                    stats["user_msgs"] += 1
                    text = _extract_text(content)
                    if text:
                        if CONTEXT_LOSS_RE.search(text):
                            stats["context_loss_count"] += 1
                        elif is_nudge(text):
                            stats["nudges"].append(text.strip()[:60])

                elif role == "assistant":
                    stats["assistant_msgs"] += 1
                    if isinstance(content, list):
                        for c in content:
                            if not isinstance(c, dict):
                                continue
                            if c.get("type") == "tool_use":
                                tool = c.get("name", "")
                                stats["tool_calls"][tool] += 1
                                if tool == "Skill":
                                    skill = (c.get("input") or {}).get("skill", "")
                                    if skill:
                                        stats["skill_calls"][skill] += 1
                                        if REVIEW_SKILL_RE.search(skill):
                                            stats["review_iterations"] += 1
                                # Detect bd close calls (beads close = work done)
                                if tool == "Bash":
                                    cmd = (c.get("input") or {}).get("command", "")
                                    if cmd and re.search(r"bd close\s+(\w+)", cmd):
                                        for m in re.finditer(r"bd close\s+([\w-]+)", cmd):
                                            stats["issues_closed"].append(m.group(1))

    except Exception as e:
        stats["errors"] += 1
        stats["parse_error"] = str(e)

    return stats


def _extract_text(content) -> str:
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        parts = []
        for c in content:
            if isinstance(c, dict) and c.get("type") == "text":
                parts.append(c.get("text", ""))
        return " ".join(parts)
    return ""


def productivity_ratio(s: dict) -> float:
    """closed issues per 100 user messages — higher is better."""
    if s["user_msgs"] == 0:
        return 0.0
    return round(len(s["issues_closed"]) / s["user_msgs"] * 100, 2)


def nudge_rate(s: dict) -> float:
    """fraction of user messages that are just nudges."""
    total = s["user_msgs"]
    if total == 0:
        return 0.0
    return round((len(s["nudges"]) + s["context_loss_count"]) / total * 100, 1)


def fmt_session_summary(s: dict, detail: bool = False) -> str:
    pid = s["session_id"][:12]
    lines = [
        f"Session {pid}  ({s['size_mb']} MB)",
        f"  Messages : user={s['user_msgs']}  assistant={s['assistant_msgs']}",
        f"  Nudges   : {len(s['nudges'])} ({nudge_rate(s):.0f}% of user msgs)",
        f"  CtxLoss  : {s['context_loss_count']}x 'Continue from where'",
        f"  Reviews  : {s['review_iterations']} review skill calls",
        f"  Closed   : {len(s['issues_closed'])} issues  (productivity={productivity_ratio(s):.2f} per 100 msgs)",
    ]
    if s["skill_calls"]:
        top = s["skill_calls"].most_common(5)
        lines.append("  Skills   : " + ", ".join(f"{k}×{v}" for k, v in top))
    if s["tool_calls"]:
        top = s["tool_calls"].most_common(5)
        lines.append("  Tools    : " + ", ".join(f"{k}×{v}" for k, v in top))
    if detail and s["nudges"]:
        lines.append("  Nudge msgs:")
        for n in s["nudges"][:10]:
            lines.append(f"    · {n}")
    return "\n".join(lines)


def aggregate(sessions: list) -> dict:
    total_nudges = sum(len(s["nudges"]) for s in sessions)
    total_ctx_loss = sum(s["context_loss_count"] for s in sessions)
    total_reviews = sum(s["review_iterations"] for s in sessions)
    total_closed = sum(len(s["issues_closed"]) for s in sessions)
    total_user = sum(s["user_msgs"] for s in sessions)

    all_skills: Counter = Counter()
    for s in sessions:
        all_skills.update(s["skill_calls"])

    nudge_heavy = sorted(sessions, key=lambda s: nudge_rate(s), reverse=True)[:3]
    ctx_loss_heavy = sorted(sessions, key=lambda s: s["context_loss_count"], reverse=True)[:3]
    productive = sorted(sessions, key=lambda s: productivity_ratio(s), reverse=True)[:3]

    return {
        "sessions_analyzed": len(sessions),
        "total_user_msgs": total_user,
        "total_nudges": total_nudges,
        "total_context_loss": total_ctx_loss,
        "total_review_iterations": total_reviews,
        "total_issues_closed": total_closed,
        "top_skills": all_skills.most_common(10),
        "nudge_heavy_sessions": [s["session_id"][:12] for s in nudge_heavy],
        "ctx_loss_sessions": [s["session_id"][:12] for s in ctx_loss_heavy],
        "most_productive_sessions": [s["session_id"][:12] for s in productive],
    }


def fmt_aggregate(agg: dict) -> str:
    lines = [
        "=" * 60,
        "SESSION AUDIT — AGGREGATE",
        "=" * 60,
        f"Sessions analyzed  : {agg['sessions_analyzed']}",
        f"Total user messages: {agg['total_user_msgs']}",
        "",
        "── Autonomy health ──────────────────────────────────────",
        f"Nudges (короткие 'ок'/'давай'): {agg['total_nudges']}",
        f"Context-loss ('Continue from'): {agg['total_context_loss']}",
        f"Review iterations             : {agg['total_review_iterations']}",
        f"Issues closed                 : {agg['total_issues_closed']}",
        "",
        "── Most-nudged sessions (worst autonomy) ────────────────",
    ]
    for sid in agg["nudge_heavy_sessions"]:
        lines.append(f"  {sid}")
    lines += [
        "",
        "── Context-loss sessions ────────────────────────────────",
    ]
    for sid in agg["ctx_loss_sessions"]:
        lines.append(f"  {sid}")
    lines += [
        "",
        "── Most productive sessions ─────────────────────────────",
    ]
    for sid in agg["most_productive_sessions"]:
        lines.append(f"  {sid}")
    lines += [
        "",
        "── Top skills called ────────────────────────────────────",
    ]
    for skill, count in agg["top_skills"]:
        lines.append(f"  {count:4d}x  {skill}")
    return "\n".join(lines)


def since_delta(spec: str) -> timedelta:
    m = re.match(r"(\d+)([dhw])", spec)
    if not m:
        raise ValueError(f"Invalid --since spec: {spec}. Use e.g. 7d, 24h, 2w")
    n, unit = int(m.group(1)), m.group(2)
    return {"d": timedelta(days=n), "h": timedelta(hours=n), "w": timedelta(weeks=n)}[unit]


def main():
    parser = argparse.ArgumentParser(description="Claude Code session analytics")
    parser.add_argument("--top", type=int, default=0, help="Limit to N largest sessions")
    parser.add_argument("--session", help="Analyze single session by ID prefix")
    parser.add_argument("--json", action="store_true", help="Output JSON")
    parser.add_argument("--since", help="Only sessions newer than e.g. 7d, 24h, 2w")
    parser.add_argument("--detail", action="store_true", help="Show nudge messages per session")
    args = parser.parse_args()

    if not PROJECT_DIR.exists():
        print(f"ERROR: session dir not found: {PROJECT_DIR}", file=sys.stderr)
        sys.exit(1)

    all_files = sorted(PROJECT_DIR.glob("*.jsonl"), key=lambda p: p.stat().st_size, reverse=True)

    if args.session:
        all_files = [f for f in all_files if f.stem.startswith(args.session)]
        if not all_files:
            print(f"No session matching '{args.session}'", file=sys.stderr)
            sys.exit(1)

    if args.since:
        cutoff = datetime.now() - since_delta(args.since)
        all_files = [f for f in all_files if datetime.fromtimestamp(f.stat().st_mtime) >= cutoff]

    if args.top:
        all_files = all_files[: args.top]

    if not all_files:
        print("No sessions found.")
        sys.exit(0)

    sessions = []
    for f in all_files:
        s = parse_session(f)
        # skip empty/tiny sessions
        if s["user_msgs"] < 3:
            continue
        sessions.append(s)

    if args.session and len(sessions) == 1:
        # single session detail mode
        s = sessions[0]
        if args.json:
            s["skill_calls"] = dict(s["skill_calls"])
            s["tool_calls"] = dict(s["tool_calls"])
            print(json.dumps(s, indent=2, default=str))
        else:
            print(fmt_session_summary(s, detail=True))
        return

    agg = aggregate(sessions)

    if args.json:
        for s in sessions:
            s["skill_calls"] = dict(s["skill_calls"])
            s["tool_calls"] = dict(s["tool_calls"])
        print(json.dumps({"aggregate": agg, "sessions": sessions}, indent=2, default=str))
        return

    print(fmt_aggregate(agg))
    print()
    print("=" * 60)
    print("PER-SESSION BREAKDOWN")
    print("=" * 60)
    for s in sessions:
        print()
        print(fmt_session_summary(s, detail=args.detail))


if __name__ == "__main__":
    main()
