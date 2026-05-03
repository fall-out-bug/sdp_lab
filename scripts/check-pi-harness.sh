#!/usr/bin/env bash
# check-pi-harness.sh — Validate Pi harness consistency with sdp.manifest.yaml
# Exit 0 if OK, 1 if drift detected.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MANIFEST="$ROOT/sdp.manifest.yaml"
PI_DIR="$ROOT/.pi"
PKG_DIR="$ROOT/pi-sdp-harness"
ERRORS=0

echo "=== Pi Harness Consistency Check ==="

# 1. Check manifest exists
if [[ ! -f "$MANIFEST" ]]; then
    echo "ERROR: $MANIFEST not found"
    exit 1
fi

# Helper: extract names under a top-level key
extract_names() {
    local key="$1"
    # Try yq first, fallback to sed
    if command -v yq &>/dev/null; then
        yq e ".${key}[].name" "$MANIFEST" 2>/dev/null || true
    else
        # Extract block between "key:" and next top-level key
        sed -n "/^${key}:/,/^[^ #]/p" "$MANIFEST" | grep '^  - { name:' | sed 's/.*name: \([^,]*\).*/\1/'
    fi
}

# 2. Check that every command in manifest has a prompt in .pi/prompts/
while IFS= read -r cmd; do
    [[ -z "$cmd" ]] && continue
    prompt_file="$PI_DIR/prompts/${cmd}.md"
    if [[ ! -f "$prompt_file" ]]; then
        echo "MISSING: prompt for command '$cmd' → $prompt_file"
        ERRORS=$((ERRORS + 1))
    fi
done < <(extract_names "commands")

# 3. Check that every skill in manifest has a skill definition
while IFS= read -r skill; do
    [[ -z "$skill" ]] && continue
    # Check in .pi/skills/ (harness-local override)
    pi_skill="$PI_DIR/skills/${skill}/SKILL.md"
    # Check in canonical prompts/skills/
    prompt_skill="$ROOT/prompts/skills/${skill}/SKILL.md"

    if [[ ! -f "$pi_skill" && ! -f "$prompt_skill" ]]; then
        echo "MISSING: skill '$skill' not in .pi/skills/ or prompts/skills/"
        ERRORS=$((ERRORS + 1))
    fi
done < <(extract_names "skills")

# 4. Check that .pi/extensions/*.ts files are valid
for ext in "$PI_DIR"/extensions/*.ts; do
    if [[ -f "$ext" ]]; then
        if ! grep -q 'export default function' "$ext"; then
            echo "INVALID: $ext missing 'export default function'"
            ERRORS=$((ERRORS + 1))
        fi
    fi
done

# 5. Check pi-sdp-harness/ is in sync with .pi/
if [[ -d "$PKG_DIR" ]]; then
    for file in "$PI_DIR"/extensions/*.ts; do
        [[ -f "$file" ]] || continue
        base="$(basename "$file")"
        pkg_file="$PKG_DIR/extensions/$base"
        if [[ ! -f "$pkg_file" ]]; then
            echo "SYNC: $pkg_file missing (run 'make sync-pi-package')"
            ERRORS=$((ERRORS + 1))
        elif ! diff -q "$file" "$pkg_file" >/dev/null 2>&1; then
            echo "SYNC: $pkg_file differs from .pi/ (run 'make sync-pi-package')"
            ERRORS=$((ERRORS + 1))
        fi
    done

    for file in "$PI_DIR"/skills/*/SKILL.md; do
        [[ -f "$file" ]] || continue
        base="$(basename "$(dirname "$file")")"
        pkg_file="$PKG_DIR/skills/$base/SKILL.md"
        if [[ ! -f "$pkg_file" ]]; then
            echo "SYNC: $pkg_file missing (run 'make sync-pi-package')"
            ERRORS=$((ERRORS + 1))
        elif ! diff -q "$file" "$pkg_file" >/dev/null 2>&1; then
            echo "SYNC: $pkg_file differs from .pi/ (run 'make sync-pi-package')"
            ERRORS=$((ERRORS + 1))
        fi
    done
fi

# 6. Check settings.json references all extensions
settings="$PI_DIR/settings.json"
for ext in "$PI_DIR"/extensions/*.ts; do
    [[ -f "$ext" ]] || continue
    base="$(basename "$ext")"
    if ! grep -q "$base" "$settings"; then
        echo "SETTINGS: $base not registered in settings.json"
        ERRORS=$((ERRORS + 1))
    fi
done

# Summary
echo ""
if [[ $ERRORS -eq 0 ]]; then
    echo "✓ Pi harness is consistent"
    exit 0
else
    echo "✗ Found $ERRORS issue(s)"
    exit 1
fi
