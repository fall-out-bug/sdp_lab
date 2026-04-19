#!/bin/bash
# PostToolUse hook - runs after any tool use
echo "🔔 PostToolUse hook triggered" >> /tmp/hook_debug.log
date >> /tmp/hook_debug.log

# Sync Beads after git commit
if [[ "$@" == *"git commit"* ]]; then
  echo "📦 Syncing Beads after commit..." >> /tmp/hook_debug.log
  scripts/beads_transport.sh export || true
fi
