#!/bin/bash
# Quality metrics checker for SDP
# Checks: 1) coverage by maturity tier  2) test/code ratio >= 1.5
#
# Coverage tiers (F150-06, sdplab-q2cb):
#   Happy-path (GA + stable surface): >= 80%
#   GA (non happy-path):              >= 60%
#   Beta:                             >= 50% (advisory)
#   Experimental:                     exempt
#
# See: docs/reference/maturity-matrix.md, docs/reference/ci-gates-map.md

set -e

echo "=== Quality Metrics Check (Tiered Coverage) ==="
echo ""

# --- Tier definitions ---
# Happy-path packages: GA packages on the canonical happy-path surface
HAPPY_PATH_PKGS="internal/scout internal/metrics internal/index internal/bootstrap internal/control internal/orchestrate internal/cli internal/manifest internal/evidence internal/guard internal/discovery internal/build"

# Beta packages (advisory only)
BETA_PKGS="internal/tower internal/dispatch internal/a2a internal/monitor internal/profile internal/policy internal/augmentation internal/mcp internal/evals internal/deploy internal/trace internal/sessionaudit"

# Experimental packages (exempt)
EXPERIMENTAL_PKGS="internal/agentloop internal/modelgateway internal/inference internal/llmclient internal/localmodel internal/memory internal/mutation internal/finetune internal/planner internal/authz internal/stream internal/secretscan internal/provenance internal/flaky internal/glob"

# Helper: check if a package is in a space-separated list
in_list() {
    local needle="$1"
    local list="$2"
    echo "$list" | grep -qw "$needle"
}

# --- 1. Coverage Check ---
echo "1. Coverage Check (tiered by maturity)"
echo "---------------------------------------"

FAILED_BLOCKING=""
FAILED_ADVISORY=""

GO_TAGS="-tags sqlite_fts5"

for pkg in $(go list ./internal/... 2>/dev/null | sort); do
    pkg_short=$(echo "$pkg" | sed 's|.*internal/|internal/|')

    # Determine tier
    if in_list "$pkg_short" "$EXPERIMENTAL_PKGS"; then
        tier="experimental"
        target=0
    elif in_list "$pkg_short" "$HAPPY_PATH_PKGS"; then
        tier="happy-path"
        target=80
    elif in_list "$pkg_short" "$BETA_PKGS"; then
        tier="beta"
        target=50
    else
        tier="ga"
        target=60
    fi

    # Skip experimental packages
    if [ "$tier" = "experimental" ]; then
        continue
    fi

    coverage=$(go test $GO_TAGS -cover "$pkg" 2>/dev/null | grep -oP 'coverage:\s*\K[0-9.]+')
    if [ -n "$coverage" ]; then
        if (( $(echo "$coverage < $target" | bc -l) )); then
            status_icon="FAIL"
            if [ "$tier" = "beta" ]; then
                echo "  ADVISORY ($tier, target >= ${target}%) $pkg_short: ${coverage}%"
                FAILED_ADVISORY="$FAILED_ADVISORY $pkg_short"
            else
                echo "  FAIL ($tier, target >= ${target}%) $pkg_short: ${coverage}%"
                FAILED_BLOCKING="$FAILED_BLOCKING $pkg_short"
            fi
        else
            echo "  PASS ($tier, >= ${target}%) $pkg_short: ${coverage}%"
        fi
    fi
done

echo ""
echo "2. Test/Code Ratio Check (target: 1.5 - 2.0)"
echo "---------------------------------------------"

check_ratio() {
    local pkg_path=$1
    local pkg_name=$2

    # Count production code lines (exclude test files)
    prod_lines=$(find "$pkg_path" -name "*.go" ! -name "*_test.go" -exec cat {} \; 2>/dev/null | grep -v "^$" | grep -v "^//" | wc -l | tr -d ' ')

    # Count test code lines
    test_lines=$(find "$pkg_path" -name "*_test.go" -exec cat {} \; 2>/dev/null | grep -v "^$" | grep -v "^//" | wc -l | tr -d ' ')

    if [ "$prod_lines" -gt 0 ]; then
        ratio=$(echo "scale=2; $test_lines / $prod_lines" | bc)
        if (( $(echo "$ratio < 1.5" | bc -l) )); then
            echo "  FAIL $pkg_name: ${ratio} (${test_lines}/${prod_lines} lines) - BELOW MINIMUM"
            return 1
        elif (( $(echo "$ratio > 2.0" | bc -l) )); then
            echo "  WARN $pkg_name: ${ratio} (${test_lines}/${prod_lines} lines) - ABOVE MAXIMUM"
            return 0  # Warning only, not a failure
        else
            echo "  PASS $pkg_name: ${ratio} (${test_lines}/${prod_lines} lines)"
            return 0
        fi
    fi
    return 0
}

FAILED_RATIO=""
for dir in internal/*/; do
    if [ -d "$dir" ]; then
        pkg_name=$(basename "$dir")
        # Skip experimental packages from ratio check too
        if in_list "internal/$pkg_name" "$EXPERIMENTAL_PKGS"; then
            continue
        fi
        if ! check_ratio "$dir" "$pkg_name"; then
            FAILED_RATIO="$FAILED_RATIO $pkg_name"
        fi
    fi
done

echo ""
echo "=== Summary ==="
if [ -z "$FAILED_BLOCKING" ] && [ -z "$FAILED_RATIO" ]; then
    echo "All blocking quality metrics passed."
    [ -n "$FAILED_ADVISORY" ] && echo "Advisory (beta coverage):$FAILED_ADVISORY"
    exit 0
else
    [ -n "$FAILED_BLOCKING" ] && echo "Blocking coverage failures:$FAILED_BLOCKING"
    [ -n "$FAILED_ADVISORY" ] && echo "Advisory (beta coverage):$FAILED_ADVISORY"
    [ -n "$FAILED_RATIO" ] && echo "Test/code ratio failures:$FAILED_RATIO"
    exit 1
fi
