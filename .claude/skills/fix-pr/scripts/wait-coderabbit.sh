#!/bin/bash
# CodeRabbitのレビュー到着を待機し、rate limitがあれば解消後にreviewを投稿するスクリプト
#
# 使用方法:
#   ./.claude/skills/fix-pr/scripts/wait-coderabbit.sh <PR番号>
#
# 環境変数（テスト用にオーバーライド可能）:
#   POLL_INTERVAL: ポーリング間隔（秒）デフォルト30
#   MAX_POLLS: 最大ポーリング回数 デフォルト10
#   RATE_LIMIT_WAIT: rate limit時のデフォルト待機時間（秒）デフォルト600
#
# 終了コード:
#   0: 正常完了（rate limitなし、またはrate limit解消後に再レビュー投稿済み）
#   1: エラー

set -euo pipefail

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <PR番号>" >&2
    exit 1
fi

PR_NUMBER="$1"
POLL_INTERVAL="${POLL_INTERVAL:-30}"
MAX_POLLS="${MAX_POLLS:-10}"
RATE_LIMIT_WAIT="${RATE_LIMIT_WAIT:-600}"

# リポジトリ情報を取得
get_repo() {
    gh repo view --json nameWithOwner --jq '.nameWithOwner' 2>/dev/null || echo ""
}

REPO=$(get_repo)

# CodeRabbitのコメント数を取得
get_coderabbit_comment_count() {
    gh pr view --repo "$REPO" "$PR_NUMBER" --json comments --jq '[.comments[] | select(.author.login=="coderabbitai")] | length'
}

# CodeRabbitのレビュー数を取得
get_coderabbit_review_count() {
    gh pr view --repo "$REPO" "$PR_NUMBER" --json reviews --jq '[.reviews[] | select(.author.login=="coderabbitai")] | length'
}

# CodeRabbitのコメント/レビューが存在するかチェック
has_coderabbit_response() {
    local comment_count
    local review_count
    comment_count=$(get_coderabbit_comment_count)
    review_count=$(get_coderabbit_review_count)
    [[ "$comment_count" -gt 0 ]] || [[ "$review_count" -gt 0 ]]
}

# CodeRabbitのレビュー状態がCHANGES_REQUESTEDかチェック
check_changes_requested() {
    local reviews
    reviews=$(gh pr view --repo "$REPO" "$PR_NUMBER" --json reviews --jq '[.reviews[] | select(.author.login=="coderabbitai")] | last | .state')
    if [[ "$reviews" == "CHANGES_REQUESTED" ]]; then
        return 0  # CHANGES_REQUESTEDあり
    fi
    return 1  # CHANGES_REQUESTEDなし
}

# CodeRabbitのコメントからrate limit情報をチェック
check_rate_limit() {
    local comments
    comments=$(gh pr view --repo "$REPO" "$PR_NUMBER" --json comments --jq '.comments[] | select(.author.login=="coderabbitai") | .body')
    if echo "$comments" | grep -qi "rate limit"; then
        return 0  # rate limitあり
    fi
    local reviews
    reviews=$(gh pr view --repo "$REPO" "$PR_NUMBER" --json reviews --jq '.reviews[] | select(.author.login=="coderabbitai") | .body')
    if echo "$reviews" | grep -qi "rate limit"; then
        return 0  # rate limitあり
    fi
    return 1  # rate limitなし
}

# CodeRabbitのレビュー到着をポーリングで待機
echo "CodeRabbitのレビュー到着を待機中..."
for i in $(seq 1 "$MAX_POLLS"); do
    if has_coderabbit_response; then
        echo "CodeRabbitのレビューを検出しました。"

        # CHANGES_REQUESTEDチェック
        if check_changes_requested; then
            echo "CodeRabbitのレビュー状態がCHANGES_REQUESTEDです。resolveを投稿します..."
            gh pr comment --repo "$REPO" "$PR_NUMBER" --body "@coderabbitai resolve"
            echo "CHANGES_REQUESTEDの解消を要求しました。"
        fi

        # rate limitチェック
        if check_rate_limit; then
            echo "rate limitが検出されました。${RATE_LIMIT_WAIT}秒待機します..."
            sleep "$RATE_LIMIT_WAIT"

            # reviewを投稿
            echo "@coderabbitai review を投稿します..."
            gh pr comment --repo "$REPO" "$PR_NUMBER" --body "@coderabbitai review"
            echo "再レビューを要求しました。"
        fi

        exit 0
    fi

    if [[ "$i" -lt "$MAX_POLLS" ]]; then
        echo "ポーリング $i/$MAX_POLLS: レビュー未到着、${POLL_INTERVAL}秒待機..."
        sleep "$POLL_INTERVAL"
    fi
done

echo "エラー: CodeRabbitのレビューがタイムアウトしました（${MAX_POLLS}回ポーリング）" >&2
exit 1
