#!/bin/bash
if [ -z "$VOX_ACTOR_WORKSPACE" ]; then
  GIT_COMMON_DIR=$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null)
  if [ -z "$GIT_COMMON_DIR" ]; then
    echo "[ERROR] 環境変数 'VOX_ACTOR_WORKSPACE' が設定されておらず、gitリポジトリ外のためデフォルト出力先を決定できません。"
    exit 1
  fi
  VOX_ACTOR_WORKSPACE="$(dirname "$GIT_COMMON_DIR")/.tmp/notify"
fi

PROBABILITY="$1"
TEXT="$2"
SPEAKER="$3"
SPEED_SCALE="$4"

if [ -z "$PROBABILITY" ]; then
  echo "[ERROR] 通知確率（第1引数）は必須です"
  exit 1
fi

if [ -z "$TEXT" ]; then
  echo "[ERROR] セリフ（第2引数）は必須です"
  exit 1
fi

if [ -z "$SPEAKER" ]; then
  echo "[ERROR] スピーカーID（第3引数）は必須です"
  exit 1
fi

if [ -z "$SPEED_SCALE" ]; then
  echo "[ERROR] 話速（第4引数）は必須です"
  exit 1
fi

if ! [[ "$PROBABILITY" =~ ^[0-9]+$ ]] || [ "$PROBABILITY" -lt 1 ] || [ "$PROBABILITY" -gt 100 ]; then
  echo "[ERROR] 通知確率は1〜100の整数で指定してください: '$PROBABILITY'"
  exit 1
fi

if ! [[ "$SPEAKER" =~ ^[0-9]+$ ]] || [ "$SPEAKER" -lt 1 ]; then
  echo "[ERROR] スピーカーIDは正の整数で指定してください: '$SPEAKER'"
  exit 1
fi

if ! [[ "$SPEED_SCALE" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
  echo "[ERROR] speedScaleは正の数値で指定してください: '$SPEED_SCALE'"
  exit 1
fi

# 乱数判定: ROLL <= PROBABILITY なら通知する
ROLL=$((RANDOM % 100 + 1))
if [ "$ROLL" -gt "$PROBABILITY" ]; then
  exit 0
fi

mkdir -p "$VOX_ACTOR_WORKSPACE"
jq -cn --argjson speaker "$SPEAKER" --arg text "$TEXT" --argjson speedScale "$SPEED_SCALE" \
  '{speaker: $speaker, text: $text, speedScale: $speedScale}' > "${VOX_ACTOR_WORKSPACE}/notify_$(($(date +%s%N)/1000000)).json"
