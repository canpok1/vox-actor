#!/usr/bin/env bash
set -euo pipefail

INTERVAL_SECONDS=60
waiting=false

# デフォルト値
ASSIGN_COUNT=2
MIN_QUEUE=0

# 引数解析
while [[ $# -gt 0 ]]; do
  case "$1" in
    --assign-count)
      if [[ -z "${2:-}" ]]; then
        echo "Error: --assign-count requires a value" >&2
        exit 1
      fi
      ASSIGN_COUNT="$2"
      shift 2
      ;;
    --min-queue)
      if [[ -z "${2:-}" ]]; then
        echo "Error: --min-queue requires a value" >&2
        exit 1
      fi
      MIN_QUEUE="$2"
      shift 2
      ;;
    *)
      echo "Usage: $0 [--assign-count N] [--min-queue N]" >&2
      exit 1
      ;;
  esac
done

if ! [[ "${ASSIGN_COUNT}" =~ ^[0-9]+$ ]]; then
  echo "Error: assign-count must be numeric" >&2
  exit 1
fi

if ! [[ "${MIN_QUEUE}" =~ ^[0-9]+$ ]]; then
  echo "Error: min-queue must be numeric" >&2
  exit 1
fi

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

# 現在のGitHubユーザーを取得
source "$SCRIPT_DIR/lib/current-user.sh"

while true; do
  # assign-to-claudeラベル付きopen Issueの件数を確認
  queue_count=$(gh issue list \
    --repo "$REPO" \
    --label "assign-to-claude" \
    --author "$CURRENT_USER" \
    --state open \
    --json number \
    --jq 'length')

  if [ "$queue_count" -le "$MIN_QUEUE" ]; then
    # キュー空の場合: 待機状態をリセットして自動選定を実行
    if [ "$waiting" = true ]; then
      echo ""
      waiting=false
    fi

    echo "----------------------------------------"
    echo "Queue is empty. Running assign-issues..."
    echo "----------------------------------------"

    "${SCRIPT_DIR}/assign-issues.sh" -p -c "$ASSIGN_COUNT" || true
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
