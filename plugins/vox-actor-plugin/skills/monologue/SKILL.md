---
name: monologue
description: 作業開始時、作業終了時、想定外のことが起こった時に呼び出す独り言スキル。キャラクターになりきった一言コメントを通知する。
argument-hint: "[キャラクター名]"
allowed-tools:
  - "Bash(${CLAUDE_PLUGIN_ROOT}/skills/monologue/scripts/monologue.sh *)"
  - "Read(${CLAUDE_PLUGIN_ROOT}/characters/*.md)"
---

作業の開始時や終了時に、キャラクターになりきった独り言を通知します。

## 通知の仕組み

1. キャラクター設定に基づいた一言コメントを生成する
2. `monologue.sh` にメモリから取得した通知確率とセリフを渡して呼び出す
3. スクリプト内で乱数判定が行われ、確率に基づいて通知される

### 通知確率

- 初期値: 100%（10回に10回）
- ユーザーの指示により調整可能（Claudeのメモリシステムに保存する）
  - 例: 「独り言の頻度を30%にして」→ メモリに `monologue_probability: 30` を保存

### 通知の実行

1. キャラクター設定ファイルの `speakers` フィールドを参照し、生成したセリフの感情に最も合うスピーカーIDを選定する
2. 通知確率（第1引数）、セリフ（第2引数）、スピーカーID（第3引数）、話速（第4引数）を `monologue.sh` に渡す

```bash
${CLAUDE_PLUGIN_ROOT}/skills/monologue/scripts/monologue.sh 通知確率 "（キャラクターの一言コメント）" スピーカーID 話速
```

- 通知確率（第1引数・必須）: 1〜100の整数。メモリに保存された確率を渡す
- セリフ（第2引数・必須）: キャラクターの一言コメント
- スピーカーID（第3引数・必須）: キャラクター設定ファイルの `speakers` フィールドから、セリフの感情に最も合うスピーカーIDを選定する
- 話速（第4引数・必須）: セリフの感情やキャラクターの状態に合わせて選定する

#### 話速の目安

| 値 | 説明 |
|----|------|
| 0.8 | ゆっくり |
| 0.9 | ちょっとゆっくり |
| 1.0 | 普通（デフォルト） |
| 1.1 | ちょっと早口 |
| 1.2 | 早口 |

スクリプト内の乱数判定により通知される。

## キャラクター設定

以下の優先順位でキャラクターを解決し、そのキャラクターになりきった一言コメントを生成する。

1. `$ARGUMENTS` が指定されていればそれを使用
2. 指定がなければメモリの `default_character` を使用（`speak` スキルと共用）
3. メモリ未設定なら `zundamon` を使用

- キャラクター設定ファイル: `${CLAUDE_PLUGIN_ROOT}/characters/{キャラクター名}.md`（例: `${CLAUDE_PLUGIN_ROOT}/characters/zundamon.md`）

## メモリに保存する設定項目

以下の設定をClaudeのメモリシステム（MEMORY.md）に保存・参照する:

| 項目 | キー | デフォルト値 | 説明 |
|------|------|-------------|------|
| デフォルトキャラクター | `default_character` | `zundamon` | `characters/<name>.md` の `<name>`。`speak` スキルと共用 |
| 通知確率 | `monologue_probability` | 100 | 1〜100の整数。通知する確率（%） |

## 前提条件

### monologue.sh の依存関係

通知の実行に使用する `${CLAUDE_PLUGIN_ROOT}/skills/monologue/scripts/monologue.sh` は以下の前提条件を必要とする:

- **`vox-actor` コマンドのインストール必須**: スクリプト冒頭で `command -v vox-actor` を検査し、未インストール時は `[ERROR] vox-actor コマンドが必要です` を stderr に出して非0終了する
- **ワークスペースの解決**: スクリプトは `vox-actor config path.queue` / `vox-actor config path.workspace` を呼ぶだけで、環境変数の分岐は CLI 側が担う。CLI の解決順は以下のとおり
  1. 環境変数 `VOX_ACTOR_WORKSPACE` が設定されていればその値をワークスペースルートとして使う（queue は `${VOX_ACTOR_WORKSPACE}/queue`）
  2. 未設定なら gitリポジトリ直下の `.vox-actor` をワークスペースルートとする（queue は `<repo>/.vox-actor/queue`）
- **git 管理外で利用する場合**: `VOX_ACTOR_WORKSPACE` の明示が必要。未指定のままだと CLI が非0終了し、そのエラーメッセージが表示されてスクリプトも終了する
- **出力先ディレクトリ**: ワークスペースルートおよび配下の `queue/` はスクリプトが必要に応じて自動作成する
- `monologue-errors.log`（directモードのエラーログ）はワークスペースルート直下に配置される

### 通知モード

通知方式は以下のルールで自動切替する:

1. 環境変数 `VOX_ACTOR_MONOLOGUE_MODE` が設定されていればそれに従う（`direct` または `file`）
2. 未設定なら `vox-actor audio-check` の終了コードで自動判定する
   - 終了コード 0（音声デバイス open 成功）→ `direct`
   - 非0（音声デバイス open 失敗）→ `file`

#### `direct` モード

`vox-actor say --speaker <スピーカーID> --speed <話速> "<セリフ>"` を同期実行し、再生の完了を待ってスクリプトが戻る。

- VOICEVOXエンジンのURLは `vox-actor` 側の環境変数 `VOX_ENGINE_URL` で解決する（非デフォルトポート時はユーザーが設定）
- 監視プロセス（`vox-actor watch`）の常駐は不要
- 同期実行のため、同一セッション内で連続して呼び出した場合は先の発話が完了してから次が再生される。ただし複数セッションから並列に呼ばれた場合はエンジンへの並列リクエストと音声再生の重なりが発生しうる。複数セッション間でも逐次再生したい場合は `VOX_ACTOR_MONOLOGUE_MODE=file` + 監視プロセス構成に切り替える
- `vox-actor say` 失敗時もスクリプト本体はエラー終了せず、標準出力/標準エラーの内容をワークスペースルート直下の `monologue-errors.log` にタイムスタンプ付きで追記する。ログは末尾200行でローテーションされる。失敗検知は `tail -f <workspace>/monologue-errors.log` で行える

#### `file` モード

`vox-actor config path.queue` で解決された queue ディレクトリに通知ファイルを書き出す。ファイル名は `{ミリ秒タイムスタンプ}_monologue.json`、形式は `{"speaker": スピーカーID, "text": "セリフ", "speedScale": 話速}`。外部の通知監視プロセス（`vox-actor watch` 等）はこの `queue/` ディレクトリを監視対象として検知・読み上げする

## キャラクター設定ファイルについて

`characters/` 配下のキャラクター設定ファイルはClaude向けの自然言語テキストとして消費される。キャラクター名や口調特徴などの主要メタデータはYAML frontmatterに構造化されており、本文にはClaude向けの詳細な性格・口調の説明や独り言の例が記載されている。

## 制約

- 通知コメントは1文程度の短い独り言にすること
- 作業の流れを妨げないよう、スキルの実行は素早く完了すること
- セリフに英単語や英語のファイル名等を含める場合は、英語のまま使わず、発音を表す日本語に置き換えること
  - 拡張子も省略せず発音を表す日本語にすること
  - 例: `README.md` → 「リードミードットエムディー」、`config.json` → 「コンフィグドットジェイソン」、`merge` → 「マージ」、`build` → 「ビルド」
- ファイルパス、URL、UUIDなど長い文字情報は発音しても分かりづらいため、セリフに含めないこと
