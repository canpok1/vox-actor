#!/bin/bash
# notify-host.sh のテスト
#
# テストリスト:
# DONE: .tmp/notify/ ディレクトリに通知ファイルが作成される
# DONE: WORKSPACE_DIR未設定時にエラー終了する

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCRIPT_UNDER_TEST="${SCRIPT_DIR}/notify-host.sh"

# テスト用の一時ディレクトリ
TEST_DIR=""

# テスト結果カウンタ
PASSED=0
FAILED=0

# セットアップ
setup() {
    TEST_DIR=$(mktemp -d)
}

# クリーンアップ
teardown() {
    rm -rf "$TEST_DIR"
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
    setup
    if $test_func; then
        echo "  PASS"
        PASSED=$((PASSED + 1))
    else
        echo "  FAIL"
        FAILED=$((FAILED + 1))
    fi
    teardown
}

# ====================
# テストケース
# ====================

echo "=== notify-host.sh テスト ==="

# テスト1: .tmp/notify/ ディレクトリに通知ファイルが作成される
test_creates_file_in_tmp_notify() {
    WORKSPACE_DIR="$TEST_DIR" "$SCRIPT_UNDER_TEST" "テストメッセージ"
    local file_count
    file_count=$(find "$TEST_DIR/.tmp/notify" -name "notify_*.txt" 2>/dev/null | wc -l)
    if [[ "$file_count" -eq 1 ]]; then
        return 0
    else
        echo "  FAIL: .tmp/notify/ に通知ファイルが見つかりません（件数: $file_count）"
        return 1
    fi
}
run_test ".tmp/notify/ディレクトリに通知ファイルが作成される" test_creates_file_in_tmp_notify

# テスト2: WORKSPACE_DIR未設定時にエラー終了する
test_error_when_workspace_dir_unset() {
    local exit_code=0
    local output
    output=$(WORKSPACE_DIR="" "$SCRIPT_UNDER_TEST" "テスト" 2>&1) || exit_code=$?
    assert_exit_code 1 "$exit_code"
}
run_test "WORKSPACE_DIR未設定時にエラー終了する" test_error_when_workspace_dir_unset

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
