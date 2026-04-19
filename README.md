# vox-actor

**LLMの作業を"声"で届ける、VOICEVOX読み上げCLI**

## ✨ 特徴

- **LLM向けに設計** — claude code連携プラグインを同梱。`vox-actor` が `PATH` 上にあれば監視プロセス不要で即座に読み上げ
- **多彩な入力** — 直接テキスト／テキストファイル／JSON台本／ディレクトリ監視に対応
- **感情制御** — キャラクター・話速・音高・抑揚をセリフ単位で調整可能

## 🧩 前提条件

- [VOICEVOXエンジン](https://voicevox.hiroshiba.jp/) が起動していること（デフォルト: `http://localhost:50021`）
  - インストーラー版: [公式サイト](https://voicevox.hiroshiba.jp/)からダウンロードしてインストール後、アプリを起動する
  - Docker版: `docker run --rm -p 50021:50021 voicevox/voicevox_engine:latest`
- Go 1.26.1 以上（ソースからビルドする場合のみ）

## 🚀 クイックスタート

claude code で作業の区切りごとに独り言を読み上げさせる最短手順です。

1. **vox-actorをインストールする（Homebrew）**

   ```bash
   brew tap canpok1/tap
   brew install --cask vox-actor
   ```

   Homebrew以外のインストール方法は[インストール方法](#-インストール方法)を参照してください。

2. **claude codeにプラグインを導入する**

   ```
   # claude code上で実行
   /plugin marketplace add canpok1/vox-actor
   /plugin install vox-actor-plugin@vox-actor-marketplace
   # 自動で独り言を生成したい場合は auto-monologue-plugin も導入
   /plugin install auto-monologue-plugin@vox-actor-marketplace
   ```

3. **作業の区切りで自動的に独り言が読み上げられる**

   `auto-monologue-plugin` を導入していれば、claude code の作業の区切りごとに自動で独り言が生成・読み上げされます。手動で呼び出す場合は `/vox-actor-plugin:monologue` を、キャラクターに概念を解説させたい場合は `/vox-actor-plugin:explain <トピック>` を実行します。

リモート環境での利用や複数セッションの音声を逐次再生したい場合は[高度な利用](#-高度な利用リモート環境監視モード)を参照してください。

## 🎯 できること（概要）

vox-actor は **CLI** と **claude code プラグイン／スキル** の 2 系統の使い方があります。

### CLI

`vox-actor` コマンドを直接呼び出してテキストや台本ファイルを読み上げる使い方です。

- `vox-actor say <text>`: コマンドライン引数のテキストを直接読み上げる
- `vox-actor act <path>`: テキストファイル／JSON台本／JSONL台本／ディレクトリを読み上げる
- `vox-actor watch <dir1> [<dir2> ...]`: 1つ以上のディレクトリを並列監視し、配置されたファイルを検知順に逐次再生する

詳細は [CLIリファレンス](#-cliリファレンス) を参照してください。

### claude code プラグイン／スキル

claude code から `/vox-actor-plugin:<skill>` 形式で呼び出すスキルです。LLM が生成したセリフをキャラクターになりきって読み上げます。

- `/vox-actor-plugin:monologue`: 作業開始／終了／想定外のことが起こった時など、節目のキャラクターの一言独り言
- `/vox-actor-plugin:explain <トピック>`: 指定したトピックを冒頭→本題→まとめの流れで、複数セリフのJSONL台本としてキャラクターに解説させる

詳細は [プラグイン／スキルリファレンス](#-プラグインスキルリファレンス) を参照してください。

### 使い分け指針

| 用途 | 使うもの |
|------|---------|
| 手元のテキストや台本ファイルを即座に読み上げたい | CLI（`say` / `act`） |
| 監視ディレクトリに配置されたファイルを逐次再生したい | CLI（`watch`） |
| claude code の作業フローに読み上げを組み込みたい | プラグイン／スキル |
| キャラクターになりきった独り言や解説を生成したい | プラグイン／スキル |

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

## 📖 CLIリファレンス

オプションの優先順位: **オプション > 環境変数 > デフォルト値**

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

ディレクトリを指定した場合、`.txt` / `.json` / `.jsonl` が辞書順で読み上げられます。

### JSONL台本モード（複数セリフを1ファイルにまとめる）

`.jsonl` ファイルを使うと、1行1JSONオブジェクトの形式で複数のセリフを1ファイルにまとめて記述できます。各行のスキーマは上記の JSON 台本と同一で、`text` のみ必須・その他のパラメータは省略可能です。

```jsonl
{"text": "こんにちは", "speaker": 3}
{"text": "また会いましょう", "speaker": 3, "speedScale": 1.2}
```

```bash
vox-actor act script.jsonl
```

### バージョン確認

```bash
vox-actor --version
```

### `say` サブコマンド

```
vox-actor say <text>
```

テキストを直接引数で渡して読み上げる。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--engine-url` | `VOX_ENGINE_URL` | `http://localhost:50021` | VOICEVOXエンジンのURL |
| `--speaker` | `VOX_SPEAKER` | `3` | キャラクターID |
| `--speed` | — | `1.0` | 話速 |
| `--pitch` | — | `0.0` | 音高 |
| `--intonation` | — | `1.0` | 抑揚 |
| `--verbose` | — | `false` | 詳細ログを出力 |
| `--dry-run` | — | `false` | VOICEVOX・音声再生を行わず、読み上げ対象をログ出力（[詳細](#-dry-runモード)） |

### `act` サブコマンド

```
vox-actor act <path>
```

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
| `--dry-run` | — | `false` | VOICEVOX・音声再生を行わず、読み上げ対象をログ出力（[詳細](#-dry-runモード)） |

### `watch` サブコマンド

```
vox-actor watch <dir1> [<dir2> ...]
```

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
| `--dry-run` | — | `false` | VOICEVOX・音声再生を行わず、読み上げ対象をログ出力（[詳細](#-dry-runモード)） |

### 🧪 dry-runモード

`--dry-run` を付けると、VOICEVOXエンジンへの通信・音声再生を一切行わず、読み上げ対象の情報のみをログ出力します。VOICEVOXエンジン未起動の環境や音声出力不可の環境（CI・リモート・コンテナなど）での動作確認に利用できます。

```bash
# テキスト指定
vox-actor say --dry-run "こんにちは"

# ファイル指定
vox-actor act --dry-run script.json

# ディレクトリ監視（ディレクトリ監視・done/移動・削除は通常通り実施）
vox-actor watch --dry-run /path/to/watch-dir
```

出力例（標準エラー出力）:

```
  [2026-04-19 15:04:05] [dry run] synthesis completed (wavSize=0)
  [2026-04-19 15:04:05] [dry run] playback completed (text=こんにちは, speaker=3, speed=1.2, pitch=0.1, intonation=1.5)
```

- 非dry-run と同じメッセージ名（`synthesis completed` / `playback completed` 等）で、`[dry run]` プレフィックスが付与されます
- `synthesis completed` の `wavSize` は `0`（実際には合成していないため）です
- `playback completed` には dry-run 時のみ `text` / `speaker` / `speed` / `pitch` / `intonation` が属性として付与されます
- 疎通確認（`HealthCheck`）も行いません
- `watch` / `act --watch` ではディレクトリ監視・`done/` への移動／削除は通常通り実施されるため、監視フロー全体を検証できます

## 🧩 プラグイン／スキルリファレンス

claude code に `vox-actor-plugin` を導入すると、以下のスラッシュコマンドが利用できます。

### `/vox-actor-plugin:monologue`

作業開始／終了／想定外のことが起こった時など、節目のキャラクターの一言独り言を読み上げます。

```
/vox-actor-plugin:monologue [キャラクター名]
```

- 1文程度の短い独り言を生成し、キャラクターになりきって読み上げます
- キャラクター設定ファイルの `speakers` からセリフの感情に合うスピーカーIDを選定します
- 再生方式は `direct` / `file` モードで自動切替されます（[高度な利用](#-高度な利用リモート環境監視モード) を参照）

#### メモリ設定

| 項目 | キー | デフォルト値 | 説明 |
|------|------|-------------|------|
| 通知確率 | `monologue_probability` | `100` | 1〜100の整数。通知する確率（%） |

ユーザー指示（例: 「独り言の頻度を30%にして」）で更新すると、以降の実行に反映されます。

### `/vox-actor-plugin:explain <トピック>`

指定したトピックを冒頭→本題→まとめの流れで、複数セリフのJSONL台本としてキャラクターに解説させます。`monologue` が1文の独り言用であるのに対し、本スキルはまとまった長さの解説向けです。

```
/vox-actor-plugin:explain <トピック>
```

- `<トピック>`: 解説してほしい概念・質問・仕様文書などを自由記述で渡します
- 生成されたJSONL台本は一時ファイルに書き出され、`vox-actor act` で再生されます
- 再生方式は `direct` / `file` モードで自動切替されます（[高度な利用](#-高度な利用リモート環境監視モード) を参照）

#### メモリ設定

以下の設定を claude code のメモリに保存しておくと、次回以降の実行に反映されます。

| 項目 | キー | デフォルト値 | 値 | 説明 |
|------|------|-------------|----|-----|
| 説明キャラクター | `explanation_character` | `zundamon` | `characters/<name>.md` の `<name>` | 例: 「説明はめたんで」→ `metan` を保存 |
| 説明の長さ | `explanation_length` | `medium` | `short` / `medium` / `long` | 下記の長さ表を参照 |

##### 説明の長さ

| 設定 | セリフ数の目安 | 想定再生時間 |
|------|---------------|------------|
| `short` | 3〜5 | 〜十数秒 |
| `medium`（既定） | 6〜10 | 30秒〜1分 |
| `long` | 10+ | 数分 |

#### 呼び出し例

```
/vox-actor-plugin:explain クロージャとは何か
```

#### JSONL出力例

短めの説明（ずんだもん）:

```jsonl
{"text": "クロージャって何なのだ？説明するのだ！", "speaker": 3, "speedScale": 1.1}
{"text": "簡単に言うと、関数が作られた時の周りの変数を覚えておく仕組みなのだ", "speaker": 3, "speedScale": 1.0}
{"text": "むむっ、ちょっと難しいけど…例えるならお弁当箱に具材を詰めて持ち歩く感じなのだ", "speaker": 3, "speedScale": 1.0}
{"text": "後から開けても中身がそのまま残ってるみたいに、変数の値も残るのだ〜", "speaker": 1, "speedScale": 0.9}
{"text": "わかったかな？お疲れ様なのだ！", "speaker": 1, "speedScale": 1.0}
```

## 🔀 高度な利用（リモート環境・監視モード）

以下のようなケースでは、`watch` コマンドを常駐させる `file` モードでの利用が適しています。

- claude code がコンテナ／リモート環境で動作し、`vox-actor` コマンドはホスト側にしか存在しない
- 複数セッションから同時に呼ばれても音声を逐次再生したい（`direct` モードは並列再生になる）

### `direct` モードと `file` モードの違い

| 観点 | `direct` モード | `file` モード |
|---|---|---|
| 読み上げ方式 | `vox-actor say` をその場で直接呼び出す | テキスト等をファイルに書き出し、別プロセスの `vox-actor watch` が読み上げる |
| 監視プロセス | 不要 | `vox-actor watch` の常駐が必要 |
| ファイル出力先 | エラーログのみ（`$VOX_ACTOR_WORKSPACE/monologue-errors.log`） | 通知ファイル（`$VOX_ACTOR_WORKSPACE/queue/notify_*.json`）＋エラーログ |
| 同時呼び出し時 | 並列再生 | 検知順に逐次再生 |
| 前提 | `vox-actor` コマンドが `PATH` 上にある | claude code 側と `vox-actor watch` 側で `VOX_ACTOR_WORKSPACE` を共有 |

> `VOX_ACTOR_WORKSPACE` は vox-actor 関連ファイルのルートディレクトリを指す（配下に `queue/` と `monologue-errors.log` が置かれる）。未指定時のデフォルトは gitリポジトリ内なら `<gitリポジトリ直下>/.vox-actor`、gitリポジトリ外なら `$PWD/.vox-actor`。`file` モードでホストとLLM実行環境を分ける場合は、双方から参照可能な共有パスを明示的に指定する。

モードは `VOX_ACTOR_MONOLOGUE_MODE` 環境変数で明示するか、未設定時は `vox-actor` コマンドの有無で自動判定されます（あり → `direct`、なし → `file`）。

### `file` モードのセットアップ

1. **ホスト側で監視プロセスを常駐させる**
   ```bash
   # vox-actor 関連ファイルのルートディレクトリを指定
   export VOX_ACTOR_WORKSPACE=/path/to/shared/directory
   # 通知ファイルは $VOX_ACTOR_WORKSPACE/queue/ に配置されるためそこを監視する
   vox-actor watch "$VOX_ACTOR_WORKSPACE/queue"
   ```

2. **claude code 側で `file` モードを明示し、共有ディレクトリを指定する**
   ```bash
   export VOX_ACTOR_WORKSPACE=/path/to/shared/directory
   # vox-actorコマンドがある環境で file モードを強制したい場合は以下も指定
   export VOX_ACTOR_MONOLOGUE_MODE=file
   ```

3. **claude code にプラグインを導入する**（[クイックスタート](#-クイックスタート)と同じ手順）

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

`watch` コマンドは `Ctrl+C`（SIGINT）または SIGTERM で停止できる。

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
