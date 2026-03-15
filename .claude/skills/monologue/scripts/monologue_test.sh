#!/bin/bash
# monologue.sh のテスト
#
# テストリスト:
# DONE: .tmp/notify/ ディレクトリにJSON通知ファイルが作成される
# DONE: WORKSPACE_DIR未設定時にエラー終了する
# DONE: スピーカーIDが指定された場合JSONに反映される
# DONE: スピーカーID省略時はデフォルト値3が使用される

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCRIPT_UNDER_TEST="${SCRIPT_DIR}/monologue.sh"

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

echo "=== monologue.sh テスト ==="

# テスト1: .tmp/notify/ ディレクトリにJSON通知ファイルが作成される
test_creates_json_file_in_tmp_notify() {
    WORKSPACE_DIR="$TEST_DIR" "$SCRIPT_UNDER_TEST" "テストメッセージ" 3
    local file_count
    file_count=$(find "$TEST_DIR/.tmp/notify" -name "notify_*.json" 2>/dev/null | wc -l)
    if [[ "$file_count" -eq 1 ]]; then
        return 0
    else
        echo "  FAIL: .tmp/notify/ にJSON通知ファイルが見つかりません（件数: $file_count）"
        return 1
    fi
}
run_test ".tmp/notify/ディレクトリにJSON通知ファイルが作成される" test_creates_json_file_in_tmp_notify

# テスト2: WORKSPACE_DIR未設定時にエラー終了する
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

test_error_when_workspace_dir_unset() {
    local exit_code=0
    local output
    output=$(WORKSPACE_DIR="" "$SCRIPT_UNDER_TEST" "テスト" 2>&1) || exit_code=$?
    assert_exit_code 1 "$exit_code" &&
    assert_output_contains "WORKSPACE_DIR" "$output"
}
run_test "WORKSPACE_DIR未設定時にエラー終了する" test_error_when_workspace_dir_unset

# テスト3: スピーカーIDが指定された場合JSONに反映される
test_speaker_id_in_json() {
    WORKSPACE_DIR="$TEST_DIR" "$SCRIPT_UNDER_TEST" "テストメッセージ" 7
    local file
    file=$(find "$TEST_DIR/.tmp/notify" -name "notify_*.json" 2>/dev/null | head -1)
    if [[ -z "$file" || ! -f "$file" ]]; then
        echo "  FAIL: 通知ファイルが見つかりません"
        return 1
    fi
    local content
    content=$(cat "$file")
    if echo "$content" | grep -qF '"speaker":7'; then
        return 0
    else
        echo "  FAIL: JSONにスピーカーID 7 が含まれていません"
        echo "  実際の内容: $content"
        return 1
    fi
}
run_test "スピーカーIDが指定された場合JSONに反映される" test_speaker_id_in_json

# テスト4: スピーカーID省略時はデフォルト値3が使用される
test_default_speaker_id() {
    WORKSPACE_DIR="$TEST_DIR" "$SCRIPT_UNDER_TEST" "テストメッセージ"
    local file
    file=$(find "$TEST_DIR/.tmp/notify" -name "notify_*.json" 2>/dev/null | head -1)
    if [[ -z "$file" || ! -f "$file" ]]; then
        echo "  FAIL: 通知ファイルが見つかりません"
        return 1
    fi
    local content
    content=$(cat "$file")
    if echo "$content" | grep -qF '"speaker":3'; then
        return 0
    else
        echo "  FAIL: デフォルトのスピーカーID 3 が含まれていません"
        echo "  実際の内容: $content"
        return 1
    fi
}
run_test "スピーカーID省略時はデフォルト値3が使用される" test_default_speaker_id

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
