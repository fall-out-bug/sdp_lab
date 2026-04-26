#!/usr/bin/env bash
# Validate skill marker syntax and idempotency
# Part of F131-01 marker convention implementation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SKILLS_DIR="$PROJECT_ROOT/.agents/skills"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Counters
TOTAL_SKILLS=0
VALID_SKILLS=0
INVALID_SKILLS=0
TOTAL_MARKERS=0

echo "🔍 Validating skill markers in $SKILLS_DIR"
echo ""

validate_markers() {
    local file="$1"
    local filename=$(basename "$file")

    # Count BEGIN/END markers (must match)
    local begin_count=$(grep -c '<!-- STACK_SPECIFIC:BEGIN' "$file" || true)
    local end_count=$(grep -c '<!-- STACK_SPECIFIC:END -->' "$file" || true)

    TOTAL_MARKERS=$((TOTAL_MARKERS + begin_count))

    local errors=0

    if [ "$begin_count" -ne "$end_count" ]; then
        echo -e "${RED}❌ $filename${NC}: Mismatched markers (BEGIN: $begin_count, END: $end_count)"
        errors=1
    fi

    # Validate each BEGIN marker has required parameters
    while IFS= read -r line; do
        if ! echo "$line" | grep -qE 'section="[^"]+"'; then
            echo -e "${RED}❌ $filename${NC}: Missing section parameter in: $line"
            errors=1
        fi
        if ! echo "$line" | grep -qE 'stack="[^"]+"'; then
            echo -e "${RED}❌ $filename${NC}: Missing stack parameter in: $line"
            errors=1
        fi

        # Validate section value
        local section=$(echo "$line" | grep -oE 'section="[^"]+"' | cut -d'"' -f2)
        if [[ ! "$section" =~ ^(test|build|lint|debug|quality-gate|coverage)$ ]]; then
            echo -e "${YELLOW}⚠️  $filename${NC}: Non-standard section '$section' in: $line"
        fi

        # Validate stack value
        local stack=$(echo "$line" | grep -oE 'stack="[^"]+"' | cut -d'"' -f2)
        if [[ ! "$stack" =~ ^(go|python|typescript|javascript|rust|java|csharp|ruby|php|swift|kotlin|dart|other)$ ]]; then
            echo -e "${YELLOW}⚠️  $filename${NC}: Non-standard stack '$stack' in: $line"
        fi
    done < <(grep '<!-- STACK_SPECIFIC:BEGIN' "$file" || true)

    if [ $errors -eq 0 ] && [ $begin_count -gt 0 ]; then
        echo -e "${GREEN}✅ $filename${NC}: $begin_count valid marker pair(s)"
        VALID_SKILLS=$((VALID_SKILLS + 1))
    elif [ $errors -eq 0 ] && [ $begin_count -eq 0 ]; then
        # No markers - this is OK for stack-agnostic skills
        echo -e "${GREEN}✅ $filename${NC}: No markers (stack-agnostic)"
        VALID_SKILLS=$((VALID_SKILLS + 1))
    else
        INVALID_SKILLS=$((INVALID_SKILLS + 1))
    fi

    TOTAL_SKILLS=$((TOTAL_SKILLS + 1))
}

# Find all markdown skill files
for skill_file in "$SKILLS_DIR"/*.md; do
    if [ -f "$skill_file" ]; then
        validate_markers "$skill_file"
    fi
done

echo ""
echo "📊 Summary:"
echo "   Total skills: $TOTAL_SKILLS"
echo "   Valid: $VALID_SKILLS"
if [ $INVALID_SKILLS -gt 0 ]; then
    echo -e "   ${RED}Invalid: $INVALID_SKILLS${NC}"
else
    echo "   Invalid: $INVALID_SKILLS"
fi
echo "   Total markers: $TOTAL_MARKERS"
echo ""

# Exit with error if any invalid skills found
if [ $INVALID_SKILLS -gt 0 ]; then
    echo -e "${RED}❌ Validation failed!${NC}"
    exit 1
else
    echo -e "${GREEN}✅ All skills validated successfully!${NC}"
    exit 0
fi
