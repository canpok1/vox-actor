#!/bin/bash
# wait-coderabbit.sh のテスト
#
# テストリスト:
# DONE: PR番号が指定されていない場合、エラー終了する（exit 1）
# DONE: CodeRabbitコメントがrate limitなしで到着した場合、正常終了する（exit 0）
# DONE: CodeRabbitコメントがrate limitありで到着した場合、待機後にfull reviewを投稿して正常終了する（exit 0）
# DONE: CodeRabbitコメントがタイムアウトしても到着しない場合、エラー終了する（exit 1）

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCRIPT_UNDER_TEST="${SCRIPT_DIR}/wait-coderabbit.sh"

# テスト用の一時ディレクトリ
TEST_DIR=""

# テスト結果カウンタ
PASSED=0
FAILED=0

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

# アサーション: ファイルに文字列が含まれるか
assert_file_contains() {
    local expected="$1"
    local file="$2"
    if grep -qF "$expected" "$file"; then
        return 0
    else
        echo "  FAIL: ファイル $file に '$expected' が含まれていません"
        echo "  ファイル内容: $(cat "$file")"
        return 1
    fi
}

# アサーション: ファイルに文字列が含まれないか
assert_file_not_contains() {
    local expected="$1"
    local file="$2"
    if ! grep -qF "$expected" "$file"; then
        return 0
    else
        echo "  FAIL: ファイル $file に '$expected' が含まれています"
        echo "  ファイル内容: $(cat "$file")"
        return 1
    fi
}

# セットアップ: テスト用のモック環境を作成
setup() {
    TEST_DIR=$(mktemp -d)
    # ghコマンドのモックを作成
    mkdir -p "$TEST_DIR/bin"
    # モックのghコマンド呼び出しログ
    touch "$TEST_DIR/gh_calls.log"
}

# クリーンアップ
teardown() {
    rm -rf "$TEST_DIR"
}

# ghモックを作成するヘルパー
# 引数: モックスクリプトの内容
create_gh_mock() {
    local mock_script="$1"
    cat > "$TEST_DIR/bin/gh" << MOCK_EOF
#!/bin/bash
echo "\$@" >> "$TEST_DIR/gh_calls.log"
$mock_script
MOCK_EOF
    chmod +x "$TEST_DIR/bin/gh"
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

echo "=== wait-coderabbit.sh テスト ==="

# テスト1: PR番号が指定されていない場合、エラー終了する
test_no_pr_number() {
    local output
    local exit_code=0
    output=$("$SCRIPT_UNDER_TEST" 2>&1) || exit_code=$?
    assert_exit_code 1 "$exit_code" && assert_output_contains "Usage:" "$output"
}
run_test "PR番号が指定されていない場合、エラー終了する" test_no_pr_number

# テスト2: CodeRabbitコメントがrate limitなしで到着した場合、正常終了する
test_no_rate_limit() {
    # ghモック: コメント取得でCodeRabbitの通常レビューコメントを返す
    create_gh_mock '
if [[ "$*" == *"comments"* ]]; then
    echo "1"
elif [[ "$*" == *"reviews"* ]]; then
    echo "0"
fi
'
    local output
    local exit_code=0
    output=$(PATH="$TEST_DIR/bin:$PATH" POLL_INTERVAL=0 MAX_POLLS=1 "$SCRIPT_UNDER_TEST" 123 2>&1) || exit_code=$?
    assert_exit_code 0 "$exit_code" && \
    # ghコマンドが呼ばれたことを確認（コメントチェックが実行された）
    assert_file_contains "pr view" "$TEST_DIR/gh_calls.log"
}
run_test "CodeRabbitコメントがrate limitなしで到着した場合、正常終了する" test_no_rate_limit

# テスト3: CodeRabbitコメントがrate limitありで到着した場合、待機後にfull reviewを投稿して正常終了する
test_with_rate_limit() {
    # ghモック: コメント取得でrate limit文言を含むレスポンスを返す
    create_gh_mock '
if [[ "$*" == *"comments"*"length"* ]]; then
    echo "1"
elif [[ "$*" == *"reviews"*"length"* ]]; then
    echo "0"
elif [[ "$*" == *"comments"*".body"* ]]; then
    echo "Rate limit exceeded. Please wait 10 minutes."
elif [[ "$*" == *"reviews"*".body"* ]]; then
    echo ""
elif [[ "$*" == *"pr comment"* ]]; then
    echo "commented"
elif [[ "$*" == *"repo view"* ]]; then
    echo "owner/repo"
fi
'
    local output
    local exit_code=0
    output=$(PATH="$TEST_DIR/bin:$PATH" POLL_INTERVAL=0 MAX_POLLS=1 RATE_LIMIT_WAIT=0 "$SCRIPT_UNDER_TEST" 123 2>&1) || exit_code=$?
    assert_exit_code 0 "$exit_code" && \
    assert_output_contains "rate limit" "$output" && \
    assert_file_contains "pr comment" "$TEST_DIR/gh_calls.log" && \
    assert_file_contains "@coderabbitai full review" "$TEST_DIR/gh_calls.log"
}
run_test "rate limitありで待機後にfull reviewを投稿して正常終了する" test_with_rate_limit

# テスト4: CodeRabbitコメントがタイムアウトしても到着しない場合、エラー終了する
test_timeout() {
    # ghモック: コメントもレビューも0件を返す
    create_gh_mock '
if [[ "$*" == *"comments"*"length"* ]]; then
    echo "0"
elif [[ "$*" == *"reviews"*"length"* ]]; then
    echo "0"
elif [[ "$*" == *"repo view"* ]]; then
    echo "owner/repo"
fi
'
    local output
    local exit_code=0
    output=$(PATH="$TEST_DIR/bin:$PATH" POLL_INTERVAL=0 MAX_POLLS=2 "$SCRIPT_UNDER_TEST" 123 2>&1) || exit_code=$?
    assert_exit_code 1 "$exit_code" && \
    assert_output_contains "タイムアウト" "$output"
}
run_test "タイムアウトしても到着しない場合、エラー終了する" test_timeout

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
