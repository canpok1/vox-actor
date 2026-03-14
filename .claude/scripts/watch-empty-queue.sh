#!/usr/bin/env bash
set -euo pipefail

INTERVAL_SECONDS=60
waiting=false

SCRIPT_DIR=$(dirname "$0")
WORKSPACE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# ロックファイルの準備
LOCK_DIR="$WORKSPACE_DIR/.tmp/locks"
mkdir -p "$LOCK_DIR"
LOCK_FILE="$LOCK_DIR/watch-empty-queue.lock"

# ロック取得（排他: 同時に1プロセスのみ）
exec 9>"$LOCK_FILE"
if ! flock -n 9; then
    echo "watch-empty-queue.sh は既に起動中です。終了します。" >&2
    exit 1
fi

# Ctrl-C（SIGINT）で正常終了するためのトラップ
trap 'if [ "$waiting" = true ]; then echo ""; fi; echo "Stopping watch-empty-queue.sh..."; exit 0' INT

# リモートURLからowner/repoを取得
source "$SCRIPT_DIR/lib/detect-repo.sh"

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
