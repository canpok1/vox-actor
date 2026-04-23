---
name: act
description: |
  vox-actorを使った音声再生の技術的実行を担うスキル。
  呼び出し元（monologue/speak/talk）が会話コンテキストで確立した
  生成種別・本文・キャラクター・長さなどの演出指示を引き継ぎ、
  JSONL/JSON台本を生成してplay-script.shで再生する。
  vox-actor CLIの利用知識（話速・モード切替・制約・エラーログ仕様）はすべてここに集約。
  直接ユーザーが呼ぶより、monologue/speak/talk経由の呼び出しを想定。
argument-hint: "<種別:monologue|speak|talk> <本文> [追加の演出指示]"
allowed-tools:
  - "Bash(${CLAUDE_PLUGIN_ROOT}/skills/act/scripts/play-script.sh *)"
  - "Bash(mkdir -p *)"
  - "Bash(date +%s%3N)"
  - "Read(${CLAUDE_PLUGIN_ROOT}/skills/act/characters/*.md)"
  - "Write(${VOX_ACTOR_WORKSPACE}/tmp/*)"
---

vox-actor を使った音声再生の技術的実行を担うスキルです。`monologue` / `speak` / `talk` の入口スキルから呼び出される前提で、台本生成と再生実行を一手に引き受けます。vox-actor CLI に関する利用知識（話速の目安・通知モード切替・ワークスペース解決・エラーログ仕様・JSON/JSONL の形式制約・キャラクター設定の扱いなど）はこのスキルに集約されています。

## 入力の引き継ぎ

呼び出し元のスキル（`monologue` / `speak` / `talk`）が会話コンテキストで確立した以下の情報を引き継ぎます。`$ARGUMENTS` には種別と本文を渡し、それ以外（キャラクター・長さ等）は会話コンテキスト経由で引き継ぎます。

| 情報 | 渡し方 | 備考 |
|------|--------|------|
| 生成種別（`monologue` / `speak` / `talk`） | `$ARGUMENTS` 先頭 | tmp ファイル名の `<kind>` にも使う |
| 本文 | `$ARGUMENTS` | ユーザーが指定した内容 |
| キャラクター | 会話コンテキスト | 呼び出し元がメモリ参照済み |
| 長さ（`short` / `medium` / `long`） | 会話コンテキスト | speak / talk の場合 |
| 通知要否の判定結果 | （暗黙） | monologue の通知スキップ時はそもそも act を呼ばない |

## 実行フロー

1. `$ARGUMENTS` から **種別** と **本文** を取り出す
2. 引き継いだキャラクター名から `${CLAUDE_PLUGIN_ROOT}/skills/act/characters/<name>.md` を読み、`speakers` 一覧と性格・口調を把握する
3. 種別に応じた台本を生成する
   - `monologue`: 1セリフだけの単一 JSON オブジェクト
   - `speak`: 冒頭→本題→まとめの流れの JSONL（複数行）
   - `talk`: 複数キャラの掛け合い JSONL（複数行）。冒頭で全キャラが1回以上登場するよう構成する
4. セリフ毎に内容の感情に合う `speaker` と `speedScale` を選定する
5. `mkdir -p "${VOX_ACTOR_WORKSPACE}/tmp"` で一時ディレクトリを作成する
6. `date +%s%3N` でユニックス時刻（ms）を取得し、一時ファイル `${VOX_ACTOR_WORKSPACE}/tmp/<unix_ms>_<kind>.<ext>` に台本を Write する
   - `monologue` → `<unix_ms>_monologue.json`
   - `speak` → `<unix_ms>_speak.jsonl`
   - `talk` → `<unix_ms>_talk.jsonl`
7. `play-script.sh <path>` を呼び出して再生する

```bash
${CLAUDE_PLUGIN_ROOT}/skills/act/scripts/play-script.sh <path>
```

- `<path>`（第1引数・必須）: 生成した台本ファイルの絶対パス

## 台本の形式

各セリフは1行の JSON オブジェクトです。`speak` / `talk` はこれを複数行並べた JSONL、`monologue` は単一オブジェクトの JSON となります。

```json
{"text": "セリフ本文", "speaker": <speaker_id>, "speedScale": <0.8〜1.2>}
```

- `text`: セリフ本文。1〜2文程度の短文
- `speaker`: キャラクター設定ファイルの `speakers` マップから感情に合うIDを選ぶ
- `speedScale`: 話速。下表の目安から感情に合う値を選ぶ

### 話速の目安

| 値 | 説明 |
|----|------|
| 0.8 | ゆっくり |
| 0.9 | ちょっとゆっくり |
| 1.0 | 普通（デフォルト） |
| 1.1 | ちょっと早口 |
| 1.2 | 早口 |

### 種別ごとの形式・例

#### `monologue`（単一 JSON）

ずんだもん（`speakers.ノーマル: 3`）の独り言例:

```json
{"text": "よーし、始めるのだ！", "speaker": 3, "speedScale": 1.1}
```

#### `speak`（JSONL・1キャラ）

冒頭の挨拶／つかみ → 本題 → まとめの流れを意識する。ずんだもんの解説例:

```jsonl
{"text": "クロージャって何なのだ？説明するのだ！", "speaker": 3, "speedScale": 1.1}
{"text": "簡単に言うと、関数が作られた時の周りの変数を覚えておく仕組みなのだ", "speaker": 3, "speedScale": 1.0}
{"text": "むむっ、ちょっと難しいけど…例えるならお弁当箱に具材を詰めて持ち歩く感じなのだ", "speaker": 3, "speedScale": 1.0}
{"text": "後から開けても中身がそのまま残ってるみたいに、変数の値も残るのだ〜", "speaker": 1, "speedScale": 0.9}
{"text": "わかったかな？お疲れ様なのだ！", "speaker": 1, "speedScale": 1.0}
```

#### `talk`（JSONL・複数キャラ）

冒頭で全キャラが登場（自己紹介・挨拶・呼びかけ）し、本題はキャラ同士の掛け合い・質問応答・補足で展開する。ずんだもん（`ノーマル: 3`、`あまあま: 1`）と四国めたん（`ノーマル: 2`、`ツンツン: 6`）の会話例:

```jsonl
{"text": "今日はクロージャについて解説するのだ！", "speaker": 3, "speedScale": 1.1}
{"text": "あら、わたくしも勉強させてもらおうかしら", "speaker": 2, "speedScale": 1.0}
{"text": "簡単に言うと、関数が作られた時の周りの変数を覚えておく仕組みなのだ", "speaker": 3, "speedScale": 1.0}
{"text": "なるほど、お弁当箱みたいなものですわね", "speaker": 2, "speedScale": 1.0}
{"text": "そう、そんな感じなのだー", "speaker": 1, "speedScale": 0.9}
{"text": "よく分かりましたわ。ありがとう、ずんだもん", "speaker": 2, "speedScale": 1.0}
```

## キャラクター設定

`${CLAUDE_PLUGIN_ROOT}/skills/act/characters/<name>.md` がキャラクター設定ファイルです。Claude 向けの自然言語テキストとして消費され、キャラクター名や口調特徴などの主要メタデータは YAML frontmatter に構造化されています。本文には性格・口調の説明とセリフ例が記載されています。

呼び出し元スキル（`monologue` / `speak` / `talk`）がメモリから読み取って引き継いだキャラクター名に対応するファイルを読み、`speakers` マップ・口調・性格を反映した台本を生成します。

## 前提条件

### play-script.sh の依存関係

再生の実行に使用する `${CLAUDE_PLUGIN_ROOT}/skills/act/scripts/play-script.sh` は以下の前提条件を必要とします。

- **`vox-actor` コマンドのインストール必須**: スクリプト冒頭で `command -v vox-actor` を検査し、未インストール時は `[ERROR] vox-actor コマンドが必要です` を stderr に出して非0終了する
- **ワークスペースの解決**: スクリプトは `vox-actor config path.queue` / `vox-actor config path.workspace` を呼ぶだけで、環境変数の分岐は CLI 側が担う。CLI の解決順は以下のとおり
  1. 環境変数 `VOX_ACTOR_WORKSPACE` が設定されていればその値をワークスペースルートとして使う（queue は `${VOX_ACTOR_WORKSPACE}/queue`）
  2. 未設定なら gitリポジトリ直下の `.vox-actor` をワークスペースルートとする（queue は `<repo>/.vox-actor/queue`）
- **git 管理外で利用する場合**: `VOX_ACTOR_WORKSPACE` の明示が必要。未指定のままだと CLI が非0終了し、そのエラーメッセージが表示されてスクリプトも終了する
- **出力先ディレクトリ**: ワークスペースルートおよび配下の `queue/` は `play-script.sh` が必要に応じて自動作成する。`tmp/` ディレクトリは Claude が Write する前に `mkdir -p "${VOX_ACTOR_WORKSPACE}/tmp"` で作成する（`tmp/` 利用時は `VOX_ACTOR_WORKSPACE` の明示が前提）
- `play-script-errors.log`（directモードのエラーログ）はワークスペースルート直下に配置される

### 通知モード

通知方式は以下のルールで自動切替する:

1. 環境変数 `VOX_ACTOR_MONOLOGUE_MODE` が設定されていればそれに従う（`direct` または `file`）
2. 未設定なら `vox-actor audio-check` の終了コードで自動判定する
   - 終了コード 0（音声デバイス open 成功）→ `direct`
   - 非0（音声デバイス open 失敗）→ `file`

#### `direct` モード

`vox-actor act <path>` を同期実行し、再生の完了を待ってスクリプトが戻る。

- VOICEVOXエンジンのURLは `vox-actor` 側の環境変数 `VOX_ENGINE_URL` で解決する（非デフォルトポート時はユーザーが設定）
- 監視プロセス（`vox-actor watch`）の常駐は不要
- 再生完了後、渡された一時ファイルを `rm -f` で削除する
- 同期実行のため、同一セッション内で連続して呼び出した場合は先の発話が完了してから次が再生される。ただし複数セッションから並列に呼ばれた場合はエンジンへの並列リクエストと音声再生の重なりが発生しうる。複数セッション間でも逐次再生したい場合は `VOX_ACTOR_MONOLOGUE_MODE=file` + 監視プロセス構成に切り替える
- `vox-actor act` 失敗時もスクリプト本体はエラー終了せず、標準出力/標準エラーの内容をワークスペースルート直下の `play-script-errors.log` にタイムスタンプ付きで追記する。ログは末尾200行でローテーションされる。失敗検知は `tail -f <workspace>/play-script-errors.log` で行える

#### `file` モード

一時ファイルを `vox-actor config path.queue` で解決された queue ディレクトリ配下へ `mv` で移動する（渡された一時ファイル名がそのまま使われるため、`<unix_ms>_monologue.json` なら `queue/<unix_ms>_monologue.json` となる）。外部の通知監視プロセス（`vox-actor watch` 等）はこの `queue/` ディレクトリを監視対象として検知・読み上げする。

## 制約

- セリフ毎に1〜2文程度の短文にすること（長文1セリフは合成・再生で詰まる）
- 英単語や英語のファイル名等を含める場合は、英語のまま使わず、発音を表す日本語に置き換えること
  - 拡張子も省略せず発音を表す日本語にすること
  - 例: `README.md` → 「リードミードットエムディー」、`config.json` → 「コンフィグドットジェイソン」、`merge` → 「マージ」、`build` → 「ビルド」
- ファイルパス、URL、UUIDなど長い文字情報は発音しても分かりづらいため、セリフに含めないこと
- 内容にコード断片や仕様文書が含まれる場合も、識別子は「読める形」に変換すること
- セリフ内のダブルクォート・バックスラッシュ等は JSON / JSONL 仕様に従って適切にエスケープすること
- `talk` では同一キャラが連続で話してもよいが、会話らしさを保つため役割配分とターン交代を意識する
- `talk` の冒頭では全キャラが1回以上登場するよう配慮する（自己紹介・挨拶・呼びかけなど）
