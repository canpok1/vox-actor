#!/bin/bash
if [ -z "$VOX_ACTOR_WORKSPACE" ]; then
  GIT_COMMON_DIR=$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null)
  if [ -n "$GIT_COMMON_DIR" ]; then
    VOX_ACTOR_WORKSPACE="$(dirname "$GIT_COMMON_DIR")/.vox-actor"
  else
    VOX_ACTOR_WORKSPACE="$PWD/.vox-actor"
  fi
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

ROLL=$((RANDOM % 100 + 1))
if [ "$ROLL" -gt "$PROBABILITY" ]; then
  exit 0
fi

MODE="${VOX_ACTOR_MONOLOGUE_MODE:-}"
if [ -z "$MODE" ]; then
  if command -v vox-actor >/dev/null 2>&1; then
    MODE="direct"
  else
    MODE="file"
  fi
fi

mkdir -p "$VOX_ACTOR_WORKSPACE"

case "$MODE" in
  direct)
    ERROR_LOG="${VOX_ACTOR_WORKSPACE}/monologue-errors.log"
    MAX_LOG_LINES=200
    OUTPUT=$(vox-actor say --speaker "$SPEAKER" --speed "$SPEED_SCALE" "$TEXT" 2>&1)
    STATUS=$?
    if [ "$STATUS" -ne 0 ]; then
      TS=$(date '+%Y-%m-%d %H:%M:%S')
      {
        echo "[$TS] exit=$STATUS speaker=$SPEAKER speed=$SPEED_SCALE text=$TEXT"
        printf '%s\n' "$OUTPUT" | sed 's/^/  /'
      } >> "$ERROR_LOG"
      LINES=$(wc -l < "$ERROR_LOG")
      if [ "$LINES" -gt "$MAX_LOG_LINES" ]; then
        TMP_LOG=$(mktemp "${ERROR_LOG}.XXXXXX")
        tail -n "$MAX_LOG_LINES" "$ERROR_LOG" > "$TMP_LOG"
        mv "$TMP_LOG" "$ERROR_LOG"
      fi
    fi
    ;;
  file)
    QUEUE_DIR="${VOX_ACTOR_WORKSPACE}/queue"
    mkdir -p "$QUEUE_DIR"
    jq -cn --argjson speaker "$SPEAKER" --arg text "$TEXT" --argjson speedScale "$SPEED_SCALE" \
      '{speaker: $speaker, text: $text, speedScale: $speedScale}' > "${QUEUE_DIR}/monologue_$(($(date +%s%N)/1000000)).json"
    ;;
  *)
    echo "[ERROR] VOX_ACTOR_MONOLOGUE_MODE は 'direct' または 'file' で指定してください: '$MODE'"
    exit 1
    ;;
esac
