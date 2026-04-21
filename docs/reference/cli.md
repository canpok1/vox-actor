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
| `--stream` | — | `false` | HTTPサーバーを起動し、SSE経由でブラウザに音声を配信（[詳細](#ストリーム配信モード)） |
| `--stream-addr` | — | `127.0.0.1:8080` | ストリーム配信用のバインドアドレス |
| `--stream-history-size` | — | `10` | サーバー側のWAVリングバッファに保持するセリフの件数上限（`--stream` と組み合わせて使用）。ブラウザ側のタイムライン表示上限は画面上のプルダウンで設定する |
| `--verbose` | — | `false` | 詳細ログを出力 |
| `--dry-run` | — | `false` | VOICEVOX・音声再生を行わず、読み上げ対象をログ出力（[詳細](#dry-runモード)） |

## ストリーム配信モード

`watch --stream` を付けると、ローカルスピーカー再生の代わりにHTTPサーバーを起動し、SSE経由でブラウザに音声を配信します。音声デバイスが利用できない環境でホスト側のブラウザに再生させるケースで利用できます。

```bash
# デフォルト（127.0.0.1:8080 にバインド）
vox-actor watch --stream /path/to/watch-dir

# バインドアドレスを変更
vox-actor watch --stream --stream-addr 0.0.0.0:8080 /path/to/watch-dir
```

- `http://<stream-addr>/` をブラウザで開き、音量スライダーを操作すると再生が解禁されます（ブラウザの自動再生ポリシー対策）。
- 画面はチャット風のタイムラインで、古いセリフが上、新しいセリフが下に並びます。再生中のセリフは背景色と `▶` アイコンで強調され、再生開始時に自動でスクロールして画面内に表示されます。
- ブラウザ側のタイムライン表示上限は画面上部のプルダウン（10 / 20 / 30 / 50 / 100 / 200、初期値20）で変更でき、選択値は localStorage に保存されてリロード後も復元されます。
- サーバー側のWAVは `--stream-history-size`（デフォルト10）まで保持され、上限を超えた古いものから破棄されます。再生中のセリフは上限に達しても削除されません。
- 複数ブラウザで同時に開いた場合、全クライアントに同じクリップが配信されます。
- `--stream` と `--dry-run` の併用はエラーになります。
- デフォルトでは `127.0.0.1` にバインドするため外部からはアクセスできません。LAN公開したい場合は `--stream-addr 0.0.0.0:8080` のように明示的に指定してください。

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
