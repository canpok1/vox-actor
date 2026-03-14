#!/bin/bash
if [ -z "$WORKSPACE_DIR" ]; then
  echo "[ERROR] 環境変数 'WORKSPACE_DIR' が設定されていません。"
  exit 1
fi

NOTIFY_DIR="${WORKSPACE_DIR}/.tmp/notify"
mkdir -p "$NOTIFY_DIR"
echo "$1" > "${NOTIFY_DIR}/notify_$(($(date +%s%N)/1000000)).txt"
