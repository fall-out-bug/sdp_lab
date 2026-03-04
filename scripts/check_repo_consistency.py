#!/usr/bin/env python3

from __future__ import annotations

import argparse
import datetime as dt
import json
import re
from dataclasses import dataclass, asdict
from pathlib import Path
from typing import Any, Dict, List, Optional


WS_RE = re.compile(r"00-\d{3}-\d{2}")


@dataclass
class Issue:
    severity: str
    code: str
    file: str
    message: str


def ws_feature_number(ws_id: str) -> Optional[int]:
    m = re.match(r"^00-(\d{3})-\d{2}$", ws_id)
    if not m:
        return None
    return int(m.group(1))


def is_legacy_ws(ws_id: str) -> bool:
    feature = ws_feature_number(ws_id)
    return feature is not None and feature < 59


def parse_date_from_header(text: str, field: str) -> Optional[dt.date]:
    m = re.search(
        rf"^>\s*\*\*{re.escape(field)}:\*\*\s*(\d{{4}}-\d{{2}}-\d{{2}})",
        text,
        re.MULTILINE,
    )
    if not m:
        return None
    return dt.date.fromisoformat(m.group(1))


def parse_index_status(index_text: str) -> Dict[str, str]:
    status = {}
    for line in index_text.splitlines():
        m = re.match(
            r"\s*\|\s*(00-\d{3}-\d{2})\s*\|\s*F\d{3}\s*\|[^|]*\|\s*([A-Za-z ]+)\s*\|",
            line,
        )
        if not m:
            continue
        ws_id = m.group(1)
        ws_status = m.group(2).strip().lower().replace(" ", "_")
        status[ws_id] = ws_status
    return status


def parse_frontmatter_status(content: str) -> Optional[str]:
    if not content.startswith("---\n"):
        return None
    end = content.find("\n---", 4)
    if end == -1:
        return None
    frontmatter = content[4:end]
    for line in frontmatter.splitlines():
        if line.strip().startswith("status:"):
            return line.split(":", 1)[1].strip().strip('"').lower()
    return None


def has_unchecked_ac(content: str) -> bool:
    if "## Acceptance Criteria" not in content:
        return False
    return "- [ ]" in content


def run_checks(root: Path, strict_ac: bool) -> Dict[str, Any]:
    docs = root / "docs"
    roadmap_path = docs / "roadmap" / "ROADMAP.md"
    index_path = docs / "workstreams" / "INDEX.md"
    backlog_dir = docs / "workstreams" / "backlog"
    mapping_path = root / ".beads-sdp-mapping.jsonl"

    issues: List[Issue] = []

    roadmap_text = roadmap_path.read_text(encoding="utf-8")
    index_text = index_path.read_text(encoding="utf-8")

    roadmap_updated = parse_date_from_header(roadmap_text, "Updated")
    index_updated = parse_date_from_header(index_text, "Updated")
    if roadmap_updated and index_updated and index_updated < roadmap_updated:
        issues.append(
            Issue(
                severity="warning",
                code="INDEX_STALE_DATE",
                file=str(index_path.relative_to(root)),
                message=(
                    f"INDEX updated date ({index_updated}) is older than ROADMAP ({roadmap_updated})"
                ),
            )
        )

    index_status = parse_index_status(index_text)

    backlog_files = sorted(backlog_dir.glob("00-*.md"))
    backlog_ws_ids = {p.stem for p in backlog_files}

    for path in backlog_files:
        ws_id = path.stem
        legacy_ws = is_legacy_ws(ws_id)
        content = path.read_text(encoding="utf-8")
        fm_status = parse_frontmatter_status(content)
        idx_status = index_status.get(ws_id)

        rel = str(path.relative_to(root))

        if idx_status is None:
            if legacy_ws:
                continue
            issues.append(
                Issue(
                    severity="error",
                    code="BACKLOG_WS_MISSING_IN_INDEX",
                    file=rel,
                    message=f"{ws_id} exists in backlog but not in INDEX status table",
                )
            )
            continue

        if not fm_status:
            if legacy_ws:
                continue
            issues.append(
                Issue(
                    severity="error",
                    code="BACKLOG_STATUS_MISSING",
                    file=rel,
                    message=f"{ws_id} has no frontmatter status",
                )
            )
            continue

        if fm_status != idx_status:
            if legacy_ws:
                continue
            issues.append(
                Issue(
                    severity="error",
                    code="STATUS_MISMATCH",
                    file=rel,
                    message=f"{ws_id}: backlog status={fm_status}, index status={idx_status}",
                )
            )

        if fm_status == "done" and has_unchecked_ac(content):
            sev = "error" if strict_ac else "warning"
            issues.append(
                Issue(
                    severity=sev,
                    code="DONE_WITH_UNCHECKED_AC",
                    file=rel,
                    message=f"{ws_id} is done but has unchecked acceptance criteria",
                )
            )

    for ws_id in sorted(index_status):
        if ws_id not in backlog_ws_ids:
            issues.append(
                Issue(
                    severity="error",
                    code="INDEX_WS_MISSING_IN_BACKLOG",
                    file=str(index_path.relative_to(root)),
                    message=f"{ws_id} appears in INDEX status table but backlog file is missing",
                )
            )

    roadmap_ws_ids = sorted(set(WS_RE.findall(roadmap_text)))
    for ws_id in roadmap_ws_ids:
        if ws_id not in backlog_ws_ids:
            issues.append(
                Issue(
                    severity="error",
                    code="ROADMAP_PHANTOM_WS",
                    file=str(roadmap_path.relative_to(root)),
                    message=f"ROADMAP references {ws_id} but backlog file does not exist",
                )
            )

    mapping_count = 0
    mapping_ws_ids: set[str] = set()
    if mapping_path.exists():
        for line in mapping_path.read_text(encoding="utf-8").splitlines():
            if not line.strip():
                continue
            mapping_count += 1
            try:
                payload = json.loads(line)
            except json.JSONDecodeError:
                continue
            ws_id = payload.get("sdp_id")
            if isinstance(ws_id, str):
                mapping_ws_ids.add(ws_id)
    else:
        issues.append(
            Issue(
                severity="error",
                code="MAPPING_FILE_MISSING",
                file=str(mapping_path.relative_to(root)),
                message=".beads-sdp-mapping.jsonl is missing",
            )
        )

    backlog_count = len(backlog_files)
    active_backlog_ws_ids = {ws for ws in backlog_ws_ids if not is_legacy_ws(ws)}
    missing_mapping_ws = sorted(active_backlog_ws_ids - mapping_ws_ids)
    if missing_mapping_ws:
        issues.append(
            Issue(
                severity="error",
                code="MAPPING_ACTIVE_WS_MISSING",
                file=str(mapping_path.relative_to(root)),
                message=f"missing mapping entries for active workstreams: {', '.join(missing_mapping_ws)}",
            )
        )

    errors = [i for i in issues if i.severity == "error"]
    warnings = [i for i in issues if i.severity == "warning"]

    return {
        "ok": len(errors) == 0,
        "summary": {
            "errors": len(errors),
            "warnings": len(warnings),
            "roadmap_ws_refs": len(roadmap_ws_ids),
            "index_ws_status_rows": len(index_status),
            "backlog_files": backlog_count,
            "beads_mapping_count": mapping_count,
        },
        "issues": [asdict(i) for i in issues],
    }


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Check roadmap/index/backlog/beads consistency"
    )
    parser.add_argument(
        "--root", default=".", help="Project root (default: current directory)"
    )
    parser.add_argument("--json", action="store_true", help="Print JSON output")
    parser.add_argument(
        "--strict-ac",
        action="store_true",
        help="Treat done-with-unchecked-AC as error",
    )
    args = parser.parse_args()

    root = Path(args.root).resolve()
    report = run_checks(root, strict_ac=args.strict_ac)

    if args.json:
        print(json.dumps(report, indent=2))
    else:
        summary = report["summary"]
        print("Repo consistency check")
        print(f"- errors: {summary['errors']}")
        print(f"- warnings: {summary['warnings']}")
        print(f"- index rows: {summary['index_ws_status_rows']}")
        print(f"- backlog files: {summary['backlog_files']}")
        print(f"- beads mapping count: {summary['beads_mapping_count']}")
        for issue in report["issues"]:
            sev = issue["severity"].upper()
            print(f"[{sev}] {issue['code']} {issue['file']}: {issue['message']}")

    return 0 if report["ok"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
