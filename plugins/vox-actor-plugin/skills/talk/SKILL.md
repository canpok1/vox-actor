---
name: talk
description: 渡されたトピックを複数キャラクターの会話形式JSONL台本として生成し、vox-actor経由で掛け合い・対話・漫才風の読み上げを届けるスキル。
argument-hint: "<内容>"
allowed-tools:
  - "Skill"
---

## 実行フロー

1. `$ARGUMENTS` を読み上げ内容として受け取る
2. メモリから `talk_characters`（既定 `[zundamon, metan]`）と `talk_length`（既定 `medium`）を読み取る
3. 内容に応じて役割配分（解説役／聞き役／ツッコミ役 など）を組み立て、「冒頭で全キャラが登場し、本題は掛け合い・質問応答・補足で会話を展開する」演出指示と、長さ設定に応じたセリフ数の目安を確立する
4. Skill ツールで `vox-actor-plugin:act` を呼び出し、種別 `talk`・本文・会話キャラクター一覧と役割配分・長さ設定を引き継ぐ

## 読み上げの長さ

| 設定 | セリフ数の目安 | 想定再生時間 |
|------|---------------|------------|
| `short` | 4〜6 | 〜30秒 |
| `medium`（既定） | 8〜12 | 1〜2分 |
| `long` | 14+ | 数分 |

## メモリに保存する設定項目

| 項目 | キー | デフォルト値 | 説明 |
|------|------|-------------|------|
| 会話キャラクター | `talk_characters` | `[zundamon, metan]` | `characters/<name>.md` の `<name>` の配列（2〜4人）。本スキル専用で `default_character` とは独立 |
| 会話の長さ | `talk_length` | `medium` | `short` / `medium` / `long` |
