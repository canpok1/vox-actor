#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ $# -gt 0 ]]; then
  echo "Usage: $0" >&2; exit 1
fi

REPO="$(gh repo view --json nameWithOwner -q '.nameWithOwner')"

RUNNING=true
trap 'RUNNING=false; echo ""; echo "Shutting down..."; exit 0' SIGINT SIGTERM

echo "Watching queued issues in ${REPO}..."

while $RUNNING; do
  # assign-to-claudeラベル付き + in-progress-by-claude未付与のIssueを検索
  ISSUE=$(gh issue list \
    --repo "$REPO" \
    --state open \
    --label "assign-to-claude" \
    --json number,title,labels \
    --jq '[.[] | select(.labels | map(.name) | index("in-progress-by-claude") | not)] | sort_by(.number) | first // empty')

  if [[ -n "$ISSUE" ]]; then
    ISSUE_NUMBER=$(echo "$ISSUE" | jq -r '.number')
    ISSUE_TITLE=$(echo "$ISSUE" | jq -r '.title')
    echo ""
    echo "#${ISSUE_NUMBER} ${ISSUE_TITLE}"
    "${SCRIPT_DIR}/solve-issue.sh" -p "$ISSUE_NUMBER"
  else
    printf "."
  fi

  sleep 60
done
