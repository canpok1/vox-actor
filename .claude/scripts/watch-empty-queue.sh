#!/usr/bin/env bash
set -euo pipefail

INTERVAL_SECONDS=60
waiting=false

# Ctrl-C（SIGINT）で正常終了するためのトラップ
SCRIPT_DIR=$(dirname "$0")

trap 'if [ "$waiting" = true ]; then echo ""; fi; echo "Stopping watch-empty-queue.sh..."; exit 0' INT

# リモートURLからowner/repoを取得
source "$(dirname "$0")/lib/detect-repo.sh"

while true; do
  # assign-to-claudeラベル付きopen Issueの件数を確認
  queue_count=$(gh issue list \
    --repo "$REPO" \
    --label "assign-to-claude" \
    --state open \
    --json number \
    --jq 'length')

  if [ "$queue_count" -eq 0 ]; then
    # キュー空の場合: 待機状態をリセットして自動選定を実行
    if [ "$waiting" = true ]; then
      echo ""
      waiting=false
    fi

    echo "----------------------------------------"
    echo "Queue is empty. Running assign-issues..."
    echo "----------------------------------------"

    "${SCRIPT_DIR}/assign-issues.sh" -p || true
  else
    # キューにIssueがある場合
    if [ "$waiting" = false ]; then
      printf "Waiting for queue to be empty..."
      waiting=true
    else
      printf "."
    fi
  fi

  # 一定時間待機
  sleep "$INTERVAL_SECONDS"
done
