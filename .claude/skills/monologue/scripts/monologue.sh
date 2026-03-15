#!/bin/bash
if [ -z "$WORKSPACE_DIR" ]; then
  echo "[ERROR] 環境変数 'WORKSPACE_DIR' が設定されていません。"
  exit 1
fi

PROBABILITY="$1"
TEXT="$2"
SPEAKER="${3:-3}"

if ! [[ "$PROBABILITY" =~ ^[0-9]+$ ]] || [ "$PROBABILITY" -lt 1 ] || [ "$PROBABILITY" -gt 100 ]; then
  echo "[ERROR] 通知確率は1〜100の整数で指定してください: '$PROBABILITY'"
  exit 1
fi

if ! [[ "$SPEAKER" =~ ^[0-9]+$ ]]; then
  echo "[ERROR] スピーカーIDは正の整数で指定してください: '$SPEAKER'"
  exit 1
fi

# 乱数判定: ROLL <= PROBABILITY なら通知する
ROLL=$((RANDOM % 100 + 1))
if [ "$ROLL" -gt "$PROBABILITY" ]; then
  exit 0
fi

NOTIFY_DIR="${WORKSPACE_DIR}/.tmp/notify"
mkdir -p "$NOTIFY_DIR"
jq -cn --argjson speaker "$SPEAKER" --arg text "$TEXT" \
  '{speaker: $speaker, text: $text}' > "${NOTIFY_DIR}/notify_$(($(date +%s%N)/1000000)).json"
