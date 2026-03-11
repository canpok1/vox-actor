#!/usr/bin/env bash
# リモートURLからowner/repoを取得（プロキシURLと通常のGitHub URL両方に対応）
# 使い方: source "$(dirname "$0")/lib/detect-repo.sh"
# 結果: REPO変数にowner/repoが設定される

REMOTE_URL=$(git remote get-url origin)
if [[ "$REMOTE_URL" == */git/* ]]; then
  REPO="${REMOTE_URL##*/git/}"
  REPO="${REPO%.git}"
else
  REPO=$(echo "$REMOTE_URL" | sed -n 's|.*github\.com[:/]\([^/]*/[^/.]*\)\(\.git\)\{0,1\}$|\1|p')
fi

if [ -z "$REPO" ]; then
  echo "Error: Could not detect owner/repo from remote URL: $REMOTE_URL" >&2
  exit 1
fi
