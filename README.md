# vox-actor

**LLMの作業を"声"で届ける、VOICEVOX読み上げCLI**

## ✨ 特徴

- **LLM向けに設計** — claude code連携プラグインを同梱。`vox-actor` が `PATH` 上にあれば監視プロセス不要で即座に読み上げ
- **多彩な入力** — 直接テキスト／テキストファイル／JSON台本／ディレクトリ監視に対応
- **感情制御** — キャラクター・話速・音高・抑揚をセリフ単位で調整可能

## 🚀 クイックスタート（LLM連携フロー）

claude code で作業の区切りごとに独り言を読み上げさせる最短手順です。`vox-actor` コマンドが `PATH` 上にある環境では `direct` モードで動作し、`vox-actor watch` の常駐や `VOX_ACTOR_WORKSPACE` の設定は不要です。

1. **VOICEVOXエンジンを起動する**

   - インストーラー版: [公式サイト](https://voicevox.hiroshiba.jp/)からダウンロードしてインストール後、アプリを起動する
   - Docker版: `docker run --rm -p 50021:50021 voicevox/voicevox_engine:latest`

2. **vox-actorをインストールする（Homebrew）**

   ```bash
   brew tap canpok1/tap
   brew install --cask vox-actor
   ```

   Homebrew以外のインストール方法は[インストール方法](#-インストール方法)を参照してください。

3. **claude codeにプラグインを導入する**

   ```
   # claude code上で実行
   /plugin marketplace add canpok1/vox-actor
   /plugin install vox-actor-plugin@vox-actor-marketplace
   # 自動で独り言を生成したい場合は auto-monologue-plugin も導入
   /plugin install auto-monologue-plugin@vox-actor-marketplace
   ```

4. **作業の区切りで自動的に独り言が読み上げられる**

   `auto-monologue-plugin` を導入していれば、claude code の作業の区切りごとに自動で独り言が生成・読み上げされます。手動で呼び出す場合は `/vox-actor-plugin:monologue` を実行します。

> **💡 リモート環境で使う場合や複数セッションの音声を逐次再生したい場合**
> `direct` モードは同時呼び出し時に音声が並列再生されます。ホストとclaude code実行環境が分かれている場合や、逐次再生したい場合は[🔀 高度な利用（リモート環境・監視モード）](#-高度な利用リモート環境監視モード)を参照してください。

## 🛠 その他の使い方

LLM連携を使わず、CLIとして単体で使う場合の例です。

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

話速・音高・抑揚を調整する場合:

```bash
vox-actor act --speed 1.2 --pitch 0.1 --intonation 1.5 script.txt
```

### JSON台本モード（感情制御パラメータ付き）

`.json` ファイルを使うと、セリフごとにキャラクターや感情パラメータを指定できます。

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

### バージョン確認

```bash
vox-actor --version
```

## 📖 コマンドリファレンス

オプションの優先順位: **CLIフラグ > 環境変数 > デフォルト値**

### `say` サブコマンド

テキストを直接引数で渡して読み上げる。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--engine-url` | `VOX_ENGINE_URL` | `http://localhost:50021` | VOICEVOXエンジンのURL |
| `--speaker` | `VOX_SPEAKER` | `3` | キャラクターID |
| `--speed` | — | `1.0` | 話速 |
| `--pitch` | — | `0.0` | 音高 |
| `--intonation` | — | `1.0` | 抑揚 |
| `--verbose` | — | `false` | 詳細ログを出力 |

### `act` サブコマンド

テキストファイル／JSON台本／ディレクトリを読み上げる。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--engine-url` | `VOX_ENGINE_URL` | `http://localhost:50021` | VOICEVOXエンジンのURL |
| `--speaker` | `VOX_SPEAKER` | `3` | キャラクターID |
| `--speed` | — | `1.0` | 話速 |
| `--pitch` | — | `0.0` | 音高 |
| `--intonation` | — | `1.0` | 抑揚 |
| `--watch` | — | `false` | ディレクトリ監視モードを有効化（後方互換。[高度な利用](#-高度な利用リモート環境監視モード)参照） |
| `--watch-delete` | — | `false` | ディレクトリ監視モード（処理済みファイルを削除。後方互換） |
| `--verbose` | — | `false` | 詳細ログを出力 |

### `watch` サブコマンド

1つ以上のディレクトリを並列監視し、配置されたファイルを検知順に逐次再生する。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--engine-url` | `VOX_ENGINE_URL` | `http://localhost:50021` | VOICEVOXエンジンのURL |
| `--speaker` | `VOX_SPEAKER` | `3` | キャラクターID |
| `--speed` | — | `1.0` | 話速 |
| `--pitch` | — | `0.0` | 音高 |
| `--intonation` | — | `1.0` | 抑揚 |
| `--delete` | — | `false` | 処理済みファイルを削除（未指定時は各ディレクトリの `done/` に移動） |
| `--verbose` | — | `false` | 詳細ログを出力 |

## 📦 インストール方法

お好みの方法でインストールできます。

- **Homebrew（macOS/Linux）**
   ```bash
   brew tap canpok1/tap
   brew install --cask vox-actor
   ```

- **バイナリダウンロード**

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

- **`go install` を使う**
   ```bash
   go install github.com/canpok1/vox-actor@latest
   ```

- **ソースからビルド**
   ```bash
   git clone https://github.com/canpok1/vox-actor.git
   cd vox-actor
   make build
   ```

## 🧩 前提条件

- [VOICEVOXエンジン](https://voicevox.hiroshiba.jp/) が起動していること（デフォルト: `http://localhost:50021`）
  - インストーラー版: [公式サイト](https://voicevox.hiroshiba.jp/)からダウンロードしてインストール後、アプリを起動する
  - Docker版: `docker run --rm -p 50021:50021 voicevox/voicevox_engine:latest`
- Go 1.26.1 以上（ソースからビルドする場合のみ）

## 🔀 高度な利用（リモート環境・監視モード）

以下のようなケースでは、`watch` コマンドを常駐させる `file` モードでの利用が適しています。

- claude code がコンテナ／リモート環境で動作し、`vox-actor` コマンドはホスト側にしか存在しない
- 複数セッションから同時に呼ばれても音声を逐次再生したい（`direct` モードは並列再生になる）

### `direct` モードと `file` モードの違い

| 観点 | `direct` モード | `file` モード |
|---|---|---|
| 読み上げ方式 | `vox-actor say` をその場で直接呼び出す | テキスト等をファイルに書き出し、別プロセスの `vox-actor watch` が読み上げる |
| 監視プロセス | 不要 | `vox-actor watch` の常駐が必要 |
| `VOX_ACTOR_WORKSPACE` | 任意（エラーログ出力先として使用） | 必須（ファイル受け渡し用ディレクトリ） |
| 同時呼び出し時 | 並列再生 | 検知順に逐次再生 |
| 前提 | `vox-actor` コマンドが `PATH` 上にある | claude code 側と `vox-actor watch` 側で `VOX_ACTOR_WORKSPACE` を共有 |

モードは `VOX_ACTOR_MONOLOGUE_MODE` 環境変数で明示するか、未設定時は `vox-actor` コマンドの有無で自動判定されます（あり → `direct`、なし → `file`）。

### `file` モードのセットアップ

1. **ホスト側で監視プロセスを常駐させる**
   ```bash
   # テキストファイルの受け渡し先ディレクトリを指定
   export VOX_ACTOR_WORKSPACE=/path/to/shared/directory
   vox-actor watch "$VOX_ACTOR_WORKSPACE"
   ```

2. **claude code 側で `file` モードを明示し、共有ディレクトリを指定する**
   ```bash
   export VOX_ACTOR_WORKSPACE=/path/to/shared/directory
   # vox-actorコマンドがある環境で file モードを強制したい場合は以下も指定
   export VOX_ACTOR_MONOLOGUE_MODE=file
   ```

3. **claude code にプラグインを導入する**（[クイックスタート](#-クイックスタートllm連携フロー)と同じ手順）

### ディレクトリ監視モードの詳細（`watch` コマンド）

`vox-actor watch` は、配置されたテキストファイルや JSON 台本を自動で読み上げます。処理済みファイルは各監視ディレクトリ直下の `done/` サブディレクトリに移動されます（例: `./dir-a/foo.txt` → `./dir-a/done/foo.txt`）。

```bash
vox-actor watch /path/to/watch-dir
```

複数ディレクトリを同時に監視する場合はスペース区切りで指定します。各ディレクトリは並列で監視され、検知したファイルは検知順に1件ずつ再生されます。

```bash
vox-actor watch /path/to/dir-a /path/to/dir-b
```

処理済みファイルを `done/` に移動する代わりに削除する場合:

```bash
vox-actor watch --delete /path/to/watch-dir
```

別のターミナルからファイルを配置すると、自動的に読み上げられます。

```bash
echo "こんばんは" > /path/to/watch-dir/sample.txt
```

### `act --watch` / `act --watch-delete`（後方互換）

従来どおり `act` コマンドでも単一ディレクトリの監視が可能です。

```bash
vox-actor act --watch /path/to/watch-dir
vox-actor act --watch-delete /path/to/watch-dir
```

`--watch` と `--watch-delete` は同時に指定できません。複数ディレクトリを同時に監視したい場合は `watch` コマンドを使ってください。

### エラーログ

`direct` モードでの `vox-actor say` 失敗は `$VOX_ACTOR_WORKSPACE/monologue-errors.log` に追記されます（ログは末尾200行でローテーション）。`tail -f` で確認できます。

## 👩‍💻 開発者向け情報

本リポジトリへのコントリビューター向けの情報です。

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
