---
name: speak
description: 渡されたトピック・メモ・作業結果などをキャラクター付きの複数セリフJSONL台本として生成し、vox-actor経由で解説・朗読・結果報告を音声で届けるスキル。
argument-hint: "<内容>"
allowed-tools:
  - "Skill"
---

ユーザーから渡された内容を、キャラクター1人がまとまった長さで読み上げる入口スキル。台本生成や再生実行は `vox-actor-plugin:act` スキルへ委譲する。

## 実行フロー

1. `$ARGUMENTS` を読み上げ内容として受け取る
2. メモリから `default_character`（既定 `zundamon`）と `speak_length`（既定 `medium`）を読み取る
3. 「冒頭の挨拶／つかみ → 本題 → まとめの流れで、キャラクター1人がまとまった長さで語る」演出指示と、長さ設定に応じたセリフ数の目安を確立する
4. Skill ツールで `vox-actor-plugin:act` を呼び出し、種別 `speak`・本文・キャラクター情報・長さ設定を引き継ぐ

## 読み上げの長さ

| 設定 | セリフ数の目安 | 想定再生時間 |
|------|---------------|------------|
| `short` | 3〜5 | 〜十数秒 |
| `medium`（既定） | 6〜10 | 30秒〜1分 |
| `long` | 10+ | 数分 |

## メモリに保存する設定項目

| 項目 | キー | デフォルト値 | 説明 |
|------|------|-------------|------|
| デフォルトキャラクター | `default_character` | `zundamon` | `characters/<name>.md` の `<name>`。`monologue` スキルと共用 |
| 読み上げの長さ | `speak_length` | `medium` | `short` / `medium` / `long` |
