#!/usr/bin/env bash
# f145-demo.sh — Day-8 acceptance demo for F145 cascade routing
#
# Builds sdp-cascade-replay, runs it on the test corpus, and verifies
# that stayed-cheap >= 40% (acceptance threshold).
#
# Exit codes:
#   0  — success and stayed-cheap threshold met
#   1  — success but threshold NOT met (fail)
#   2  — error (build/runtime failure)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

BIN_PATH="${REPO_ROOT}/bin/sdp-cascade-replay"
CORPUS_PATH="${REPO_ROOT}/testdata/cascade_corpus.json"

# Build sdp-cascade-replay
echo "Building sdp-cascade-replay..."
if ! go build -o "$BIN_PATH" "./cmd/sdp-cascade-replay"; then
	echo "ERROR: build failed" >&2
	exit 2
fi

# Verify corpus exists
if [ ! -f "$CORPUS_PATH" ]; then
	echo "ERROR: corpus not found at $CORPUS_PATH" >&2
	exit 2
fi

echo "Running cascade-replay on $CORPUS_PATH..."
echo ""

# Run replay and capture output
OUTPUT=$("$BIN_PATH" -corpus "$CORPUS_PATH" 2>&1)
EXIT_CODE=$?

if [ $EXIT_CODE -ne 0 ]; then
	echo "ERROR: cascade-replay failed with exit code $EXIT_CODE" >&2
	echo "$OUTPUT" >&2
	exit 2
fi

echo "$OUTPUT"

# Extract stayed-cheap percentage
STAYED_CHEAP=$(echo "$OUTPUT" | grep "stayed-cheap" | awk '{print $2}')

if [ -z "$STAYED_CHEAP" ]; then
	echo "ERROR: could not extract stayed-cheap percentage" >&2
	exit 2
fi

# Remove '%' if present
STAYED_CHEAP="${STAYED_CHEAP%\%}"

echo ""
echo "Acceptance Criteria: stayed-cheap >= 40%"
echo "Result: $STAYED_CHEAP%"

# Parse as float and compare (bash limitation: use bc or awk)
THRESHOLD="40"
if (( $(echo "$STAYED_CHEAP >= $THRESHOLD" | bc -l) )); then
	echo "Status: PASS"
	exit 0
else
	echo "Status: FAIL"
	exit 1
fi
