#!/usr/bin/env bash
set -euo pipefail

cat > /dev/null

CONTEXT=$(cat <<'EOF'
以下のタイミングでは /vox-actor-plugin:monologue スキルを必ず呼び出してキャラクターの独り言を通知してください。呼び出しをスキップしてはいけません。

- セッション開始直後の最初の行動として必ず呼び出すこと
- 会話や作業の区切りがいいとき
- 方針を転換したとき
- タスクが完了したとき
- エラーが発生したとき、想定外の結果が返ってきたときなど、想定外のことが起きたとき
EOF
)

jq -n --arg ctx "$CONTEXT" '{
  hookSpecificOutput: {
    hookEventName: "SessionStart",
    additionalContext: $ctx
  }
}'
