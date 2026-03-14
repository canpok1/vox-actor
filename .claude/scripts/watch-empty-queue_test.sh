#!/bin/bash
# watch-empty-queue.sh のロック機構テスト
#
# テストリスト:
# DONE: 既にロックが取得されている場合、エラーメッセージを出して終了する
# DONE: ロックファイルが .tmp/locks/ 配下に作成される

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCRIPT_UNDER_TEST="${SCRIPT_DIR}/watch-empty-queue.sh"
WORKSPACE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# テスト結果カウンタ
PASSED=0
FAILED=0

# クリーンアップ対象のプロセスID
CLEANUP_PIDS=()

# アサーション: 出力に文字列が含まれるか
assert_output_contains() {
    local expected="$1"
    local actual="$2"
    if echo "$actual" | grep -qF "$expected"; then
        return 0
    else
        echo "  FAIL: 出力に '$expected' が含まれていません"
        echo "  実際の出力: $actual"
        return 1
    fi
}

# アサーション: 終了コードの確認
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
    # テスト後にバックグラウンドプロセスをクリーンアップ
    for pid in "${CLEANUP_PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
    done
    CLEANUP_PIDS=()
}

# ====================
# テストケース
# ====================

echo "=== watch-empty-queue.sh ロック機構テスト ==="

# テスト1: 既にロックが取得されている場合、エラーメッセージを出して終了する
test_exits_when_lock_held() {
    local lock_dir="$WORKSPACE_DIR/.tmp/locks"
    mkdir -p "$lock_dir"
    local lock_file="$lock_dir/watch-empty-queue.lock"

    # バックグラウンドプロセスでロックを保持（sleepで保持し続ける）
    flock -n "$lock_file" sleep 30 &
    local holder_pid=$!
    CLEANUP_PIDS+=("$holder_pid")
    sleep 0.5

    # ロック保持中にスクリプトを実行
    local exit_code=0
    local output=""
    output=$(timeout 5 "$SCRIPT_UNDER_TEST" 2>&1) || exit_code=$?

    assert_exit_code 1 "$exit_code" &&
    assert_output_contains "既に起動中です" "$output"
}
run_test "既にロックが取得されている場合、エラーメッセージを出して終了する" test_exits_when_lock_held

# テスト2: ロックファイルが .tmp/locks/ 配下に作成される
test_lock_file_in_tmp_locks() {
    local lock_file="$WORKSPACE_DIR/.tmp/locks/watch-empty-queue.lock"

    # 既存のロックファイルを削除
    rm -f "$lock_file"

    # スクリプトをバックグラウンドで短時間実行
    "$SCRIPT_UNDER_TEST" &>/dev/null &
    local pid=$!
    CLEANUP_PIDS+=("$pid")
    sleep 2

    # ロックファイルが存在するか確認
    if [[ -f "$lock_file" ]]; then
        return 0
    else
        echo "  FAIL: ロックファイルが存在しません: $lock_file"
        return 1
    fi
}
run_test "ロックファイルが .tmp/locks/ 配下に作成される" test_lock_file_in_tmp_locks

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
