---
name: talk
description: 指示に沿う形のセリフを生成して再生するスキル。
argument-hint: "指示（テーマ、キャラクターなど）"
allowed-tools: Bash(vox-actor *) Bash(scripts/play-script.sh *)
---

## 手順
1. 指示内容に従ってセリフを生成する。
2. `vox-actor script write $(vox-actor config path.tmp)/$(date +%s%3N).jsonl --json '[...]'` でセリフファイルを一括作成する。
3. `scripts/play-script.sh {セリフファイルパス}` で再生する。

## 利用可能なスクリプト
スキルディレクトリの `scripts` フォルダ内に以下のスクリプトを格納しています。
- play-script.sh: セリフファイルを再生するスクリプト。引数にセリフファイルパスを取る。

## コンテキスト
- 使用可能なキャラクター一覧: `vox-actor speakers list`
- キャラクター設定の確認方法: `vox-actor speakers profile --id {profile_id}`
    - `profile_id` は `vox-actor speakers list` の結果から選ぶ
- セリフファイルの一括作成方法: `vox-actor script write {セリフファイルパス} --json '[{セリフオブジェクト}, ...]'`
    - `--json` の各オブジェクトのキー:
        - `text` (string, 必須): セリフ本文
        - `speaker` (int, 省略可): `vox-actor speakers profile` の `speakers` フィールドの値
        - `intonation` (float, 省略可): 抑揚。目安: 0.0（棒読み） 〜 1.0（標準） 〜 1.5（表現豊か）
        - `pitch` (float, 省略可): 音高。目安: -0.05（低い） 〜 0.0（標準） 〜 0.05（高い）
        - `speed` (float, 省略可): 話速。目安: 0.8（ゆっくり） 〜 1.0（標準） 〜 1.2（早口）
- セリフファイルの再生方法: `scripts/play-script.sh {セリフファイルパス}`
- 指示内容: $ARGUMENTS

## 制約
- キャラクターのセリフにはキャラクター設定を反映させる。
- セリフ内容に合わせて抑揚・音高・話速を調整する。
- 各セリフは1〜2文の短文とする。
- 英単語・英語ファイル名は発音を表す日本語に置き換える（拡張子も省略しない）
  - 例: `README.md` → 「リードミードットエムディー」、`config.json` → 「コンフィグドットジェイソン」、`merge` → 「マージ」
- ファイルパス・URL・UUID 等の長い文字列はセリフに含めない
- セリフ内のダブルクォート・バックスラッシュは JSON / JSONL 仕様でエスケープ
- 利用可能なキャラクターが存在しない場合は、 `--speaker` を省略してデフォルト声でセリフ生成する
