#!/usr/bin/env bash
set -euo pipefail

# オプション解析
USE_PRINT_MODE=false
while getopts "p" opt; do
  case "$opt" in
    p) USE_PRINT_MODE=true ;;
    *) echo "Usage: $0 [-p]" >&2; exit 1 ;;
  esac
done

echo "Issue自動選定を開始します"

# リモートURLからowner/repoを取得
source "$(dirname "$0")/lib/detect-repo.sh"

# readyラベル付きのIssue数を確認し、0件ならスキップ
READY_COUNT=$(gh issue list --repo "$REPO" --state open --label "ready" --json number --jq 'length')
if [ "$READY_COUNT" -eq 0 ]; then
  echo "readyラベル付きのIssueがないため、スキップします"
  exit 0
fi

# Claudeでissueを選定・ラベル付与（コード変更不要のため--worktreeは不使用）
if "${USE_PRINT_MODE}"; then
  claude --dangerously-skip-permissions -p "/assign-issues"
else
  claude --dangerously-skip-permissions "/assign-issues"
fi
