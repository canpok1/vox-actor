#!/usr/bin/env bash
set -euo pipefail

cat > /dev/null

jq -n '{
  hookSpecificOutput: {
    hookEventName: "PreToolUse",
    additionalContext: "これから実行しようとしているスキルの状況に合わせて、/vox-actor-plugin:talk スキルを必ず呼び出してキャラクターの短い独り言を生成・再生してください。呼び出しをスキップしてはいけません。"
  }
}'
