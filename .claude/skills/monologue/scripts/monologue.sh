#!/bin/bash
if [ -z "$WORKSPACE_DIR" ]; then
  echo "[ERROR] 環境変数 'WORKSPACE_DIR' が設定されていません。"
  exit 1
fi

TEXT="$1"
SPEAKER="${2:-3}"

NOTIFY_DIR="${WORKSPACE_DIR}/.tmp/notify"
mkdir -p "$NOTIFY_DIR"
jq -cn --argjson speaker "$SPEAKER" --arg text "$TEXT" \
  '{speaker: $speaker, text: $text}' > "${NOTIFY_DIR}/notify_$(($(date +%s%N)/1000000)).json"
