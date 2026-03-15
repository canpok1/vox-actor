#!/bin/bash
# create-version-tag.sh のテスト
# --dry-run モードを使ってテストする
#
# テストリスト:
# DONE: タグが存在しない場合、v0.0.1 を新しいバージョンとする
# DONE: v0.0.1 が存在する場合、v0.0.2 を新しいバージョンとする
# DONE: v1.2.3 が存在する場合、v1.2.4 を新しいバージョンとする
# DONE: プレリリースタグ（v1.0.0-rc1）が存在する場合、無視される
# DONE: 複数タグが存在する場合、最新のものからインクリメントする
# DONE: --dry-run モードではタグの作成・プッシュをスキップする
# DONE: 不明なオプションが渡された場合、エラー終了する

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCRIPT_UNDER_TEST="${SCRIPT_DIR}/create-version-tag.sh"

# テスト用の一時ディレクトリ
TEST_DIR=""

# テスト結果カウンタ
PASSED=0
FAILED=0

# セットアップ: テスト用のgitリポジトリを作成
setup() {
    TEST_DIR=$(mktemp -d)
    cd "$TEST_DIR"
    git init --initial-branch=main
    git config user.email "test@example.com"
    git config user.name "Test User"
    # 最低1つのコミットが必要
    echo "initial" > README.md
    git add README.md
    git commit -m "Initial commit"
}

# クリーンアップ
teardown() {
    cd /
    rm -rf "$TEST_DIR"
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

echo "=== create-version-tag.sh テスト ==="

# テスト1: タグが存在しない場合、v0.0.1 を新しいバージョンとする
test_no_existing_tags() {
    local output
    output=$("$SCRIPT_UNDER_TEST" --dry-run 2>&1)
    assert_output_contains "新しいバージョン: v0.0.1" "$output"
}
run_test "タグが存在しない場合、v0.0.1を新しいバージョンとする" test_no_existing_tags

# テスト2: v0.0.1 が存在する場合、v0.0.2 を新しいバージョンとする
test_increment_patch_from_001() {
    git tag v0.0.1
    local output
    output=$("$SCRIPT_UNDER_TEST" --dry-run 2>&1)
    assert_output_contains "新しいバージョン: v0.0.2" "$output"
}
run_test "v0.0.1が存在する場合、v0.0.2を新しいバージョンとする" test_increment_patch_from_001

# テスト3: v1.2.3 が存在する場合、v1.2.4 を新しいバージョンとする
test_increment_patch_from_123() {
    git tag v1.2.3
    local output
    output=$("$SCRIPT_UNDER_TEST" --dry-run 2>&1)
    assert_output_contains "新しいバージョン: v1.2.4" "$output"
}
run_test "v1.2.3が存在する場合、v1.2.4を新しいバージョンとする" test_increment_patch_from_123

# テスト4: プレリリースタグが存在する場合、無視される
test_ignore_prerelease_tags() {
    git tag v1.0.0-rc1
    local output
    output=$("$SCRIPT_UNDER_TEST" --dry-run 2>&1)
    # プレリリースタグしかないので、v0.0.1 になるはず
    assert_output_contains "新しいバージョン: v0.0.1" "$output"
}
run_test "プレリリースタグが存在する場合、無視される" test_ignore_prerelease_tags

# テスト5: 複数タグが存在する場合、最新のものからインクリメントする
test_multiple_tags_uses_latest() {
    git tag v0.1.0
    git tag v0.2.0
    git tag v0.1.5
    local output
    output=$("$SCRIPT_UNDER_TEST" --dry-run 2>&1)
    # v0.2.0 が最新なので、v0.2.1 になるはず
    assert_output_contains "新しいバージョン: v0.2.1" "$output"
}
run_test "複数タグが存在する場合、最新のものからインクリメントする" test_multiple_tags_uses_latest

# テスト6: --dry-run モードではタグの作成・プッシュをスキップする
test_dry_run_skips_tag_creation() {
    local output
    output=$("$SCRIPT_UNDER_TEST" --dry-run 2>&1)
    assert_output_contains "[ドライラン] タグの作成とプッシュをスキップします。" "$output"
    # タグが作成されていないことを確認
    local tag_count
    tag_count=$(git tag -l | wc -l)
    if [[ "$tag_count" -ne 0 ]]; then
        echo "  FAIL: ドライランモードでタグが作成されています（タグ数: $tag_count）"
        return 1
    fi
}
run_test "--dry-runモードではタグの作成・プッシュをスキップする" test_dry_run_skips_tag_creation

# テスト7: 不明なオプションが渡された場合、エラー終了する
test_unknown_option_exits_with_error() {
    local output
    local exit_code=0
    output=$("$SCRIPT_UNDER_TEST" --unknown 2>&1) || exit_code=$?
    assert_exit_code 1 "$exit_code" && assert_output_contains "不明なオプション: --unknown" "$output"
}
run_test "不明なオプションが渡された場合、エラー終了する" test_unknown_option_exits_with_error

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
