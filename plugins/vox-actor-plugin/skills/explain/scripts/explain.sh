#!/bin/bash
if [ -z "$VOX_ACTOR_WORKSPACE" ]; then
  GIT_COMMON_DIR=$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null)
  if [ -n "$GIT_COMMON_DIR" ]; then
    VOX_ACTOR_WORKSPACE="$(dirname "$GIT_COMMON_DIR")/.vox-actor"
  else
    VOX_ACTOR_WORKSPACE="$PWD/.vox-actor"
  fi
fi

JSONL_PATH="$1"

if [ -z "$JSONL_PATH" ]; then
  echo "[ERROR] JSONLファイルのパス（第1引数）は必須です"
  exit 1
fi

if [ ! -f "$JSONL_PATH" ]; then
  echo "[ERROR] JSONLファイルが存在しません: '$JSONL_PATH'"
  exit 1
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
    ERROR_LOG="${VOX_ACTOR_WORKSPACE}/explain-errors.log"
    MAX_LOG_LINES=200
    OUTPUT=$(vox-actor act "$JSONL_PATH" 2>&1)
    STATUS=$?
    if [ "$STATUS" -ne 0 ]; then
      TS=$(date '+%Y-%m-%d %H:%M:%S')
      {
        echo "[$TS] exit=$STATUS path=$JSONL_PATH"
        printf '%s\n' "$OUTPUT" | sed 's/^/  /'
      } >> "$ERROR_LOG"
      LINES=$(wc -l < "$ERROR_LOG")
      if [ "$LINES" -gt "$MAX_LOG_LINES" ]; then
        TMP_LOG=$(mktemp "${ERROR_LOG}.XXXXXX")
        tail -n "$MAX_LOG_LINES" "$ERROR_LOG" > "$TMP_LOG"
        mv "$TMP_LOG" "$ERROR_LOG"
      fi
    fi
    rm -f "$JSONL_PATH"
    ;;
  file)
    QUEUE_DIR="${VOX_ACTOR_WORKSPACE}/queue"
    mkdir -p "$QUEUE_DIR"
    DEST="${QUEUE_DIR}/explain_$(($(date +%s%N)/1000000)).jsonl"
    mv "$JSONL_PATH" "$DEST"
    ;;
  *)
    echo "[ERROR] VOX_ACTOR_MONOLOGUE_MODE は 'direct' または 'file' で指定してください: '$MODE'"
    exit 1
    ;;
esac
