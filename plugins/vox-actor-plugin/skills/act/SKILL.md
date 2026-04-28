---
name: act
description: 指示に沿う形のセリフを生成して再生するスキル。
argument-hint: "指示（テーマ、キャラクターなど）"
allowed-tools:
  - "Bash(vox-actor *)"
  - "Bash(${CLAUDE_PLUGIN_ROOT}/skills/act/scripts/play-script.sh *)"
---

## 手順
1. 指示内容に従ってセリフを生成する。
2. セリフファイルにセリフを追記する。
3. セリフファイルを再生する。

## コンテキスト
- セリフファイルパス: !`vox-actor config path.tmp`/!`date +%s%3N`.jsonl
- セリフファイルへのセリフ追記方法: `vox-actor say -o {セリフファイルパス} {オプション} "{セリフ}"`
    - 話者指定オプション（省略化）: `--speaker {speaker_id}`
        - `speaker_id` は `vox-actor speakers list` の結果から選ぶ
    - 抑揚変更オプション（省略化）: `--intonation {0.0〜2.0}`
        - 目安: 0.0（棒読み） 〜 1.0（標準） 〜 2.0（表現豊か）
    - 音高変更オプション（省略化）: `--pitch {-0.1〜0.1}`
        - 目安: -0.1（低い） 〜 0.0（標準） 〜 0.1（高い）
    - 話速変更オプション（省略化）: `--speed {0.8〜1.2}`
        - 目安: 0.8（ゆっくり） 〜 1.0（標準） 〜 1.2（早口）
- セリフファイルの再生方法: `play-script.sh {セリフファイルパス}`
- キャラクター設定の確認方法: `vox-actor speakers profile --id {speaker_id}`
    - `speaker_id` は `vox-actor speakers list` の結果から選ぶ
- 指示内容: $ARGUMENTS

## 制約
- キャラクターのセリフにはキャラクター設定を反映させる。
- セリフ内容に合わせて抑揚・音高・話速を調整する。
- 各セリフは1〜2文の短文とする。
- 英単語・英語ファイル名は発音を表す日本語に置き換える（拡張子も省略しない）
  - 例: `README.md` → 「リードミードットエムディー」、`config.json` → 「コンフィグドットジェイソン」、`merge` → 「マージ」
- ファイルパス・URL・UUID 等の長い文字列はセリフに含めない
- セリフ内のダブルクォート・バックスラッシュは JSON / JSONL 仕様でエスケープ
- `vox-actor` 未インストール時はスクリプトが `[ERROR] vox-actor コマンドが必要です` を出して非0終了する
- `speakers list` / `profile` 失敗時、または該当キャラ無しの場合は利用可能な id/name 一覧を提示してエラー終了する
