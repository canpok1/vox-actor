#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

PRINT_MODE=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    -p) PRINT_MODE=true; shift ;;
    *)
      if [[ -z "${ISSUE_NUMBER:-}" ]] && [[ "$1" =~ ^[0-9]+$ ]]; then
        ISSUE_NUMBER="$1"; shift
      else
        echo "Usage: $0 [-p] <issue_number>" >&2; exit 1
      fi
      ;;
  esac
done

if [[ -z "${ISSUE_NUMBER:-}" ]]; then
  echo "Error: issue number is required" >&2
  echo "Usage: $0 [-p] <issue_number>" >&2
  exit 1
fi

REPO="$(gh repo view --json nameWithOwner -q '.nameWithOwner')"

# ロックディレクトリの作成
LOCK_DIR="${WORKSPACE_DIR}/.tmp/locks"
mkdir -p "$LOCK_DIR"
lock_file="${LOCK_DIR}/${ISSUE_NUMBER}"

# ファイルロックで重複実行を防止
exec 9>"$lock_file"
if ! flock -n 9; then
  echo "Issue #${ISSUE_NUMBER} is already being processed"
  exit 1
fi

# クリーンアップ: in-progress-by-claudeラベルを除去
cleanup() {
  echo "Removing in-progress-by-claude label from #${ISSUE_NUMBER}..."
  gh issue edit "$ISSUE_NUMBER" --repo "$REPO" --remove-label "in-progress-by-claude" 2>/dev/null || true
}
trap cleanup EXIT

# in-progress-by-claudeラベルを付与
echo "Adding in-progress-by-claude label to #${ISSUE_NUMBER}..."
gh issue edit "$ISSUE_NUMBER" --repo "$REPO" --add-label "in-progress-by-claude"

# mainブランチを最新化
cd "$WORKSPACE_DIR"
git checkout main
git pull origin main

# Claude実行（worktreeモード）
if [[ "$PRINT_MODE" == "true" ]]; then
  "${SCRIPT_DIR}/claude-stream.sh" --worktree "issue-${ISSUE_NUMBER}" --dangerously-skip-permissions --model sonnet -p "/base-tools:solve-issue ${ISSUE_NUMBER}"
else
  claude --worktree "issue-${ISSUE_NUMBER}" --dangerously-skip-permissions --model sonnet "/base-tools:solve-issue ${ISSUE_NUMBER}"
fi
