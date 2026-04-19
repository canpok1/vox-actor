---
name: monologue
description: 作業開始時、作業終了時、想定外のことが起こった時に呼び出す独り言スキル。キャラクターになりきった一言コメントを通知する。
argument-hint: "[キャラクター名]"
allowed-tools:
  - "Bash(${CLAUDE_PLUGIN_ROOT}/skills/monologue/scripts/monologue.sh *)"
  - "Read(${CLAUDE_PLUGIN_ROOT}/skills/monologue/characters/*.md)"
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

`$ARGUMENTS` で指定されたキャラクターの設定ファイルを読み込み、そのキャラクターになりきった一言コメントを生成する。

- `$ARGUMENTS` が空の場合: デフォルトキャラクター「ずんだもん」を使用
- キャラクター設定ファイル: `${CLAUDE_PLUGIN_ROOT}/skills/monologue/characters/{キャラクター名}.md`（例: `${CLAUDE_PLUGIN_ROOT}/skills/monologue/characters/zundamon.md`）

## メモリに保存する設定項目

以下の設定をClaudeのメモリシステム（MEMORY.md）に保存・参照する:

| 項目 | キー | デフォルト値 | 説明 |
|------|------|-------------|------|
| 通知確率 | `monologue_probability` | 100 | 1〜100の整数。通知する確率（%） |

## 前提条件

### monologue.sh の依存関係

通知の実行に使用する `${CLAUDE_PLUGIN_ROOT}/skills/monologue/scripts/monologue.sh` は以下の前提条件を必要とする:

- **環境変数 `VOX_ACTOR_WORKSPACE`**: vox-actor関連ファイルのルートディレクトリを指定する。配下に `queue/`（fileモードの通知JSON出力先）および `monologue-errors.log`（directモードのエラーログ）が置かれる。未設定の場合のデフォルト値は以下のとおり
  - gitリポジトリ内: `<gitリポジトリ直下>/.vox-actor`
  - gitリポジトリ外: `$PWD/.vox-actor`
- **出力先ディレクトリ**: 上記のルートディレクトリおよび配下の `queue/` はスクリプトが必要に応じて自動作成する

### 通知モード

通知方式は以下のルールで自動切替する:

1. 環境変数 `VOX_ACTOR_MONOLOGUE_MODE` が設定されていればそれに従う（`direct` または `file`）
2. 未設定なら `vox-actor` コマンドの有無で判定する（あり → `direct`、なし → `file`）

#### `direct` モード

`vox-actor say --speaker <スピーカーID> --speed <話速> "<セリフ>"` をバックグラウンドで起動し、スクリプトは即座に戻る。

- VOICEVOXエンジンのURLは `vox-actor` 側の環境変数 `VOX_ENGINE_URL` で解決する（非デフォルトポート時はユーザーが設定）
- 監視プロセス（`vox-actor watch`）の常駐は不要
- 複数セッションから並列に呼ばれても許容する（エンジンへの並列リクエストと音声再生の重なりは発生するがエラーにはならない）。逐次再生が必要な場合は `VOX_ACTOR_MONOLOGUE_MODE=file` + 監視プロセス構成に切り替えれば対応可能
- `vox-actor say` 失敗時もスクリプト本体はエラー終了せず、標準出力/標準エラーの内容を `${VOX_ACTOR_WORKSPACE}/monologue-errors.log` にタイムスタンプ付きで追記する。ログは末尾200行でローテーションされる。失敗検知は `tail -f ${VOX_ACTOR_WORKSPACE}/monologue-errors.log` で行える

#### `file` モード

`${VOX_ACTOR_WORKSPACE}/queue/` に通知ファイルを書き出す。ファイル名は `notify_{ミリ秒タイムスタンプ}.json`、形式は `{"speaker": スピーカーID, "text": "セリフ", "speedScale": 話速}`。外部の通知監視プロセス（`vox-actor watch` 等）はこの `queue/` ディレクトリを監視対象として検知・読み上げする

## キャラクター設定ファイルについて

`characters/` 配下のキャラクター設定ファイルはClaude向けの自然言語テキストとして消費される。キャラクター名や口調特徴などの主要メタデータはYAML frontmatterに構造化されており、本文にはClaude向けの詳細な性格・口調の説明や独り言の例が記載されている。

## 制約

- 通知コメントは1文程度の短い独り言にすること
- 作業の流れを妨げないよう、スキルの実行は素早く完了すること
- セリフに英単語や英語のファイル名等を含める場合は、英語のまま使わず、発音を表す日本語に置き換えること
  - 拡張子も省略せず発音を表す日本語にすること
  - 例: `README.md` → 「リードミードットエムディー」、`config.json` → 「コンフィグドットジェイソン」、`merge` → 「マージ」、`build` → 「ビルド」
- ファイルパス、URL、UUIDなど長い文字情報は発音しても分かりづらいため、セリフに含めないこと
