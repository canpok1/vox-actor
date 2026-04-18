#!/usr/bin/env bash
set -euo pipefail

cat > /dev/null

jq -n '{
  hookSpecificOutput: {
    hookEventName: "PreToolUse",
    additionalContext: "現在の作業が以下のいずれかのタイミングに該当する場合、/vox-actor-plugin:monologue スキルを呼び出してキャラクターの独り言を通知してください。\n- 作業開始時・終了時\n- ステップの開始時・終了時\n- 想定外のことが起こった時\n- ユーザー確認が必要な時\n- 新しい発見があったとき\n\n該当しない場合は無視してください。"
  }
}'
