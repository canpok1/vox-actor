# Worktreeブランチ運用ルール

## ブランチ命名規則

- worktreeで作業する場合、ブランチ名は `worktree-issue-{Issue番号}` とする
- worktreeブランチで作業している場合、別途 `feature/*` ブランチを作成してはならない
- 1つのIssueに対して作成するブランチは1つのみとする

## PR作成前の重複チェック（必須）

PR作成前に、以下のコマンドで同一Issueに対する既存PRを確認すること:

```bash
gh pr list --repo {owner}/{repo} --search "issue-{番号}" --state all
```

以下のいずれかに該当する場合、**新しいPRを作成してはならない**:

- 同一Issueに対するPRが既にopen状態で存在する
- 同一Issueに対するPRが既にmerged状態で存在する

該当するPRが見つかった場合は、ユーザーに報告して指示を仰ぐこと。
