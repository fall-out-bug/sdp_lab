#!/usr/bin/env bash
# =============================================================================
# sdp-publish.sh — Export SDP protocol artifacts from sdp_lab to the public sdp repo
#
# Usage:
#   ./scripts/sdp-publish.sh [OPTIONS]
#
# Options:
#   --dry-run          Show what would be copied without making changes
#   --check            Compare sdp_lab and sdp repo for drift (exit 1 if different)
#   --pr               Create a pull request after pushing
#   --target-dir PATH  Override default temp directory for sdp repo clone
#   --help             Show this help message
#
# Artifact mapping (sdp_lab source -> sdp repo destination):
#   prompts/           -> prompts/
#   schema/            -> schema/
#   templates/         -> templates/
#   scripts/hooks/     -> hooks/
#   .claude/hooks/     -> .claude/hooks/
#   .claude/patterns/  -> .claude/patterns/
#   .agents/skills/    -> prompts/skills/
# =============================================================================
set -euo pipefail

# ---- Configuration ----------------------------------------------------------
SDP_REPO="https://github.com/fall-out-bug/sdp.git"
SDP_REPO_SSH="git@github.com:fall-out-bug/sdp.git"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMP_DIR=""
DRY_RUN=false
CHECK_ONLY=false
CREATE_PR=false
TODAY="$(date +%Y-%m-%d)"
BRANCH_NAME="publish/${TODAY}"
CLEANUP_NEEDED=false

# ---- Colors -----------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ---- Artifact mapping -------------------------------------------------------
# Format: "source_dir:dest_dir"
# Source dirs are relative to PROJECT_ROOT; dest dirs are relative to sdp repo root.
ARTIFACT_MAP=(
  "prompts:prompts"
  "schema:schema"
  "templates:templates"
  "scripts/hooks:hooks"
  ".claude/hooks:.claude/hooks"
  ".claude/patterns:.claude/patterns"
  ".agents/skills:prompts/skills"
)

# ---- Functions --------------------------------------------------------------

usage() {
  cat <<'HEREDOC'
Usage: sdp-publish.sh [OPTIONS]

Export SDP protocol artifacts from sdp_lab to the public sdp repo.

Options:
  --dry-run          Show what would be copied without making changes
  --check            Compare sdp_lab and sdp repo for drift (exit 1 if different)
  --pr               Create a pull request after pushing
  --target-dir PATH  Override default temp directory for sdp repo clone
  --help             Show this help message

Artifact mapping (sdp_lab -> sdp repo):
  prompts/           -> prompts/
  schema/            -> schema/
  templates/         -> templates/
  scripts/hooks/     -> hooks/
  .claude/hooks/     -> .claude/hooks/
  .claude/patterns/  -> .claude/patterns/
  .agents/skills/    -> prompts/skills/
HEREDOC
}

log_info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
log_success() { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }

cleanup() {
  if [[ "$CLEANUP_NEEDED" == true && -n "${TEMP_DIR:-}" && -d "$TEMP_DIR" ]]; then
    log_info "Cleaning up temp directory: $TEMP_DIR"
    rm -rf "$TEMP_DIR"
  fi
}

# Resolve a source path, following symlinks. Returns empty string if not found.
resolve_source_dir() {
  local rel_path="$1"
  local full_path="${PROJECT_ROOT}/${rel_path}"

  if [[ -d "$full_path" ]]; then
    # Resolve symlinks with realpath
    realpath "$full_path" 2>/dev/null || echo "$full_path"
  elif [[ -L "$full_path" ]]; then
    # Broken symlink
    local target
    target="$(readlink "$full_path")"
    if [[ -d "$target" ]]; then
      echo "$target"
    else
      echo ""
    fi
  else
    echo ""
  fi
}

# Count files in a directory (recursively), excluding .git entries
count_files() {
  local dir="$1"
  if [[ -d "$dir" ]]; then
    find "$dir" -type f ! -path '*/.git/*' 2>/dev/null | wc -l | tr -d ' '
  else
    echo "0"
  fi
}

# List files relative to a base directory
list_files_relative() {
  local dir="$1"
  if [[ -d "$dir" ]]; then
    (cd "$dir" && find . -type f ! -path '*/.git/*' | sort)
  fi
}

do_dry_run() {
  log_info "=== DRY RUN ==="
  log_info "Source: ${PROJECT_ROOT}"
  log_info "Target repo: ${SDP_REPO}"
  log_info "Branch: ${BRANCH_NAME}"
  echo ""

  local total_files=0
  local missing=0

  for mapping in "${ARTIFACT_MAP[@]}"; do
    local src_rel="${mapping%%:*}"
    local dst_rel="${mapping##*:}"
    local src_resolved
    src_resolved="$(resolve_source_dir "$src_rel")"

    echo -e "${BLUE}--- ${src_rel}/ -> ${dst_rel}/ ---${NC}"

    if [[ -z "$src_resolved" ]]; then
      log_warn "Source directory not found: ${src_rel}/"
      missing=$((missing + 1))
    else
      local count
      count="$(count_files "$src_resolved")"
      total_files=$((total_files + count))
      log_info "  ${count} files would be copied"
      list_files_relative "$src_resolved" | while read -r f; do
        echo "    ${dst_rel}/${f#./}"
      done
    fi
    echo ""
  done

  if [[ $missing -gt 0 ]]; then
    log_warn "${missing} source directories missing"
  fi

  log_info "Total files to publish: ${total_files}"
}

do_check() {
  log_info "=== CHECK MODE: Comparing sdp_lab with sdp repo ==="
  log_info "Cloning sdp repo to compare..."

  # Clone the sdp repo to a temp directory
  TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/sdp-publish-check.XXXXXX")"
  CLEANUP_NEEDED=true
  trap cleanup EXIT

  if ! git clone --quiet --depth 1 "$SDP_REPO" "$TEMP_DIR/sdp" 2>/dev/null; then
    log_error "Failed to clone sdp repo. Check network access and repo URL."
    exit 1
  fi

  local sdp_root="${TEMP_DIR}/sdp"
  local drift_found=false
  local missing_sources=0

  for mapping in "${ARTIFACT_MAP[@]}"; do
    local src_rel="${mapping%%:*}"
    local dst_rel="${mapping##*:}"
    local src_resolved
    src_resolved="$(resolve_source_dir "$src_rel")"

    if [[ -z "$src_resolved" ]]; then
      log_warn "Source directory missing in sdp_lab: ${src_rel}/"
      missing_sources=$((missing_sources + 1))
      continue
    fi

    local sdp_dest="${sdp_root}/${dst_rel}"

    if [[ ! -d "$sdp_dest" ]]; then
      log_warn "Destination directory does not exist in sdp repo: ${dst_rel}/"
      log_warn "  This is new content (not drift)."
      continue
    fi

    # Compare file lists
    local src_files dst_files
    src_files="$(list_files_relative "$src_resolved")"
    dst_files="$(cd "$sdp_dest" && find . -type f ! -path '*/.git/*' | sort)"

    if [[ "$src_files" != "$dst_files" ]]; then
      log_warn "File list differs for: ${src_rel}/ -> ${dst_rel}/"
      # Show which files differ
      diff <(echo "$src_files") <(echo "$dst_files") | while IFS= read -r line; do
        echo "    $line"
      done
      drift_found=true
    else
      # Check file contents
      local has_content_diff=false
      while IFS= read -r f; do
        [[ -z "$f" ]] && continue
        local rel="${f#./}"
        if [[ ! -f "${sdp_dest}/${rel}" ]]; then
          continue
        fi
        if ! diff -q "${src_resolved}/${rel}" "${sdp_dest}/${rel}" &>/dev/null; then
          if [[ "$has_content_diff" == false ]]; then
            log_warn "Content differs for: ${src_rel}/ -> ${dst_rel}/"
            has_content_diff=true
          fi
          echo "    CHANGED: ${dst_rel}/${rel}"
          drift_found=true
        fi
      done < <(echo "$src_files")
      if [[ "$has_content_diff" == false ]]; then
        log_success "In sync: ${src_rel}/ -> ${dst_rel}/"
      fi
    fi
  done

  echo ""
  if [[ "$drift_found" == true ]]; then
    log_warn "Drift detected between sdp_lab and sdp repo."
    exit 1
  elif [[ $missing_sources -gt 0 ]]; then
    log_warn "${missing_sources} source directories missing in sdp_lab."
    exit 1
  else
    log_success "All artifacts are in sync."
    exit 0
  fi
}

do_publish() {
  log_info "=== PUBLISH: Exporting sdp_lab artifacts to sdp repo ==="

  # Validate source directories exist
  local missing=0
  for mapping in "${ARTIFACT_MAP[@]}"; do
    local src_rel="${mapping%%:*}"
    local src_resolved
    src_resolved="$(resolve_source_dir "$src_rel")"
    if [[ -z "$src_resolved" ]]; then
      log_error "Source directory missing: ${src_rel}/"
      missing=$((missing + 1))
    fi
  done

  if [[ $missing -gt 0 ]]; then
    log_error "Cannot publish: ${missing} source directories missing."
    exit 1
  fi

  # Clone the sdp repo
  TEMP_DIR="${TEMP_DIR:-$(mktemp -d "${TMPDIR:-/tmp}/sdp-publish.XXXXXX")}"
  CLEANUP_NEEDED=true
  trap cleanup EXIT

  local sdp_root="${TEMP_DIR}/sdp"

  if [[ -d "$sdp_root" ]]; then
    log_info "Using existing directory: $sdp_root"
    cd "$sdp_root"
    git fetch origin
    git checkout main
    git reset --hard origin/main
  else
    log_info "Cloning sdp repo..."
    if ! git clone "$SDP_REPO" "$sdp_root" 2>/dev/null; then
      log_error "Failed to clone sdp repo."
      exit 1
    fi
  fi

  cd "$sdp_root"

  # Create a publish branch
  local branch="${BRANCH_NAME}"
  local branch_suffix=1
  while git show-ref --verify --quiet "refs/heads/${branch}" 2>/dev/null; do
    branch="${BRANCH_NAME}-${branch_suffix}"
    branch_suffix=$((branch_suffix + 1))
  done

  log_info "Creating branch: ${branch}"
  git checkout -b "$branch"

  # Copy artifacts
  local total_copied=0
  for mapping in "${ARTIFACT_MAP[@]}"; do
    local src_rel="${mapping%%:*}"
    local dst_rel="${mapping##*:}"
    local src_resolved
    src_resolved="$(resolve_source_dir "$src_rel")"

    local dst_path="${sdp_root}/${dst_rel}"
    local count

    log_info "Copying ${src_rel}/ -> ${dst_rel}/"

    # Create destination directory
    mkdir -p "$dst_path"

    # Remove old contents (to detect deletions)
    rm -rf "${dst_path:?}"/*

    # Copy new contents, preserving structure
    # Use rsync if available, otherwise cp -r
    if command -v rsync &>/dev/null; then
      rsync -a --exclude='.git' "${src_resolved}/" "$dst_path/"
    else
      # cp approach: copy contents of source dir into dest
      (cd "$src_resolved" && find . -type f ! -path '*/.git/*' | while read -r f; do
        mkdir -p "${dst_path}/$(dirname "$f")"
        cp "${f}" "${dst_path}/${f}"
      done)
    fi

    count="$(count_files "$dst_path")"
    total_copied=$((total_copied + count))
    log_info "  ${count} files copied"
  done

  log_info "Total files copied: ${total_copied}"

  # Check if there are changes to commit
  if git diff --quiet && git diff --cached --quiet; then
    log_success "No changes detected. sdp repo is already up to date."
    exit 0
  fi

  # Stage and commit
  git add -A
  git commit -m "publish: sync protocol artifacts from sdp_lab (${TODAY})"

  # Push
  log_info "Pushing branch ${branch}..."
  local push_url
  # Try SSH first, fall back to HTTPS
  if git remote get-url origin 2>/dev/null | grep -q 'https://'; then
    push_url="$SDP_REPO"
  else
    push_url="$SDP_REPO_SSH"
  fi

  git push -u origin "$branch"

  log_success "Pushed branch: ${branch}"

  # Create PR if requested
  if [[ "$CREATE_PR" == true ]]; then
    if command -v gh &>/dev/null; then
      log_info "Creating pull request..."
      gh pr create \
        --repo fall-out-bug/sdp \
        --title "publish: sync protocol artifacts (${TODAY})" \
        --body "$(cat <<PRBODY
## Summary

Automated sync of protocol artifacts from sdp_lab.

### Artifacts synced
- prompts/
- schema/
- templates/
- hooks/
- .claude/hooks/
- .claude/patterns/
- prompts/skills/

Generated by \`scripts/sdp-publish.sh\`
PRBODY
)" \
        --base main \
        --head "$branch"
      log_success "Pull request created."
    else
      log_error "gh CLI not found. Cannot create PR. Install: https://cli.github.com/"
      log_info "You can create a PR manually at:"
      log_info "  https://github.com/fall-out-bug/sdp/compare/main...${branch}"
      exit 1
    fi
  fi

  log_success "Publish complete."
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --dry-run)
        DRY_RUN=true
        shift
        ;;
      --check)
        CHECK_ONLY=true
        shift
        ;;
      --pr)
        CREATE_PR=true
        shift
        ;;
      --target-dir)
        if [[ -z "${2:-}" ]]; then
          log_error "--target-dir requires a path argument."
          exit 1
        fi
        TEMP_DIR="$2"
        shift 2
        ;;
      --help|-h)
        usage
        exit 0
        ;;
      *)
        log_error "Unknown option: $1"
        usage
        exit 1
        ;;
    esac
  done
}

main() {
  parse_args "$@"

  # Verify we are in a valid project root
  if [[ ! -f "${PROJECT_ROOT}/AGENTS.md" ]]; then
    log_error "AGENTS.md not found at ${PROJECT_ROOT}. Are you in sdp_lab?"
    exit 1
  fi

  if [[ "$DRY_RUN" == true ]]; then
    do_dry_run
  elif [[ "$CHECK_ONLY" == true ]]; then
    do_check
  else
    do_publish
  fi
}

main "$@"
