#!/usr/bin/env bash
set -euo pipefail

INTERVAL_SECONDS=60
waiting=false

SCRIPT_DIR=$(dirname "$0")
WORKSPACE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# ロックファイル用ディレクトリの準備
LOCK_DIR="$WORKSPACE_DIR/.tmp/locks"
mkdir -p "$LOCK_DIR"

# Ctrl-C（SIGINT）で正常終了するためのトラップ
trap 'if [ "$waiting" = true ]; then echo ""; fi; echo "Stopping watch-queued-issues.sh..."; exit 0' INT

# リモートURLからowner/repoを取得
source "$SCRIPT_DIR/lib/detect-repo.sh"

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
    # Issue番号単位のロックを取得し、保持した状態で処理を実行
    lock_file="$LOCK_DIR/issue-${issue_number}.lock"
    (
      flock -n 9 || { echo "Issue #$issue_number is locked by another process. Skipping."; exit 1; }

      # Issue検出時: 待機中だった場合は改行
      if [ "$waiting" = true ]; then
        echo ""
      fi

      echo "----------------------------------------"
      echo "Processing issue #$issue_number"
      echo "----------------------------------------"

      # in-progress-by-claudeラベルを付与
      gh issue edit --repo "$REPO" "$issue_number" --add-label "in-progress-by-claude"

      # 処理開始メッセージ
      echo "Starting Claude for issue #$issue_number..."

      # solve-issue.shを実行（エラーでも継続）
      "$SCRIPT_DIR/solve-issue.sh" -p "$issue_number" || true

      # 処理完了後、ラベルを外す（成否を問わず）
      gh issue edit --repo "$REPO" "$issue_number" --remove-label "in-progress-by-claude" || true

      # 処理完了メッセージ
      echo "Completed processing issue #$issue_number."
    ) 9>"$lock_file"
    lock_result=$?

    # ロック取得成功時のみ待機状態をリセット
    if [ "$lock_result" -eq 0 ]; then
      waiting=false
    fi
  fi

  # 一定時間待機
  sleep "$INTERVAL_SECONDS"
done
