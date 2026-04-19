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
   curl -fLo vox-actor.tar.gz https://github.com/canpok1/vox-actor/releases/latest/download/vox-actor_Linux_x86_64.tar.gz
   tar xzf vox-actor.tar.gz
   sudo mv vox-actor /usr/local/bin/

   # macOS (Apple Silicon) の場合
   curl -fLo vox-actor.tar.gz https://github.com/canpok1/vox-actor/releases/latest/download/vox-actor_Darwin_arm64.tar.gz
   tar xzf vox-actor.tar.gz
   sudo mv vox-actor /usr/local/bin/

   # macOS (Intel) の場合
   curl -fLo vox-actor.tar.gz https://github.com/canpok1/vox-actor/releases/latest/download/vox-actor_Darwin_x86_64.tar.gz
   tar xzf vox-actor.tar.gz
   sudo mv vox-actor /usr/local/bin/
   ```

3. **テキストを読み上げる**

   ```bash
   vox-actor say "こんにちは"
   ```

4. **ディレクトリを監視して自動読み上げする**

   ```bash
   vox-actor watch /path/to/watch-dir
   ```

   複数ディレクトリを同時に監視する場合は、スペース区切りでパスを追加できます。

   ```bash
   vox-actor watch /path/to/dir-a /path/to/dir-b
   ```

   別のターミナルからテキストファイルを配置すると、自動的に読み上げられます。

   ```bash
   echo "こんばんは" > /path/to/watch-dir/sample.txt
   ```

## 前提条件

- Go 1.26.1 以上（ソースからビルドする場合）
- [VOICEVOX エンジン](https://voicevox.hiroshiba.jp/)が起動していること（デフォルト: `http://localhost:50021`）
  - インストーラー版: [公式サイト](https://voicevox.hiroshiba.jp/)からダウンロードしてインストール後、アプリを起動する
  - Docker版: `docker run --rm -p 50021:50021 voicevox/voicevox_engine:latest`

## インストール

- バイナリをダウンロードする場合
   - [GitHub Releases](https://github.com/canpok1/vox-actor/releases)からビルド済みバイナリをダウンロードして配置してください。

- Homebrewを使う場合（macOS/Linux）:
   ```bash
   brew tap canpok1/tap
   brew install --cask vox-actor
   ```

- Goの`install`コマンドを使う場合
   ```bash
   go install github.com/canpok1/vox-actor@latest
   ```

- ソースからビルドする場合
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

ディレクトリを監視し、ファイルが配置されると自動的に読み上げます。処理済みファイルは各監視ディレクトリ内の `done/` サブディレクトリに移動されます。

```bash
vox-actor watch /path/to/watch-dir
```

複数ディレクトリを同時に監視する場合:

```bash
vox-actor watch /path/to/dir-a /path/to/dir-b
```

各ディレクトリは並列で監視され、検知したファイルは検知順に1件ずつ再生されます。処理済みファイルは各ディレクトリ直下の `done/` に移動されます（例: `./dir-a/foo.txt` → `./dir-a/done/foo.txt`）。

処理済みファイルを `done/` に移動する代わりに削除する場合:

```bash
vox-actor watch --delete /path/to/watch-dir
```

#### `act --watch` / `act --watch-delete`（後方互換）

従来どおり `act` コマンドでも単一ディレクトリの監視が可能です。

```bash
vox-actor act --watch /path/to/watch-dir
vox-actor act --watch-delete /path/to/watch-dir
```

`--watch` と `--watch-delete` は同時に指定できません。複数ディレクトリを同時に監視したい場合は `watch` コマンドを使ってください。

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

### `watch` サブコマンド

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--engine-url` | `VOX_ENGINE_URL` | `http://localhost:50021` | VOICEVOXエンジンのURL |
| `--speaker` | `VOX_SPEAKER` | `3` | キャラクターID |
| `--speed` | — | `1.0` | 話速 |
| `--pitch` | — | `0.0` | 音高 |
| `--intonation` | — | `1.0` | 抑揚 |
| `--delete` | — | `false` | 処理済みファイルを削除（未指定時は各ディレクトリの `done/` に移動） |
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

## 活用方法

### claude codeなどのLLMの作業状況を音声で通知する

LLMの作業状況を指定ディレクトリにテキストファイルとして出力し、vox-actorの監視モードで読み上げることで、作業の進捗を音声で把握できます。

claude code向けプラグインを同梱しているので、claude codeでは簡単に導入できます。同梱プラグインは以下の2種類です。

- `vox-actor-plugin`: 独り言スキル（`/vox-actor-plugin:monologue`）を提供するプラグイン
- `auto-monologue-plugin`: Stop hookでClaudeに独り言スキルの活用を促すプラグイン（`vox-actor-plugin` と併用することで、作業の区切りで独り言が自動生成されるようになる）

1. 環境変数を設定する。
   ```
   # テキストファイルの出力先ディレクトリを指定
   export VOX_ACTOR_WORKSPACE=/path/to/directory
   ```

2. vox-actorを監視モードで起動する。
   ```
   vox-actor watch $VOX_ACTOR_WORKSPACE
   ```

3. claude codeにプラグインを導入する。
   ```
   # claude code上で実行
   /plugin marketplace add canpok1/vox-actor
   /plugin install vox-actor-plugin@vox-actor-marketplace
   # 自動で独り言を生成したい場合は auto-monologue-plugin も導入する
   /plugin install auto-monologue-plugin@vox-actor-marketplace
   ```

4. 独り言スキル（monologue）を実行する。
   ```
   # claude code上で実行
   /vox-actor-plugin:monologue
   ```
   `auto-monologue-plugin` を導入している場合は、作業の区切りで自動的に独り言が生成される。


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
