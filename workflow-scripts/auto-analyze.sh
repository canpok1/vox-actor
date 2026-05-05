#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [[ $# -ne 1 ]] || ! [[ "$1" =~ ^[0-9]+$ ]] || [[ "$1" -eq 0 ]]; then
  echo "Usage: $0 <min-count>" >&2
  exit 1
fi

MIN_COUNT="$1"

LOCK_DIR="${WORKSPACE_DIR}/.tmp/locks"
mkdir -p "$LOCK_DIR"
lock_file="${LOCK_DIR}/auto-analyze"

exec 9>"$lock_file"
if ! flock -n 9; then
  echo "auto-analyze is already running"
  exit 1
fi

trap 'echo ""; echo "Shutting down..."; exit 0' SIGINT SIGTERM

MEMO_DONE_DIR="${WORKSPACE_DIR}/.tmp/memo/done"

echo "Watching ${MEMO_DONE_DIR} (min-count: ${MIN_COUNT})..."

PREV_STATE=""

while true; do
  if [[ -d "$MEMO_DONE_DIR" ]]; then
    FILE_COUNT=$(( $(find "$MEMO_DONE_DIR" -maxdepth 1 -type f | wc -l) ))
  else
    FILE_COUNT=0
  fi

  if [[ "$FILE_COUNT" -ge "$MIN_COUNT" ]]; then
    CURRENT_STATE="analyzing"
    echo ""
    echo "File count: ${FILE_COUNT} (>= ${MIN_COUNT}), starting analyze-work-memo..."

    "${SCRIPT_DIR}/claude-stream.sh" -p "/base-tools:analyze-work-memo"
  else
    CURRENT_STATE="waiting"
    if [[ "$PREV_STATE" != "$CURRENT_STATE" ]]; then
      echo ""
      echo "File count: ${FILE_COUNT} (< ${MIN_COUNT}), waiting..."
    else
      printf "."
    fi
  fi

  PREV_STATE="$CURRENT_STATE"
  sleep 60
done
