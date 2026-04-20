---
name: speak
description: 渡されたトピック・メモ・作業結果などをキャラクター付きの複数セリフJSONL台本として生成し、vox-actor経由で解説・朗読・結果報告を音声で届けるスキル。
argument-hint: "<内容>"
allowed-tools:
  - "Bash(${CLAUDE_PLUGIN_ROOT}/skills/speak/scripts/speak.sh *)"
  - "Bash(mkdir -p *)"
  - "Read(${CLAUDE_PLUGIN_ROOT}/characters/*.md)"
  - "Write(${VOX_ACTOR_WORKSPACE}/tmp/*)"
---

ユーザーから渡された内容を、キャラクターになりきった複数セリフのJSONL台本として生成し、音声で届けます。解説・朗読・作業の結果報告・ストーリーテリングなど、まとまった長さの音声アウトプット全般に利用できます。`monologue` が1文の独り言用であるのに対し、本スキルは冒頭→本題→まとめのまとまった長さを扱います。

## 実行フロー

1. `$ARGUMENTS` を読み上げ内容として受け取る
2. メモリから `explanation_character`（既定 `zundamon`）と `explanation_length`（既定 `medium`）を読み取る
3. `${CLAUDE_PLUGIN_ROOT}/characters/<name>.md` を読み、`speakers` 一覧と性格を把握する
4. 長さ設定に応じた **JSONL台本** を生成する（各行: `{"text":..., "speaker":..., "speedScale":...}`）
   - 冒頭の挨拶／つかみ → 本題 → まとめの流れを意識する
   - セリフ毎に内容の感情に合う `speaker` と `speedScale` を選定する
5. `mkdir -p "${VOX_ACTOR_WORKSPACE}/tmp"` で一時ディレクトリを作成する
6. 一時ファイル `${VOX_ACTOR_WORKSPACE}/tmp/speak_<unix_ms>.jsonl` に Write する
7. `speak.sh <path>` を呼び出す

```bash
${CLAUDE_PLUGIN_ROOT}/skills/speak/scripts/speak.sh <jsonl_path>
```

- `<jsonl_path>`（第1引数・必須）: 生成した JSONL 台本の絶対パス

## 読み上げの長さ

| 設定 | セリフ数の目安 | 想定再生時間 |
|------|---------------|------------|
| `short` | 3〜5 | 〜十数秒 |
| `medium`（既定） | 6〜10 | 30秒〜1分 |
| `long` | 10+ | 数分 |

ユーザー指示「読み上げを短めにして」等でメモリを更新すれば次回以降に反映される。

## 話速の目安

| 値 | 説明 |
|----|------|
| 0.8 | ゆっくり |
| 0.9 | ちょっとゆっくり |
| 1.0 | 普通（デフォルト） |
| 1.1 | ちょっと早口 |
| 1.2 | 早口 |

## キャラクター設定

メモリで指定されたキャラクターの設定ファイルを読み込み、そのキャラクターになりきった台本を生成する。

- メモリ未設定時のデフォルト: ずんだもん（`zundamon`）
- キャラクター設定ファイル: `${CLAUDE_PLUGIN_ROOT}/characters/{キャラクター名}.md`（例: `${CLAUDE_PLUGIN_ROOT}/characters/zundamon.md`）

## メモリに保存する設定項目

以下の設定をClaudeのメモリシステム（MEMORY.md）に保存・参照する:

| 項目 | キー | デフォルト値 | 値 | 説明 |
|------|------|-------------|----|-----|
| 読み上げキャラクター | `explanation_character` | `zundamon` | `characters/<name>.md` の `<name>` | 例: 「読み上げはめたんで」→ `metan` を保存 |
| 読み上げの長さ | `explanation_length` | `medium` | `short` / `medium` / `long` | 上記の長さ表を参照 |

`monologue` 側の `monologue_probability` とは衝突しない。

## 前提条件

### speak.sh の依存関係

再生の実行に使用する `${CLAUDE_PLUGIN_ROOT}/skills/speak/scripts/speak.sh` は以下の前提条件を必要とする:

- **環境変数 `VOX_ACTOR_WORKSPACE`**: vox-actor関連ファイルのルートディレクトリを指定する。配下に `tmp/`（JSONL台本の一時格納先）、`queue/`（fileモードの通知ファイル出力先）、`speak-errors.log`（directモードのエラーログ）が置かれる。未設定の場合のデフォルト値は以下のとおり
  - gitリポジトリ内: `<gitリポジトリ直下>/.vox-actor`
  - gitリポジトリ外: `$PWD/.vox-actor`
- **出力先ディレクトリ**: 上記のルートディレクトリおよび配下の `queue/` は `speak.sh` が必要に応じて自動作成する。`tmp/` ディレクトリは Claude が Write する前に `mkdir -p "${VOX_ACTOR_WORKSPACE}/tmp"` で作成する

### 通知モード

通知方式は以下のルールで自動切替する:

1. 環境変数 `VOX_ACTOR_MONOLOGUE_MODE` が設定されていればそれに従う（`direct` または `file`）
2. 未設定なら `vox-actor` コマンドの有無で判定する（あり → `direct`、なし → `file`）

#### `direct` モード

`vox-actor act <jsonl_path>` を同期実行し、再生の完了を待ってスクリプトが戻る。

- VOICEVOXエンジンのURLは `vox-actor` 側の環境変数 `VOX_ENGINE_URL` で解決する（非デフォルトポート時はユーザーが設定）
- 監視プロセス（`vox-actor watch`）の常駐は不要
- 再生完了後、渡された一時ファイルを `rm -f` で削除する
- 同期実行のため、同一セッション内で連続して呼び出した場合は先の再生が完了してから次が再生される。ただし複数セッションから並列に呼ばれた場合はエンジンへの並列リクエストと音声再生の重なりが発生しうる。複数セッション間でも逐次再生したい場合は `VOX_ACTOR_MONOLOGUE_MODE=file` + 監視プロセス構成に切り替える
- `vox-actor act` 失敗時もスクリプト本体はエラー終了せず、標準出力/標準エラーの内容を `${VOX_ACTOR_WORKSPACE}/speak-errors.log` にタイムスタンプ付きで追記する。ログは末尾200行でローテーションされる。失敗検知は `tail -f ${VOX_ACTOR_WORKSPACE}/speak-errors.log` で行える

#### `file` モード

一時ファイルを `${VOX_ACTOR_WORKSPACE}/queue/speak_{ミリ秒タイムスタンプ}.jsonl` へ `mv` で移動する。外部の通知監視プロセス（`vox-actor watch` 等）はこの `queue/` ディレクトリを監視対象として検知・読み上げする。

## JSONL 出力例

解説用途の例（ずんだもん）。朗読・結果報告・ストーリーテリングでも同じJSONL形式で台本を生成する:

```jsonl
{"text": "クロージャって何なのだ？説明するのだ！", "speaker": 3, "speedScale": 1.1}
{"text": "簡単に言うと、関数が作られた時の周りの変数を覚えておく仕組みなのだ", "speaker": 3, "speedScale": 1.0}
{"text": "むむっ、ちょっと難しいけど…例えるならお弁当箱に具材を詰めて持ち歩く感じなのだ", "speaker": 3, "speedScale": 1.0}
{"text": "後から開けても中身がそのまま残ってるみたいに、変数の値も残るのだ〜", "speaker": 1, "speedScale": 0.9}
{"text": "わかったかな？お疲れ様なのだ！", "speaker": 1, "speedScale": 1.0}
```

## キャラクター設定ファイルについて

`characters/` 配下のキャラクター設定ファイルはClaude向けの自然言語テキストとして消費される。キャラクター名や口調特徴などの主要メタデータはYAML frontmatterに構造化されており、本文にはClaude向けの詳細な性格・口調の説明やセリフ例が記載されている。`monologue` スキルと共用する。

## 制約

- セリフ毎に1〜2文程度の短文にすること（長文1セリフは合成・再生で詰まる）
- 英単語や英語のファイル名等を含める場合は、英語のまま使わず、発音を表す日本語に置き換えること
  - 拡張子も省略せず発音を表す日本語にすること
  - 例: `README.md` → 「リードミードットエムディー」、`config.json` → 「コンフィグドットジェイソン」、`merge` → 「マージ」、`build` → 「ビルド」
- ファイルパス、URL、UUIDなど長い文字情報は発音しても分かりづらいため、セリフに含めないこと
- 内容にコード断片や仕様文書が含まれる場合も、識別子は「読める形」に変換すること
- セリフ内のダブルクォート・バックスラッシュ等は JSONL 仕様に従って適切にエスケープすること
- 同じ `speaker` を連続で使ってもよいが、感情に合わせて切り替えるとキャラクターらしさが出る
