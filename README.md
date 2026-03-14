# vox-actor

テキストファイルをVOICEVOXエンジンで音声合成し、読み上げるCLIツールです。

## 前提条件

- Go 1.26.1 以上
- [VOICEVOX エンジン](https://voicevox.hiroshiba.jp/)が起動していること（デフォルト: `http://localhost:50021`）

## インストール

```bash
go install github.com/canpok1/vox-actor@latest
```

または、ソースからビルドする場合:

```bash
git clone https://github.com/canpok1/vox-actor.git
cd vox-actor
make build
```

## 使い方

### テキストファイルの読み上げ

```bash
vox-actor act script.txt
```

VOICEVOXエンジンのURLやキャラクターを指定する場合:

```bash
vox-actor act --engine-url http://localhost:50021 --speaker 3 script.txt
```

話速・音高・抑揚を調整する場合:

```bash
vox-actor act --speed 1.2 --pitch 0.1 --intonation 1.5 script.txt
```

### ディレクトリ監視モード

ディレクトリを監視し、ファイルが配置されると自動的に読み上げます:

```bash
vox-actor act --watch /path/to/watch-dir
```

詳細ログを出力する場合:

```bash
vox-actor act --verbose script.txt
```

## オプション一覧

### `act` サブコマンド

| オプション | デフォルト値 | 説明 |
|---|---|---|
| `--engine-url` | `http://localhost:50021` | VOICEVOXエンジンのURL |
| `--speaker` | `3` | キャラクターID |
| `--speed` | `1.0` | 話速 |
| `--pitch` | `0.0` | 音高 |
| `--intonation` | `1.0` | 抑揚 |
| `--watch` | `false` | ディレクトリ監視モードを有効化 |
| `--verbose` | `false` | 詳細ログを出力 |

## 開発

```bash
# セットアップ（linter等のインストール）
make setup

# ビルド
make build

# テスト
make test

# フォーマット
make fmt

# Lint
make lint
```

## ライセンス

[MIT License](LICENSE)
