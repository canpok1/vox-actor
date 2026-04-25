#!/usr/bin/env bash
set -euo pipefail

cat > /dev/null

jq -n '{
  hookSpecificOutput: {
    hookEventName: "PostToolUse",
    additionalContext: "タスクに進捗があったので /vox-actor-plugin:monologue スキルを必ず呼び出してキャラクターの独り言を通知してください。呼び出しをスキップしてはいけません。"
  }
}'
