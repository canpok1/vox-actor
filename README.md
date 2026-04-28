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

手元で順に試して動作確認しながら、CLI単体〜claude code プラグインまで一通り体験できる手順です。

1. **CLIのセットアップ（Homebrew）**

   ```bash
   brew tap canpok1/tap
   brew install --cask vox-actor
   ```

   Homebrew以外のインストール方法は[インストール方法](#-インストール方法)を参照してください。

2. **CLIの実行で疎通確認**

   ```bash
   vox-actor say "こんにちは"
   ```

   VOICEVOXエンジンが起動していれば音声が再生されます。

3. **claude code プラグインのインストール**

   ```
   # claude code上で実行
   /plugin marketplace add canpok1/vox-actor
   /plugin install vox-actor-plugin@vox-actor-marketplace
   # 自動で独り言を生成したい場合は auto-monologue-plugin も導入
   /plugin install auto-monologue-plugin@vox-actor-marketplace
   ```

4. **スキルの実行**

   ```
   # 指示に沿うセリフを生成して再生
   /vox-actor-plugin:talk クロージャとは何か
   ```

   `auto-monologue-plugin` を導入していれば、claude code の作業の区切りごとに自動で独り言が生成・読み上げされます。

## 🎯 できること

vox-actor は **`vox-actor` CLI** と **claude code プラグイン／スキル** の2系統を提供しています。プラグイン／スキルも内部では `vox-actor` CLI を利用するため、CLIはどのパターンでも必須です。

### 利用パターン

| 利用パターン | 使うもの | 代表ユースケース |
|---|---|---|
| CLI単体 | CLI | テキスト即時読み上げ／台本ファイル再生／ディレクトリ監視 |
| claude code経由での利用（音声デバイス利用可能環境） | CLI ＋ プラグイン／スキル | 作業区切りの独り言／解説・結果報告／複数キャラの会話読み上げ |
| claude code経由での利用（音声デバイス利用不可環境） | ホスト: CLI（`watch`） ＋ コンテナ: プラグイン／スキル | 上記をコンテナ／リモート上の claude code から利用 |

### 各利用パターンの使い方

#### CLI単体

`vox-actor` コマンドを直接呼び出してテキストや台本ファイルを読み上げます。

```bash
# 直接テキストを読み上げ
vox-actor say "こんにちは"

# テキスト／JSON／JSONL 台本ファイルを読み上げ
vox-actor act script.jsonl

# ディレクトリを並列監視し、配置されたファイルを検知順に読み上げ
vox-actor watch /path/to/watch-dir
```

詳細は [CLIリファレンス](./docs/reference/cli.md) を参照してください。

#### claude code経由での利用（音声デバイス利用可能環境）

claude code と `vox-actor` CLI が同じ環境で動作するケースです。プラグインをインストールするだけで、スキルから CLI が直接呼び出されます。

```
/vox-actor-plugin:talk <内容>                 # 指示に沿うセリフを生成して再生（独り言・解説・会話など）
```

セットアップ手順は上記 [クイックスタート](#-クイックスタート) を参照してください。スキルの詳細は [プラグイン／スキルリファレンス](./docs/reference/plugins.md) を参照してください。

#### claude code経由での利用（音声デバイス利用不可環境）

claude code がコンテナ／リモートで動作し、音声デバイスはホスト側のみにある構成です。ホスト側で `vox-actor watch` を常駐させ、claude code側は共有ディレクトリへ通知ファイルを書き出すことで、読み上げが逐次再生されます。

```bash
# ホスト側: 共有ディレクトリを監視（VOX_ACTOR_WORKSPACE は vox-actor CLI が解釈する）
export VOX_ACTOR_WORKSPACE=/path/to/shared/directory
vox-actor watch "$(vox-actor config path.queue)"
```

```bash
# claude code 側: 同じ共有ディレクトリを指定
export VOX_ACTOR_WORKSPACE=/path/to/shared/directory
export VOX_ACTOR_MONOLOGUE_MODE=file
```

詳細なセットアップ手順は [プラグイン／スキルリファレンス](./docs/reference/plugins.md) を参照してください。

ホスト側にも音声デバイスがない場合は `vox-actor viewer` でブラウザに音声を配信する構成も利用できます。詳細は [CLIリファレンスのストリーム配信モード](./docs/reference/cli.md#ストリーム配信モード) を参照してください。

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

## 📚 ドキュメント

- [CLIリファレンス](./docs/reference/cli.md) — `say` / `script` / `act` / `watch` サブコマンドの詳細
- [プラグイン／スキルリファレンス](./docs/reference/plugins.md) — 対応キャラクター、スキルごとの仕様、再生モード、音声デバイス利用不可環境のセットアップ
- [開発者向け情報](./docs/development/contributing.md) — ビルド・テスト・Lint等のコマンド、レイヤー構成と責務ルール

## ライセンス

[MIT License](LICENSE)

キャラクター設定は `vox-actor assets download` で別リポジトリから取得します。利用規約とクレジット表記は取得元リポジトリの案内に従ってください。

`--scope` フラグで配置先を切り替えられます。

| `--scope` | 配置先 |
|-----------|--------|
| 省略（既定） | `<repoRoot>/.vox-actor/assets/`（git リポジトリ外は `<cwd>/.vox-actor/assets/`） |
| `project` | 同上 |
| `home` | `~/.vox-actor/assets/`（複数プロジェクト共用に便利） |

`speakers list` / `speakers profile` / viewer 画面はプロジェクト・ホーム両方の assets をマージして返します（同一 ID はプロジェクト優先）。
