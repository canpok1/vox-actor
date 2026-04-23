# プラグイン／スキルリファレンス

claude code に `vox-actor-plugin` を導入すると、以下のスラッシュコマンドが利用できます。

## 対応キャラクター一覧

`monologue` / `speak` / `talk` スキルで利用できるキャラクターは `plugins/vox-actor-plugin/skills/act/characters/` に設定ファイルとして同梱されています。キャラクター名（`<name>`）は `/vox-actor-plugin:monologue <name>` の引数や、`monologue` / `speak` 共通の `default_character`、`talk` 用の `talk_characters` メモリ設定で指定します。

| キャラクター名 | `<name>` | 分類 | 特徴 |
|---|---|---|---|
| ずんだもん（既定） | `zundamon` | — | 元気で明るい／語尾「〜のだ」 |
| 四国めたん | `metan` | 女性 | お嬢様口調／ずんだもんの定番相方 |
| 春日部つむぎ | `tsumugi` | 女性 | 元気な女子高生／親しみやすい |
| 青山龍星 | `ryusei` | 男性 | 低音ボイス／落ち着いた口調 |
| 玄野武宏 | `takehiro` | 男性 | 熱血系／感情バリエーション豊富 |
| ナースロボ＿タイプＴ | `nurserobo_t` | 機械 | 無感情寄り／ロボット的 |

## `/vox-actor-plugin:monologue`

作業開始／終了／想定外のことが起こった時など、節目のキャラクターの一言独り言を読み上げます。

```
/vox-actor-plugin:monologue [キャラクター名]
```

- 1文程度の短い独り言を生成し、キャラクターになりきって読み上げます
- キャラクター設定ファイルの `speakers` からセリフの感情に合うスピーカーIDを選定します
- 再生方式は `direct` / `file` モードで自動切替されます（[再生モード](#再生モードdirectfile) を参照）

### メモリ設定

| 項目 | キー | デフォルト値 | 説明 |
|------|------|-------------|------|
| デフォルトキャラクター | `default_character` | `zundamon` | `characters/<name>.md` の `<name>`。`speak` スキルと共用。引数指定があればそちらが優先 |
| 通知確率 | `monologue_probability` | `100` | 1〜100の整数。通知する確率（%） |

ユーザー指示（例: 「独り言の頻度を30%にして」）で更新すると、以降の実行に反映されます。

## `/vox-actor-plugin:speak <内容>`

渡した内容を冒頭→本題→まとめの流れで、複数セリフのJSONL台本としてキャラクターに読み上げさせます。解説・朗読・作業結果の報告・ストーリーテリングなど、まとまった長さの音声アウトプット全般に利用できます。`monologue` が1文の独り言用であるのに対し、本スキルはまとまった長さを扱います。

```
/vox-actor-plugin:speak <内容>
```

- `<内容>`: 読み上げてほしい概念・メモ・調査結果・文章などを自由記述で渡します
- 生成されたJSONL台本は一時ファイルに書き出され、`vox-actor act` で再生されます
- 再生方式は `direct` / `file` モードで自動切替されます（[再生モード](#再生モードdirectfile) を参照）

### メモリ設定

以下の設定を claude code のメモリに保存しておくと、次回以降の実行に反映されます。

| 項目 | キー | デフォルト値 | 値 | 説明 |
|------|------|-------------|----|-----|
| デフォルトキャラクター | `default_character` | `zundamon` | `characters/<name>.md` の `<name>` | `monologue` スキルと共用。例: 「読み上げはめたんで」→ `metan` を保存 |
| 読み上げの長さ | `speak_length` | `medium` | `short` / `medium` / `long` | 下記の長さ表を参照 |

#### 読み上げの長さ

| 設定 | セリフ数の目安 | 想定再生時間 |
|------|---------------|------------|
| `short` | 3〜5 | 〜十数秒 |
| `medium`（既定） | 6〜10 | 30秒〜1分 |
| `long` | 10+ | 数分 |

### 呼び出し例

```
/vox-actor-plugin:speak クロージャとは何か
```

### JSONL出力例

解説用途の例（ずんだもん）。朗読・結果報告・ストーリーテリングでも同じJSONL形式で台本が生成されます:

```jsonl
{"text": "クロージャって何なのだ？説明するのだ！", "speaker": 3, "speedScale": 1.1}
{"text": "簡単に言うと、関数が作られた時の周りの変数を覚えておく仕組みなのだ", "speaker": 3, "speedScale": 1.0}
{"text": "むむっ、ちょっと難しいけど…例えるならお弁当箱に具材を詰めて持ち歩く感じなのだ", "speaker": 3, "speedScale": 1.0}
{"text": "後から開けても中身がそのまま残ってるみたいに、変数の値も残るのだ〜", "speaker": 1, "speedScale": 0.9}
{"text": "わかったかな？お疲れ様なのだ！", "speaker": 1, "speedScale": 1.0}
```

## `/vox-actor-plugin:talk <内容>`

渡した内容を、複数キャラクターが会話する形式のJSONL台本として生成し、掛け合い・対話・漫才風・ニュース番組風などの読み上げを行います。`speak` が1キャラでまとまった長さを語るのに対し、本スキルは2〜4人のキャラによる役割配分と会話の流れを構成します。

```
/vox-actor-plugin:talk <内容>
```

- `<内容>`: 会話のトピックにしたい概念・メモ・調査結果・文章などを自由記述で渡します
- 生成されたJSONL台本は一時ファイルに書き出され、`vox-actor act` で再生されます
- 再生方式は `direct` / `file` モードで自動切替されます（[再生モード](#再生モードdirectfile) を参照）

### メモリ設定

以下の設定を claude code のメモリに保存しておくと、次回以降の実行に反映されます。

| 項目 | キー | デフォルト値 | 値 | 説明 |
|------|------|-------------|----|-----|
| 会話キャラクター | `talk_characters` | `[zundamon, metan]` | `characters/<name>.md` の `<name>` の配列（2〜4人） | 本スキル専用。`default_character` とは別に管理 |
| 会話の長さ | `talk_length` | `medium` | `short` / `medium` / `long` | 下記の長さ表を参照 |

#### 会話の長さ

| 設定 | セリフ数の目安 | 想定再生時間 |
|------|---------------|------------|
| `short` | 4〜6 | 〜30秒 |
| `medium`（既定） | 8〜12 | 1〜2分 |
| `long` | 14+ | 数分 |

### 呼び出し例

```
/vox-actor-plugin:talk クロージャとは何か
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

`monologue` / `speak` / `talk` スキルの再生方式は、実行環境に応じて2つのモードを自動切替します。

以下のようなケースでは、`watch` コマンドを常駐させる `file` モードが適しています。

- claude code がコンテナ／リモート環境で動作し、`vox-actor` コマンドはホスト側にしか存在しない
- 複数セッションから同時に呼ばれても音声を逐次再生したい（`direct` モードは同一セッション内では逐次だが、セッション間では並列再生になる）

### `direct` モードと `file` モードの違い

| 観点 | `direct` モード | `file` モード |
|---|---|---|
| 読み上げ方式 | `vox-actor say` をその場で直接呼び出す | テキスト等をファイルに書き出し、別プロセスの `vox-actor watch` が読み上げる |
| 監視プロセス | 不要 | `vox-actor watch` の常駐が必要 |
| ファイル出力先 | エラーログのみ（`monologue` / `speak` / `talk` 共通でワークスペースルート直下の `play-script-errors.log`） | 通知ファイル（ワークスペース配下の `queue/*_monologue.json` / `*_speak.jsonl` / `*_talk.jsonl`）＋エラーログ |
| 同時呼び出し時 | 同一セッション内は逐次再生、複数セッション間は並列再生 | 検知順に逐次再生 |
| 前提 | `vox-actor` コマンドが `PATH` 上にあり、音声デバイスが利用可能（`vox-actor audio-check` が成功） | claude code 側と `vox-actor watch` 側で `VOX_ACTOR_WORKSPACE` を共有 |

> **前提**: `vox-actor` コマンドのインストールは必須です（両スクリプトは起動時に `command -v vox-actor` で存在確認し、未インストール時は `[ERROR] vox-actor コマンドが必要です` を stderr に出して非0終了します）。
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

- `monologue` / `speak` / `talk` スキル共通: `<workspace>/play-script-errors.log`
