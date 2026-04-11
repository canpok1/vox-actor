#!/bin/bash
# 作業メモファイルを ${WORKSPACE_DIR}/.tmp/memo/<dest-subdir>/ へ移動するスクリプト
#
# 同名ファイルが既に存在する場合はタイムスタンプ付き（<basename>.YYYYMMDDHHMMSS.md）で
# リネームして衝突を回避する。
#
# 使用方法:
#   ./.claude/skills/work-memo/scripts/move-memo.sh <src-abs-path> <dest-subdir>
#
# 引数:
#   <src-abs-path>   移動元のメモファイル絶対パス
#   <dest-subdir>    メモディレクトリ配下のサブディレクトリ名（例: done, issued）
#
# 出力:
#   移動先の絶対パスを標準出力に1行出力する
#
# 終了コード:
#   0: 正常完了
#   1: エラー（WORKSPACE_DIR未設定、引数不足、移動元不在など）

set -euo pipefail

if [[ -z "${WORKSPACE_DIR:-}" ]]; then
    echo "Error: environment variable WORKSPACE_DIR is not set" >&2
    exit 1
fi

if [[ $# -ne 2 ]]; then
    echo "Usage: $0 <src-abs-path> <dest-subdir>" >&2
    exit 1
fi

SRC=$1
DEST_SUBDIR=$2

if [[ ! -f "$SRC" ]]; then
    echo "Error: source file does not exist: $SRC" >&2
    exit 1
fi

DEST_DIR="${WORKSPACE_DIR}/.tmp/memo/${DEST_SUBDIR}"
mkdir -p "$DEST_DIR"

BASE=$(basename "$SRC" .md)
TARGET="${DEST_DIR}/${BASE}.md"

if [[ -e "$TARGET" ]]; then
    TS=$(date +%Y%m%d%H%M%S)
    TARGET="${DEST_DIR}/${BASE}.${TS}.md"
fi

mv "$SRC" "$TARGET"
echo "$TARGET"
