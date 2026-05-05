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
  "speed": 1.2,
  "pitch": 0.1,
  "intonation": 1.5
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
{"text": "また会いましょう", "speaker": 3, "speed": 1.2}
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
| `--viewer-url` | `VOX_VIEWER_URL` | (未指定) | viewer の HTTP エンドポイント URL (例: `http://192.168.1.10:8080`)。指定時は lockfile auto-detect をスキップして明示 URL の viewer に POST `/api/play` する。接続失敗時はローカル再生にフォールバックせずエラー終了する。 |
| `--speaker` | `VOX_SPEAKER` | `3` | キャラクターID |
| `--speed` | — | `1.0` | 話速 |
| `--pitch` | — | `0.0` | 音高 |
| `--intonation` | — | `1.0` | 抑揚 |
| `--verbose` | — | `false` | 詳細ログを出力 |
| `--dry-run` | — | `false` | VOICEVOX・音声再生を行わず、読み上げ対象をログ出力（[詳細](#dry-runモード)） |
| `--save-wav` | `VOX_SAVE_WAV` | (未指定) | 合成した WAV をファイルに保存するパス。ローカル合成時のみ有効（viewer 送信経路では保存しない）。親ディレクトリが存在しない場合は自動作成。既存ファイルは上書き。`--dry-run` 時は保存しない。 |

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
  - `.json` → 1ファイル1スクリプトのJSON。`text` に加え、明示指定された `--speaker` / `--speed` / `--pitch` / `--intonation` を `speaker` / `speed` / `pitch` / `intonation` として保存
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

### `script write` サブコマンド

```
vox-actor script write <file> --json '<JSON配列>'
```

JSON 配列で渡した複数のセリフを指定ファイルへ一括書き込みする（上書き）。VOICEVOX接続・音声再生は行わない。`talk` スキルなど複数セリフを 1 回で書き出したいユースケースに利用できる。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--json` | — | （必須） | セリフの JSON 配列 |
| `--verbose` | — | `false` | 詳細ログを出力 |

`--json` の各オブジェクトのキー:

| キー | 型 | 必須 | 説明 |
|---|---|---|---|
| `text` | string | ✓ | セリフ本文 |
| `speaker` | int | | キャラクターID |
| `speed` | float | | 話速 |
| `pitch` | float | | 音高 |
| `intonation` | float | | 抑揚 |

- 出力先ファイルを**新規作成または上書き**します（`append` と異なり既存内容は削除されます）。
- 省略したキーは出力 JSONL に含まれません（既存の JSONL フォーマットと同一）。
- `--json` の JSON パースエラー時は終了コード 2 で失敗します。

```bash
vox-actor script write sample.jsonl --json '[
  {"text":"おはようなのだ","speaker":3,"intonation":1.1},
  {"text":"今日もがんばるのだ","speaker":3,"speed":1.0}
]'
```

出力（`sample.jsonl`）:

```jsonl
{"text":"おはようなのだ","speaker":3,"intonation":1.1}
{"text":"今日もがんばるのだ","speaker":3,"speed":1.0}
```

## `act` サブコマンド

```
vox-actor act <path>
```

テキストファイル／JSON台本／JSONL台本／ディレクトリを読み上げる。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--engine-url` | `VOX_ENGINE_URL` | `http://localhost:50021` | VOICEVOXエンジンのURL |
| `--viewer-url` | `VOX_VIEWER_URL` | (未指定) | viewer の HTTP エンドポイント URL (例: `http://192.168.1.10:8080`)。指定時は lockfile auto-detect をスキップして明示 URL の viewer に POST `/api/play` する。接続失敗時はローカル再生にフォールバックせずエラー終了する。 |
| `--speaker` | `VOX_SPEAKER` | `3` | キャラクターID |
| `--speed` | — | `1.0` | 話速 |
| `--pitch` | — | `0.0` | 音高 |
| `--intonation` | — | `1.0` | 抑揚 |
| `--verbose` | — | `false` | 詳細ログを出力 |
| `--dry-run` | — | `false` | VOICEVOX・音声再生を行わず、読み上げ対象をログ出力（[詳細](#dry-runモード)） |

> **注意**: `act --watch` および `act --watch-delete` は削除されました。ディレクトリ監視は [`watch` サブコマンド](#watch-サブコマンド)を使用してください。
>
> | 旧コマンド | 新コマンド |
> |---|---|
> | `act --watch <dir>` | `watch <dir>` |
> | `act --watch-delete <dir>` | `watch --delete <dir>` |

## `watch` サブコマンド

```
vox-actor watch <dir1> [<dir2> ...]
vox-actor watch --queue
```

1つ以上のディレクトリを並列監視し、配置されたファイルを検知順に逐次再生する。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--engine-url` | `VOX_ENGINE_URL` | `http://localhost:50021` | VOICEVOXエンジンのURL |
| `--viewer-url` | `VOX_VIEWER_URL` | (未指定) | viewer の HTTP エンドポイント URL (例: `http://192.168.1.10:8080`)。指定時は lockfile auto-detect をスキップして明示 URL の viewer に POST `/api/play` する。 |
| `--speaker` | `VOX_SPEAKER` | `3` | キャラクターID |
| `--speed` | — | `1.0` | 話速 |
| `--pitch` | — | `0.0` | 音高 |
| `--intonation` | — | `1.0` | 抑揚 |
| `--delete` | — | `false` | 処理済みファイルを削除（未指定時は各ディレクトリの `done/` に移動） |
| `--queue` | — | `false` | `vox-actor config path.queue` で解決される queue ディレクトリを監視対象に自動選択（[詳細](#queueオプション)） |
| `--save-wav-dir` | `VOX_SAVE_WAV_DIR` | (未指定) | 合成した WAV を保存するディレクトリ。ファイル名は `<UnixMs>_<text先頭20文字>.wav`（ローカル合成時のみ保存。未作成なら自動作成。`--dry-run` および viewer 送信時は保存しない） |
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

> **破壊的変更**: `--watch` / `--watch-queue` / `--delete` は削除されました。ディレクトリ監視は [`watch` サブコマンド](#watch-サブコマンド)を使用してください。
>
> | 旧コマンド | 新コマンド |
> |---|---|
> | `viewer --watch <dir>` | `viewer &` + `watch <dir>` |
> | `viewer --watch-queue` | `viewer &` + `watch --queue` |
> | `viewer --watch <dir> --delete` | `viewer &` + `watch --delete <dir>` |

```
vox-actor viewer [--host <host>] [--port <port>] \
                 [--engine-url <url>] [--speaker <id>] \
                 [--speed N] [--pitch N] [--intonation N] [--verbose] \
                 [--save-wav-dir <dir>]
```

HTTPサーバーとブラウザUIを起動し、SSE経由でブラウザに音声を配信する。ディレクトリ監視が必要な場合は `vox-actor watch` を別途起動してください。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--host` | — | `127.0.0.1` | HTTPサーバーのバインドホスト |
| `--port` | — | `8080` | HTTPサーバーのバインドポート（1〜65535） |
| `--engine-url` | `VOX_ENGINE_URL` | `http://localhost:50021` | VOICEVOXエンジンのURL |
| `--speaker` | `VOX_SPEAKER` | `3` | 既定話者ID |
| `--speed` | — | `1.0` | 話速 |
| `--pitch` | — | `0.0` | 音高 |
| `--intonation` | — | `1.0` | 抑揚 |
| `--verbose` | — | `false` | 詳細ログを出力 |
| `--save-wav-dir` | `VOX_SAVE_WAV_DIR` | (未指定) | 合成した WAV を保存するディレクトリ。ファイル名は `<UnixMs>_<text先頭20文字>.wav`。保存先が存在しない場合は自動作成。 |

### 起動パターン

```bash
# HTTP+UI のみ起動
vox-actor viewer

# バインドアドレスを変更（LAN公開）
vox-actor viewer --host 0.0.0.0 --port 8080

# ディレクトリ監視と並行運用（viewer と watch を別プロセスで起動）
vox-actor viewer &
vox-actor watch /path/to/dir

# queue ディレクトリ監視と並行運用
vox-actor viewer &
vox-actor watch --queue
```

### ロックファイルと再生履歴

viewer は端末内で `127.0.0.1:8080` にバインドする前提のため、排他粒度は**ユーザー（端末）スコープ**です。

| 項目 | パス | 説明 |
|---|---|---|
| ロックファイル | `~/.vox-actor/viewer/viewer.lock` | 起動中に保持。同一ユーザーで 2 つ目の viewer 起動はエラーになる |
| 再生履歴 | `~/.vox-actor/viewer/history/YYYY-MM-DD.jsonl` | `GET /api/history` で返す履歴の読み書き先 |

- `VOX_ACTOR_WORKSPACE` はロック / 履歴の出力先に影響しません。
- **破壊的変更**: `v0.x` 以前の `<repoRoot>/.vox-actor/viewer/history/` 配下のファイルは読み込まれません。必要な場合は `~/.vox-actor/viewer/history/` へ手動でコピーしてください。

### エラー条件

| 状況 | 終了コード | エラー出力 |
|---|---|---|
| `--port` が 1〜65535 範囲外 | 2 (`ErrUsage`) | `Error: invalid port: <n>` |
| HTTPサーバー起動失敗（ポート占有等） | 1 | `Error: failed to start stream server: ...` |
| 同一ユーザーで viewer が既に起動中 | 1 | `Error: viewer は既に起動中です...` |

### HTTP API

viewer が起動する HTTP サーバーが提供するエンドポイントの仕様一覧です。

#### POST /api/play

再生リクエストを受け付ける非同期エンドポイントです。リクエスト受付後すぐに 200 を返し、音声合成・配信はバックグラウンド worker が処理します。

**リクエスト**

```json
{
  "clips": [
    {
      "text": "読み上げテキスト",
      "speaker_id": 3,
      "speed": 1.0,
      "pitch": 0.0,
      "intonation": 1.0
    }
  ]
}
```

`speed` / `pitch` / `intonation` は省略可能です。省略時は viewer 起動時のデフォルト値が使用されます。

**レスポンス**

| ステータスコード | 条件 |
|---|---|
| 200 | 正常受付（無音モード時も 200） |
| 400 | リクエストボディが不正な JSON |
| 405 | GET など POST 以外のメソッド |

200 レスポンスの JSON:

```json
{
  "playback_id": "1746316414387",
  "clip_count": 1,
  "silent": false,
  "silent_reason": ""
}
```

- `silent` が `true` の場合、音声合成は行われず SSE `clip` イベントも配信されません。`silent_reason` に理由文面が入ります。
- VOICEVOX 合成エラーは非同期 worker が処理するため、合成失敗時もレスポンスは 200 です。

#### GET /api/status

サーバーの現在状態（無音フラグ・理由文面・話者一覧）を返します。レスポンスは起動時に一度だけ生成されキャッシュされます。

**レスポンス**

| ステータスコード | 条件 |
|---|---|
| 200 | 常時 |

```json
{
  "silent": false,
  "silentReason": "",
  "speakers": [
    { "id": 3, "speakerName": "ずんだもん", "styleName": "ノーマル" }
  ]
}
```

無音モード時は `silent=true`・`silentReason` に理由文面・`speakers=[]` になります。

> **廃止**: `/speakers.json` は廃止されました。`GET /api/status` を使用してください。

#### GET /api/history

再生履歴（`~/.vox-actor/viewer/history/YYYY-MM-DD.jsonl`）を返します。

**レスポンス**

| ステータスコード | 条件 |
|---|---|
| 200 | 常時（履歴がない場合は空配列） |

```json
{
  "entries": [
    {
      "text": "読み上げテキスト",
      "speakerName": "ずんだもん",
      "styleName": "ノーマル",
      "timestamp": 1746316414387
    }
  ]
}
```

`timestamp` は配信時刻の Unix ms（UTC）です。WAV URL は viewer 再起動で失効するため履歴には含まれません。

#### GET /api/playback/{id}

`POST /api/play` が返した `playback_id` を指定して、再生の完了状態をポーリングするエンドポイントです。`vox-actor playback wait` コマンドが内部で使用します。

**レスポンス**

| ステータスコード | 条件 |
|---|---|
| 200 | 常時 |

```json
{
  "id": "1746316414387",
  "status": "completed",
  "clip_count": 1,
  "completed_clips": 1,
  "started_at": 1746316414387,
  "finished_at": 1746316415000,
  "failed_reason": ""
}
```

`status` は `pending` / `playing` / `completed` / `failed` のいずれかです。

#### GET /api/characters

キャラクター設定（`speaker.json` + 画像）の一覧を返します。レスポンスは起動時に一度だけ生成されキャッシュされます。

**レスポンス**

| ステータスコード | 条件 |
|---|---|
| 200 | 常時 |
| Content-Type | `application/json; charset=utf-8` |

```json
{
  "enabled": true,
  "characters": [
    {
      "speakerName": "ずんだもん",
      "styleName": "ノーマル",
      "mouthClosed": "/assets/images/zundamon/mouth_closed.png",
      "mouthOpen": "/assets/images/zundamon/mouth_open.png"
    }
  ]
}
```

- `enabled` はキャラクター設定が1件以上ある場合のみ `true` になります。assets ディレクトリが存在しない場合は `enabled=false` で `characters=[]` を返します。
- 複数スタイルを持つキャラクターはスタイルごとにエントリが展開されます。
- 不正な `speaker.json` を持つキャラクターはスキップされ、有効なキャラクターのみ返します。
- キャラクター設定の優先順位はプロジェクト assets（`VOX_ACTOR_WORKSPACE/assets/`）> HOME assets（`~/.vox-actor/assets/`）です。

#### GET /test-clip

指定した話者 ID でテスト用 WAV を合成して返します。初回合成結果は viewer 終了まで話者ごとにキャッシュされ、2 回目以降は同一バイト列を返します（冪等）。

**クエリパラメーター**

| パラメーター | 必須 | 説明 |
|---|---|---|
| `speaker` | 必須 | 話者 ID（整数） |

**レスポンス**

| ステータスコード | 条件 |
|---|---|
| 200 | 正常（WAV バイナリ） |
| 400 | 無音モード時（話者情報が空のため `speaker not found`） |

正常時のヘッダー:

| ヘッダー | 値 |
|---|---|
| `Content-Type` | `audio/wav` |
| `Content-Length` | WAV のバイト数 |

デフォルトの合成フレーズは「音量テストです」です。

#### GET /clips/{id}.wav

`POST /api/play` の結果として SSE `clip` イベントで通知される URL から WAV を取得するエンドポイントです。

**パスパラメーター**

| パラメーター | 説明 |
|---|---|
| `id` | Unix ミリ秒タイムスタンプ（SSE `clip` イベントの `url` フィールドから取得） |

**レスポンス**

| ステータスコード | 条件 |
|---|---|
| 200 | WAV が存在する |
| 404 | ID がタイムスタンプ形式でない / 対応 WAV が存在しない / 無音モード |

正常時のヘッダー:

| ヘッダー | 値 |
|---|---|
| `Content-Type` | `audio/wav` |
| `Cache-Control` | `public, max-age=31536000, immutable` |

WAV はサーバー内メモリにリングバッファ（固定 20 件）で保持されます。上限を超えた古いものから破棄されるため、古い URL は 404 になります。無音モード時は WAV が生成されないため常に 404 を返します。

#### GET /events（SSE）

Server-Sent Events ストリームです。ブラウザが接続し、再生クリップや合成エラーのイベントをリアルタイムで受信します。複数クライアントが同時に接続可能で、全クライアントに同一イベントが配信されます。

**レスポンスヘッダー**

| ヘッダー | 値 |
|---|---|
| `Content-Type` | `text/event-stream` |
| `Cache-Control` | `no-cache` |
| `Connection` | `keep-alive` |

**イベント種別**

`clip` イベント（音声クリップ配信時）:

```
event: clip
data: {"url":"/clips/1746316414387.wav","text":"読み上げテキスト","speakerName":"ずんだもん","styleName":"ノーマル","timestamp":1746316414387}
```

| フィールド | 説明 |
|---|---|
| `url` | WAV の取得 URL（無音モード時は空文字） |
| `text` | 読み上げテキスト |
| `speakerName` | 話者名（エンジン未登録 ID は `話者#<ID>`） |
| `styleName` | スタイル名（無音モード時は空文字） |
| `timestamp` | 配信時刻の Unix ms（UTC） |

`error` イベント（合成エラー等発生時）:

```
event: error
data: {"id":1,"category":"synthesis","message":"...","timestamp":1746316414387,"path":"","text":"読み上げテキスト"}
```

サーバーシャットダウン時は接続中のすべての SSE クライアントが切断されます。無音モードでは `clip` イベントは配信されません。

#### GET /assets/...（静的アセット・キャラクター画像）

**フロントエンドアセット** (`/assets/<hash>.js`, `/assets/<hash>.css` 等):

Vite でビルドされたハッシュ付きアセットを配信します。ファイルが存在する場合は長期キャッシュヘッダーが付与されます。

| ヘッダー | 値 |
|---|---|
| `Cache-Control` | `public, max-age=31536000, immutable` |

| ステータスコード | 条件 |
|---|---|
| 200 | ファイルが存在する |
| 404 | ファイルが存在しない |

**キャラクター画像** (`/assets/images/<charId>/<filename>`):

`GET /api/characters` のレスポンスに含まれる `mouthClosed` / `mouthOpen` パスで参照される画像を配信します。

| ステータスコード | 条件 |
|---|---|
| 200 | 画像が存在する |
| 404 | キャラクターが存在しない・assets ディレクトリがない |
| 4xx | パストラバーサル（`../` を含むパス）は拒否される |

優先順位はプロジェクト assets（`VOX_ACTOR_WORKSPACE/assets/`）> HOME assets（`~/.vox-actor/assets/`）です。

### フロントエンド UI 仕様

viewer が配信するブラウザ UI の仕様です。

#### タブ構成

| タブ名 | 表示条件 |
|---|---|
| 配信 | 常時表示 |
| 音声テスト | 通常モード時のみ。無音モード（`GET /api/status` の `silent=true`）時は非表示 |

初期表示は「配信」タブです。選択中のタブは `localStorage["vox-actor.stream.activeTab"]` に保存され、リロード後も復元されます。

無音モード時は画面上部に無音モードバッジが表示されます。

#### 配信タブ

##### SSE 接続ステータス

`GET /events` に接続した後、ページ上部にステータス表示が更新されます。

| 状態 | 表示 |
|---|---|
| 接続中 | 「接続中」 |
| 切断 | 「切断」（その後自動再接続） |

##### タイムライン

`clip` / `error` イベントを受信するたびにタイムラインへ追加されます。ページロード時は `GET /api/history` の履歴をタイムラインに表示し、リスト末尾（最新エントリ）にスクロールします（履歴エントリは再生キューに積まれません）。

**clip アイテムの表示内容**

| 項目 | デフォルト | localStorage キー |
|---|---|---|
| テキスト | 常時表示 | — |
| キャラ名（話者名） | ON | `vox-actor.stream.showSpeakerName` |
| スタイル名 | ON | `vox-actor.stream.showStyleName` |
| 時刻（HH:MM:SS） | ON | `vox-actor.stream.showTimestamp` |

**error アイテムのラベル**

| `category` | ラベル |
|---|---|
| `synthesis` | 合成エラー |
| `file` | ファイルエラー |
| `connection` | 接続エラー |

`speakerName` / `text` が含まれる場合はラベルと合わせて表示されます。

**履歴件数上限**

| 設定 | デフォルト | localStorage キー | 備考 |
|---|---|---|---|
| 履歴件数上限 | 20 | `vox-actor.stream.historySize` | 上限を超えると古いエントリから削除される |

##### 音量・消音コントロール

| コントロール | 初期値 | localStorage キー |
|---|---|---|
| 消音チェックボックス | ON（ミュート） | — |
| 音量スライダー（0〜100） | 50 | `vox-actor.stream.volume` |

消音チェックボックスのオン/オフは `audio.muted` と連動します。muted 状態でも clip はタイムラインに追加されます。音量スライダーの値はリロード後も復元されます。

##### キャラクター画像（口パク）

`GET /api/characters` のレスポンスに応じて「キャラ画像」チェックボックスの表示/非表示が切り替わります。

| `enabled` | チェックボックス | 口パク画像エリア |
|---|---|---|
| `true` | 表示 | チェックON時に clip 受信後表示 |
| `false` | 非表示 | 非表示 |

「キャラ画像」チェックボックスの状態は `vox-actor.stream.showCharacters` に保存されリロード後も復元されます（デフォルト ON）。チェックをオンにした状態で、受信した `clip` の `speakerName` に対応するキャラクターが登録されている場合、`mouthClosed` / `mouthOpen` 画像が口パク表示されます。対応キャラクターがない場合は画像エリアが表示されません（タイムラインへの追加は通常どおり行われます）。チェックをオフにすると画像エリアが非表示になります。

##### 再生キュー

URL 付きの `clip` を受信すると、ブラウザが WAV を先読み（fetch）してから `audio.src` に blob URL を設定します。URL が空の `clip` はタイムラインに表示されますが `audio.src` は更新されません。表示順は受信順（timestamp 昇順）です。

#### 音声テストタブ

話者セレクタで話者を選択して「テスト再生」ボタンを押すと、`GET /test-clip?speaker=<id>` を呼び出して WAV を取得し `audio.src` を更新します。同一話者の 2 回目以降の再生はサーバー側キャッシュが返るため合成は実行されません。

| 設定 | localStorage キー |
|---|---|
| 選択中の話者ID | `vox-actor.stream.testSpeakerId` |

話者IDの選択状態はリロード後も復元されます。

#### localStorage 永続化キー一覧

| キー | 型 | デフォルト | 用途 |
|---|---|---|---|
| `vox-actor.stream.activeTab` | string | `"stream"` | アクティブタブ（`"stream"` / `"test"`） |
| `vox-actor.stream.volume` | number | `50` | 音量スライダー値（0〜100） |
| `vox-actor.stream.historySize` | number | `20` | タイムライン履歴件数上限 |
| `vox-actor.stream.showSpeakerName` | boolean | `true` | キャラ名表示トグル |
| `vox-actor.stream.showStyleName` | boolean | `true` | スタイル名表示トグル |
| `vox-actor.stream.showTimestamp` | boolean | `true` | 時刻表示トグル |
| `vox-actor.stream.showCharacters` | boolean | `true` | キャラ画像表示トグル |
| `vox-actor.stream.testSpeakerId` | string | `""` | 音声テストタブの選択話者ID |

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
3. git管理外の場合はカレントディレクトリの `.vox-actor` サブディレクトリをワークスペースルートとして扱う

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

音声出力デバイスを **open → 即 close** のみ行い、実再生を伴わずに可用性を**終了コード**で返す診断用サブコマンド。シェルスクリプトからのモード判定（direct / file）に利用できる。

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

## `viewer-check` サブコマンド

```
vox-actor viewer-check [-v|--verbose]
```

viewer のロックファイルと `/api/status` への疎通確認を行い、起動有無を**終了コード**で返す診断用サブコマンド。シェルスクリプトからのモード判定（direct / file）に利用できる。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--verbose` / `-v` | — | `false` | 診断メッセージを標準出力に出力する |

**出力と終了コード:**

| 状況 | 終了コード | stdout | stderr |
|---|---|---|---|
| viewer 起動中（ロックファイルあり＋ `/api/status` 応答 200） | `0` | 空 | 空 |
| viewer 未起動 or 応答なし | `1` | 空 | 空 |

- `--verbose` 指定時は起動中の場合のみ `viewer addr: <addr>` を stdout に出力する。
- 機械可読な判定は**終了コード**で行う。
- `/api/status` への HTTP プローブが入るため、呼び出しごとに数十〜数百ms のオーバーヘッドが発生する。

**使用例:**

```bash
# 直接実行（viewer 起動中）
$ vox-actor viewer-check
$ echo $?
0

# verbose 出力
$ vox-actor viewer-check -v
viewer addr: 127.0.0.1:8080
$ echo $?
0

# viewer 未起動時
$ vox-actor viewer-check
$ echo $?
1
```

## `playback` サブコマンド

再生状態を管理するサブコマンド群。

### `playback wait` サブコマンド

```
vox-actor playback wait <id>
```

`playback_id` を指定して再生が完了するまでポーリングする。`local_playback` を渡すと即時終了する。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--viewer-url` | `VOX_VIEWER_URL` | (未指定) | viewer の HTTP エンドポイント URL (例: `http://192.168.1.10:8080`)。指定時は lockfile auto-detect をスキップして明示 URL の viewer に接続する。 |
| `--server-down-timeout` | — | `30s` | サーバーへの接続失敗が連続してこの時間を超えた場合にエラー終了する |

**引数:**

| 引数 | 必須 | 説明 |
|---|---|---|
| `<id>` | ✓ | 再生ID。`local_playback` を渡すと即時終了する。 |

**使用例:**

```bash
# 再生完了を待機
vox-actor playback wait <playback_id>

# viewer URL を明示して待機
vox-actor playback wait --viewer-url http://192.168.1.10:8080 <playback_id>

# ローカル再生モード（即時終了）
vox-actor playback wait local_playback
```

## `assets` サブコマンド

キャラクター画像などのアセットを管理するサブコマンド群。

### `assets download` サブコマンド

```
vox-actor assets download <repo-url>
```

指定した git リポジトリから `vox-actor-assets.json` を読み取り、キャラクター画像を `.vox-actor/assets/` へコピーする。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--speaker` | — | (未指定) | ダウンロード対象の speaker 名（リピート可、カンマ区切り可）。省略時は全 speaker をダウンロードする。 |
| `--force` | — | `false` | ローカルに同名 speaker が存在する場合に上書きする |
| `--scope` | — | `project` | 配置先スコープ（`home`: `~/.vox-actor/assets/`、`project`: プロジェクト配下の `.vox-actor/assets/`） |
| `--verbose` | — | `false` | 詳細ログを出力 |

**引数:**

| 引数 | 必須 | 説明 |
|---|---|---|
| `<repo-url>` | ✓ | キャラクター設定を含む git リポジトリの URL |

**使用例:**

```bash
# 全 speaker をプロジェクトスコープにダウンロード
vox-actor assets download https://example.com/character-repo.git

# 特定の speaker のみダウンロード
vox-actor assets download --speaker ずんだもん https://example.com/character-repo.git

# 複数の speaker をカンマ区切りで指定
vox-actor assets download --speaker "ずんだもん,四国めたん" https://example.com/character-repo.git

# ホームスコープにダウンロード（全プロジェクトで共有）
vox-actor assets download --scope home https://example.com/character-repo.git

# 既存 speaker を強制上書き
vox-actor assets download --force https://example.com/character-repo.git
```

## `speakers` サブコマンド

利用可能なキャラクター（スピーカー）を管理するサブコマンド群。

### `speakers list` サブコマンド

```
vox-actor speakers list
```

利用可能なキャラクター一覧をJSON形式で出力する。オプションはなし。

**出力サンプル:**

```json
[{"id":"metan","name":"四国めたん"},{"id":"zundamon","name":"ずんだもん"}]
```

**使用例:**

```bash
vox-actor speakers list
```

### `speakers profile` サブコマンド

```
vox-actor speakers profile (--id <id> | --name <name>)
```

指定したキャラクターのプロフィールをJSON形式で返す。`--id` と `--name` はどちらか一方のみ指定する（両方または両方省略はエラー）。

| オプション | 環境変数 | デフォルト値 | 説明 |
|---|---|---|---|
| `--id` | — | (未指定) | assets 配下のディレクトリ名でキャラクターを指定する |
| `--name` | — | (未指定) | `speaker.json` の `name` フィールドでキャラクターを指定する |

**出力サンプル:**

```json
{"id":"zundamon","name":"ずんだもん","pronoun":"ボク","speechSuffix":["〜のだ","〜なのだ"],"personality":["元気","明るい","素直","前向き","おっちょこちょい","好奇心旺盛"],"speakers":{"ノーマル":3,"あまあま":1},"styles":["ノーマル","あまあま"],"description":"# ずんだもん キャラクター設定\n..."}
```

**使用例:**

```bash
# ID でプロフィールを取得
vox-actor speakers profile --id zundamon

# 名前でプロフィールを取得
vox-actor speakers profile --name ずんだもん
```

## ストリーム配信モード

HTTPサーバーを起動し、SSE経由でブラウザに音声を配信します。音声デバイスが利用できない環境でホスト側のブラウザに再生させるケースで利用できます。

[`viewer` サブコマンド](#viewer-サブコマンド)を使ってください。

```bash
# viewer と watch を並行起動
vox-actor viewer &
vox-actor watch /path/to/watch-dir

# viewer でバインドアドレスを変更して並行起動
vox-actor viewer --host 0.0.0.0 --port 8080 &
vox-actor watch /path/to/watch-dir
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
- `watch` ではディレクトリ監視・`done/` への移動／削除は通常通り実施されるため、監視フロー全体を検証できます
