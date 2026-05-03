#!/usr/bin/env bash
# lint-skills.sh — Cross-harness skill validation (F127-08)
#
# Standalone, portable bash alternative to `sdp-protocol-check --lint-skills`.
# No Go toolchain required. Suitable for pre-commit hooks and CI.
#
# Checks performed:
#   ERROR:   missing required frontmatter keys (name, description, version)
#   ERROR:   version is not semver (major.minor.patch)
#   WARNING: missing 'compatibility' frontmatter
#   WARNING: frontmatter name does not match filename (without .md)
#   WARNING: description outside recommended 60-120 character window
#   WARNING: body contains hardcoded harness-specific phrases
#
# Exit codes:
#   0 — all skills pass (warnings are allowed)
#   1 — one or more errors found
#   2 — warnings only (no errors), use --strict to promote warnings to errors
#
# Usage:
#   ./scripts/lint-skills.sh              # check all skills
#   ./scripts/lint-skills.sh --strict     # treat warnings as errors
#   ./scripts/lint-skills.sh --json       # JSON output for CI
#   ./scripts/lint-skills.sh --skills-dir .agents/skills   # custom dir
#
# Pre-commit hook integration:
#   Add to .git/hooks/pre-commit or use scripts/install-hooks.sh:
#     scripts/lint-skills.sh || exit 1
#
# GitHub Actions integration:
#   - name: Lint skills
#     run: ./scripts/lint-skills.sh --strict
#
# The Go-based equivalent (for richer output):
#   go run ./cmd/sdp-protocol-check --lint-skills
#   go run ./cmd/sdp-protocol-check --lint-skills --format json
#
# Note: fm_value extracts only the first line of YAML values. Multi-line YAML values
# (> or | syntax) are not fully supported. For full YAML validation, use the Go-based
# sdp-protocol-check.

set -euo pipefail

# --- Defaults ---
SKILLS_DIR=""
STRICT=0
JSON_OUTPUT=0
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# --- Parse args ---
while [[ $# -gt 0 ]]; do
    case "$1" in
        --strict)
            STRICT=1
            shift
            ;;
        --json)
            JSON_OUTPUT=1
            shift
            ;;
        --skills-dir)
            SKILLS_DIR="$2"
            shift 2
            ;;
        -h|--help)
            sed -n '2,/^$/p' "$0" | sed 's/^# \?//'
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

# Resolve skills directory
if [[ -z "$SKILLS_DIR" ]]; then
    # Try relative to script location (scripts/ -> project root)
    PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
    SKILLS_DIR="$PROJECT_ROOT/.agents/skills"
fi

if [[ ! -d "$SKILLS_DIR" ]]; then
    echo "SKIP: .agents/skills/ directory not found at $SKILLS_DIR"
    exit 0
fi

# --- State ---
ERRORS=0
WARNINGS=0
# JSON accumulators
JSON_ISSUES=""

# --- Helpers ---
add_issue() {
    local severity="$1" file="$2" message="$3"
    if [[ "$severity" == "error" ]]; then
        ((ERRORS++)) || true
    else
        ((WARNINGS++)) || true
    fi

    if [[ "$JSON_OUTPUT" -eq 1 ]]; then
        local escaped_msg
        escaped_msg=$(printf '%s' "$message" | sed 's/\\/\\\\/g; s/"/\\"/g')
        local escaped_file
        escaped_file=$(printf '%s' "$file" | sed 's/\\/\\\\/g; s/"/\\"/g')
        # Append comma-separated JSON objects (strip leading comma later)
        JSON_ISSUES="${JSON_ISSUES},{\"severity\":\"${severity}\",\"file\":\"${escaped_file}\",\"message\":\"${escaped_msg}\"}"
    else
        printf '  [%s] %s: %s\n' "$severity" "$file" "$message" >&2
    fi
}

# Extract frontmatter value for a key.
# Usage: fm_value "$content" "key"
# Uses awk for portability across BSD and GNU sed.
fm_value() {
    local content="$1" key="$2"
    # Match "key:" at start of line (possibly with whitespace), capture value
    # Handles: key: value, key: "value", key: 'value'
    echo "$content" | awk -v k="$key" '
        /^---$/ { in_fm++; next }
        in_fm == 1 && $0 ~ "^[[:space:]]*" k ":" {
            sub(/^[[:space:]]*[^:]+:[[:space:]]*/, "")
            gsub(/^["'"'"']|["'"'"']$/, "")
            print
            exit
        }
    ' | head -1
}

# Check if compatibility key exists in frontmatter
has_compatibility() {
    local content="$1"
    echo "$content" | awk '
        /^---$/ { in_fm++; next }
        in_fm == 1 && /^[[:space:]]*compatibility:/ { found=1; exit }
        END { exit !found }
    '
}

# Extract body (everything after closing ---)
get_body() {
    local content="$1"
    # Use awk to skip the frontmatter block
    echo "$content" | awk '
        /^---$/ { count++; next }
        count >= 2 { print }
    '
}

# --- Harness-specific phrase patterns ---
# These are the same patterns as the Go implementation (skill_lint.go).
HARNESS_PHRASES=(
    '[Cc]laude [Cc]ode [Oo]nly'
    '[Oo]pen[Cc]ode [Oo]nly'
    '[Cc]ursor [Oo]nly'
    '[Cc]odex [Oo]nly'
    '[Ii]n [Cc]laude [Cc]ode[, ]'
    '[Ii]n [Oo]pen[Cc]ode[, ]'
    '[Uu]se the [Tt]ask [Tt]ool'
)

# --- Main ---
SKILL_COUNT=0

for skill_file in "$SKILLS_DIR"/*.md; do
    [[ -f "$skill_file" ]] || continue

    filename="$(basename "$skill_file")"

    # Skip module docs — they are documentation/agent contracts, not skills.
    [[ "$filename" == "README.md" ]] && continue
    [[ "$filename" == "AGENTS.md" ]] && continue

    ((SKILL_COUNT++)) || true
    rel_path=".agents/skills/${filename}"

    content=""
    content=$(<"$skill_file")

    # --- Check: frontmatter exists ---
    if ! echo "$content" | head -1 | grep -q '^---$'; then
        add_issue "error" "$rel_path" "missing YAML frontmatter (must start with ---)"
        continue
    fi

    # Check closing ---
    if ! echo "$content" | awk 'NR>1 && /^---$/ { found=1; exit } END { exit !found }'; then
        add_issue "error" "$rel_path" "frontmatter opening --- found but no closing ---"
        continue
    fi

    # --- Check: required fields ---
    name_val=$(fm_value "$content" "name")
    desc_val=$(fm_value "$content" "description")
    ver_val=$(fm_value "$content" "version")

    if [[ -z "$name_val" ]]; then
        add_issue "error" "$rel_path" "missing required frontmatter key \"name\""
    fi

    if [[ -z "$desc_val" ]]; then
        add_issue "error" "$rel_path" "missing required frontmatter key \"description\""
    fi

    if [[ -z "$ver_val" ]]; then
        add_issue "error" "$rel_path" "missing required frontmatter key \"version\""
    else
        # --- Check: semver format (major.minor.patch) ---
        if ! echo "$ver_val" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+'; then
            add_issue "error" "$rel_path" "version \"${ver_val}\" is not semver (expected major.minor.patch)"
        fi
    fi

    # --- Check: compatibility recommended ---
    if ! has_compatibility "$content"; then
        add_issue "warning" "$rel_path" "missing 'compatibility' frontmatter — skill portability across harnesses is not declared"
    fi

    # --- Check: name matches filename ---
    if [[ -n "$name_val" ]]; then
        base="${filename%.md}"
        if [[ "$name_val" != "$base" ]]; then
            add_issue "warning" "$rel_path" "frontmatter name \"${name_val}\" does not match filename base \"${base}\""
        fi
    fi

    # --- Check: description length window ---
    if [[ -n "$desc_val" ]]; then
        desc_len=${#desc_val}
        if [[ "$desc_len" -lt 60 || "$desc_len" -gt 120 ]]; then
            add_issue "warning" "$rel_path" "description length ${desc_len} outside recommended 60-120 character window"
        fi
    fi

    # --- Check: harness-specific phrases in body ---
    body=""
    body=$(get_body "$content")

    for pattern in "${HARNESS_PHRASES[@]}"; do
        # Find matching lines, report unique matches
        match=$(echo "$body" | grep -oiE "$pattern" | sort -u | head -5 || true)
        if [[ -n "$match" ]]; then
            while IFS= read -r phrase; do
                [[ -z "$phrase" ]] && continue
                add_issue "warning" "$rel_path" "harness-specific phrase \"${phrase}\" — prefer harness-neutral prose"
            done <<< "$match"
        fi
    done
done

# --- Output ---
if [[ "$JSON_OUTPUT" -eq 1 ]]; then
    # Build JSON array
    if [[ -n "$JSON_ISSUES" ]]; then
        # Strip leading comma
        JSON_ISSUES="${JSON_ISSUES#,}"
        printf '{"issues":[%s],"errors":%d,"warnings":%d,"skills_checked":%d}\n' \
            "$JSON_ISSUES" "$ERRORS" "$WARNINGS" "$SKILL_COUNT"
    else
        printf '{"issues":[],"errors":0,"warnings":0,"skills_checked":%d}\n' "$SKILL_COUNT"
    fi
else
    echo ""
    echo "Skill lint: ${SKILL_COUNT} skill(s) checked, ${ERRORS} error(s), ${WARNINGS} warning(s)"
fi

# --- Exit code ---
if [[ "$ERRORS" -gt 0 ]]; then
    exit 1
elif [[ "$STRICT" -eq 1 && "$WARNINGS" -gt 0 ]]; then
    exit 2
fi

exit 0
