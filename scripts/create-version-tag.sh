#!/bin/bash
# mainブランチにバージョンタグを付与するスクリプト
# 最新のリリースバージョンのパッチバージョンを1つ進めたタグを作成する
#
# 使用方法:
#   ./scripts/create-version-tag.sh           # 実際にタグを作成してプッシュ
#   ./scripts/create-version-tag.sh --dry-run # ドライランモード（タグ作成・プッシュをスキップ）

set -euo pipefail

DRY_RUN=false

# 引数の解析
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        *)
            echo "不明なオプション: $1" >&2
            exit 1
            ;;
    esac
done

# 最新のバージョンタグを取得（セマンティックバージョニング形式、プレリリースタグを除外）
# pipefail環境でのSIGPIPE問題を回避するため、grepの結果を一度変数に格納
ALL_TAGS=$(git tag -l "v[0-9]*.[0-9]*.[0-9]*" --sort=-v:refname | { grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' || true; })
LATEST_TAG=$(echo "$ALL_TAGS" | head -n 1)

if [[ -z "$LATEST_TAG" ]]; then
    echo "最新のタグが見つかりません。初期バージョン v0.0.1 を使用します。"
    NEW_VERSION="v0.0.1"
else
    echo "最新のタグ: $LATEST_TAG"

    # バージョン番号を解析（vプレフィックスを除去）
    IFS='.' read -r MAJOR MINOR PATCH <<< "${LATEST_TAG#v}"

    # パッチバージョンをインクリメント
    NEW_PATCH=$((PATCH + 1))
    NEW_VERSION="v${MAJOR}.${MINOR}.${NEW_PATCH}"
fi

echo "新しいバージョン: $NEW_VERSION"

if [[ "$DRY_RUN" == "true" ]]; then
    echo "[ドライラン] タグの作成とプッシュをスキップします。"
else
    # タグを作成
    git tag "$NEW_VERSION"
    echo "タグ $NEW_VERSION を作成しました。"

    # タグをプッシュ
    git push origin "$NEW_VERSION"
    echo "タグ $NEW_VERSION をプッシュしました。"
fi
