#!/bin/bash
# fix-pr.sh のテスト
#
# テストリスト:
# DONE: PR番号が指定されていない場合、エラー終了する（exit 1）
# DONE: ブランチが最新の場合、CI待機とマージに進む
# DONE: コンフリクトが発生した場合、git merge --abortしてexit 1する
# DONE: CIが失敗した場合、exit 3する
# DONE: PRマージが成功した場合、exit 0する
# DONE: 未解決レビューが原因でマージ失敗した場合、exit 2する
# DONE: その他のエラーでマージ失敗した場合、exit 3する（stderrにエラー内容を出力）
# DONE: ブランチが遅れている場合、git merge mainしてpushする

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCRIPT_UNDER_TEST="${SCRIPT_DIR}/fix-pr.sh"

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

# セットアップ
setup() {
    TEST_DIR=$(mktemp -d)
    mkdir -p "$TEST_DIR/bin"
    touch "$TEST_DIR/gh_calls.log"
    touch "$TEST_DIR/git_calls.log"
}

# クリーンアップ
teardown() {
    rm -rf "$TEST_DIR"
}

# ghモックを作成するヘルパー
create_gh_mock() {
    local mock_script="$1"
    cat > "$TEST_DIR/bin/gh" << MOCK_EOF
#!/bin/bash
echo "\$@" >> "$TEST_DIR/gh_calls.log"
$mock_script
MOCK_EOF
    chmod +x "$TEST_DIR/bin/gh"
}

# gitモックを作成するヘルパー
create_git_mock() {
    local mock_script="$1"
    cat > "$TEST_DIR/bin/git" << MOCK_EOF
#!/bin/bash
echo "\$@" >> "$TEST_DIR/git_calls.log"
$mock_script
MOCK_EOF
    chmod +x "$TEST_DIR/bin/git"
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

echo "=== fix-pr.sh テスト ==="

# テスト1: PR番号が指定されていない場合、エラー終了する
test_no_pr_number() {
    local output
    local exit_code=0
    output=$("$SCRIPT_UNDER_TEST" 2>&1) || exit_code=$?
    assert_exit_code 1 "$exit_code" && assert_output_contains "Usage:" "$output"
}
run_test "PR番号が指定されていない場合、エラー終了する" test_no_pr_number

# テスト2: ブランチ最新+CI成功+マージ成功の場合、exit 0する
test_happy_path() {
    # gitモック: ブランチ最新
    create_git_mock '
if [[ "$*" == *"fetch"* ]]; then
    exit 0
elif [[ "$*" == *"rev-parse HEAD"* ]]; then
    echo "abc123"
elif [[ "$*" == *"merge-base"* ]]; then
    echo "abc123"
elif [[ "$*" == *"rev-parse"*"main"* ]]; then
    echo "def456"
fi
'
    # ghモック: CI成功+マージ成功
    create_gh_mock '
if [[ "$*" == *"pr checks"*"--watch"* ]]; then
    exit 0
elif [[ "$*" == *"pr merge"* ]]; then
    echo "merged"
    exit 0
elif [[ "$*" == *"repo view"* ]]; then
    echo "owner/repo"
elif [[ "$*" == *"api repos"* ]]; then
    echo "{\"mergeCommit\":true,\"squash\":true,\"rebase\":true}"
fi
'
    local output
    local exit_code=0
    output=$(PATH="$TEST_DIR/bin:$PATH" "$SCRIPT_UNDER_TEST" 123 2>&1) || exit_code=$?
    assert_exit_code 0 "$exit_code" && \
    assert_file_contains "pr merge" "$TEST_DIR/gh_calls.log"
}
run_test "ブランチ最新+CI成功+マージ成功の場合、exit 0する" test_happy_path

# テスト3: コンフリクトが発生した場合、exit 1する
test_conflict() {
    # gitモック: merge失敗（コンフリクト）
    create_git_mock '
if [[ "$*" == *"fetch"* ]]; then
    exit 0
elif [[ "$*" == *"rev-parse HEAD"* ]]; then
    echo "abc123"
elif [[ "$*" == *"merge-base"* ]]; then
    echo "abc123"
elif [[ "$*" == *"rev-parse"*"main"* ]]; then
    echo "def456"
elif [[ "$*" == *"merge origin/main"* ]]; then
    exit 1
elif [[ "$*" == *"merge --abort"* ]]; then
    exit 0
fi
'
    create_gh_mock '
if [[ "$*" == *"repo view"* ]]; then
    echo "owner/repo"
fi
'
    local output
    local exit_code=0
    output=$(PATH="$TEST_DIR/bin:$PATH" "$SCRIPT_UNDER_TEST" 123 2>&1) || exit_code=$?
    assert_exit_code 1 "$exit_code"
}
run_test "コンフリクトが発生した場合、exit 1する" test_conflict

# テスト4: CIが失敗した場合、exit 3する
test_ci_failure() {
    # gitモック: ブランチ最新
    create_git_mock '
if [[ "$*" == *"fetch"* ]]; then
    exit 0
elif [[ "$*" == *"rev-parse HEAD"* ]]; then
    echo "abc123"
elif [[ "$*" == *"merge-base"* ]]; then
    echo "abc123"
elif [[ "$*" == *"rev-parse"*"main"* ]]; then
    echo "def456"
fi
'
    # ghモック: CI失敗
    create_gh_mock '
if [[ "$*" == *"pr checks"*"--watch"* ]]; then
    exit 1
elif [[ "$*" == *"repo view"* ]]; then
    echo "owner/repo"
fi
'
    local output
    local exit_code=0
    output=$(PATH="$TEST_DIR/bin:$PATH" "$SCRIPT_UNDER_TEST" 123 2>&1) || exit_code=$?
    assert_exit_code 3 "$exit_code"
}
run_test "CIが失敗した場合、exit 3する" test_ci_failure

# テスト5: 未解決レビューが原因でマージ失敗した場合、exit 2する
test_unresolved_reviews() {
    # gitモック: ブランチ最新
    create_git_mock '
if [[ "$*" == *"fetch"* ]]; then
    exit 0
elif [[ "$*" == *"rev-parse HEAD"* ]]; then
    echo "abc123"
elif [[ "$*" == *"merge-base"* ]]; then
    echo "abc123"
elif [[ "$*" == *"rev-parse"*"main"* ]]; then
    echo "def456"
fi
'
    # ghモック: マージ失敗（未解決レビュー）
    create_gh_mock '
if [[ "$*" == *"pr checks"*"--watch"* ]]; then
    exit 0
elif [[ "$*" == *"pr merge"* ]]; then
    echo "unresolved review" >&2
    exit 1
elif [[ "$*" == *"repo view"* ]]; then
    echo "owner/repo"
elif [[ "$*" == *"api repos"* ]]; then
    echo "{\"mergeCommit\":true,\"squash\":true,\"rebase\":true}"
fi
'
    local output
    local exit_code=0
    output=$(PATH="$TEST_DIR/bin:$PATH" "$SCRIPT_UNDER_TEST" 123 2>&1) || exit_code=$?
    assert_exit_code 2 "$exit_code"
}
run_test "未解決レビューが原因でマージ失敗した場合、exit 2する" test_unresolved_reviews

# テスト6: その他のエラーでマージ失敗した場合、exit 3する
test_other_merge_error() {
    # gitモック: ブランチ最新
    create_git_mock '
if [[ "$*" == *"fetch"* ]]; then
    exit 0
elif [[ "$*" == *"rev-parse HEAD"* ]]; then
    echo "abc123"
elif [[ "$*" == *"merge-base"* ]]; then
    echo "abc123"
elif [[ "$*" == *"rev-parse"*"main"* ]]; then
    echo "def456"
fi
'
    # ghモック: マージ失敗（その他エラー）
    create_gh_mock '
if [[ "$*" == *"pr checks"*"--watch"* ]]; then
    exit 0
elif [[ "$*" == *"pr merge"* ]]; then
    echo "some other error" >&2
    exit 1
elif [[ "$*" == *"repo view"* ]]; then
    echo "owner/repo"
elif [[ "$*" == *"api repos"* ]]; then
    echo "{\"mergeCommit\":true,\"squash\":true,\"rebase\":true}"
fi
'
    local output
    local exit_code=0
    output=$(PATH="$TEST_DIR/bin:$PATH" "$SCRIPT_UNDER_TEST" 123 2>&1) || exit_code=$?
    assert_exit_code 3 "$exit_code"
}
run_test "その他のエラーでマージ失敗した場合、exit 3する" test_other_merge_error

# テスト7: ブランチが遅れている場合、mergeしてpushする
test_branch_behind() {
    # gitモック: merge-baseがremote mainと異なる（遅れている）
    create_git_mock '
if [[ "$*" == *"fetch"* ]]; then
    exit 0
elif [[ "$*" == *"rev-parse HEAD"* ]]; then
    echo "abc123"
elif [[ "$*" == *"merge-base"* ]]; then
    echo "old000"
elif [[ "$*" == *"rev-parse"*"main"* ]]; then
    echo "def456"
elif [[ "$*" == *"merge origin/main"* ]]; then
    exit 0
elif [[ "$*" == *"push"* ]]; then
    exit 0
fi
'
    # ghモック: CI成功+マージ成功
    create_gh_mock '
if [[ "$*" == *"pr checks"*"--watch"* ]]; then
    exit 0
elif [[ "$*" == *"pr merge"* ]]; then
    echo "merged"
    exit 0
elif [[ "$*" == *"repo view"* ]]; then
    echo "owner/repo"
elif [[ "$*" == *"api repos"* ]]; then
    echo "{\"mergeCommit\":true,\"squash\":true,\"rebase\":true}"
fi
'
    local output
    local exit_code=0
    output=$(PATH="$TEST_DIR/bin:$PATH" "$SCRIPT_UNDER_TEST" 123 2>&1) || exit_code=$?
    assert_exit_code 0 "$exit_code" && \
    assert_output_contains "マージします" "$output" && \
    assert_file_contains "merge origin/main" "$TEST_DIR/git_calls.log" && \
    assert_file_contains "push" "$TEST_DIR/git_calls.log"
}
run_test "ブランチが遅れている場合、mergeしてpushする" test_branch_behind

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
