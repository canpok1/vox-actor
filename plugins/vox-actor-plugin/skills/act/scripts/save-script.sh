#!/bin/bash
set -e

# 種別引数の検証
kind="${1:?種別は必須です (monologue|speak|talk)}"

case "$kind" in
  monologue) ext="json" ;;
  speak|talk) ext="jsonl" ;;
  *)
    echo "[ERROR] 不明な種別: $kind" >&2
    exit 1
    ;;
esac

# tmpディレクトリを取得
tmp_dir=$(vox-actor config path.tmp) || exit 1

# tmpディレクトリを作成
mkdir -p "$tmp_dir"

# ユニックス時刻(ms)を取得
unix_ms=$(date +%s%3N)

# 出力ファイルのパス
outfile="${tmp_dir}/${unix_ms}_${kind}.${ext}"

# stdinから台本を読み込んでファイルに書き込む
cat > "$outfile"

# ファイルパスを標準出力に出力
echo "$outfile"
