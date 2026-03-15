#!/usr/bin/env bash
# 現在のGitHubユーザーのログイン名を取得する
# 使い方: source "$(dirname "$0")/lib/current-user.sh"
# 結果: CURRENT_USER変数にログイン名が設定される

CURRENT_USER=$(gh api user --jq '.login')

if [ -z "$CURRENT_USER" ]; then
  echo "Error: Could not detect current GitHub user" >&2
  exit 1
fi
