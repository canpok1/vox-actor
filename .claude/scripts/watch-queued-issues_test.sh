#!/bin/bash
# watch-queued-issues.sh のロック機構テスト
#
# watch-queued-issues.sh は無限ループかつ gh コマンドに依存するため、
# ロック機構のみを抽出してテストする。
#
# テストリスト:
# DONE: Issue番号のロックが取得されている場合、そのIssueをスキップする
# DONE: 処理完了後にロックが解放される（別プロセスがロック取得可能）

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WORKSPACE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
LOCK_DIR="$WORKSPACE_DIR/.tmp/locks"

# テスト結果カウンタ
PASSED=0
FAILED=0

# クリーンアップ対象
CLEANUP_PIDS=()

# アサーション
assert_exit_code() {
    local expected="$1"
    local actual="$2"
    if [[ "$actual" -eq "$expected" ]]; then
        return 0
    else
        echo "  FAIL: 終了コードが $expected ではなく $actual です"
        return 1
    fi
}

# テスト実行ヘルパー
run_test() {
    local test_name="$1"
    local test_func="$2"

    echo "--- $test_name"
    if $test_func; then
        echo "  PASS"
        PASSED=$((PASSED + 1))
    else
        echo "  FAIL"
        FAILED=$((FAILED + 1))
    fi
    for pid in "${CLEANUP_PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
    done
    CLEANUP_PIDS=()
}

# ====================
# テストケース
# ====================

echo "=== watch-queued-issues.sh ロック機構テスト ==="

# テスト1: Issue番号のロックが取得されている場合、flock -n が失敗する
test_issue_lock_conflict() {
    mkdir -p "$LOCK_DIR"
    local lock_file="$LOCK_DIR/issue-42.lock"

    # バックグラウンドでロックを保持
    bash -c "flock -n '$lock_file' sleep 30" &
    local holder_pid=$!
    CLEANUP_PIDS+=("$holder_pid")
    sleep 1

    # 同じロックを取得しようとすると失敗するはず
    local exit_code=0
    flock -n "$lock_file" echo "acquired" 2>/dev/null || exit_code=$?

    assert_exit_code 1 "$exit_code"
}
run_test "Issue番号のロックが取得されている場合、flock -n が失敗する" test_issue_lock_conflict

# テスト2: 処理完了後にロックが解放される（別プロセスがロック取得可能）
test_lock_released_after_completion() {
    mkdir -p "$LOCK_DIR"
    local lock_file="$LOCK_DIR/issue-99.lock"

    # ロックを取得して即解放（短時間保持）
    bash -c "flock -n '$lock_file' sleep 1" &
    local holder_pid=$!
    sleep 2
    wait "$holder_pid" 2>/dev/null || true

    # 解放後に再取得できるはず
    local exit_code=0
    flock -n "$lock_file" echo "re-acquired" >/dev/null 2>&1 || exit_code=$?

    assert_exit_code 0 "$exit_code"
}
run_test "処理完了後にロックが解放される" test_lock_released_after_completion

# テスト結果のサマリー
print_summary() {
    echo ""
    echo "=== テスト結果 ==="
    echo "PASSED: $PASSED"
    echo "FAILED: $FAILED"
    if [[ "$FAILED" -gt 0 ]]; then
        exit 1
    fi
}

trap print_summary EXIT
