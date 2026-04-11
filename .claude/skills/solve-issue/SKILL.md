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

1. `/check-issue-status` でIssueの対応状況を確認する
  - `CLOSED`: 「Issue #$ARGUMENTS は既にクローズ済みです」と報告して処理を終了する
  - `ALREADY_DONE`: 以下の手順でIssueをクローズし、ステップ8（振り返り）のみ実施して完了する
    - `gh issue close $ARGUMENTS --repo {owner}/{repo} --comment "対応済みPR #{PR番号} が既にマージされているためクローズします。\n\nPR: {PR URL}"`
  - `IN_PROGRESS`: 新しい実装・PR作成はスキップし、既存PR番号を使ってステップ7（`/autofix-pr`）から再開する
  - `PREVIOUSLY_ABANDONED` / `NOT_STARTED`: 次のステップに進む
2. Issue の内容を把握する
3. 実装する
4. `/simplify` で変更コードの再利用性・品質・効率の観点からレビュー・改善を行う
5. 実装内容をレビューする
  - 指摘があれば内容を精査して必要に応じて修正する
6. `/commit-push-pr` でPRを作成する
7. `/autofix-pr` でCI待機・レビュー対応・マージを行う
8. `/retro` で振り返りを行う
