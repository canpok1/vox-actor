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
  - "Bash(vox-actor speakers *)"
  - "Bash(${CLAUDE_PLUGIN_ROOT}/skills/act/scripts/save-script.sh *)"
  - "Bash(${CLAUDE_PLUGIN_ROOT}/skills/act/scripts/play-script.sh *)"
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
2. 引き継いだキャラクター名をもとに、`vox-actor speakers list` で利用可能なキャラクター一覧を取得し、入力がマッチするか確認する
   - list の結果から id/name が入力と一致するキャラクターを特定
   - 一致するキャラクターがない場合は、利用可能な id/name 一覧を提示してエラー終了
3. 確定した id（または name）を `vox-actor speakers profile --id <id>` に渡して詳細を取得し、JSON 出力を jq でパースする
   - `speakers`: スピーカーIDマップ（スタイル名 → ID）
   - `pronoun`: 人称
   - `speechSuffix`: 語尾パターン
   - `personality`: 性格特性
   - `description`: キャラクター説明本文（性格・口調・セリフ例を含む）を読み取る
4. 種別に応じた台本を生成する
   - `monologue`: 1セリフだけの単一 JSON オブジェクト
   - `speak`: 冒頭→本題→まとめの流れの JSONL（複数行）
   - `talk`: 複数キャラの掛け合い JSONL（複数行）。冒頭で全キャラが1回以上登場するよう構成する
5. セリフ毎に内容の感情に合う `speaker` と `speedScale` を選定する
6. 台本をstdinから `save-script.sh` に渡して、一時ファイルの絶対パスを取得する

```bash
SCRIPT_PATH=$(echo "$script_content" | ${CLAUDE_PLUGIN_ROOT}/skills/act/scripts/save-script.sh <kind>) || exit 1
```

- `<kind>`: 種別（`monologue` / `speak` / `talk`）
- 返り値: `<workspace>/tmp/<unix_ms>_<kind>.<ext>` の絶対パス

7. `play-script.sh <path>` を呼び出して再生する

```bash
${CLAUDE_PLUGIN_ROOT}/skills/act/scripts/play-script.sh <path>
```

- `<path>`（第1引数・必須）: `save-script.sh` が返した台本ファイルの絶対パス

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

キャラクター設定は `vox-actor speakers profile` コマンド経由で取得します。コマンドが返す JSON に以下の情報が含まれます：

- `id`: アセットディレクトリ内のディレクトリ名
- `name`: キャラクター表示名
- `pronoun`: 人称（例："ボク"）
- `speechSuffix`: 語尾パターン（配列、例：["〜のだ", "〜なのだ"]）
- `personality`: 性格特性（配列、例：["元気", "明るい"]）
- `speakers`: スピーカーIDマップ（スタイル名 → VOICEVOX エンジンのスピーカーID）
- `styles`: 利用可能なスタイル名の配列
- `description`: キャラクター説明本文（性格・口調の詳細とセリフ例を含む）

呼び出し元スキル（`monologue` / `speak` / `talk`）がメモリから読み取って引き継いだキャラクター名をもとに、speakers コマンドでプロフィールを取得し、`speakers` マップ・口調・性格を反映した台本を生成します。

## キャラクター取得フロー（list → profile の2段階）

キャラクター指定が曖昧またはタイポの場合に有用なエラーメッセージを提供するため、以下の2段階フローを採用しています：

1. **`vox-actor speakers list`**: 軽量な id/name リストを取得
   - JSON 形式: `[{"id":"zundamon","name":"ずんだもん"}, ...]`
   - 入力されたキャラクター名が完全一致するか確認

2. **存在確認 → `vox-actor speakers profile --id <id>`**: 詳細プロフィール取得
   - 一致した id を使って profile コマンドを呼び出す
   - list に存在しないキャラクターが指定された場合は、利用可能な id/name 一覧を提示してエラー終了

### エラーハンドリング

- `speakers list` コマンド実行失敗時: スキルはエラーメッセージを出力して終了
- list の結果に該当キャラクターがない場合: 利用可能なキャラクター id/name 一覧を提示してエラー終了
- `speakers profile` コマンド実行失敗時: スキルはエラーメッセージを出力して終了
- JSON パース失敗時: エラーの詳細をログ出力して終了

## 前提条件

### CLI コマンドの依存関係

このスキルが使用する CLI コマンドについて、ワークスペース解決および一時ファイル管理は CLI 側に統一されています。詳細は `docs/reference/cli.md` を参照してください。

- **`vox-actor` コマンドのインストール必須**: スクリプト冒頭で `command -v vox-actor` を検査し、未インストール時は `[ERROR] vox-actor コマンドが必要です` を stderr に出して非0終了する
- **ワークスペースの自動解決**: `vox-actor config path.*` は環境変数・git リポジトリ・カレントディレクトリの順で自動判定する
  - 詳細は `docs/reference/cli.md` の `config` サブコマンドセクションを参照
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
