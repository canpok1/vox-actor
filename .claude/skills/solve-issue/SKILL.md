---
name: solve-issue
description: ユーザーが `/solve-issue` で手動実行した場合のみ使用。GitHub Issueの対応を行うスキル。実装、自己レビュー、PR作成、マージ、振り返りまで一連の流れを一気に行う。
agent: general-purpose
allowed-tools: Skill, Agent, Bash, Read, Grep, Glob, Write, Edit
disable-model-invocation: true
user-invocable: true
argument-hint: "[issue-number]"
---

GitHub Issue $ARGUMENTS を対応します。

1. `check-issue-status` スキルでIssueの対応状況を確認する
  - `CLOSED`: 「Issue #$ARGUMENTS は既にクローズ済みです」と報告して処理を終了する
  - `ALREADY_DONE`: 以下の手順でIssueをクローズし、ステップ7（振り返り）のみ実施して完了する
    - `gh issue close $ARGUMENTS --repo {owner}/{repo} --comment "対応済みPR #{PR番号} が既にマージされているためクローズします。\n\nPR: {PR URL}"`
  - `IN_PROGRESS`: 新しい実装・PR作成はスキップし、既存PR番号を使ってステップ6（fix-pr）から再開する
  - `PREVIOUSLY_ABANDONED` / `NOT_STARTED`: 次のステップに進む
2. Issue の内容を把握する
3. 実装する
4. `/review` スキルで自己レビュー（コード品質 + ドキュメント整合性チェック）を行う
5. `commit-push-pr` スキルでPRを作成する
6. `/fix-pr` スキルでCI待機・レビュー対応・マージを行う
  - 引数にPR番号を渡す
7. `/retro` スキルで振り返りを行う
