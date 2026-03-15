#!/bin/bash
if [ -z "$WORKSPACE_DIR" ]; then
  echo "[ERROR] 環境変数 'WORKSPACE_DIR' が設定されていません。"
  exit 1
fi

TEXT="$1"
SPEAKER="${2:-3}"

if ! [[ "$SPEAKER" =~ ^[0-9]+$ ]]; then
  echo "[ERROR] スピーカーIDは正の整数で指定してください: '$SPEAKER'"
  exit 1
fi

NOTIFY_DIR="${WORKSPACE_DIR}/.tmp/notify"
mkdir -p "$NOTIFY_DIR"
jq -cn --argjson speaker "$SPEAKER" --arg text "$TEXT" \
  '{speaker: $speaker, text: $text}' > "${NOTIFY_DIR}/notify_$(($(date +%s%N)/1000000)).json"
