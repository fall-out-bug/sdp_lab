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

print_assessment() {
    local axis="$1"
    local state="$2"
    local detail="$3"
    printf '  %-28s %-15s %s\n' "$axis" "$state" "$detail"
}

has_linter_token() {
    local token="$1"
    [ -f .golangci.yml ] && grep -Eq "^[[:space:]]*-[[:space:]]*${token}\$|^[[:space:]]*${token}:" .golangci.yml
}

echo "0. Deterministic Quality Matrix"
echo "--------------------------------"
print_assessment "go_build_test_vet_lint" "evidence_only" "covered by run_go_quality_gates.sh/CI build-test, not by this script"
print_assessment "coverage_baseline_delta" "evidence_only" "covered by CI coverage-gate; this script checks package tiers only"
if [ "${SDP_QUALITY_MATRIX_ONLY:-0}" = "1" ]; then
    print_assessment "maturity_tier_coverage" "evidence_only" "available with --full; default report does not run per-package coverage"
    print_assessment "test_code_ratio" "evidence_only" "available with --full; default report does not run ratio checks"
else
    print_assessment "maturity_tier_coverage" "evidence_only" "checked below from go test -cover per package"
    print_assessment "test_code_ratio" "evidence_only" "checked below as local evidence; not wired into CI"
fi

if has_linter_token "gocognit" || has_linter_token "gocyclo"; then
    print_assessment "cognitive_complexity" "evidence_only" "linter token found in root .golangci.yml; verify CI wiring before treating as blocking"
else
    print_assessment "cognitive_complexity" "not_assessed" "root .golangci.yml does not enable gocognit/gocyclo thresholds"
fi

print_assessment "crap_score" "not_assessed" "no selected Go CRAP formula/tool is configured"

if command -v go >/dev/null 2>&1; then
    print_assessment "modern_go" "evidence_only" "go vet/golangci evidence exists; staticcheck/gosimple/ineffassign are disabled in root config"
else
    print_assessment "modern_go" "cannot_verify" "go toolchain not available"
fi

if [ -d docs/workstreams ] && [ -f docs/workstreams/INDEX.md ]; then
    print_assessment "spec_drift" "evidence_only" "protocol/doc consistency tools own this; this script does not run them"
else
    print_assessment "spec_drift" "cannot_verify" "workstream docs are unavailable"
fi

BASE_REF="${SDP_BASE_REF:-origin/main}"
if git rev-parse --verify "$BASE_REF" >/dev/null 2>&1; then
    CHANGED_FILES="$(git diff --name-only "$BASE_REF"...HEAD 2>/dev/null || true)"
    if echo "$CHANGED_FILES" | grep -q '^\.sdp/checkpoints/.*\.json$'; then
        print_assessment "work_without_spec" "evidence_only" "checkpoint files changed; CI scope-gate is the authority"
    else
        print_assessment "work_without_spec" "cannot_verify" "no checkpoint evidence in diff against ${BASE_REF}"
    fi
else
    print_assessment "work_without_spec" "cannot_verify" "base ref ${BASE_REF} is unavailable"
fi

echo ""

if [ "${SDP_QUALITY_MATRIX_ONLY:-0}" = "1" ]; then
    exit 0
fi

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

    coverage_output=$(go test $GO_TAGS -cover "$pkg" 2>/dev/null || true)
    coverage=$(printf '%s\n' "$coverage_output" | awk '
        /coverage:/ {
            for (i = 1; i <= NF; i++) {
                if ($i == "coverage:") {
                    pct = $(i + 1)
                    gsub(/%/, "", pct)
                    if (pct ~ /^[0-9]+(\.[0-9]+)?$/) {
                        print pct
                    }
                }
            }
        }
    ' | tail -n 1)
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
