#!/bin/bash
# verify_pricing_freshness.sh - CI check for model pricing freshness
# Fails if any model pricing entry is older than 60 days

set -euo pipefail

PRICING_FILE="${PRICING_FILE:-configs/model_pricing.json}"
MAX_AGE_DAYS="${MAX_AGE_DAYS:-60}"

if [ ! -f "$PRICING_FILE" ]; then
  echo "ERROR: Pricing file not found: $PRICING_FILE"
  exit 1
fi

# Get current date in seconds since epoch
current_epoch=$(date +%s)
max_age_seconds=$((MAX_AGE_DAYS * 86400))

# Check each model's effective date
outdated_models=()
while IFS= read -r effective_date; do
  if [ -n "$effective_date" ]; then
    # Parse date and convert to seconds since epoch
    if date_epoch=$(date -j -f "%Y-%m-%d" "$effective_date" +%s 2>/dev/null); then
      age_seconds=$((current_epoch - date_epoch))
      age_days=$((age_seconds / 86400))

      if [ $age_seconds -gt $max_age_seconds ]; then
        outdated_models+=("$effective_date ($age_days days old)")
      fi
    else
      echo "WARNING: Invalid date format: $effective_date"
    fi
  fi
done < <(jq -r '.models[]?.effective // empty' "$PRICING_FILE" 2>/dev/null || echo "")

if [ ${#outdated_models[@]} -gt 0 ]; then
  echo "ERROR: Model pricing is outdated (> $MAX_AGE_DAYS days)"
  echo "Outdated entries:"
  printf '  - %s\n' "${outdated_models[@]}"
  echo ""
  echo "Update by:"
  echo "  1. Check provider pricing pages"
  echo "  2. Update 'effective' date and prices in $PRICING_FILE"
  echo "  3. Run this script to verify"
  exit 1
fi

echo "OK: All model pricing entries are within $MAX_AGE_DAYS days"
exit 0
