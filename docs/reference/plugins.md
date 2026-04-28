# プラグイン／スキルリファレンス

claude code に `vox-actor-plugin` を導入すると、以下のスラッシュコマンドが利用できます。

## 対応キャラクター

キャラクター設定は本リポジトリには同梱しておらず、`vox-actor assets download` で別リポジトリから取得します。利用規約とクレジット表記は取得元リポジトリの案内に従ってください。

利用可能なキャラクター一覧は `vox-actor speakers list`、各キャラクターの詳細プロフィール（声のスタイル、性格、口調）は `vox-actor speakers profile --id <id>` で確認できます。

## `/vox-actor-plugin:talk <指示>`

指示に沿うセリフを生成して再生する汎用スキルです。1文の独り言から、複数キャラクターの掛け合い、解説・朗読・結果報告まで、指示内容に応じて構成を変えて読み上げます。

```
/vox-actor-plugin:talk <指示>
```

- `<指示>`: テーマ、登場キャラクター、用途（独り言／会話／解説 など）を自由記述で渡します
- セリフは JSONL 台本として一時ファイルに書き出され、`scripts/play-script.sh` 経由で再生されます
- 再生方式は `direct` / `file` モードで自動切替されます（[再生モード](#再生モードdirectfile) を参照）

### 呼び出し例

```
# 1文の独り言
/vox-actor-plugin:talk ずんだもんがタスク完了の独り言

# 複数キャラの会話
/vox-actor-plugin:talk ずんだもんとめたんでクロージャを解説

# まとまった長さの解説
/vox-actor-plugin:talk クロージャとは何か、ずんだもんが解説
```

### JSONL出力例

ずんだもん（`speakers.ノーマル: 3`、`あまあま: 1`）と四国めたん（`ノーマル: 2`、`ツンツン: 6`）の会話例:

```jsonl
{"text": "今日はクロージャについて解説するのだ！", "speaker": 3, "speedScale": 1.1}
{"text": "あら、わたくしも勉強させてもらおうかしら", "speaker": 2, "speedScale": 1.0}
{"text": "簡単に言うと、関数が作られた時の周りの変数を覚えておく仕組みなのだ", "speaker": 3, "speedScale": 1.0}
{"text": "なるほど、お弁当箱みたいなものですわね", "speaker": 2, "speedScale": 1.0}
{"text": "そう、そんな感じなのだー", "speaker": 1, "speedScale": 0.9}
{"text": "よく分かりましたわ。ありがとう、ずんだもん", "speaker": 2, "speedScale": 1.0}
```

## 再生モード（direct／file）

`talk` スキルの再生方式は、実行環境に応じて2つのモードを自動切替します。

以下のようなケースでは、`watch` コマンドを常駐させる `file` モードが適しています。

- claude code がコンテナ／リモート環境で動作し、`vox-actor` コマンドはホスト側にしか存在しない
- 複数セッションから同時に呼ばれても音声を逐次再生したい（`direct` モードは同一セッション内では逐次だが、セッション間では並列再生になる）

### `direct` モードと `file` モードの違い

| 観点 | `direct` モード | `file` モード |
|---|---|---|
| 読み上げ方式 | `vox-actor act` をその場で直接呼び出す | JSONL ファイルをキューに移動し、別プロセスの `vox-actor watch` が読み上げる |
| 監視プロセス | 不要 | `vox-actor watch` の常駐が必要 |
| ファイル出力先 | エラーログのみ（ワークスペースルート直下の `play-script-errors.log`） | 通知ファイル（ワークスペース配下の `queue/*.jsonl`）＋エラーログ |
| 同時呼び出し時 | 同一セッション内は逐次再生、複数セッション間は並列再生 | 検知順に逐次再生 |
| 前提 | `vox-actor` コマンドが `PATH` 上にあり、音声デバイスが利用可能（`vox-actor audio-check` が成功） | claude code 側と `vox-actor watch` 側で `VOX_ACTOR_WORKSPACE` を共有 |

> **前提**: `vox-actor` コマンドのインストールは必須です（`scripts/play-script.sh` は起動時に `command -v vox-actor` で存在確認し、未インストール時は `[ERROR] vox-actor コマンドが必要です` を stderr に出して非0終了します）。
>
> **ワークスペースの解決**: スクリプトは `vox-actor config path.workspace` / `vox-actor config path.queue` を呼ぶだけで、環境変数の分岐は CLI 側が担います。CLI の解決順は以下のとおりです。
> 1. `VOX_ACTOR_WORKSPACE` が設定されていればその値をワークスペースルートとして使う（queue は `${VOX_ACTOR_WORKSPACE}/queue`）
> 2. 未設定なら gitリポジトリ直下の `.vox-actor` をワークスペースルートとする（queue は `<repo>/.vox-actor/queue`）
>
> git 管理外ディレクトリで `VOX_ACTOR_WORKSPACE` を明示しないまま実行すると CLI が非0終了し、そのエラーがユーザーに表示されてスクリプトも終了します。git 外で利用する場合は `VOX_ACTOR_WORKSPACE` を明示してください。`file` モードでホストとLLM実行環境を分ける場合も、双方から参照可能な共有パスを明示的に指定します。

モードは `VOX_ACTOR_MONOLOGUE_MODE` 環境変数で明示するか、未設定時は `vox-actor audio-check` の終了コードで自動判定されます（0 → `direct`、非0 → `file`）。

### `file` モードのセットアップ（音声デバイス利用不可環境での利用）

claude code をコンテナ／リモートで動かし、音声デバイスはホスト側にのみ存在する構成などで利用します。

1. **ホスト側で監視プロセスを常駐させる**
   ```bash
   # vox-actor 関連ファイルのルートディレクトリを指定（vox-actor CLI が VOX_ACTOR_WORKSPACE を解釈する）
   export VOX_ACTOR_WORKSPACE=/path/to/shared/directory
   # 通知ファイルの配置先は vox-actor config path.queue で解決できる
   vox-actor watch "$(vox-actor config path.queue)"
   ```

2. **claude code 側で `file` モードを明示し、共有ディレクトリを指定する**
   ```bash
   export VOX_ACTOR_WORKSPACE=/path/to/shared/directory
   # vox-actorコマンドがある環境で file モードを強制したい場合は以下も指定
   export VOX_ACTOR_MONOLOGUE_MODE=file
   ```

3. **claude code にプラグインを導入する**（README のクイックスタートと同じ手順）

### ディレクトリ監視モードの詳細（`watch` コマンド）

`vox-actor watch` は、配置されたテキストファイルや JSON 台本を自動で読み上げます。処理済みファイルは各監視ディレクトリ直下の `done/` サブディレクトリに移動されます（例: `./dir-a/foo.txt` → `./dir-a/done/foo.txt`）。

```bash
vox-actor watch /path/to/watch-dir
```

複数ディレクトリを同時に監視する場合はスペース区切りで指定します。各ディレクトリは並列で監視され、検知したファイルは検知順に1件ずつ再生されます。

```bash
vox-actor watch /path/to/dir-a /path/to/dir-b
```

処理済みファイルを `done/` に移動する代わりに削除する場合:

```bash
vox-actor watch --delete /path/to/watch-dir
```

別のターミナルからファイルを配置すると、自動的に読み上げられます。

```bash
echo "こんばんは" > /path/to/watch-dir/sample.txt
```

`watch` コマンドは `Ctrl+C`（SIGINT）または SIGTERM で停止できる。

### `act --watch` / `act --watch-delete`（後方互換）

従来どおり `act` コマンドでも単一ディレクトリの監視が可能です。

```bash
vox-actor act --watch /path/to/watch-dir
vox-actor act --watch-delete /path/to/watch-dir
```

`--watch` と `--watch-delete` は同時に指定できません。複数ディレクトリを同時に監視したい場合は `watch` コマンドを使ってください。

### エラーログ

`direct` モードでの失敗は以下のログに追記されます（末尾200行でローテーション）。`tail -f` で確認できます。`vox-actor config path.workspace` で解決されるワークスペースルート直下に配置されます。

- `<workspace>/play-script-errors.log`
