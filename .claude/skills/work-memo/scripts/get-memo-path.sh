#!/bin/bash
# 現在のgitブランチに対応する作業メモファイルの絶対パスを出力するスクリプト
#
# 使用方法:
#   ./.claude/skills/work-memo/scripts/get-memo-path.sh
#
# 出力:
#   メモファイルの絶対パスを標準出力に出力する。
#   メモディレクトリは必要に応じて自動作成される。
#   （メモファイル自体は作成しない。存在しない場合もパスだけを返す）
#
# 終了コード:
#   0: 正常完了
#   1: エラー（WORKSPACE_DIR未設定、gitリポジトリ外など）

set -euo pipefail

if [[ -z "${WORKSPACE_DIR:-}" ]]; then
    echo "Error: environment variable WORKSPACE_DIR is not set" >&2
    exit 1
fi

BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null) || {
    echo "Error: failed to get current git branch (not a git repository?)" >&2
    exit 1
}

# ブランチ名の /:*?"<>|\ をハイフン - に変換してファイル名として安全な形に正規化
SANITIZED=$(echo "$BRANCH" | tr '/:*?"<>|\\' '-')

MEMO_DIR="${WORKSPACE_DIR}/.tmp/memo"
MEMO_FILE="${MEMO_DIR}/${SANITIZED}.md"

mkdir -p "$MEMO_DIR"

echo "$MEMO_FILE"
