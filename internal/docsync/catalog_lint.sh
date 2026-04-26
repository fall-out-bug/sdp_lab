#!/bin/bash
# SDP Catalog Lint
# Validates skill catalog parity across harnesses and documentation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
ERRORS=0
WARNINGS=0

echo "🔍 SDP Catalog Lint"
echo "===================="
echo ""

# Load canonical catalog
CATALOG_JSON="$PROJECT_ROOT/.agents/skills/index.json"
if [[ ! -f "$CATALOG_JSON" ]]; then
  echo -e "${RED}ERROR: Catalog file not found: $CATALOG_JSON${NC}"
  exit 1
fi

# Helper functions
error() {
  echo -e "${RED}ERROR:${NC} $1"
  ((ERRORS++))
}

warning() {
  echo -e "${YELLOW}WARNING:${NC} $1"
  ((WARNINGS++))
}

success() {
  echo -e "${GREEN}✓${NC} $1"
}

# Extract active skills from catalog
ACTIVE_SKILLS=$(jq -r '.skills[] | select(.status == "active") | .name' "$CATALOG_JSON")
DEPRECATED_SKILLS=$(jq -r '.deprecated[] | .name' "$CATALOG_JSON")
REMOVED_SKILLS=$(jq -r '.removed[] | .name' "$CATALOG_JSON")

echo "Catalog Stats:"
echo "  Active skills: $(echo "$ACTIVE_SKILLS" | wc -l | tr -d ' ')"
echo "  Deprecated skills: $(echo "$DEPRECATED_SKILLS" | wc -l | tr -d ' ')"
echo "  Removed skills: $(echo "$REMOVED_SKILLS" | wc -l | tr -d ' ')"
echo ""

# Check 1: Active skills exist in .agents/skills/
echo "Check 1: Active skills exist in .agents/skills/"
for skill in $ACTIVE_SKILLS; do
  skill_file="$PROJECT_ROOT/.agents/skills/${skill}.md"
  if [[ ! -f "$skill_file" ]]; then
    error "Active skill '$skill' not found in .agents/skills/"
  else
    success "Found $skill"
  fi
done
echo ""

# Check 2: Deprecated skills have deprecation notices
echo "Check 2: Deprecated skills have deprecation notices"
for skill in $DEPRECATED_SKILLS; do
  skill_file="$PROJECT_ROOT/.agents/skills/${skill}.md"
  if [[ -f "$skill_file" ]]; then
    if grep -q "deprecated: true" "$skill_file"; then
      success "$skill has deprecation notice"
    else
      warning "$skill missing deprecation notice in frontmatter"
    fi
  fi
done
echo ""

# Check 3: Removed skills should not exist
echo "Check 3: Removed skills should not exist in .agents/skills/"
for skill in $REMOVED_SKILLS; do
  skill_file="$PROJECT_ROOT/.agents/skills/${skill}.md"
  if [[ -f "$skill_file" ]]; then
    error "Removed skill '$skill' still exists in .agents/skills/"
  else
    success "$skill correctly removed"
  fi
done
echo ""

# Check 4: No extra skills in .agents/skills/ beyond catalog
echo "Check 4: No unexpected skills in .agents/skills/"
ALL_CATALOG_SKILLS=$(jq -r '.skills[], .deprecated[], .removed[] | .name' "$CATALOG_JSON" | sort -u)
for skill_file in "$PROJECT_ROOT"/.agents/skills/*.md; do
  if [[ -f "$skill_file" ]]; then
    skill_name=$(basename "$skill_file" .md)
    if [[ "$skill_name" != "README" ]] && [[ "$skill_name" != "index" ]]; then
      if ! echo "$ALL_CATALOG_SKILLS" | grep -q "^${skill_name}$"; then
        warning "Skill '$skill_name' not in catalog"
      fi
    fi
  fi
done
echo ""

# Check 5: Documentation references use canonical skills
echo "Check 5: Checking documentation for deprecated skill usage"
DEPRECATED_PATTERN="@scout|@architect|@metrics|@landscape|@feature|@idea|@design|@ux|@vision|@prototype|@oneshot|@hotfix|@bugfix|@issue|@debug|@reality-check|@verify-workstream|@deploy|@ci-triage|@plan"
DOCS_WITH_DEPRECATED=$(grep -r -l -E "$DEPRECATED_PATTERN" "$PROJECT_ROOT/docs/reference" 2>/dev/null | grep -v "migration-guide" | grep -v "skill-catalog-inventory" || true)

if [[ -n "$DOCS_WITH_DEPRECATED" ]]; then
  warning "Found deprecated skill references in reference docs:"
  echo "$DOCS_WITH_DEPRECATED" | while read -r file; do
    echo "  - $file"
  done
else
  success "No deprecated skill references in reference docs"
fi
echo ""

# Check 6: prompts/skills/ directory parity
echo "Check 6: Checking prompts/skills/ directory parity"
if [[ -d "$PROJECT_ROOT/prompts/skills" ]]; then
  for skill_dir in "$PROJECT_ROOT"/prompts/skills/*/; do
    if [[ -d "$skill_dir" ]]; then
      skill_name=$(basename "$skill_dir")
      skill_file="$skill_dir/SKILL.md"
      if [[ -f "$skill_file" ]]; then
        # Check if skill is in catalog
        if ! echo "$ALL_CATALOG_SKILLS" | grep -q "^${skill_name}$"; then
          warning "prompts/skills/$skill_name/SKILL.md not in catalog"
        else
          success "prompts/skills/$skill_name/ in catalog"
        fi
      fi
    fi
  done
else
  warning "prompts/skills/ directory not found"
fi
echo ""

# Check 7: .codex/skills/ directory parity
echo "Check 7: Checking .codex/skills/ directory parity"
if [[ -d "$PROJECT_ROOT/.codex/skills" ]]; then
  for skill_file in "$PROJECT_ROOT"/.codex/skills/*.md; do
    if [[ -f "$skill_file" ]]; then
      skill_name=$(basename "$skill_file" .md)
      if [[ "$skill_name" != "README" ]]; then
        if ! echo "$ALL_CATALOG_SKILLS" | grep -q "^${skill_name}$"; then
          warning ".codex/skills/$skill_name.md not in catalog"
        else
          success ".codex/skills/$skill_name.md in catalog"
        fi
      fi
    fi
  done
else
  warning ".codex/skills/ directory not found"
fi
echo ""

# Check 8: .opencode/skill/ directory parity
echo "Check 8: Checking .opencode/skill/ directory parity"
if [[ -d "$PROJECT_ROOT/.opencode/skill" ]]; then
  for skill_file in "$PROJECT_ROOT"/.opencode/skill/*.md; do
    if [[ -f "$skill_file" ]]; then
      skill_name=$(basename "$skill_file" .md)
      if [[ "$skill_name" != "README" ]]; then
        if ! echo "$ALL_CATALOG_SKILLS" | grep -q "^${skill_name}$"; then
          warning ".opencode/skill/$skill_name.md not in catalog"
        else
          success ".opencode/skill/$skill_name.md in catalog"
        fi
      fi
    fi
  done
else
  warning ".opencode/skill/ directory not found"
fi
echo ""

# Summary
echo "===================="
echo "Summary:"
echo "  Errors: $ERRORS"
echo "  Warnings: $WARNINGS"

if [[ $ERRORS -gt 0 ]]; then
  echo -e "${RED}FAILED${NC}"
  exit 1
elif [[ $WARNINGS -gt 0 ]]; then
  echo -e "${YELLOW}PASSED WITH WARNINGS${NC}"
  exit 0
else
  echo -e "${GREEN}PASSED${NC}"
  exit 0
fi
