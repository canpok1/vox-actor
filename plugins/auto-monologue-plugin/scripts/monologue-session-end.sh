#!/usr/bin/env bash
set -euo pipefail

cat > /dev/null

jq -n '{
  hookSpecificOutput: {
    hookEventName: "SessionEnd",
    additionalContext: "セッションを終了する前に /vox-actor-plugin:monologue スキルを必ず呼び出してキャラクターの締めの独り言を通知してください。呼び出しをスキップしてはいけません。"
  }
}'
