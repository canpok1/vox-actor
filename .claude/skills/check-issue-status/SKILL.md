---
name: check-issue-status
description: GitHub Issueの対応状況を確認するスキル。Issue自体のopen/closed状態と同一Issueに対する既存PRの有無を調べ、対応状況を判定して返す。状態の判定のみ行い、Issueクローズなどの副作用は発生させない。
agent: general-purpose
allowed-tools: Bash(gh issue view *), Bash(gh pr list *)
argument-hint: "<issue-number>"
---

Issue $ARGUMENTS の対応状況を確認します。

## 手順

1. Issue自体の状態を取得する
  - `gh issue view $ARGUMENTS --json state --jq .state` で `OPEN` / `CLOSED` を確認する
  - `CLOSED` の場合: 対応状況を `CLOSED` として報告し、以降の手順はスキップする

2. 同一Issueに対する既存PRを検索する
  - ブランチ名検索（第一手段）: `gh pr list --repo {owner}/{repo} --head "worktree-issue-$ARGUMENTS" --state all`
  - テキスト検索（フォールバック）: `gh pr list --repo {owner}/{repo} --search "#$ARGUMENTS" --state all`
  - 両方の結果を合わせて判断する

3. 以下の優先順位で対応状況を判定する

| 対応状況 | 判定条件 | 追加情報 |
|---|---|---|
| `ALREADY_DONE` | マージ済みPRが存在する | PR番号・PR URL |
| `IN_PROGRESS` | open状態のPRが存在する | PR番号（複数あれば最新のもの） |
| `PREVIOUSLY_ABANDONED` | closed（マージされていない）PRのみ存在する | なし |
| `NOT_STARTED` | 既存PRなし | なし |

  - 判定に使うコマンド例:
    - マージ済みPR情報: `gh pr list --repo {owner}/{repo} --search "#$ARGUMENTS" --state merged --json number,url --jq '.[0]'`
    - open PR番号: `gh pr list --repo {owner}/{repo} --head "worktree-issue-$ARGUMENTS" --state open --json number --jq '.[0].number'`

## 出力形式

呼び出し元に以下の情報を報告する。

- **対応状況**: `CLOSED` / `ALREADY_DONE` / `IN_PROGRESS` / `PREVIOUSLY_ABANDONED` / `NOT_STARTED` のいずれか
- **PR番号**: `ALREADY_DONE` / `IN_PROGRESS` の場合のみ
- **PR URL**: `ALREADY_DONE` の場合のみ

## 禁止事項

- このスキルは**チェックのみ**を行う。Issueのクローズ、コメント投稿、PR操作などの副作用は一切発生させないこと
- 副作用が必要な処理（例: `ALREADY_DONE` 時のIssueクローズ）は呼び出し元の責務とする
