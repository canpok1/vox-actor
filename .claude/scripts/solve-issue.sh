#!/usr/bin/env bash
set -euo pipefail

# オプション解析
USE_PRINT_MODE=false
while getopts "p" opt; do
  case "$opt" in
    p) USE_PRINT_MODE=true ;;
    *) echo "Usage: $0 [-p] <issue_number>" >&2; exit 1 ;;
  esac
done
shift $((OPTIND - 1))

# 引数チェック
if [ $# -ne 1 ]; then
  echo "Usage: $0 [-p] <issue_number>" >&2
  exit 1
fi

ISSUE_NUMBER="$1"
if ! [[ "${ISSUE_NUMBER}" =~ ^[0-9]+$ ]]; then
  echo "Error: issue_number must be numeric" >&2
  exit 1
fi

echo "Issue #${ISSUE_NUMBER} の処理を開始します"

# mainブランチに切り替えて最新化
git checkout main
git pull origin main

# Claudeでissueを解決（--worktreeで自動的にブランチとワークツリーを作成）
if "${USE_PRINT_MODE}"; then
  claude --worktree "issue-${ISSUE_NUMBER}" --dangerously-skip-permissions -p "/solve-issue ${ISSUE_NUMBER}"
else
  claude --worktree "issue-${ISSUE_NUMBER}" --dangerously-skip-permissions "/solve-issue ${ISSUE_NUMBER}"
fi
