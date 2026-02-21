#!/usr/bin/env bash
# Register a project in the registry.
set -e

PROJECT_ID="${1:?usage: $0 <project_id> [repo_url] [branch]}"
REPO_URL="${2:-.}"
BRANCH="${3:-main}"

REGISTRY_URL="${REGISTRY_AGENT_URL:-http://localhost:8080}"

curl -s -X POST "$REGISTRY_URL/api/v1/projects" \
  -H "Content-Type: application/json" \
  -d "{\"id\":\"$PROJECT_ID\",\"repo_url\":\"$REPO_URL\",\"repo_branch\":\"$BRANCH\",\"language\":\"go\",\"workstreams\":[\"workstream:generic\",\"workstream:builder\"]}" \
  | jq .

echo "Registered project $PROJECT_ID"
