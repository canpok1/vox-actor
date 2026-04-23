---
name: monologue
description: 作業開始時、作業終了時、想定外のことが起こった時に呼び出す独り言スキル。キャラクターになりきった一言コメントを通知する。
argument-hint: "[キャラクター名]"
allowed-tools:
  - "Bash(${CLAUDE_PLUGIN_ROOT}/scripts/play-script.sh *)"
  - "Bash(mkdir -p *)"
  - "Bash(echo $((RANDOM % 100 + 1)))"
  - "Read(${CLAUDE_PLUGIN_ROOT}/characters/*.md)"
  - "Write(${VOX_ACTOR_WORKSPACE}/tmp/*)"
---

作業の開始時や終了時に、キャラクターになりきった独り言を通知します。

## 実行フロー

1. `$ARGUMENTS` を独り言のテーマとして受け取る（未指定なら作業の文脈から生成する）
2. メモリから `default_character`（既定 `zundamon`、`speak` スキルと共用）と `monologue_probability`（既定 `100`）を読み取る
3. **確率判定**: `echo $((RANDOM % 100 + 1))` を Bash で実行し、1〜100 の判定値を取得する。判定値が `monologue_probability` より大きい場合はここで終了する（後続のキャラクター設定読み込みも `play-script.sh` 呼び出しも行わない）
4. `${CLAUDE_PLUGIN_ROOT}/characters/<name>.md` を読み、`speakers` 一覧と性格を把握する
5. キャラクターになりきった 1 文程度の独り言セリフを生成し、感情に合う `speaker` と `speedScale` を選定する
6. `mkdir -p "${VOX_ACTOR_WORKSPACE}/tmp"` で一時ディレクトリを作成する
7. 一時ファイル `${VOX_ACTOR_WORKSPACE}/tmp/<unix_ms>_monologue.json` に単一JSONを Write する（形式: `{"text": "...", "speaker": ..., "speedScale": ...}`）
8. `play-script.sh <path>` を呼び出す

```bash
${CLAUDE_PLUGIN_ROOT}/scripts/play-script.sh <json_path>
```

- `<json_path>`（第1引数・必須）: 生成した JSON ファイルの絶対パス

### 通知確率

- 初期値: 100%（10回に10回）
- ユーザーの指示により調整可能（Claudeのメモリシステムに保存する）
  - 例: 「独り言の頻度を30%にして」→ メモリに `monologue_probability: 30` を保存

### 話速の目安

| 値 | 説明 |
|----|------|
| 0.8 | ゆっくり |
| 0.9 | ちょっとゆっくり |
| 1.0 | 普通（デフォルト） |
| 1.1 | ちょっと早口 |
| 1.2 | 早口 |

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
| 通知確率 | `monologue_probability` | 100 | 0〜100の整数。通知する確率（%）。`0` で常に通知されない、`100` で必ず通知される |

## 前提条件

### play-script.sh の依存関係

通知の実行に使用する `${CLAUDE_PLUGIN_ROOT}/scripts/play-script.sh` は以下の前提条件を必要とする:

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

`vox-actor act <json_path>` を同期実行し、再生の完了を待ってスクリプトが戻る。

- VOICEVOXエンジンのURLは `vox-actor` 側の環境変数 `VOX_ENGINE_URL` で解決する（非デフォルトポート時はユーザーが設定）
- 監視プロセス（`vox-actor watch`）の常駐は不要
- 再生完了後、渡された一時ファイルを `rm -f` で削除する
- 同期実行のため、同一セッション内で連続して呼び出した場合は先の発話が完了してから次が再生される。ただし複数セッションから並列に呼ばれた場合はエンジンへの並列リクエストと音声再生の重なりが発生しうる。複数セッション間でも逐次再生したい場合は `VOX_ACTOR_MONOLOGUE_MODE=file` + 監視プロセス構成に切り替える
- `vox-actor act` 失敗時もスクリプト本体はエラー終了せず、標準出力/標準エラーの内容をワークスペースルート直下の `play-script-errors.log` にタイムスタンプ付きで追記する。ログは末尾200行でローテーションされる。失敗検知は `tail -f <workspace>/play-script-errors.log` で行える

#### `file` モード

一時ファイルを `vox-actor config path.queue` で解決された queue ディレクトリ配下へ `mv` で移動する（渡された一時ファイル名がそのまま使われるため、`<unix_ms>_monologue.json` なら `queue/<unix_ms>_monologue.json` となる）。外部の通知監視プロセス（`vox-actor watch` 等）はこの `queue/` ディレクトリを監視対象として検知・読み上げする。

## JSON 出力例

ずんだもん（`speakers.ノーマル: 3`）の独り言例:

```json
{"text": "よーし、始めるのだ！", "speaker": 3, "speedScale": 1.1}
```

## キャラクター設定ファイルについて

`characters/` 配下のキャラクター設定ファイルはClaude向けの自然言語テキストとして消費される。キャラクター名や口調特徴などの主要メタデータはYAML frontmatterに構造化されており、本文にはClaude向けの詳細な性格・口調の説明や独り言の例が記載されている。

## 制約

- 通知コメントは1文程度の短い独り言にすること
- 作業の流れを妨げないよう、スキルの実行は素早く完了すること
- セリフに英単語や英語のファイル名等を含める場合は、英語のまま使わず、発音を表す日本語に置き換えること
  - 拡張子も省略せず発音を表す日本語にすること
  - 例: `README.md` → 「リードミードットエムディー」、`config.json` → 「コンフィグドットジェイソン」、`merge` → 「マージ」、`build` → 「ビルド」
- ファイルパス、URL、UUIDなど長い文字情報は発音しても分かりづらいため、セリフに含めないこと
- JSON 内のダブルクォート・バックスラッシュ等は JSON 仕様に従って適切にエスケープすること
