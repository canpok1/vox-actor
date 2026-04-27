#!/bin/sh
# PreToolUse hook: block `git commit` when current branch is main

set -eu

input=$(cat)
if [ -z "$input" ]; then
  printf 'Error: hook に stdin が渡されていません。\n' >&2
  exit 2
fi

command=$(printf '%s' "$input" | jq -r '.tool_input.command // ""')

case "$command" in
  "git commit"*) ;;
  *) exit 0 ;;
esac

branch=$(git symbolic-ref --short HEAD 2>/dev/null || true)
if [ "$branch" = "main" ]; then
  printf 'Error: mainブランチへの直接コミットは禁止です。\n' >&2
  exit 2
fi

exit 0
