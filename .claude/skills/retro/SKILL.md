---
name: retro
description: 直前の作業内容を振り返り、改善点を分析してGitHub Issueを作成します。作業完了後に使用してください。
context: fork
agent: general-purpose
allowed-tools: Read, Grep, Glob, Bash(gh issue create *), Bash(gh issue list *)
disable-model-invocation: false
user-invocable: true
---

振り返りを行います。
実装は行いません。振り返りとIssue作成のみに専念します。

## 禁止事項

- **コードの実装・修正を絶対に行わないこと**（ファイルの編集、Write、Edit ツールの使用は禁止）
- **コミット・プッシュを行わないこと**
- **ブランチの作成・切り替えを行わないこと**
- 改善案は必ずIssueとして作成し、直接実装しないこと
- 振り返り後に「続けて実装しましょう」等と提案しないこと

## ワークフロー

1. 直前の作業内容を振り返る
  - 実施タスク
  - 使用したスキル/エージェント
  - スムーズだった点
  - 問題点
2. 改善案をまとめる
  - 改善することがなければ改善案なしとする
3. 既存のgithub issueを確認し、改善案が既にissue化されていないか確認する
4. issue化されていない改善案をgithub issueとして作成する
5. 振り返りの内容をユーザーに報告する

## 完了条件

- 振り返り内容の報告とIssue作成をもって**完了**とする
- 実装には進まず、ここで終了すること
