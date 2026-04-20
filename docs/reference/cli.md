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
| `--verbose` | — | `false` | 詳細ログを出力 |
| `--dry-run` | — | `false` | VOICEVOX・音声再生を行わず、読み上げ対象をログ出力（[詳細](#dry-runモード)） |

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
