#!/usr/bin/env bash
# vox-actor バイナリを GitHub Releases から取得して INSTALL_DIR に配置する。
#
# このスクリプトはリリースアセットとして配布される専用スクリプトです。
# リリースページから直接ダウンロードして実行してください。
#
# 使い方:
#   bash install.sh
#   bash install.sh --help
#
# 環境変数:
#   INSTALL_DIR  バイナリ設置先ディレクトリ (デフォルト: $HOME/.local/bin)
set -euo pipefail

REPO="canpok1/vox-actor"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# リリース時に置換されるバージョンプレースホルダ
VERSION="__VERSION__"

usage() {
  echo "Usage: $0" >&2
  echo "" >&2
  echo "このスクリプトはリリースアセットとして配布されます。" >&2
  echo "リリースページから install.sh をダウンロードして実行してください。" >&2
  echo "" >&2
  echo "Environment variables:" >&2
  echo "  INSTALL_DIR  Installation directory (default: \$HOME/.local/bin)" >&2
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

# ---- プレースホルダ未置換チェック ----
if [[ ! "$VERSION" =~ ^v[0-9] ]]; then
  echo "ERROR: バージョンが埋め込まれていません。" >&2
  echo "       リリースページから install.sh をダウンロードして実行してください。" >&2
  exit 1
fi

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "ERROR: $1 が必要です" >&2
    exit 1
  fi
}

# ---- 必須コマンド検出 ----
if command -v curl >/dev/null 2>&1; then
  DOWNLOADER="curl"
elif command -v wget >/dev/null 2>&1; then
  DOWNLOADER="wget"
else
  echo "ERROR: curl または wget が必要です" >&2
  exit 1
fi

require_cmd tar

# sha256 検証コマンドを選択
if command -v sha256sum >/dev/null 2>&1; then
  SHA256CMD="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  SHA256CMD="shasum -a 256"
else
  echo "ERROR: sha256sum または shasum が必要です" >&2
  exit 1
fi

# ---- OS・arch 判定 ----
os_raw="$(uname -s)"
case "$os_raw" in
  Linux)  OS="Linux" ;;
  Darwin) OS="Darwin" ;;
  *)
    echo "ERROR: 未対応の OS: $os_raw (Linux / macOS のみサポート)" >&2
    exit 1
    ;;
esac

arch_raw="$(uname -m)"
case "$arch_raw" in
  x86_64 | amd64)   ARCH="x86_64" ;;
  aarch64 | arm64)  ARCH="arm64" ;;
  *)
    echo "ERROR: 未対応の arch: $arch_raw (x86_64 / arm64 のみサポート)" >&2
    exit 1
    ;;
esac

# ---- ダウンロードヘルパー ----
download() {
  local url="$1"
  local out="$2"
  if [[ "$DOWNLOADER" == "curl" ]]; then
    curl -fsSL -o "$out" "$url"
  else
    wget -q -O "$out" "$url"
  fi
}

# VERSION から先頭 "v" を除いたもの（checksums ファイル名用）
VERSION_NUM="${VERSION#v}"

ASSET_NAME="vox-actor_${OS}_${ARCH}.tar.gz"
CHECKSUMS_NAME="vox-actor_${VERSION_NUM}_checksums.txt"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
TARBALL_URL="${BASE_URL}/${ASSET_NAME}"
CHECKSUMS_URL="${BASE_URL}/${CHECKSUMS_NAME}"

# ---- インストール済みバージョン確認・分岐 ----
INSTALLED_VERSION=""
if command -v vox-actor >/dev/null 2>&1; then
  raw_ver="$(vox-actor --version 2>/dev/null || true)"
  # cobra デフォルト形式: "vox-actor version 0.0.1" → "v0.0.1" に正規化
  INSTALLED_VERSION="$(printf '%s' "$raw_ver" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)"
  INSTALLED_VERSION="${INSTALLED_VERSION:+v${INSTALLED_VERSION}}"
  INSTALLED_VERSION="${INSTALLED_VERSION:-$raw_ver}"
fi

if [[ -n "$INSTALLED_VERSION" ]]; then
  if [[ ! "$INSTALLED_VERSION" =~ ^v[0-9] ]]; then
    echo "警告: インストール済みバージョン ($INSTALLED_VERSION) が semver 形式ではありません。上書きインストールします。"
  elif [[ "$INSTALLED_VERSION" == "$VERSION" ]]; then
    echo "$VERSION は既にインストール済みです。スキップします。"
    exit 0
  else
    LOWER="$(printf '%s\n%s\n' "$VERSION" "$INSTALLED_VERSION" | sort -V | head -1)"
    if [[ "$LOWER" == "$VERSION" ]]; then
      echo "警告: より新しいバージョン ($INSTALLED_VERSION) がインストール済みです。インストールをスキップします。"
      exit 0
    else
      echo "古いバージョン ($INSTALLED_VERSION) から $VERSION へ更新します。"
    fi
  fi
else
  echo "$VERSION を新規インストールします。"
fi

# ---- 一時ディレクトリ（EXIT 時に自動削除） ----
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

# ---- ダウンロード ----
echo "ダウンロード中: $TARBALL_URL"
download "$TARBALL_URL" "$WORK_DIR/$ASSET_NAME"

echo "ダウンロード中: $CHECKSUMS_URL"
download "$CHECKSUMS_URL" "$WORK_DIR/$CHECKSUMS_NAME"

# ---- sha256 検証 ----
echo "チェックサムを検証しています..."
checksum_line="$(grep "$ASSET_NAME" "$WORK_DIR/$CHECKSUMS_NAME" || true)"
if [[ -z "$checksum_line" ]]; then
  echo "ERROR: checksums.txt に $ASSET_NAME のエントリが見つかりません" >&2
  exit 1
fi
echo "$checksum_line" | (cd "$WORK_DIR" && $SHA256CMD --check -)
echo "チェックサム OK"

# ---- 展開 ----
tar -xzf "$WORK_DIR/$ASSET_NAME" -C "$WORK_DIR" vox-actor

# ---- インストール ----
# 設置先を作成する（既存なら何もしない。$HOME/.local/bin はデフォルトでは無いことが多い）
if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
  echo "ERROR: INSTALL_DIR を作成できません: $INSTALL_DIR" >&2
  echo "       書き込み可能なディレクトリを INSTALL_DIR で指定してください。" >&2
  exit 1
fi

if [[ ! -w "$INSTALL_DIR" ]]; then
  echo "ERROR: INSTALL_DIR に書き込み権限がありません: $INSTALL_DIR" >&2
  echo "       書き込み可能なディレクトリを INSTALL_DIR で指定してください。" >&2
  echo "       例: curl -fsSL <URL> | INSTALL_DIR=\$HOME/.local/bin bash" >&2
  exit 1
fi

install -m 0755 "$WORK_DIR/vox-actor" "$INSTALL_DIR/vox-actor"

echo ""
echo "インストール完了: $INSTALL_DIR/vox-actor"

# ---- Linux ランタイム依存（ALSA）の案内 ----
# Linux 向けバイナリは ALSA ランタイム (libasound2 / alsa-lib) に動的リンクされている。
# 未インストールだと --version すら起動しないため、案内のうえで起動確認はスキップ可能にする。
if [[ "$OS" == "Linux" ]]; then
  echo ""
  echo "注意: Linux では ALSA ランタイムライブラリが必要です（音声再生・起動に必須）。"
  echo "      未インストールの場合は以下のいずれかを実行してください。"
  echo "        Debian/Ubuntu (23.10 以前): sudo apt install libasound2"
  echo "        Ubuntu 24.04 Noble 以降    : sudo apt install libasound2t64"
  echo "        RHEL/Fedora 系             : sudo dnf install alsa-lib"
fi

# 起動確認（ALSA 未導入の Linux 等で失敗しても致命扱いにしない）
if ! "$INSTALL_DIR/vox-actor" --version; then
  echo "警告: vox-actor の起動確認に失敗しました。上記の依存ライブラリを確認してください。" >&2
fi

# ---- PATH 案内 ----
# INSTALL_DIR が PATH に含まれていなければ追加方法を案内する（設定ファイルは書き換えない）
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*)
    ;;
  *)
    echo ""
    echo "注意: $INSTALL_DIR は PATH に含まれていません。"
    echo "      vox-actor コマンドを使うには、以下を shell の設定ファイル（例: ~/.bashrc, ~/.zshrc）に追記してください。"
    echo ""
    echo "      export PATH=\"$INSTALL_DIR:\$PATH\""
    ;;
esac
