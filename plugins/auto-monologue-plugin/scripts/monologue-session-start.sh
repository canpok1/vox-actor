#!/usr/bin/env bash
set -euo pipefail

cat > /dev/null

jq -n '{
  hookSpecificOutput: {
    hookEventName: "SessionStart",
    additionalContext: "会話の区切りや作業の節目で /vox-actor-plugin:monologue スキルを LLM 判断で活用してください。長時間沈黙する前、方針転換時、タスク完了時などには積極的に呼び出してください。"
  }
}'
