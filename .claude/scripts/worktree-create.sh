#!/usr/bin/env bash
# WorktreeCreate hook: worktree 作成後に make build を自動実行する。
# デフォルトの git worktree 作成処理を完全に置き換える。
#
# 入力 JSON（実機確認済みフィールド）:
#   {
#     "hook_event_name": "WorktreeCreate",
#     "name": "<worktree名>",   # ".claude/worktrees/<name>/" のディレクトリ名かつブランチ名
#     ...                        # その他フィールドはスクリプト先頭のデバッグログを参照
#   }
#
# 出力:
#   stdout: 作成した worktree のパス（hook 仕様で必須）
#   stderr: 進捗・エラーメッセージ

set -euo pipefail

input="$(cat)"

# DEBUG=1 時のみ入力 JSON を stderr に出力（フィールド構成の実機確認用）
[ "${DEBUG:-}" = "1" ] && printf '[worktree-create] input JSON: %s\n' "$input" >&2

name="$(jq -r '.name // empty' <<< "$input")"

if [ -z "$name" ]; then
  printf '[worktree-create] Error: "name" field missing or empty in input JSON\n' >&2
  exit 1
fi

repo_root="$(git rev-parse --show-toplevel)"
worktree_path="${repo_root}/.claude/worktrees/${name}"

printf '[worktree-create] Creating worktree: %s (branch: %s)\n' "$worktree_path" "$name" >&2

{
  git worktree add "$worktree_path" -b "$name"
  printf '[worktree-create] Running make build in worktree...\n'
  make -C "$worktree_path" build
} >&2

# stdout には worktree パスのみ出力（hook 仕様）
echo "$worktree_path"
