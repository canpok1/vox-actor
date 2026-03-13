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

検索結果に複数のPRが含まれる場合、以下の優先順位で判断する:

1. **merged状態のPRが存在する場合**: 既に対応済みのため、処理をスキップして完了する
2. **open状態のPRが存在する場合**: 新しいPRを作成せず、既存PRに対してレビュー対応やマージなどの後続処理を継続する
   - 複数のopen PRがある場合は、最新のものを対象とする
   - PR番号の取得例: `gh pr list --repo {owner}/{repo} --search "issue-{番号}" --state open --json number --jq '.[0].number'`
3. **closed状態（マージされずにクローズ）のPRのみ存在する場合**: 既存PRなしとして扱い、新規PR作成に進む
