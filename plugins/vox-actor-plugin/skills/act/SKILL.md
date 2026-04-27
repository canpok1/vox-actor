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

## 入力

- `$ARGUMENTS` 先頭から **種別**（`monologue` / `speak` / `talk`）と **本文** を取り出す
- **キャラクター名**・**長さ**（`short` / `medium` / `long`、speak/talk の場合）は会話コンテキストから引き継ぐ（呼び出し元がメモリ参照済み）

## 実行フロー

1. `vox-actor speakers list` で利用可能キャラ一覧（`[{"id":..., "name":...}, ...]`）を取得し、引き継いだキャラクター名と id/name の完全一致を確認する。一致なしなら一覧を提示してエラー終了
2. `vox-actor speakers profile --id <id>` で詳細を取得し、jq でパースする。台本生成では特に以下を使う
   - `speakers`: スタイル名 → スピーカーID のマップ（`speaker` 値の選定に使用）
   - `pronoun` / `speechSuffix` / `personality` / `description`: 口調・性格の反映
3. 種別に応じて台本を生成する（形式は後述）
   - `monologue`: 単一 JSON
   - `speak`: 冒頭→本題→まとめの JSONL
   - `talk`: 複数キャラの掛け合い JSONL（冒頭で全キャラが1回以上登場）
4. セリフ毎に内容の感情に合う `speaker` と `speedScale` を選ぶ
5. 台本を stdin から `save-script.sh` に渡し、一時ファイルの絶対パスを取得する

   ```bash
   SCRIPT_PATH=$(echo "$script_content" | ${CLAUDE_PLUGIN_ROOT}/skills/act/scripts/save-script.sh <kind>) || exit 1
   ```

   - `<kind>`: `monologue` / `speak` / `talk`
   - 返り値: `<workspace>/tmp/<unix_ms>_<kind>.<ext>` の絶対パス

6. `play-script.sh <path>` で再生する

   ```bash
   ${CLAUDE_PLUGIN_ROOT}/skills/act/scripts/play-script.sh <path>
   ```

   通知モードの自動切替（direct/file）・ワークスペース解決・エラーログ等は CLI 側で扱う。詳細は `docs/reference/cli.md` を参照。

## 台本の形式

各セリフは1行の JSON オブジェクト。`speak` / `talk` は複数行の JSONL、`monologue` は単一 JSON。

```json
{"text": "セリフ本文", "speaker": <speaker_id>, "speedScale": <0.8〜1.2>}
```

- `text`: 1〜2文の短文
- `speaker`: profile の `speakers` マップから感情に合う ID を選ぶ
- `speedScale`: 下表から選ぶ

| 値 | 0.8 | 0.9 | 1.0 | 1.1 | 1.2 |
|----|-----|-----|-----|-----|-----|
| 説明 | ゆっくり | ちょっとゆっくり | 普通 | ちょっと早口 | 早口 |

### 例

`monologue`（ずんだもん `ノーマル: 3`）:

```json
{"text": "よーし、始めるのだ！", "speaker": 3, "speedScale": 1.1}
```

`speak`（1キャラ・冒頭→本題→まとめ）:

```jsonl
{"text": "クロージャって何なのだ？説明するのだ！", "speaker": 3, "speedScale": 1.1}
{"text": "簡単に言うと、関数が作られた時の周りの変数を覚えておく仕組みなのだ", "speaker": 3, "speedScale": 1.0}
{"text": "むむっ、ちょっと難しいけど…例えるならお弁当箱に具材を詰めて持ち歩く感じなのだ", "speaker": 3, "speedScale": 1.0}
{"text": "後から開けても中身がそのまま残ってるみたいに、変数の値も残るのだ〜", "speaker": 1, "speedScale": 0.9}
{"text": "わかったかな？お疲れ様なのだ！", "speaker": 1, "speedScale": 1.0}
```

`talk`（ずんだもん `ノーマル: 3`/`あまあま: 1` × 四国めたん `ノーマル: 2`/`ツンツン: 6`）:

```jsonl
{"text": "今日はクロージャについて解説するのだ！", "speaker": 3, "speedScale": 1.1}
{"text": "あら、わたくしも勉強させてもらおうかしら", "speaker": 2, "speedScale": 1.0}
{"text": "簡単に言うと、関数が作られた時の周りの変数を覚えておく仕組みなのだ", "speaker": 3, "speedScale": 1.0}
{"text": "なるほど、お弁当箱みたいなものですわね", "speaker": 2, "speedScale": 1.0}
{"text": "そう、そんな感じなのだー", "speaker": 1, "speedScale": 0.9}
{"text": "よく分かりましたわ。ありがとう、ずんだもん", "speaker": 2, "speedScale": 1.0}
```

## 制約

- セリフ毎に1〜2文の短文（長文1セリフは合成・再生で詰まる）
- 英単語・英語ファイル名は発音を表す日本語に置き換える（拡張子も省略しない）
  - 例: `README.md` → 「リードミードットエムディー」、`config.json` → 「コンフィグドットジェイソン」、`merge` → 「マージ」
- ファイルパス・URL・UUID 等の長い文字列はセリフに含めない
- セリフ内のダブルクォート・バックスラッシュは JSON / JSONL 仕様でエスケープ
- `talk` は同一キャラの連続発話可だが、冒頭で全キャラが1回以上登場し、ターン交代を意識する
- `vox-actor` 未インストール時はスクリプトが `[ERROR] vox-actor コマンドが必要です` を出して非0終了する
- `speakers list` / `profile` 失敗時、または該当キャラ無しの場合は利用可能な id/name 一覧を提示してエラー終了する
