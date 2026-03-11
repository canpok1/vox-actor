#!/usr/bin/env bash
set -euo pipefail

INTERVAL_SECONDS=60
waiting=false

# Ctrl-C（SIGINT）で正常終了するためのトラップ
trap 'if [ "$waiting" = true ]; then echo ""; fi; echo "Stopping watch-queued-issues.sh..."; exit 0' INT

# リモートURLからowner/repoを取得
source "$(dirname "$0")/lib/detect-repo.sh"

while true; do
  # assign-to-claudeラベル付き、かつin-progress-by-claudeラベルが付いていないissueを1件取得（古い順）
  issue_number=$(gh issue list \
    --repo "$REPO" \
    --label "assign-to-claude" \
    --search "sort:created-asc" \
    --json number,labels \
    --jq '.[] | select(.labels | map(.name) | contains(["in-progress-by-claude"]) | not) | .number' \
    | head -n 1)

  # 対象issueが存在しない場合
  if [ -z "$issue_number" ]; then
    if [ "$waiting" = false ]; then
      printf "Waiting for issues..."
      waiting=true
    else
      printf "."
    fi
  else
    # Issue検出時: 待機中だった場合は改行
    if [ "$waiting" = true ]; then
      echo ""
      waiting=false
    fi

    echo "----------------------------------------"
    echo "Processing issue #$issue_number"
    echo "----------------------------------------"

    # in-progress-by-claudeラベルを付与
    gh issue edit --repo "$REPO" "$issue_number" --add-label "in-progress-by-claude"

    # 処理開始メッセージ
    echo "Starting Claude for issue #$issue_number..."

    # solve-issue.shを実行（エラーでも継続）
    "$(dirname "$0")/solve-issue.sh" -p "$issue_number" || true

    # 処理完了後、ラベルを外す（成否を問わず）
    gh issue edit --repo "$REPO" "$issue_number" --remove-label "in-progress-by-claude" || true

    # 処理完了メッセージ
    echo "Completed processing issue #$issue_number."
  fi

  # 一定時間待機
  sleep "$INTERVAL_SECONDS"
done
