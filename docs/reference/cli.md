# CLIリファレンス

オプションの優先順位: **オプション > 環境変数 > デフォルト値**

## テキストの直接読み上げ

```bash
vox-actor say "こんにちは"
```

キャラクターや音声パラメータを指定する場合:

```bash
vox-actor say --speaker 3 --speed 1.2 "こんにちは"
```

## テキストファイルの読み上げ

```bash
vox-actor act script.txt
```

話速・音高・抑揚を調整する場合:

```bash
vox-actor act --speed 1.2 --pitch 0.1 --intonation 1.5 script.txt
```

## JSON台本モード（感情制御パラメータ付き）

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

## JSONL台本モード（複数セリフを1ファイルにまとめる）

`.jsonl` ファイルを使うと、1行1JSONオブジェクトの形式で複数のセリフを1ファイルにまとめて記述できます。各行のスキーマは上記の JSON 台本と同一で、`text` のみ必須・その他のパラメータは省略可能です。

```jsonl
{"text": "こんにちは", "speaker": 3}
{"text": "また会いましょう", "speaker": 3, "speedScale": 1.2}
```

```bash
vox-actor act script.jsonl
```

## バージョン確認

```bash
vox-actor --version
```

## `say` サブコマンド

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
| `--dry-run` | — | `false` | VOICEVOX・音声再生を行わず、読み上げ対象をログ出力（[詳細](#dry-runモード)） |

## `script` サブコマンド

セリフファイルを操作するサブコマンド群。

### `script append` サブコマンド

```
vox-actor script append <file> <text>
```

指定したセリフファイルにテキストを追記する。VOICEVOX接続・音声再生は行わない。`vox-actor watch --queue` と組み合わせ、別プロセスで再生させたいユースケースに利用できる。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--speaker` | `VOX_SPEAKER` | `3` | キャラクターID |
| `--speed` | — | `1.0` | 話速 |
| `--pitch` | — | `0.0` | 音高 |
| `--intonation` | — | `1.0` | 抑揚 |
| `--verbose` | — | `false` | 詳細ログを出力 |

- 出力先ファイルの拡張子で書き出し形式を自動判定します。
  - `.json` → 1ファイル1スクリプトのJSON。`text` に加え、明示指定された `--speaker` / `--speed` / `--pitch` / `--intonation` を `speaker` / `speedScale` / `pitchScale` / `intonationScale` として保存
  - `.jsonl` → 1行JSON（フィールドは `.json` と同じ）
  - `.txt` および **未知の拡張子** → 本文のみのテキストファイル（拡張子はそのまま）
    - このとき `--speaker` / `--speed` / `--pitch` / `--intonation` がコマンドラインで明示指定されていたら、これらは保存できない旨を WARN ログで通知します（処理は継続）
- **既存ファイルとの衝突回避**: 出力先に同名ファイルが存在する場合は、`<name>_<UnixNano><ext>` の形式で連番を付与してユニーク化します。
- **親ディレクトリの扱い**: 親ディレクトリが存在しない場合はエラーになります（任意の深さを自動作成しません）。

```bash
# ホスト側 / 別ターミナル: queue を監視
vox-actor watch --queue

# claude code 等から: queue にテキストを書き出すと watch が拾って読み上げる
vox-actor script append "$(vox-actor config path.queue)/01.txt" "こんにちは"

# パラメータを保持したい場合は .json / .jsonl で書き出す
vox-actor script append "$(vox-actor config path.queue)/01.json" --speaker 5 --speed 1.2 "こんにちは"
```

## `act` サブコマンド

```
vox-actor act <path>
```

テキストファイル／JSON台本／JSONL台本／ディレクトリを読み上げる。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--engine-url` | `VOX_ENGINE_URL` | `http://localhost:50021` | VOICEVOXエンジンのURL |
| `--speaker` | `VOX_SPEAKER` | `3` | キャラクターID |
| `--speed` | — | `1.0` | 話速 |
| `--pitch` | — | `0.0` | 音高 |
| `--intonation` | — | `1.0` | 抑揚 |
| `--watch` | — | `false` | ディレクトリ監視モードを有効化（後方互換。[plugins.md の該当節](./plugins.md#act---watch--act---watch-delete後方互換)参照） |
| `--watch-delete` | — | `false` | ディレクトリ監視モード（処理済みファイルを削除。後方互換） |
| `--verbose` | — | `false` | 詳細ログを出力 |
| `--dry-run` | — | `false` | VOICEVOX・音声再生を行わず、読み上げ対象をログ出力（[詳細](#dry-runモード)） |

## `watch` サブコマンド

```
vox-actor watch <dir1> [<dir2> ...]
vox-actor watch --queue
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
| `--queue` | — | `false` | `vox-actor config path.queue` で解決される queue ディレクトリを監視対象に自動選択（[詳細](#queueオプション)） |
| `--verbose` | — | `false` | 詳細ログを出力 |
| `--dry-run` | — | `false` | VOICEVOX・音声再生を行わず、読み上げ対象をログ出力（[詳細](#dry-runモード)） |

### `--queue`オプション

`vox-actor watch --queue` を指定すると、`vox-actor config path.queue` と同じロジックで解決される queue ディレクトリを監視対象として自動選択します。

- 位置引数（ディレクトリパス）との**併用はエラー**になります。
- `VOX_ACTOR_WORKSPACE` 未設定かつカレントディレクトリがgitリポジトリ外、または `git` コマンドがPATH上に無い場合は起動時にエラー終了します。
- queue ディレクトリが存在しない場合は起動時に自動作成されます（`os.MkdirAll(..., 0o755)`）。
- worktree 上で実行した場合でも、`git rev-parse --git-common-dir` で本体リポジトリの `.git` を参照するため、**本体リポジトリ直下**の `.vox-actor/queue` が選ばれます。
- `--delete` / `--dry-run` などの既存オプションとは自由に併用できます。

```bash
# config path.queue で解決される queue ディレクトリを監視（done/ 移動モード）
vox-actor watch --queue

# 削除モードで監視
vox-actor watch --queue --delete
```

## `viewer` サブコマンド

```
vox-actor viewer [--watch <dir>]... [--watch-queue] [--delete] \
                 [--host <host>] [--port <port>] \
                 [--engine-url <url>] [--speaker <id>] \
                 [--speed N] [--pitch N] [--intonation N] [--verbose]
```

HTTPサーバーとブラウザUIを起動し、SSE経由でブラウザに音声を配信する。ブラウザ配信専用のサブコマンドで、`--watch` や `--watch-queue` でディレクトリを監視しながら配信できる。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--watch <dir>` | — | （未指定可・複数可） | 監視対象ディレクトリ（`--watch dir1 --watch dir2` のように複数回指定） |
| `--watch-queue` | — | `false` | `vox-actor config path.queue` で解決される queue ディレクトリを監視対象に追加 |
| `--delete` | — | `false` | 処理済みファイルを削除（未指定時は各ディレクトリの `done/` に移動） |
| `--host` | — | `127.0.0.1` | HTTPサーバーのバインドホスト |
| `--port` | — | `8080` | HTTPサーバーのバインドポート（1〜65535） |
| `--engine-url` | `VOX_ENGINE_URL` | `http://localhost:50021` | VOICEVOXエンジンのURL |
| `--speaker` | `VOX_SPEAKER` | `3` | 既定話者ID |
| `--speed` | — | `1.0` | 話速 |
| `--pitch` | — | `0.0` | 音高 |
| `--intonation` | — | `1.0` | 抑揚 |
| `--verbose` | — | `false` | 詳細ログを出力 |

### 起動パターン

```bash
# 監視なし（HTTP+UIのみ）。「音声テスト」タブだけが機能する
vox-actor viewer

# ディレクトリを監視しながら配信
vox-actor viewer --watch /path/to/dir

# 複数ディレクトリを並列監視
vox-actor viewer --watch /path/to/dir1 --watch /path/to/dir2

# queue ディレクトリを監視
vox-actor viewer --watch-queue

# --watch と --watch-queue の併用
vox-actor viewer --watch /extra/dir --watch-queue

# バインドアドレスを変更（LAN公開）
vox-actor viewer --host 0.0.0.0 --port 8080
```

### エラー条件

| 状況 | 終了コード | エラー出力 |
|---|---|---|
| `--watch` のパスがディレクトリ以外 | 2 (`ErrUsage`) | `Error: <path> is not a directory` |
| `--watch-queue` 指定時に git管理外かつ `VOX_ACTOR_WORKSPACE` 未設定 | 2 (`ErrUsage`) | `Error: gitコマンドが見つかりません` 等 |
| `--port` が 1〜65535 範囲外 | 2 (`ErrUsage`) | `Error: invalid port: <n>` |
| HTTPサーバー起動失敗（ポート占有等） | 1 | `Error: failed to start stream server: ...` |

ブラウザUI・SSE・`/api/status`・無音モードの挙動は[ストリーム配信モード](#ストリーム配信モード)と同一です。

## `config` サブコマンド

```
vox-actor config <key>
```

指定したキーの設定値を取得して標準出力に返す、読み取り専用のサブコマンド。`git config` 風のインターフェースで、初期リリースではパス解決のみをサポートする。

**サポートキー:**

| キー | 返却値 |
|---|---|
| `path.workspace` | vox-actor のワークスペースルート絶対パス |
| `path.queue` | ワークスペースルート配下の `queue` ディレクトリ絶対パス |
| `path.tmp` | ワークスペースルート配下の `tmp` ディレクトリ絶対パス |

**解決順（全キー共通）:**

1. 環境変数 `VOX_ACTOR_WORKSPACE` が設定されていればその値をワークスペースルートとして扱う
2. gitリポジトリ内であれば `git rev-parse --path-format=absolute --git-common-dir` の結果の親ディレクトリ配下の `.vox-actor` をワークスペースルートとする
3. git管理外の場合はカレントディレクトリをワークスペースルートとして扱う

`path.queue` と `path.tmp` はワークスペースルートに各サブディレクトリを結合した値を返すため、常に以下の関係が成立します:
- `path.workspace`/queue = `path.queue`
- `path.workspace`/tmp = `path.tmp`

**出力:**

- 成功時: 標準出力に絶対パスを1行出力（末尾LFあり）
- 失敗時: 標準エラーに日本語メッセージを出力

**エラー条件:**

| 状況 | 終了コード | エラー出力 |
|---|---|---|
| 成功 | 0 | — |
| 未知のキー | 2 (`ErrUsage`) | `unknown key: <key>` とサポートキー一覧 |
| `git` コマンドがPATH上に無い | 1 | `Error: gitコマンドが見つかりません` |
| カレント ディレクトリ取得失敗（通常発生しない） | 1 | `Error: カレントディレクトリが取得できません` |

**使用例:**

```bash
# gitリポジトリ内で実行
$ cd /path/to/repo && vox-actor config path.workspace
/path/to/repo/.vox-actor
$ cd /path/to/repo && vox-actor config path.queue
/path/to/repo/.vox-actor/queue
$ cd /path/to/repo && vox-actor config path.tmp
/path/to/repo/.vox-actor/tmp

# git管理外で実行（カレントディレクトリにフォールバック）
$ cd /tmp/non-git && vox-actor config path.workspace
/tmp/non-git
$ cd /tmp/non-git && vox-actor config path.queue
/tmp/non-git/queue
$ cd /tmp/non-git && vox-actor config path.tmp
/tmp/non-git/tmp

# VOX_ACTOR_WORKSPACE 明示
$ VOX_ACTOR_WORKSPACE=/custom vox-actor config path.workspace
/custom
$ VOX_ACTOR_WORKSPACE=/custom vox-actor config path.queue
/custom/queue
$ VOX_ACTOR_WORKSPACE=/custom vox-actor config path.tmp
/custom/tmp

# 未知のキー
$ vox-actor config path.unknown
Error: unknown key: path.unknown
supported keys:
  path.queue
  path.tmp
  path.workspace
```

`config` サブコマンド自体は**パスの返却のみ**を行い、ディレクトリ作成は行いません。worktree 上で実行した場合も本体リポジトリ直下のパスが返るため、外部スクリプトから共通のワークスペースを参照できます。

## `audio-check` サブコマンド

```
vox-actor audio-check [-v|--verbose]
```

音声出力デバイスを **open → 即 close** のみ行い、実再生を伴わずに可用性を**終了コード**で返す診断用サブコマンド。シェルスクリプトから `VOX_ACTOR_MONOLOGUE_MODE` 未指定時のモード判定（direct / file）に利用できる。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--verbose` / `-v` | — | `false` | 診断メッセージを標準エラー出力に出力する |

**出力と終了コード:**

| 状況 | 終了コード | stdout | stderr (通常) | stderr (`--verbose`) |
|---|---|---|---|---|
| デバイス open 成功 | `0` | 空 | 空 | `audio backend: <name>` |
| デバイス open 失敗 | `1` | 空 | 空 | `Error: <エラー内容>` |
| バックエンド未対応 OS | `1` | 空 | 空 | `Error: unsupported platform: <GOOS>` |

- `stdout` には一切出力しない。機械可読な判定は**終了コード**で行う。
- 失敗理由の細分化は `--verbose` 指定時の stderr 出力で確認する。
- 呼び出しはミリ秒オーダーで完了する想定（内部に2秒のタイムアウトを持つ）。
- 本サブコマンドは**ローカルの出力デバイスのみ**を対象とする。VOICEVOX エンジンの疎通確認は行わない。

**バックエンド名:**

| GOOS | backend |
|---|---|
| `darwin` | `coreaudio` |
| `linux` | `pulseaudio` |
| `windows` | `wasapi` |

**使用例:**

```bash
# 直接実行（成功）
$ vox-actor audio-check
$ echo $?
0

# verbose 出力
$ vox-actor audio-check -v
audio backend: coreaudio
$ echo $?
0

# 音声デバイスが使えない環境（例: CI コンテナ）
$ vox-actor audio-check -v
Error: failed to initialize audio device: no output device available
$ echo $?
1
```

**シェルスクリプトからのモード判定例:**

```bash
MODE="${VOX_ACTOR_MONOLOGUE_MODE:-}"
if [ -z "$MODE" ]; then
  if vox-actor audio-check >/dev/null 2>&1; then
    MODE="direct"
  else
    MODE="file"
  fi
fi
```

## ストリーム配信モード

HTTPサーバーを起動し、SSE経由でブラウザに音声を配信します。音声デバイスが利用できない環境でホスト側のブラウザに再生させるケースで利用できます。

[`viewer` サブコマンド](#viewer-サブコマンド)を使ってください。

```bash
# viewer
vox-actor viewer --watch /path/to/watch-dir

# viewer でバインドアドレスを変更
vox-actor viewer --host 0.0.0.0 --port 8080 --watch /path/to/watch-dir
```

> **注意**: `watch --stream` および `watch --stream-addr` は削除されました。
> `vox-actor viewer` を使ってください。

- `http://<host>:<port>/` をブラウザで開き、音量スライダーを操作すると再生が解禁されます（ブラウザの自動再生ポリシー対策）。
- 画面上部のタブで「配信」「音声テスト」を切り替えます。タブ切替時は再生中の音声が即停止し、再生キューも破棄されます。選択中のタブは localStorage に保存されてリロード後も復元されます（初期タブは「配信」）。
- 「配信」タブはチャット風のタイムラインで、古いセリフが上、新しいセリフが下に並びます。再生中のセリフは背景色と `▶` アイコンで強調され、再生開始時に自動でスクロールして画面内に表示されます。
- 各タイムライン項目はメタ行（時刻 / 話者名 / `[スタイル名]`）と本文の 2 段構成です。`☑話者名` `☑スタイル` `☑時刻` のチェックボックスで個別に表示/非表示を切り替えでき、状態は localStorage に保存されてリロード後も復元されます。
- 「音声テスト」タブでは話者を選択して `▶ テスト再生` ボタンで合成済みフレーズ（既定: 「音量テストです」）を試聴できます。選択した話者は localStorage に保存されます。音声テストタブ中に SSE で届いた配信 clip はタイムラインには追加されますが再生はされず、配信タブに戻ると以降の clip から通常再生に戻ります。
- 音量スライダーと消音チェックは両タブ共通で、消音 ON のときはテスト再生も鳴りません。
- h1 の右側には SSE 接続状態のバッジが表示され、接続中は緑の `● 接続中`、切断時は赤の `● 切断` に切り替わります。
- 話者名・スタイル名は watch 起動時にエンジンの `/speakers` から取得した一覧で解決されます。エンジン未登録の SpeakerID は `話者#<ID>` というフォールバック表示になります。
- ブラウザ側のタイムライン表示上限は画面上部のプルダウン（10 / 20 / 30 / 50 / 100 / 200、初期値20）で変更でき、選択値は localStorage に保存されてリロード後も復元されます。
- サーバー側のWAVリングバッファは固定容量（20件）で、上限を超えた古いものから破棄されます。
- テスト再生用の WAV は話者ごとに初回合成時のみ VOICEVOX へ問い合わせ、以降は watch プロセス終了までキャッシュから返されます。
- 複数ブラウザで同時に開いた場合、全クライアントに同じクリップが配信されます。
- デフォルトでは `127.0.0.1` にバインドするため外部からはアクセスできません。LAN公開したい場合は `--host 0.0.0.0` を指定してください。

### 無音モード（VOICEVOX 未起動時の自動フォールバック）

`viewer` 起動時に VOICEVOX エンジンへの `HealthCheck` または `/speakers` 取得が失敗した場合、エラー終了せず**無音モード**で起動を継続します。VOICEVOX を立ち上げずにテキストだけをブラウザで読みたいときに有効です。

- 判定は **起動時のみ** 行われます。起動後にエンジンが切断されても無音モードへは切り替わりません（リカバリ対象外）。
- フォールバック発動時、標準エラー出力に WARN ログ `VOICEVOX engine unreachable, continuing in silent mode (no audio will be played)` が1回出力されます。`error` 属性で `HealthCheck` 失敗か `GetSpeakers` 失敗かを識別できます。
- 無音モード時は通常モードと比べて次のような挙動になります:
  - WAV の合成・配信は行われず、`clipEvent.url` は空文字で配信されます。`/clips/{id}.wav` へのリクエストは 404 を返します。
  - セリフの配信間隔は WAV 長ではなく固定 0.5 秒になります（タイムラインが一瞬で流れ去らないための暫定値）。
  - 話者名は全 SpeakerID について `話者#<ID>` にフォールバックします（スタイル名は空文字）。
  - 「音声テスト」タブは画面上に表示されません。
  - ヘッダーの SSE 接続バッジの右隣に `🔇 無音モード` バッジが表示され、ホバーまたはタップで無音モードに入った理由と対処法（VOICEVOX を起動した状態で `viewer` を起動し直すなど）をツールチップで確認できます。
- サーバーの現在の状態は `GET /api/status` で取得できます。無音フラグ・理由文面・話者一覧をまとめて返す単一エンドポイントで、既存の `/speakers.json` は廃止されています。
- ディレクトリ監視・`done/` への移動／削除は無音モードでも通常通り実施されます。
- `watch` / `say` / `act` では従来通り `HealthCheck` 失敗時に即エラー終了します（本フォールバックは `viewer` のみ）。

## dry-runモード

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
