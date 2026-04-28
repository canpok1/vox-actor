#!/usr/bin/env bash
set -euo pipefail

INPUT=$(cat)

STOP_HOOK_ACTIVE=$(echo "$INPUT" | jq -r '.stop_hook_active // false')

if [ "$STOP_HOOK_ACTIVE" = "true" ]; then
  exit 0
fi

jq -n '{
  decision: "block",
  reason: "区切りがいいので/vox-actor-plugin:talkスキルでキャラクターの短い独り言を生成・再生して。"
}'
