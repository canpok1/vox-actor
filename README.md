# vox-actor

テキストファイルやテキストをVOICEVOXエンジンで音声合成し、読み上げるCLIツールです。

## クイックスタート

1. **VOICEVOXエンジンを起動する**

   - インストーラー版: [公式サイト](https://voicevox.hiroshiba.jp/)からダウンロードしてインストール後、アプリを起動する
   - Docker版: `docker run --rm -p 50021:50021 voicevox/voicevox_engine:latest`

2. **vox-actorをインストールする**

   [GitHub Releases](https://github.com/canpok1/vox-actor/releases)からビルド済みバイナリをダウンロードして配置します。

   ```bash
   # Linux (x86_64) の場合
   curl -Lo vox-actor.tar.gz https://github.com/canpok1/vox-actor/releases/latest/download/vox-actor_Linux_x86_64.tar.gz
   tar xzf vox-actor.tar.gz
   sudo mv vox-actor /usr/local/bin/

   # macOS (Apple Silicon) の場合
   curl -Lo vox-actor.tar.gz https://github.com/canpok1/vox-actor/releases/latest/download/vox-actor_Darwin_arm64.tar.gz
   tar xzf vox-actor.tar.gz
   sudo mv vox-actor /usr/local/bin/
   ```

3. **テキストを読み上げる**

   ```bash
   vox-actor say "こんにちは"
   ```

4. **ディレクトリを監視して自動読み上げする**

   ```bash
   vox-actor act --watch /path/to/watch-dir
   ```

   別のターミナルからテキストファイルを配置すると、自動的に読み上げられます。

   ```bash
   echo "こんばんは" > /path/to/watch-dir/sample.txt
   ```

## 前提条件

- Go 1.26.1 以上
- [VOICEVOX エンジン](https://voicevox.hiroshiba.jp/)が起動していること（デフォルト: `http://localhost:50021`）
  - インストーラー版: [公式サイト](https://voicevox.hiroshiba.jp/)からダウンロードしてインストール後、アプリを起動する
  - Docker版: `docker run --rm -p 50021:50021 voicevox/voicevox_engine:latest`

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

## バージョン確認

```bash
vox-actor --version
```

## 使い方

### テキストの直接読み上げ

```bash
vox-actor say "こんにちは"
```

キャラクターや音声パラメータを指定する場合:

```bash
vox-actor say --speaker 3 --speed 1.2 "こんにちは"
```

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

### JSON台本モード（感情制御パラメータ付き）

`.json` ファイルを使うと、セリフごとにキャラクターや感情パラメータを指定できます:

```json
{
  "text": "こんにちは",
  "speaker": 3,
  "speedScale": 1.2,
  "pitchScale": 0.1,
  "intonationScale": 1.5
}
```

```bash
vox-actor act script.json
```

`text` のみ必須で、他のパラメータは省略可能です。省略した場合はCLIオプションのデフォルト値が使われます。

ディレクトリを指定した場合、`.txt` と `.json` の両方が辞書順で読み上げられます。

### ディレクトリ監視モード

ディレクトリを監視し、ファイルが配置されると自動的に読み上げます。処理済みファイルは監視ディレクトリ内の `done/` サブディレクトリに移動されます。

```bash
vox-actor act --watch /path/to/watch-dir
```

処理済みファイルを `done/` に移動する代わりに削除する場合:

```bash
vox-actor act --watch-delete /path/to/watch-dir
```

`--watch` と `--watch-delete` は同時に指定できません。

詳細ログを出力する場合:

```bash
vox-actor act --verbose script.txt
```

## オプション一覧

### `act` サブコマンド

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--engine-url` | `VOX_ENGINE_URL` | `http://localhost:50021` | VOICEVOXエンジンのURL |
| `--speaker` | `VOX_SPEAKER` | `3` | キャラクターID |
| `--speed` | — | `1.0` | 話速 |
| `--pitch` | — | `0.0` | 音高 |
| `--intonation` | — | `1.0` | 抑揚 |
| `--watch` | — | `false` | ディレクトリ監視モードを有効化 |
| `--watch-delete` | — | `false` | ディレクトリ監視モード（処理済みファイルを削除） |
| `--verbose` | — | `false` | 詳細ログを出力 |

### `say` サブコマンド

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--engine-url` | `VOX_ENGINE_URL` | `http://localhost:50021` | VOICEVOXエンジンのURL |
| `--speaker` | `VOX_SPEAKER` | `3` | キャラクターID |
| `--speed` | — | `1.0` | 話速 |
| `--pitch` | — | `0.0` | 音高 |
| `--intonation` | — | `1.0` | 抑揚 |
| `--verbose` | — | `false` | 詳細ログを出力 |

オプションの優先順位: **CLIフラグ > 環境変数 > デフォルト値**

## 開発

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

## ライセンス

[MIT License](LICENSE)
