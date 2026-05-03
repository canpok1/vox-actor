#!/bin/bash
if ! command -v vox-actor >/dev/null 2>&1; then
  echo "[ERROR] vox-actor コマンドが必要です。インストール方法は README を参照してください" >&2
  exit 1
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

QUEUE_DIR=$(vox-actor config path.queue) || exit 1
WORKSPACE_DIR=$(vox-actor config path.workspace) || exit 1

if vox-actor audio-check >/dev/null 2>&1 || vox-actor viewer-check >/dev/null 2>&1; then
  MODE="direct"
else
  MODE="file"
fi

case "$MODE" in
  direct)
    mkdir -p "$WORKSPACE_DIR"
    ERROR_LOG="${WORKSPACE_DIR}/play-script-errors.log"
    MAX_LOG_LINES=200
    ID=$(vox-actor act "$JSONL_PATH" 2>>"$ERROR_LOG")
    STATUS=$?
    if [ "$STATUS" -ne 0 ]; then
      TS=$(date '+%Y-%m-%d %H:%M:%S')
      echo "[$TS] act exit=$STATUS path=$JSONL_PATH" >> "$ERROR_LOG"
      LINES=$(wc -l < "$ERROR_LOG")
      if [ "$LINES" -gt "$MAX_LOG_LINES" ]; then
        TMP_LOG=$(mktemp "${ERROR_LOG}.XXXXXX")
        tail -n "$MAX_LOG_LINES" "$ERROR_LOG" > "$TMP_LOG"
        mv "$TMP_LOG" "$ERROR_LOG"
      fi
      rm -f "$JSONL_PATH"
      exit "$STATUS"
    fi
    rm -f "$JSONL_PATH"
    vox-actor playback wait "$ID" 2>>"$ERROR_LOG"
    ;;
  file)
    mkdir -p "$QUEUE_DIR"
    DEST="${QUEUE_DIR}/$(basename "$JSONL_PATH")"
    mv "$JSONL_PATH" "$DEST"
    ;;
esac
