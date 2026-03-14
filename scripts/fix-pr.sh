#!/bin/bash
# ブランチ最新化・CI待機・PRマージを行う汎用スクリプト
#
# 使用方法:
#   ./scripts/fix-pr.sh <PR番号>
#
# 終了コード:
#   0: マージ完了
#   1: コンフリクト要解消
#   2: 未解決レビューが原因でマージ不可
#   3: その他エラー（stderrにエラー内容を出力）

set -euo pipefail

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <PR番号>" >&2
    exit 1
fi

PR_NUMBER="$1"

# リポジトリ情報を取得
get_repo() {
    gh repo view --json nameWithOwner --jq '.nameWithOwner' 2>/dev/null || echo ""
}

REPO=$(get_repo)

# ========================================
# Step 1: ブランチ最新化チェック
# ========================================
echo "ブランチ最新化チェック中..."

git fetch origin main 2>/dev/null || true

# mainとの差分を確認
MERGE_BASE=$(git merge-base HEAD origin/main 2>/dev/null || echo "")
REMOTE_MAIN=$(git rev-parse origin/main 2>/dev/null || echo "")

if [[ "$MERGE_BASE" != "$REMOTE_MAIN" ]]; then
    echo "ブランチがmainより遅れています。マージします..."
    if ! git merge origin/main 2>/dev/null; then
        echo "コンフリクトが発生しました。マージを中断します。" >&2
        git merge --abort 2>/dev/null || true
        exit 1
    fi
    echo "mainをマージしました。プッシュします..."
    git push 2>/dev/null || true
else
    echo "ブランチは最新です。"
fi

# ========================================
# Step 2: CI終了待機
# ========================================
echo "CI終了を待機中..."

if ! gh pr checks --repo "$REPO" "$PR_NUMBER" --watch; then
    echo "CIが失敗しました。" >&2
    exit 3
fi

echo "CIが成功しました。"

# ========================================
# Step 3: PRマージ試行
# ========================================
echo "PRマージを試行中..."

# マージ方式を確認
MERGE_METHODS=$(gh api "repos/$REPO" --jq '{mergeCommit: .allow_merge_commit, squash: .allow_squash_merge, rebase: .allow_rebase_merge}')

MERGE_OPTION=""
if echo "$MERGE_METHODS" | grep -q '"squash":true'; then
    MERGE_OPTION="--squash"
elif echo "$MERGE_METHODS" | grep -q '"mergeCommit":true'; then
    MERGE_OPTION="--merge"
elif echo "$MERGE_METHODS" | grep -q '"rebase":true'; then
    MERGE_OPTION="--rebase"
else
    echo "許可されたマージ方式がありません。リポジトリ設定を確認してください。" >&2
    exit 3
fi

echo "マージ方式: $MERGE_OPTION"

# マージ実行
MERGE_OUTPUT=""
MERGE_EXIT=0
MERGE_OUTPUT=$(gh pr merge --repo "$REPO" "$PR_NUMBER" "$MERGE_OPTION" 2>&1) || MERGE_EXIT=$?

if [[ "$MERGE_EXIT" -eq 0 ]]; then
    echo "PRをマージしました。"
    exit 0
fi

# マージ失敗の原因を判定
if echo "$MERGE_OUTPUT" | grep -qi "unresolved review\|review.*required\|review.*not.*resolved\|CHANGES_REQUESTED"; then
    echo "未解決レビューが原因でマージできません。" >&2
    echo "$MERGE_OUTPUT" >&2
    exit 2
fi

# その他のエラー
echo "マージに失敗しました。" >&2
echo "$MERGE_OUTPUT" >&2
exit 3
