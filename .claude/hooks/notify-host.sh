#!/bin/bash
if [ -z "$WORKSPACE_DIR" ]; then
  echo "[ERROR] 環境変数 'WORKSPACE_DIR' が設定されていません。"
  exit 1
fi

if ! command -v jq &> /dev/null; then
  echo "[ERROR] jq が見つかりません。jq をインストールしてください。"
  exit 1
fi

jq -n --arg msg "${1:-Done}" --arg title "${2:-Dev Container}" \
  '{message: $msg, title: $title}' > "${WORKSPACE_DIR}/.devcontainer/host-notifier.json"
