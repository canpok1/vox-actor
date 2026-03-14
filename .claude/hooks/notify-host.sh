#!/bin/bash
if [ -z "$WORKSPACE_DIR" ]; then
  echo "[ERROR] 環境変数 'WORKSPACE_DIR' が設定されていません。"
  exit 1
fi

if ! command -v jq &> /dev/null; then
  echo "[ERROR] jq が見つかりません。jq をインストールしてください。"
  exit 1
fi

echo "$1" > "${WORKSPACE_DIR}/.notify/notify_$(($(date +%s%N)/1000000)).txt"
