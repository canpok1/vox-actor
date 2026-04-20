---
name: talk
description: 渡されたトピックを複数キャラクターの会話形式JSONL台本として生成し、vox-actor経由で掛け合い・対話・漫才風の読み上げを届けるスキル。
argument-hint: "<内容>"
allowed-tools:
  - "Bash(${CLAUDE_PLUGIN_ROOT}/scripts/play-script.sh *)"
  - "Bash(mkdir -p *)"
  - "Read(${CLAUDE_PLUGIN_ROOT}/characters/*.md)"
  - "Write(${VOX_ACTOR_WORKSPACE}/tmp/*)"
---

ユーザーから渡された内容を、複数キャラクターが会話する形式のJSONL台本として生成し、音声で届けます。掛け合いや対話、漫才、ニュース番組風の読み上げなど、複数人の語りが求められる音声アウトプットに利用できます。`speak` スキルが1キャラで冒頭→本題→まとめを語るのに対し、本スキルは複数キャラの役割配分と会話の流れを構成します。

## 実行フロー

1. `$ARGUMENTS` を読み上げ内容として受け取る
2. メモリから `talk_characters`（既定 `[zundamon, metan]`）と `talk_length`（既定 `medium`）を読み取る
3. `talk_characters` の各キャラについて `${CLAUDE_PLUGIN_ROOT}/characters/<name>.md` を読み、`speakers` 一覧と性格を把握する
4. 内容に応じて役割配分（例: 解説役／聞き役／ツッコミ役 など）をその場で組み立てる
5. 長さ設定に応じた **JSONL台本** を生成する（各行: `{"text":..., "speaker":..., "speedScale":...}`）
   - 冒頭で全キャラが登場する（自己紹介・挨拶など）よう意識する
   - 本題ではキャラ同士の掛け合い・質問応答・補足などで会話を展開する
   - セリフ毎に担当キャラの `speakers` から感情に合うIDを選定し、`speedScale` も感情に合わせる
6. `mkdir -p "${VOX_ACTOR_WORKSPACE}/tmp"` で一時ディレクトリを作成する
7. 一時ファイル `${VOX_ACTOR_WORKSPACE}/tmp/talk_<unix_ms>.jsonl` に Write する
8. `play-script.sh <path>` を呼び出す

```bash
${CLAUDE_PLUGIN_ROOT}/scripts/play-script.sh <jsonl_path>
```

- `<jsonl_path>`（第1引数・必須）: 生成した JSONL 台本の絶対パス

## 読み上げの長さ

| 設定 | セリフ数の目安 | 想定再生時間 |
|------|---------------|------------|
| `short` | 4〜6 | 〜30秒 |
| `medium`（既定） | 8〜12 | 1〜2分 |
| `long` | 14+ | 数分 |

ユーザー指示「会話を短めにして」等でメモリを更新すれば次回以降に反映される。

## 話速の目安

| 値 | 説明 |
|----|------|
| 0.8 | ゆっくり |
| 0.9 | ちょっとゆっくり |
| 1.0 | 普通（デフォルト） |
| 1.1 | ちょっと早口 |
| 1.2 | 早口 |

## キャラクター設定

メモリ `talk_characters` に指定された2〜4人のキャラクター設定ファイルを読み込み、それぞれになりきった台本を生成する。

- メモリ未設定時のデフォルト: ずんだもん（`zundamon`）と四国めたん（`metan`）の2人
- キャラクター設定ファイル: `${CLAUDE_PLUGIN_ROOT}/characters/{キャラクター名}.md`（例: `${CLAUDE_PLUGIN_ROOT}/characters/zundamon.md`）

## メモリに保存する設定項目

以下の設定をClaudeのメモリシステム（MEMORY.md）に保存・参照する:

| 項目 | キー | デフォルト値 | 値 | 説明 |
|------|------|-------------|----|-----|
| 会話キャラクター | `talk_characters` | `[zundamon, metan]` | `characters/<name>.md` の `<name>` の配列（2〜4人） | 本スキル専用。`default_character` とは別に管理 |
| 会話の長さ | `talk_length` | `medium` | `short` / `medium` / `long` | 上記の長さ表を参照 |

`talk_characters` は本スキル固有の設定で、`monologue` / `speak` の `default_character` とは独立している。

## 前提条件

### play-script.sh の依存関係

再生の実行に使用する `${CLAUDE_PLUGIN_ROOT}/scripts/play-script.sh` は以下の前提条件を必要とする:

- **環境変数 `VOX_ACTOR_WORKSPACE`**: vox-actor関連ファイルのルートディレクトリを指定する。配下に `tmp/`（JSONL台本の一時格納先）、`queue/`（fileモードの通知ファイル出力先）、`play-script-errors.log`（directモードのエラーログ）が置かれる。未設定の場合のデフォルト値は以下のとおり
  - gitリポジトリ内: `<gitリポジトリ直下>/.vox-actor`
  - gitリポジトリ外: `$PWD/.vox-actor`
- **出力先ディレクトリ**: 上記のルートディレクトリおよび配下の `queue/` は `play-script.sh` が必要に応じて自動作成する。`tmp/` ディレクトリは Claude が Write する前に `mkdir -p "${VOX_ACTOR_WORKSPACE}/tmp"` で作成する

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
- `vox-actor act` 失敗時もスクリプト本体はエラー終了せず、標準出力/標準エラーの内容を `${VOX_ACTOR_WORKSPACE}/play-script-errors.log` にタイムスタンプ付きで追記する。ログは末尾200行でローテーションされる。失敗検知は `tail -f ${VOX_ACTOR_WORKSPACE}/play-script-errors.log` で行える

#### `file` モード

一時ファイルを `${VOX_ACTOR_WORKSPACE}/queue/$(basename <jsonl_path>)` へ `mv` で移動する（渡された一時ファイル名がそのまま使われるため、`talk_<ms>.jsonl` なら `queue/talk_<ms>.jsonl` となる）。外部の通知監視プロセス（`vox-actor watch` 等）はこの `queue/` ディレクトリを監視対象として検知・読み上げする。

## JSONL 出力例

ずんだもん（`speakers.ノーマル: 3`、`あまあま: 1`）と四国めたん（`ノーマル: 2`、`ツンツン: 6`）の会話例:

```jsonl
{"text": "今日はクロージャについて解説するのだ！", "speaker": 3, "speedScale": 1.1}
{"text": "あら、わたくしも勉強させてもらおうかしら", "speaker": 2, "speedScale": 1.0}
{"text": "簡単に言うと、関数が作られた時の周りの変数を覚えておく仕組みなのだ", "speaker": 3, "speedScale": 1.0}
{"text": "なるほど、お弁当箱みたいなものですわね", "speaker": 2, "speedScale": 1.0}
{"text": "そう、そんな感じなのだー", "speaker": 1, "speedScale": 0.9}
{"text": "よく分かりましたわ。ありがとう、ずんだもん", "speaker": 2, "speedScale": 1.0}
```

## キャラクター設定ファイルについて

`characters/` 配下のキャラクター設定ファイルはClaude向けの自然言語テキストとして消費される。キャラクター名や口調特徴などの主要メタデータはYAML frontmatterに構造化されており、本文にはClaude向けの詳細な性格・口調の説明やセリフ例が記載されている。`monologue` / `speak` スキルと同一のファイルを共用する。

## 制約

- セリフ毎に1〜2文程度の短文にすること（長文1セリフは合成・再生で詰まる）
- 英単語や英語のファイル名等を含める場合は、英語のまま使わず、発音を表す日本語に置き換えること
  - 拡張子も省略せず発音を表す日本語にすること
  - 例: `README.md` → 「リードミードットエムディー」、`config.json` → 「コンフィグドットジェイソン」、`merge` → 「マージ」、`build` → 「ビルド」
- ファイルパス、URL、UUIDなど長い文字情報は発音しても分かりづらいため、セリフに含めないこと
- 内容にコード断片や仕様文書が含まれる場合も、識別子は「読める形」に変換すること
- セリフ内のダブルクォート・バックスラッシュ等は JSONL 仕様に従って適切にエスケープすること
- 同一キャラが連続で話してもよいが、会話らしさを保つため役割配分とターン交代を意識する
- 冒頭では全キャラが1回以上登場するよう配慮する（自己紹介・挨拶・呼びかけなど）
