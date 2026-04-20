# 開発者向け情報

本リポジトリへのコントリビューター向けの情報です。

## 前提条件

- Go 1.26.1 以上

## よく使うコマンド

```bash
# セットアップ（linter等のインストール）
make setup

# ビルド
make build

# テスト
make test

# E2Eテスト
make test-e2e

# フォーマット
make fmt

# Lint
make lint

# 依存チェック
make depcheck

# ビルド成果物の削除
make clean
```

## アーキテクチャ

レイヤー構成と責務ルールは [layer-rules.md](./layer-rules.md) を参照してください。
